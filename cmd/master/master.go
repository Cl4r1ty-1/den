package master

import (
	"context"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"html/template"
	"syscall"
	"time"
	crand "crypto/rand"
	"encoding/hex"
	"math"
    "strings"

	"github.com/den/internal/auth"
	"github.com/den/internal/database"
	"github.com/den/internal/dns"
	"github.com/den/internal/handlers"
	"github.com/den/internal/storage"
	"github.com/den/internal/ssh"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/lib/pq"
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
    go func() {
        for {
            if err := runJobOnce(db); err != nil {
                log.Printf("job worker error: %v", err)
                time.Sleep(2 * time.Second)
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

func runJobOnce(db *database.DB) error {
    tx, err := db.Begin()
    if err != nil { return err }
    defer tx.Rollback()
    var id int
    var jtype string
    var payloadStr string
    err = tx.QueryRow(`SELECT id, type, payload FROM jobs WHERE status='queued' AND run_after <= NOW() ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&id, &jtype, &payloadStr)
    if err != nil {
        if err.Error() == "sql: no rows in result set" { time.Sleep(1 * time.Second); return nil }
        return err
    }
    if _, err := tx.Exec(`UPDATE jobs SET status='running', attempts=attempts+1, updated_at=NOW() WHERE id=$1`, id); err != nil { return err }
    if err := tx.Commit(); err != nil { return err }

    switch jtype {
    case "export_container":
        return handleExportJob(db, id, []byte(payloadStr))
    case "create_container":
        return handleCreateContainerJob(db, id, []byte(payloadStr))
    case "delete_container":
        return handleDeleteContainerJob(db, id, []byte(payloadStr))
    default:
        _, _ = db.Exec(`UPDATE jobs SET status='failed', error=$2, updated_at=NOW() WHERE id=$1`, id, "unknown job type")
        return nil
    }
}

func handleExportJob(db *database.DB, jobID int, payload []byte) error {
    var p struct {
        ExportID    int    `json:"export_id"`
        UserID      int    `json:"user_id"`
        ContainerID string `json:"container_id"`
        NodeHostname string `json:"node_hostname"`
        ObjectKey   string `json:"object_key"`
        TTLDays     int    `json:"ttl_days"`
    }
    if err := json.Unmarshal(payload, &p); err != nil { return err }

    r2, err := storage.NewR2ClientFromEnv()
    if err != nil { _, _ = db.Exec(`UPDATE exports SET status='failed', error=$2 WHERE id=$1`, p.ExportID, "storage not configured"); return nil }
    putURL, err := r2.PresignedPut(context.Background(), p.ObjectKey, 2*time.Hour)
    if err != nil { _, _ = db.Exec(`UPDATE exports SET status='failed', error=$2 WHERE id=$1`, p.ExportID, "presign put failed"); return nil }
    _, _ = db.Exec(`UPDATE exports SET status='uploading', updated_at=NOW() WHERE id=$1`, p.ExportID)

    slaveURL := fmt.Sprintf("http://%s:8081/api/export", p.NodeHostname)
    body, _ := json.Marshal(map[string]string{"container_id": p.ContainerID, "put_url": putURL})
    resp, err := http.Post(slaveURL, "application/json", bytes.NewBuffer(body))
    if err != nil { _, _ = db.Exec(`UPDATE exports SET status='failed', error=$2 WHERE id=$1`, p.ExportID, err.Error()); return finalizeJob(db, jobID, false, err.Error(), nil) }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        _, _ = db.Exec(`UPDATE exports SET status='failed', error=$2 WHERE id=$1`, p.ExportID, string(b))
        return finalizeJob(db, jobID, false, string(b), nil)
    }
    getURL, err := r2.PresignedGet(context.Background(), p.ObjectKey, time.Duration(p.TTLDays)*24*time.Hour)
    if err != nil { _, _ = db.Exec(`UPDATE exports SET status='failed', error=$2 WHERE id=$1`, p.ExportID, "presign get failed"); return finalizeJob(db, jobID, false, "presign get failed", nil) }
    _, _ = db.Exec(`UPDATE exports SET status='complete', updated_at=NOW() WHERE id=$1`, p.ExportID)
    res := map[string]string{"download_url": getURL}
    rb, _ := json.Marshal(res)
    return finalizeJob(db, jobID, true, "", rb)
}

func finalizeJob(db *database.DB, jobID int, success bool, errMsg string, result []byte) error {
	if success {
		_, err := db.Exec(`UPDATE jobs SET status='success', result=$2, updated_at=NOW() WHERE id=$1`, jobID, string(result))
		return err
	}
	var attempts, maxAttempts int
	if err := db.QueryRow(`SELECT attempts, COALESCE(max_attempts, 0) FROM jobs WHERE id=$1`, jobID).Scan(&attempts, &maxAttempts); err == nil {
		if maxAttempts == 0 { maxAttempts = 3 }
		if attempts < maxAttempts {
			backoff := int(math.Pow(2, float64(attempts-1)) * 5)
			if backoff > 300 { backoff = 300 }
			delay := time.Duration(backoff) * time.Second
			_, _ = db.Exec(`UPDATE jobs SET status='queued', run_after=NOW()+($2::interval), error=$3, updated_at=NOW() WHERE id=$1`, jobID, fmt.Sprintf("%d seconds", int(delay.Seconds())), errMsg)
			return nil
		}
	}
	_, err := db.Exec(`UPDATE jobs SET status='failed', error=$2, updated_at=NOW() WHERE id=$1`, jobID, errMsg)
	return err
}

func handleCreateContainerJob(db *database.DB, jobID int, payload []byte) error {
    var p struct {
        UserID   int    `json:"user_id"`
        Username string `json:"username"`
    }
    if err := json.Unmarshal(payload, &p); err != nil { return finalizeJob(db, jobID, false, "invalid payload", nil) }

    var nodeID int
    var nodeHostname string
    if err := db.QueryRow(`SELECT id, hostname FROM nodes WHERE is_online = true ORDER BY id LIMIT 1`).Scan(&nodeID, &nodeHostname); err != nil {
        return finalizeJob(db, jobID, false, "no available nodes", nil)
    }
    slaveURL := fmt.Sprintf("http://%s:8081", nodeHostname)

    reqBody, _ := json.Marshal(map[string]interface{}{ "user_id": p.UserID, "username": p.Username })
    resp, err := http.Post(slaveURL+"/api/containers", "application/json", bytes.NewBuffer(reqBody))
    if err != nil {
        return finalizeJob(db, jobID, false, err.Error(), nil)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return finalizeJob(db, jobID, false, string(b), nil)
    }
    var containerInfo map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&containerInfo); err != nil {
        return finalizeJob(db, jobID, false, "decode response failed", nil)
    }

    containerID, _ := containerInfo["ID"].(string)
    name, _ := containerInfo["Name"].(string)
    ipAny, _ := containerInfo["IP"]
    sshPortAny, _ := containerInfo["SSHPort"]
    var ip *string
    if s, ok := ipAny.(string); ok { ip = &s }
    sshPort := 0
    switch v := sshPortAny.(type) {
    case float64:
        sshPort = int(v)
    case int:
        sshPort = v
    }

    tokenBytes := make([]byte, 24)
    if _, err := crand.Read(tokenBytes); err != nil { return finalizeJob(db, jobID, false, "token gen failed", nil) }
    containerToken := hex.EncodeToString(tokenBytes)

    if _, err := db.Exec(`INSERT INTO containers (id, user_id, node_id, name, status, ip_address, ssh_port, memory_mb, cpu_cores, storage_gb, allocated_ports, container_token) VALUES ($1,$2,$3,$4,'RUNNING',$5,$6,4096,4,15,$7,$8)`,
        containerID, p.UserID, nodeID, name, ip, sshPort, pq.Array([]int{}), containerToken); err != nil {
        return finalizeJob(db, jobID, false, "db insert failed", nil)
    }
    if _, err := db.Exec(`UPDATE users SET container_id = $1, updated_at = NOW() WHERE id = $2`, containerID, p.UserID); err != nil {
        return finalizeJob(db, jobID, false, "db update user failed", nil)
    }

    {
        body, _ := json.Marshal(map[string]string{"container_id": containerID, "token": containerToken, "username": p.Username})
        resp, err := http.Post(slaveURL+"/api/cli/token", "application/json", bytes.NewBuffer(body))
        if err != nil {
            log.Printf("post /api/cli/token failed for %s: %v", containerID, err)
        } else {
            defer resp.Body.Close()
            if resp.StatusCode < 200 || resp.StatusCode >= 300 {
                b, _ := io.ReadAll(resp.Body)
                log.Printf("/api/cli/token non-200 for %s: %d %s", containerID, resp.StatusCode, strings.TrimSpace(string(b)))
            }
        }
    }
    {
        installBody, _ := json.Marshal(map[string]string{"container_id": containerID})
        resp, err := http.Post(slaveURL+"/api/cli/install", "application/json", bytes.NewBuffer(installBody))
        if err != nil {
            log.Printf("post /api/cli/install failed for %s: %v", containerID, err)
        } else {
            defer resp.Body.Close()
            if resp.StatusCode < 200 || resp.StatusCode >= 300 {
                b, _ := io.ReadAll(resp.Body)
                log.Printf("/api/cli/install non-200 for %s: %d %s", containerID, resp.StatusCode, strings.TrimSpace(string(b)))
            }
        }
    }

    res := map[string]interface{}{ "container_id": containerID, "ip_address": ip, "ssh_port": sshPort, "container_token": containerToken }
    rb, _ := json.Marshal(res)
    return finalizeJob(db, jobID, true, "", rb)
}

func handleDeleteContainerJob(db *database.DB, jobID int, payload []byte) error {
    var p struct {
        UserID      int    `json:"user_id"`
        ContainerID string `json:"container_id"`
        NodeHostname string `json:"node_hostname"`
        Username    string `json:"username"`
    }
    if err := json.Unmarshal(payload, &p); err != nil { return finalizeJob(db, jobID, false, "invalid payload", nil) }

    slaveURL := fmt.Sprintf("http://%s:8081/api/containers/%s", p.NodeHostname, p.ContainerID)
    req, _ := http.NewRequest(http.MethodDelete, slaveURL, nil)
    client := &http.Client{Timeout: 2 * time.Minute}
    resp, err := client.Do(req)
    if err != nil {
        return finalizeJob(db, jobID, false, err.Error(), nil)
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return finalizeJob(db, jobID, false, string(b), nil)
    }

    if rows, qerr := db.Query(`SELECT subdomain, subdomain_type FROM subdomains WHERE user_id = $1`, p.UserID); qerr == nil {
        defer rows.Close()
        for rows.Next() {
            var sub, subType string
            if err := rows.Scan(&sub, &subType); err == nil {
                _ = dns.NewService().DeleteDNSRecord(sub, p.Username, subType)
            }
        }
        _, _ = db.Exec("DELETE FROM subdomains WHERE user_id = $1", p.UserID)
    }

    _, _ = db.Exec("DELETE FROM containers WHERE id = $1", p.ContainerID)
    _, _ = db.Exec("UPDATE users SET container_id = NULL, updated_at = NOW() WHERE id = $1", p.UserID)

    return finalizeJob(db, jobID, true, "", nil)
}

func setupRouter(authService *auth.Service, db *database.DB) *gin.Engine {
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()
    // i have no idea how to use prometheus
    requestCounter := prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "den_http_requests_total", Help: "HTTP requests"},
        []string{"method","path","status"},
    )
    durationHist := prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "den_http_request_duration_seconds", Help: "HTTP duration", Buckets: prometheus.DefBuckets},
        []string{"method","path"},
    )
    prometheus.MustRegister(requestCounter, durationHist)
    nodeContainers := prometheus.NewGauge(prometheus.GaugeOpts{Name: "den_master_total_containers", Help: "Total containers across cluster"})
    prometheus.MustRegister(nodeContainers)
    go func(){
        for {
            var total int
            db.QueryRow("SELECT COUNT(*) FROM containers").Scan(&total)
            nodeContainers.Set(float64(total))
            time.Sleep(15 * time.Second)
        }
    }()
    go func(){
        for {
            _, _ = db.Exec("UPDATE nodes SET is_online=false WHERE last_seen < NOW() - INTERVAL '90 seconds'")
            time.Sleep(30 * time.Second)
        }
    }()
    r.GET("/metrics", gin.WrapH(promhttp.Handler()))
    r.GET("/healthz", func(c *gin.Context){ c.JSON(http.StatusOK, gin.H{"ok": true}) })
    r.Use(func(c *gin.Context) {
        b := make([]byte, 8)
        if _, err := crand.Read(b); err == nil {
            rid := hex.EncodeToString(b)
            c.Set("request_id", rid)
            c.Writer.Header().Set("X-Request-ID", rid)
        }
        start := time.Now()
        c.Next()
        status := c.Writer.Status()
        rid, _ := c.Get("request_id")
        log.Printf("rid=%v %s %s status=%d duration=%s", rid, c.Request.Method, c.FullPath(), status, time.Since(start))
        path := c.FullPath()
        if path == "" { path = c.Request.URL.Path }
        requestCounter.WithLabelValues(c.Request.Method, path, fmt.Sprintf("%d", status)).Inc()
        durationHist.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
    })
    r.Use(gin.Recovery())

    r.SetFuncMap(template.FuncMap{
        "toJson": func(v interface{}) template.JS {
            b, _ := json.Marshal(v)
            return template.JS(string(b))
        },
    })
    r.LoadHTMLGlob("web/templates/*")
	r.Static("/static", "./web/static")
	r.Static("/assets", "./webapp/dist/assets")
	r.StaticFile("/vite.svg", "./webapp/dist/vite.svg")
    r.Static("/downloads/cli", "./cli/dist")

    h := handlers.New(authService, db)

	r.GET("/", h.Home)
	r.GET("/login", h.LoginPage)
	r.GET("/legal", h.LegalPage)
	r.GET("/logout", h.Logout)
	r.GET("/auth/github", h.GitHubAuth)
	r.GET("/auth/callback", h.GitHubCallback)

	userGroup := r.Group("/user")
	userGroup.Use(h.RequireAuth())
	{
		userGroup.GET("/aup", h.AUPPage)
		userGroup.POST("/aup/accept", h.AUPAccept)
		userGroup.GET("/dashboard", h.UserDashboard)
		userGroup.GET("/container", h.ContainerStatus)
		userGroup.GET("/container/stats", h.ContainerStats)
		userGroup.GET("/container/shell", h.GetContainerShell)
		userGroup.POST("/container/shell", h.SetContainerShell)
		userGroup.POST("/container/start", h.ContainerStart)
		userGroup.POST("/container/stop", h.ContainerStop)
		userGroup.POST("/container/restart", h.ContainerRestart)
		userGroup.GET("/container/token", h.ContainerToken)
		userGroup.POST("/container/token/rotate", h.RotateContainerToken)
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
		adminGroup.POST("/users/:id/rotate-token", h.AdminRotateUserContainerToken)
		adminGroup.POST("/users/:id/reinstall-cli", h.AdminReinstallUserCLI)
		adminGroup.POST("/users/:id/approve", h.AdminApproveUser)
		adminGroup.POST("/users/:id/reject", h.AdminRejectUser)
		adminGroup.POST("/users/:id/export", h.AdminExportUserContainer)
		adminGroup.GET("/jobs", h.AdminListJobs)
		adminGroup.GET("/jobs/:id", h.AdminGetJob)
	}

    cliGroup := r.Group("/cli")
    cliGroup.Use(h.RequireCLIAuth())
    {
        cliGroup.GET("/me", h.CLIMe)
        cliGroup.GET("/container/stats", h.CLIContainerStats)
        cliGroup.POST("/container/:action", h.CLIContainerControl)
        cliGroup.GET("/container/ports", h.CLIContainerPorts)
        cliGroup.POST("/container/ports/new", h.CLIContainerNewPort)
    }

	apiGroup := r.Group("/api")
	{
		apiGroup.POST("/nodes/register", h.APIRegisterNode)
		apiGroup.POST("/nodes/heartbeat", h.APINodeHeartbeat)
		apiGroup.POST("/containers/:id/status", h.APIUpdateContainerStatus)
	}
	
	apiProtected := r.Group("/api")
	apiProtected.Use(h.RequireNodeAuth())
	{
		apiProtected.POST("/containers", h.APICreateContainer)
		apiProtected.GET("/containers/:id", h.APIGetContainer)
		apiProtected.DELETE("/containers/:id", h.APIDeleteContainer)
	}

	r.NoRoute(h.NotFound)

	return r
}
