package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/den/internal/auth"
	"github.com/den/internal/database"
	"github.com/den/internal/dns"
	"github.com/den/internal/models"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	auth *auth.Service
	db   *database.DB
	dns  *dns.Service
}

func New(authService *auth.Service, db *database.DB) *Handler {
	return &Handler{
		auth: authService,
		db:   db,
		dns:  dns.NewService(),
	}
}
func (h *Handler) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		user, err := h.auth.GetUserBySession(sessionID)
		if err != nil {
			c.SetCookie("session", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}

func (h *Handler) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		u := user.(*models.User)
		if !u.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "admin required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (h *Handler) RequireNodeAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}
		var nodeID int
		err := h.db.QueryRow("SELECT id FROM nodes WHERE token = $1", token).Scan(&nodeID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Set("node_id", nodeID)
		c.Next()
	}
}
func (h *Handler) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "home.html", gin.H{
		"title": "den - a cozy pubnix",
	})
}

func (h *Handler) LoginPage(c *gin.Context) {
	state := generateState()
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)
	
	authURL := h.auth.GetAuthURL(state)
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title":    "login to den",
		"auth_url": authURL,
	})
}

func (h *Handler) SlackAuth(c *gin.Context) {
	state := generateState()
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)
	
	authURL := h.auth.GetAuthURL(state)
	c.Redirect(http.StatusFound, authURL)
}

func (h *Handler) SlackCallback(c *gin.Context) {
	storedState, err := c.Cookie("oauth_state")
	if err != nil || storedState != c.Query("state") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	user, err := h.auth.HandleCallback(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sessionID, err := h.auth.CreateSession(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}
	c.SetCookie("session", sessionID, 3600*24*7, "/", "", false, true)
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)

	c.Redirect(http.StatusFound, "/user/dashboard")
}
func (h *Handler) UserDashboard(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	var container *models.Container
	if user.ContainerID != nil {
		container = &models.Container{}
		err := h.db.QueryRow(`
			SELECT id, user_id, node_id, name, status, ip_address, ssh_port,
				   memory_mb, cpu_cores, storage_gb, created_at, updated_at
			FROM containers WHERE id = $1
		`, *user.ContainerID).Scan(
			&container.ID, &container.UserID, &container.NodeID, &container.Name,
			&container.Status, &container.IPAddress, &container.SSHPort,
			&container.MemoryMB, &container.CPUCores, &container.StorageGB,
			&container.CreatedAt, &container.UpdatedAt,
		)
		if err != nil {
			container = nil
		}
	}
	rows, err := h.db.Query(`
		SELECT id, subdomain, target_port, is_active, created_at
		FROM subdomains WHERE user_id = $1
		ORDER BY created_at DESC
	`, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var subdomains []models.Subdomain
	for rows.Next() {
		var subdomain models.Subdomain
		err := rows.Scan(&subdomain.ID, &subdomain.Subdomain, &subdomain.TargetPort,
			&subdomain.IsActive, &subdomain.CreatedAt)
		if err != nil {
			continue
		}
		subdomain.UserID = user.ID
		subdomains = append(subdomains, subdomain)
	}

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title":      "Dashboard",
		"user":       user,
		"container":  container,
		"subdomains": subdomains,
	})
}

func (h *Handler) ContainerStatus(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	
	if user.ContainerID == nil {
		c.JSON(http.StatusOK, gin.H{"status": "none"})
		return
	}

	var container models.Container
	err := h.db.QueryRow(`
		SELECT id, status, ip_address, ssh_port, memory_mb, cpu_cores, storage_gb
		FROM containers WHERE id = $1
	`, *user.ContainerID).Scan(
		&container.ID, &container.Status, &container.IPAddress,
		&container.SSHPort, &container.MemoryMB, &container.CPUCores, &container.StorageGB,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, container)
}

func (h *Handler) CreateContainer(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	
	if user.ContainerID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container already exists"})
		return
	}
	var nodeID int
	var nodeHostname string
	err := h.db.QueryRow(`
		SELECT id, hostname FROM nodes 
		WHERE is_online = true 
		ORDER BY id LIMIT 1
	`).Scan(&nodeID, &nodeHostname)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no available nodes"})
		return
	}
	containerID := fmt.Sprintf("den-%s", user.Username)

	// Create container record
	_, err = h.db.Exec(`
		INSERT INTO containers (id, user_id, node_id, name, status, memory_mb, cpu_cores, storage_gb)
		VALUES ($1, $2, $3, $4, 'creating', 4096, 4, 15)
	`, containerID, user.ID, nodeID, containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create container record"})
		return
	}
	_, err = h.db.Exec(`
		UPDATE users SET container_id = $1, updated_at = NOW() WHERE id = $2
	`, containerID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}
	// todo: do this stuff (make lxc)
	c.JSON(http.StatusOK, gin.H{
		"message":      "Container creation started",
		"container_id": containerID,
	})
}

func (h *Handler) SubdomainManagement(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	
	rows, err := h.db.Query(`
		SELECT id, subdomain, target_port, is_active, created_at
		FROM subdomains WHERE user_id = $1
		ORDER BY created_at DESC
	`, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var subdomains []models.Subdomain
	for rows.Next() {
		var subdomain models.Subdomain
		err := rows.Scan(&subdomain.ID, &subdomain.Subdomain, &subdomain.TargetPort,
			&subdomain.IsActive, &subdomain.CreatedAt)
		if err != nil {
			continue
		}
		subdomain.UserID = user.ID
		subdomains = append(subdomains, subdomain)
	}

	c.JSON(http.StatusOK, gin.H{"subdomains": subdomains})
}

func (h *Handler) CreateSubdomain(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	
	var req struct {
		Subdomain  string `json:"subdomain" binding:"required"`
		TargetPort int    `json:"target_port" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.dns.ValidateSubdomain(req.Subdomain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.dns.ValidatePort(req.TargetPort); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var exists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM subdomains WHERE subdomain = $1)", req.Subdomain).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "subdomain already taken"})
		return
	}
	var containerIP string
	if user.ContainerID != nil {
		err = h.db.QueryRow("SELECT ip_address FROM containers WHERE id = $1", *user.ContainerID).Scan(&containerIP)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "container not ready"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no container available"})
		return
	}
	var subdomainID int
	err = h.db.QueryRow(`
		INSERT INTO subdomains (user_id, subdomain, target_port, is_active)
		VALUES ($1, $2, $3, true)
		RETURNING id
	`, user.ID, req.Subdomain, req.TargetPort).Scan(&subdomainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subdomain"})
		return
	}
	if err := h.dns.CreateDNSRecord(req.Subdomain, containerIP, req.TargetPort); err != nil {
		h.db.Exec("DELETE FROM subdomains WHERE id = $1", subdomainID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create DNS record"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Subdomain created successfully",
		"subdomain":  req.Subdomain + ".hack.kim",
		"target_port": req.TargetPort,
	})
}

func (h *Handler) DeleteSubdomain(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	subdomainID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subdomain ID"})
		return
	}
	var subdomain string
	err = h.db.QueryRow(`
		SELECT subdomain FROM subdomains 
		WHERE id = $1 AND user_id = $2
	`, subdomainID, user.ID).Scan(&subdomain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subdomain not found"})
		return
	}
	_, err = h.db.Exec("DELETE FROM subdomains WHERE id = $1 AND user_id = $2", subdomainID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	h.dns.DeleteDNSRecord(subdomain)

	c.JSON(http.StatusOK, gin.H{"message": "Subdomain deleted successfully"})
}

func (h *Handler) SSHSetup(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	
	c.HTML(http.StatusOK, "ssh_setup.html", gin.H{
		"title": "SSH Setup",
		"user":  user,
	})
}

func (h *Handler) ConfigureSSH(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	
	var req struct {
		Method    string `json:"method" binding:"required"`
		Password  string `json:"password"`
		PublicKey string `json:"public_key"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Method == "password" && req.Password != "" {
		if err := h.auth.SetSSHPassword(user.ID, req.Password); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set password"})
			return
		}
	} else if req.Method == "key" && req.PublicKey != "" {
		if err := h.auth.SetSSHPublicKey(user.ID, req.PublicKey); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set public key"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "SSH configuration updated"})
}
func (h *Handler) AdminDashboard(c *gin.Context) {
	var userCount, nodeCount, containerCount int
	h.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	h.db.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&nodeCount)
	h.db.QueryRow("SELECT COUNT(*) FROM containers").Scan(&containerCount)

	c.HTML(http.StatusOK, "admin_dashboard.html", gin.H{
		"title":          "Admin Dashboard",
		"user_count":     userCount,
		"node_count":     nodeCount,
		"container_count": containerCount,
	})
}

func (h *Handler) NodeManagement(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, name, hostname, max_memory_mb, max_cpu_cores, max_storage_gb,
			   is_online, last_seen, created_at
		FROM nodes ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var nodes []models.Node
	for rows.Next() {
		var node models.Node
		err := rows.Scan(&node.ID, &node.Name, &node.Hostname, &node.MaxMemoryMB,
			&node.MaxCPUCores, &node.MaxStorageGB, &node.IsOnline, &node.LastSeen, &node.CreatedAt)
		if err != nil {
			continue
		}
		nodes = append(nodes, node)
	}

	c.JSON(http.StatusOK, gin.H{"nodes": nodes})
}

func (h *Handler) CreateNode(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Hostname     string `json:"hostname" binding:"required"`
		MaxMemoryMB  int    `json:"max_memory_mb"`
		MaxCPUCores  int    `json:"max_cpu_cores"`
		MaxStorageGB int    `json:"max_storage_gb"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.MaxMemoryMB == 0 {
		req.MaxMemoryMB = 4096
	}
	if req.MaxCPUCores == 0 {
		req.MaxCPUCores = 4
	}
	if req.MaxStorageGB == 0 {
		req.MaxStorageGB = 15
	}
	token := generateNodeToken()
	var nodeID int
	err := h.db.QueryRow(`
		INSERT INTO nodes (name, hostname, token, max_memory_mb, max_cpu_cores, max_storage_gb)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, req.Name, req.Hostname, token, req.MaxMemoryMB, req.MaxCPUCores, req.MaxStorageGB).Scan(&nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create node"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Node created successfully",
		"node_id": nodeID,
		"token":   token,
	})
}

func (h *Handler) GenerateNodeToken(c *gin.Context) {
	nodeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node ID"})
		return
	}

	token := generateNodeToken()
	
	_, err = h.db.Exec("UPDATE nodes SET token = $1, updated_at = NOW() WHERE id = $2", token, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func (h *Handler) DeleteNode(c *gin.Context) {
	nodeID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node ID"})
		return
	}

	_, err = h.db.Exec("DELETE FROM nodes WHERE id = $1", nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Node deleted successfully"})
}

func (h *Handler) UserManagement(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT id, username, email, display_name, is_admin, container_id, created_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName,
			&user.IsAdmin, &user.ContainerID, &user.CreatedAt)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *Handler) DeleteUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	_, err = h.db.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func (h *Handler) APICreateContainer(c *gin.Context) {
	// implement this, implement that, etc.
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

func (h *Handler) APIGetContainer(c *gin.Context) {
	// blah blah blah
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

func (h *Handler) APIDeleteContainer(c *gin.Context) {
	// im lazy, sue me
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

func (h *Handler) APIUpdateContainerStatus(c *gin.Context) {
	// implement this you buffoon
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
func generateState() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateNodeToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
