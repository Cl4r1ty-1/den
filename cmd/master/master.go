package master

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"html/template"
	"syscall"
	"time"
    "crypto/rand"
    "encoding/hex"

	"github.com/den/internal/auth"
	"github.com/den/internal/database"
	"github.com/den/internal/dns"
	"github.com/den/internal/handlers"
    "github.com/den/internal/storage"
	"github.com/den/internal/ssh"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func Run() error {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}

	db, err := database.Initialize()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()
	if err := database.RunMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	dnsService := dns.NewService()
	go func() {
		time.Sleep(2 * time.Second)
		if err := dnsService.RebuildRoutesFromDatabase(db.DB); err != nil {
			log.Printf("failed to sync caddy routes: %v", err)
		}
	}()

	authService := auth.NewService(db)
	
	sshGateway := ssh.NewGateway(db)
	go func() {
		if err := sshGateway.Start(); err != nil {
			log.Printf("ssh gateway error: %v", err)
		}
	}()
    if _, err := storage.NewR2ClientFromEnv(); err != nil {
        log.Printf("warning: R2 not configured: %v", err)
    }
    router := setupRouter(authService, db)
    go func() {
        ticker := time.NewTicker(30 * time.Minute)
        defer ticker.Stop()
        for range ticker.C {
            if err := cleanupExpiredExports(db); err != nil {
                log.Printf("export cleanup error: %v", err)
            }
        }
    }()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		log.Println("starting master on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("server exited")
	return nil
}

func cleanupExpiredExports(db *database.DB) error {
    r2, err := storage.NewR2ClientFromEnv()
    if err != nil { return nil }
    rows, err := db.Query(`SELECT id, object_key FROM exports WHERE status IN ('complete') AND expires_at < NOW()`)
    if err != nil { return err }
    defer rows.Close()
    for rows.Next() {
        var id int; var key string
        if err := rows.Scan(&id, &key); err != nil { continue }
        _ = r2.DeleteObject(context.Background(), key)
        _, _ = db.Exec(`UPDATE exports SET status='expired', updated_at=NOW() WHERE id=$1`, id)
    }
    return nil
}

func setupRouter(authService *auth.Service, db *database.DB) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
    r := gin.New()
    r.Use(func(c *gin.Context) {
        b := make([]byte, 8)
        if _, err := rand.Read(b); err == nil {
            rid := hex.EncodeToString(b)
            c.Set("request_id", rid)
            c.Writer.Header().Set("X-Request-ID", rid)
        }
        start := time.Now()
        c.Next()
        status := c.Writer.Status()
        rid, _ := c.Get("request_id")
        log.Printf("rid=%v %s %s status=%d duration=%s", rid, c.Request.Method, c.FullPath(), status, time.Since(start))
    })
    r.Use(gin.Recovery())

    r.SetFuncMap(template.FuncMap{
        "toJson": func(v interface{}) template.JS {
            b, _ := json.Marshal(v)
            return template.JS(string(b))
        },
    })
    r.LoadHTMLGlob("web/templates/*")
    r.Static("/assets", "./web/static/app/assets")
    r.GET("/", func(c *gin.Context) { c.File("web/static/app/index.html") })
    r.NoRoute(func(c *gin.Context) { c.File("web/static/app/index.html") })

    h := handlers.New(authService, db)

	r.GET("/", h.Home)
	r.GET("/login", h.LoginPage)
	r.GET("/auth/slack", h.SlackAuth)
	r.GET("/auth/callback", h.SlackCallback)

	userGroup := r.Group("/user")
	userGroup.Use(h.RequireAuth())
	{
		userGroup.GET("/aup", h.AUPPage)
		userGroup.GET("/aup/questions", h.AUPQuestions)
		userGroup.POST("/aup/accept", h.AUPAccept)
		userGroup.GET("/me", h.Me)
		userGroup.GET("/dashboard", h.UserDashboard)
		userGroup.GET("/container", h.ContainerStatus)
		userGroup.POST("/container/create", h.CreateContainer)
		userGroup.POST("/container/ports/new", h.GetNewPort)
		userGroup.GET("/subdomains", h.SubdomainManagement)
		userGroup.POST("/subdomains", h.CreateSubdomain)
		userGroup.DELETE("/subdomains/:id", h.DeleteSubdomain)
		userGroup.GET("/api/subdomains", h.GetUserSubdomains)
		userGroup.GET("/ssh-setup", h.SSHSetup)
		userGroup.POST("/ssh-setup", h.ConfigureSSH)
		userGroup.POST("/aup/validate", h.AUPValidate)
	}

	adminGroup := r.Group("/admin")
	adminGroup.Use(h.RequireAuth())
	adminGroup.Use(h.RequireAdmin())
	{
		adminGroup.GET("/", h.AdminDashboard)
		adminGroup.GET("/nodes", h.NodeManagement)
		adminGroup.POST("/nodes", h.CreateNode)
		adminGroup.GET("/nodes/:id/token", h.GenerateNodeToken)
		adminGroup.DELETE("/nodes/:id", h.DeleteNode)
		adminGroup.GET("/users", h.UserManagement)
		adminGroup.DELETE("/users/:id", h.DeleteUser)
		adminGroup.DELETE("/users/:id/container", h.AdminDeleteUserContainer)
        adminGroup.POST("/users/:id/export", h.AdminExportUserContainer)
	}

	apiGroup := r.Group("/api")
	{
		apiGroup.POST("/nodes/register", h.APIRegisterNode)
		apiGroup.POST("/nodes/heartbeat", h.APINodeHeartbeat)
	}
	
	apiProtected := r.Group("/api")
	apiProtected.Use(h.RequireNodeAuth())
	{
		apiProtected.POST("/containers", h.APICreateContainer)
		apiProtected.GET("/containers/:id", h.APIGetContainer)
		apiProtected.DELETE("/containers/:id", h.APIDeleteContainer)
		apiProtected.POST("/containers/:id/status", h.APIUpdateContainerStatus)
	}

	return r
}
