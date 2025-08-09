package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/den/internal/auth"
	"github.com/den/internal/database"
	"github.com/den/internal/dns"
	"github.com/den/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
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
				   memory_mb, cpu_cores, storage_gb, allocated_ports, created_at, updated_at
			FROM containers WHERE id = $1
		`, *user.ContainerID).Scan(
			&container.ID, &container.UserID, &container.NodeID, &container.Name,
			&container.Status, &container.IPAddress, &container.SSHPort,
			&container.MemoryMB, &container.CPUCores, &container.StorageGB,
			pq.Array(&container.AllocatedPorts), &container.CreatedAt, &container.UpdatedAt,
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
		SELECT id, status, ip_address, ssh_port, memory_mb, cpu_cores, storage_gb, allocated_ports
		FROM containers WHERE id = $1
	`, *user.ContainerID).Scan(
		&container.ID, &container.Status, &container.IPAddress,
		&container.SSHPort, &container.MemoryMB, &container.CPUCores, &container.StorageGB,
		pq.Array(&container.AllocatedPorts),
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
	slaveURL := fmt.Sprintf("http://%s:8081", nodeHostname)
	payload := map[string]interface{}{
		"user_id":   user.ID,
		"username":  user.Username,
	}
	
	data, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal request"})
		return
	}
	
	resp, err := http.Post(slaveURL+"/api/containers", "application/json", bytes.NewBuffer(data))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to communicate with slave node"})
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "slave node failed to create container"})
		return
	}
	
	var containerInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&containerInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode container info"})
		return
	}
	containerID := containerInfo["ID"].(string)
	var allocatedPorts []int
	if portsInterface, exists := containerInfo["AllocatedPorts"]; exists {
		if portsSlice, ok := portsInterface.([]interface{}); ok {
			for _, port := range portsSlice {
				if portFloat, ok := port.(float64); ok {
					allocatedPorts = append(allocatedPorts, int(portFloat))
				}
			}
		}
	}
	
	_, err = h.db.Exec(`
		INSERT INTO containers (id, user_id, node_id, name, status, ip_address, ssh_port, memory_mb, cpu_cores, storage_gb, allocated_ports)
		VALUES ($1, $2, $3, $4, 'running', $5, $6, $7, $8, $9, $10)
	`, containerID, user.ID, nodeID, containerInfo["Name"], 
		containerInfo["IP"], containerInfo["SSHPort"], 4096, 4, 15, pq.Array(allocatedPorts))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store container info"})
		return
	}
	_, err = h.db.Exec("UPDATE users SET container_id = $1 WHERE id = $2", containerID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":        "Container created successfully",
		"container_id":   containerID,
		"ip_address":     containerInfo["IP"],
		"allocated_ports": allocatedPorts,
		"ssh_port":     containerInfo["SSHPort"],
	})
}

func (h *Handler) SubdomainManagement(c *gin.Context) {
	if c.GetHeader("Accept") == "application/json" || c.Query("format") == "json" {
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
		return
	}
	c.HTML(http.StatusOK, "subdomains.html", gin.H{
		"title": "Subdomain Management",
	})
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
		SELECT id, name, hostname, public_hostname, max_memory_mb, max_cpu_cores, max_storage_gb,
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
		err := rows.Scan(&node.ID, &node.Name, &node.Hostname, &node.PublicHostname, &node.MaxMemoryMB,
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
		Name           string  `json:"name" binding:"required"`
		Hostname       string  `json:"hostname" binding:"required"`
		PublicHostname *string `json:"public_hostname"`
		MaxMemoryMB    int     `json:"max_memory_mb"`
		MaxCPUCores    int     `json:"max_cpu_cores"`
		MaxStorageGB   int     `json:"max_storage_gb"`
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
		INSERT INTO nodes (name, hostname, public_hostname, token, max_memory_mb, max_cpu_cores, max_storage_gb)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, req.Name, req.Hostname, req.PublicHostname, token, req.MaxMemoryMB, req.MaxCPUCores, req.MaxStorageGB).Scan(&nodeID)
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
	var req struct {
		UserID   int    `json:"user_id" binding:"required"`
		Username string `json:"username" binding:"required"`
		NodeID   int    `json:"node_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var nodeHostname string
	err := h.db.QueryRow("SELECT hostname FROM nodes WHERE id = $1", req.NodeID).Scan(&nodeHostname)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node ID"})
		return
	}
	slaveURL := fmt.Sprintf("http://%s:8081", nodeHostname)
	payload := map[string]interface{}{
		"user_id":   req.UserID,
		"username":  req.Username,
	}
	
	data, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal request"})
		return
	}
	
	resp, err := http.Post(slaveURL+"/api/containers", "application/json", bytes.NewBuffer(data))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to communicate with slave node"})
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "slave node failed to create container"})
		return
	}
	
	var containerInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&containerInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode container info"})
		return
	}
	containerID := containerInfo["ID"].(string)
	_, err = h.db.Exec(`
		INSERT INTO containers (id, user_id, node_id, name, status, ip_address, ssh_port, memory_mb, cpu_cores, storage_gb)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, containerID, req.UserID, req.NodeID, containerInfo["Name"], "running", 
		containerInfo["IP"], containerInfo["SSHPort"], 4096, 4, 15)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store container info"})
		return
	}
	_, err = h.db.Exec("UPDATE users SET container_id = $1 WHERE id = $2", containerID, req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}
	
	c.JSON(http.StatusOK, containerInfo)
}

func (h *Handler) APIGetContainer(c *gin.Context) {
	containerID := c.Param("id")
	
	var container models.Container
	err := h.db.QueryRow(`
		SELECT id, user_id, node_id, name, status, ip_address, ssh_port, memory_mb, cpu_cores, storage_gb, allocated_ports, created_at, updated_at
		FROM containers WHERE id = $1
	`, containerID).Scan(&container.ID, &container.UserID, &container.NodeID, &container.Name,
		&container.Status, &container.IPAddress, &container.SSHPort, &container.MemoryMB,
		&container.CPUCores, &container.StorageGB, pq.Array(&container.AllocatedPorts), &container.CreatedAt, &container.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}
	
	c.JSON(http.StatusOK, container)
}

func (h *Handler) TraefikConfig(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT s.subdomain, s.target_port, n.public_hostname, n.hostname
		FROM subdomains s
		JOIN users u ON s.user_id = u.id
		JOIN containers co ON u.container_id = co.id
		JOIN nodes n ON co.node_id = n.id
		WHERE s.is_active = true
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch subdomains"})
		return
	}
	defer rows.Close()

	config := map[string]interface{}{
		"http": map[string]interface{}{
			"routers":  map[string]interface{}{},
			"services": map[string]interface{}{},
		},
	}

	routers := config["http"].(map[string]interface{})["routers"].(map[string]interface{})
	services := config["http"].(map[string]interface{})["services"].(map[string]interface{})

	for rows.Next() {
		var subdomain, hostname string
		var publicHostname *string
		var targetPort int
		
		if err := rows.Scan(&subdomain, &targetPort, &publicHostname, &hostname); err != nil {
			continue
		}

		nodeHost := hostname
		if publicHostname != nil && *publicHostname != "" {
			nodeHost = *publicHostname
		}

		routerName := fmt.Sprintf("subdomain-%s", subdomain)
		serviceName := fmt.Sprintf("service-%s", subdomain)
		routers[routerName] = map[string]interface{}{
			"rule":         fmt.Sprintf("Host(`%s`)", subdomain),
			"service":      serviceName,
			"entrypoints":  []string{"websecure"},
			"tls": map[string]interface{}{
				"certresolver": "letsencrypt",
			},
		}
		services[serviceName] = map[string]interface{}{
			"loadBalancer": map[string]interface{}{
				"servers": []map[string]interface{}{
					{
						"url": fmt.Sprintf("http://%s:%d", nodeHost, targetPort),
					},
				},
			},
		}
	}

	c.JSON(http.StatusOK, config)
}
func (h *Handler) CreateSubdomain(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	
	if user.ContainerID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no container found"})
		return
	}

	var req struct {
		Subdomain  string `json:"subdomain" binding:"required"`
		TargetPort int    `json:"target_port" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !h.dns.IsValidSubdomain(req.Subdomain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subdomain format"})
		return
	}
	var existingID int
	err := h.db.QueryRow("SELECT id FROM subdomains WHERE subdomain = $1", req.Subdomain).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "subdomain already exists"})
		return
	}
	var container models.Container
	err = h.db.QueryRow(`
		SELECT allocated_ports FROM containers WHERE id = $1
	`, *user.ContainerID).Scan(pq.Array(&container.AllocatedPorts))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get container ports"})
		return
	}
	portFound := false
	for _, port := range container.AllocatedPorts {
		if port == req.TargetPort {
			portFound = true
			break
		}
	}
	if !portFound {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target port not allocated to your container"})
		return
	}
	_, err = h.db.Exec(`
		INSERT INTO subdomains (user_id, subdomain, target_port, is_active)
		VALUES ($1, $2, $3, true)
	`, user.ID, req.Subdomain, req.TargetPort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subdomain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "subdomain created successfully",
		"subdomain":   req.Subdomain,
		"target_port": req.TargetPort,
	})
}
func (h *Handler) DeleteSubdomain(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	subdomainID := c.Param("id")
	_, err := h.db.Exec(`
		DELETE FROM subdomains 
		WHERE id = $1 AND user_id = $2
	`, subdomainID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete subdomain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "subdomain deleted successfully"})
}
func (h *Handler) GetUserSubdomains(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	rows, err := h.db.Query(`
		SELECT id, subdomain, target_port, is_active, created_at
		FROM subdomains
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch subdomains"})
		return
	}
	defer rows.Close()

	var subdomains []models.Subdomain
	for rows.Next() {
		var subdomain models.Subdomain
		if err := rows.Scan(&subdomain.ID, &subdomain.Subdomain, &subdomain.TargetPort, 
			&subdomain.IsActive, &subdomain.CreatedAt); err != nil {
			continue
		}
		subdomains = append(subdomains, subdomain)
	}

	c.JSON(http.StatusOK, gin.H{"subdomains": subdomains})
}

func (h *Handler) APIDeleteContainer(c *gin.Context) {
	containerID := c.Param("id")
	var nodeHostname string
	err := h.db.QueryRow(`
		SELECT n.hostname FROM nodes n
		JOIN containers c ON c.node_id = n.id
		WHERE c.id = $1
	`, containerID).Scan(&nodeHostname)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}
	slaveURL := fmt.Sprintf("http://%s:8081", nodeHostname)
	req, err := http.NewRequest("DELETE", slaveURL+"/api/containers/"+containerID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to communicate with slave node"})
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "slave node failed to delete container"})
		return
	}
	_, err = h.db.Exec("DELETE FROM containers WHERE id = $1", containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove container from database"})
		return
	}
	_, err = h.db.Exec("UPDATE users SET container_id = NULL WHERE container_id = $1", containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "container deleted successfully"})
}

func (h *Handler) APIUpdateContainerStatus(c *gin.Context) {
	containerID := c.Param("id")
	
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	_, err := h.db.Exec("UPDATE containers SET status = $1, updated_at = NOW() WHERE id = $2", req.Status, containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update container status"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "status updated successfully"})
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

func (h *Handler) APIRegisterNode(c *gin.Context) {
	var req struct {
		NodeID       string `json:"node_id" binding:"required"`
		NodeToken    string `json:"node_token" binding:"required"`
		MaxMemoryMB  int    `json:"max_memory_mb"`
		MaxCPUCores  int    `json:"max_cpu_cores"`
		MaxStorageGB int    `json:"max_storage_gb"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var nodeID int
	err := h.db.QueryRow("SELECT id FROM nodes WHERE token = $1", req.NodeToken).Scan(&nodeID)
	if err == nil {
		_, err = h.db.Exec(`
			UPDATE nodes SET is_online = true, last_seen = NOW(), updated_at = NOW() 
			WHERE id = $1
		`, nodeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update node"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "node registered successfully"})
		return
	}
	_, err = h.db.Exec(`
		INSERT INTO nodes (name, hostname, token, max_memory_mb, max_cpu_cores, max_storage_gb, is_online, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, true, NOW())
	`, req.NodeID, req.NodeID, req.NodeToken, req.MaxMemoryMB, req.MaxCPUCores, req.MaxStorageGB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register node"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "node registered successfully"})
}

func (h *Handler) APINodeHeartbeat(c *gin.Context) {
	var req struct {
		NodeID     string      `json:"node_id" binding:"required"`
		NodeToken  string      `json:"node_token" binding:"required"`
		Containers interface{} `json:"containers"`
		Timestamp  int64       `json:"timestamp"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.db.Exec(`
		UPDATE nodes SET is_online = true, last_seen = NOW(), updated_at = NOW() 
		WHERE token = $1
	`, req.NodeToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid node token"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "heartbeat received"})
}
