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
	"bytes"

	"github.com/den/internal/auth"
	"github.com/den/internal/database"
	"github.com/den/internal/dns"
	"github.com/den/internal/handlers"
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

    router := setupRouter(authService, db)
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        defer ticker.Stop()
        time.Sleep(5 * time.Second)
        for {
            if err := enforceAUPCompliance(db); err != nil {
                log.Printf("aup enforcement error: %v", err)
            }
            select {
            case <-ticker.C:
                continue
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

func setupRouter(authService *auth.Service, db *database.DB) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

    r.SetFuncMap(template.FuncMap{
        "toJson": func(v interface{}) template.JS {
            b, _ := json.Marshal(v)
            return template.JS(string(b))
        },
    })
    r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "./web/static")

	h := handlers.New(authService, db)

	r.GET("/", h.Home)
	r.GET("/login", h.LoginPage)
	r.GET("/auth/slack", h.SlackAuth)
	r.GET("/auth/callback", h.SlackCallback)

	userGroup := r.Group("/user")
	userGroup.Use(h.RequireAuth())
	{
		userGroup.GET("/aup", h.AUPPage)
		userGroup.POST("/aup/accept", h.AUPAccept)
		userGroup.GET("/dashboard", h.UserDashboard)
		userGroup.GET("/container", h.ContainerStatus)
		userGroup.POST("/container/create", h.CreateContainer)
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

func enforceAUPCompliance(db *database.DB) error {
    rows, err := db.DB.Query(`
        SELECT u.id, u.username, c.id AS container_id, n.hostname
        FROM users u
        JOIN containers c ON u.container_id = c.id
        JOIN nodes n ON c.node_id = n.id
        WHERE (u.agreed_to_tos = FALSE OR u.agreed_to_privacy = FALSE)
          AND c.status = 'running'
    `)
    if err != nil {
        return err
    }
    defer rows.Close()

    type item struct{ userID int; username, containerID, hostname string }
    var items []item
    for rows.Next() {
        var it item
        if err := rows.Scan(&it.userID, &it.username, &it.containerID, &it.hostname); err == nil {
            items = append(items, it)
        }
    }
    if len(items) == 0 {
        return nil
    }

    client := &http.Client{Timeout: 15 * time.Second}
    for _, it := range items {
        url := fmt.Sprintf("http://%s:8081/api/control/containers/%s", it.hostname, it.containerID)
        body := map[string]string{"action": "stop"}
        b, _ := json.Marshal(body)
        req, _ := http.NewRequest("POST", url, bytes.NewBuffer(b))
        req.Header.Set("Content-Type", "application/json")
        resp, err := client.Do(req)
        if err != nil {
            log.Printf("failed to stop container %s on %s: %v", it.containerID, it.hostname, err)
            continue
        }
        resp.Body.Close()
        if resp.StatusCode >= 400 {
            log.Printf("slave returned %d for %s", resp.StatusCode, it.containerID)
            continue
        }
        if _, err := db.DB.Exec(`UPDATE containers SET status = 'stopped', updated_at = NOW() WHERE id = $1`, it.containerID); err != nil {
            log.Printf("failed to update DB for %s: %v", it.containerID, err)
        } else {
            log.Printf("stopped container %s for user %s", it.containerID, it.username)
        }
    }
    return nil
}
