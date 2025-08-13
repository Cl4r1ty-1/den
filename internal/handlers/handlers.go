package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
	"strings"
    "unicode"
	"log"
	
	"github.com/lib/pq"

	"github.com/den/internal/auth"
	"github.com/den/internal/database"
	"github.com/den/internal/dns"
	"github.com/den/internal/models"
    "github.com/den/internal/storage"
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
        if !user.AgreedToTOS || !user.AgreedToPrivacy {
            if c.FullPath() != "/user/aup/accept" && c.FullPath() != "/user/aup/questions" && c.FullPath() != "/user/aup/validate" {
                c.Redirect(http.StatusFound, "/aup")
                c.Abort()
                return
            }
        }
        c.Next()
	}
}

func (h *Handler) AUPPage(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    questions, _ := h.ensureAssignedQuestions(user.ID, 3)
    c.HTML(http.StatusOK, "aup.html", gin.H{
        "title": "terms & privacy",
        "user":  user,
        "quiz_questions": questions,
    })
}

func (h *Handler) AUPQuestions(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    questions, err := h.ensureAssignedQuestions(user.ID, 3)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load questions"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"questions": questions})
}

func (h *Handler) Me(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    c.JSON(http.StatusOK, gin.H{
        "id": user.ID,
        "username": user.Username,
        "display_name": user.DisplayName,
        "email": user.Email,
        "is_admin": user.IsAdmin,
        "container_id": user.ContainerID,
        "agreed_to_tos": user.AgreedToTOS,
        "agreed_to_privacy": user.AgreedToPrivacy,
    })
}

type quizAnswer struct { ID int `json:"id"`; Answer string `json:"answer"` }

func (h *Handler) AUPAccept(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    var req struct {
        AcceptTOS     bool `json:"accept_tos"`
        AcceptPrivacy bool `json:"accept_privacy"`
        Answers       []quizAnswer `json:"answers"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }
    if !req.AcceptTOS || !req.AcceptPrivacy {
        c.JSON(http.StatusBadRequest, gin.H{"error": "you must accept both the terms and privacy policy"})
        return
    }
    if err := h.validateQuizAnswers(user.ID, req.Answers); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if _, err := h.db.Exec(`UPDATE users SET agreed_to_tos = TRUE, agreed_to_privacy = TRUE, updated_at = NOW() WHERE id = $1`, user.ID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save acceptance"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AUPValidate(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    var req struct { Answers []quizAnswer `json:"answers"` }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }
    if err := h.validateQuizAnswers(user.ID, req.Answers); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) ensureAssignedQuestions(userID int, n int) ([]models.Question, error) {
    var assignedIDs pq.Int64Array
    if err := h.db.QueryRow(`SELECT tos_questions FROM users WHERE id = $1`, userID).Scan(&assignedIDs); err != nil {
        return nil, err
    }
    ids := make([]int, 0, len(assignedIDs))
    for _, v := range assignedIDs { ids = append(ids, int(v)) }

    if len(ids) == 0 {
        rows, err := h.db.Query(`SELECT id, prompt FROM questions WHERE is_active = TRUE ORDER BY random() LIMIT $1`, n)
        if err != nil { return nil, err }
        defer rows.Close()
        var newIDs []int
        var questions []models.Question
        for rows.Next() {
            var q models.Question
            if err := rows.Scan(&q.ID, &q.Prompt); err != nil { continue }
            newIDs = append(newIDs, q.ID)
            questions = append(questions, q)
        }
        if len(newIDs) > 0 {
            int64IDs := make([]int64, len(newIDs)); for i, v := range newIDs { int64IDs[i] = int64(v) }
            _, _ = h.db.Exec(`UPDATE users SET tos_questions = $1, updated_at = NOW() WHERE id = $2`, pq.Array(int64IDs), userID)
        }
        return questions, nil
    }
    rows, err := h.db.Query(`SELECT id, prompt FROM questions WHERE id = ANY($1) AND is_active = TRUE`, pq.Array(ids))
    if err != nil { return nil, err }
    defer rows.Close()
    var questions []models.Question
    for rows.Next() {
        var q models.Question
        if err := rows.Scan(&q.ID, &q.Prompt); err != nil { continue }
        questions = append(questions, q)
    }
    return questions, nil
}

func (h *Handler) validateQuizAnswers(userID int, answers []quizAnswer) error {
    if len(answers) == 0 {
        return fmt.Errorf("please answer the questions")
    }
    var assigned pq.Int64Array
    if err := h.db.QueryRow(`SELECT tos_questions FROM users WHERE id = $1`, userID).Scan(&assigned); err != nil {
        return fmt.Errorf("failed to load assigned questions")
    }
    if len(assigned) == 0 {
        return fmt.Errorf("no questions assigned; please reload the page")
    }
    assignedSet := map[int]struct{}{}
    for _, v := range assigned { assignedSet[int(v)] = struct{}{} }
    ids := make([]int, 0, len(assigned))
    for k := range assignedSet { ids = append(ids, k) }
    rows, err := h.db.Query(`SELECT id, correct_answer FROM questions WHERE id = ANY($1) AND is_active = TRUE`, pq.Array(ids))
    if err != nil { return fmt.Errorf("failed to load questions") }
    defer rows.Close()
    correct := map[int]string{}
    for rows.Next() {
        var id int; var ans string
        if err := rows.Scan(&id, &ans); err == nil {
            correct[id] = strings.TrimSpace(strings.ToLower(ans))
        }
    }
    if len(correct) == 0 {
        return fmt.Errorf("no active questions available")
    }
    incorrect := []int{}
    provided := map[int]string{}
    for _, a := range answers { provided[a.ID] = a.Answer }
    for id, want := range correct {
        got, ok := provided[id]
        if !ok || !isFuzzyCorrect(got, want) { incorrect = append(incorrect, id) }
    }
    if len(incorrect) > 0 {
        return fmt.Errorf("one or more answers are incorrect")
    }
    return nil
}
func isFuzzyCorrect(got, want string) bool {
    g := normalizeAnswer(got)
    w := normalizeAnswer(want)
    if g == "" || w == "" {
        return false
    }
    if g == w { return true }
    if extractDigits(g) == extractDigits(w) && extractDigits(w) != "" {
        return true
    }
    if len(w) <= 6 && (strings.Contains(g, w) || strings.Contains(w, g)) {
        return true
    }
    for _, alt := range synonymsFor(w) {
        if g == alt || (len(alt) <= 6 && strings.Contains(g, alt)) {
            return true
        }
    }
    dist := levenshtein(g, w)
    maxErr := 1
    if len(w) >= 8 { maxErr = 2 }
    if len(w) >= 14 { maxErr = 3 }
    return dist <= maxErr
}

func normalizeAnswer(s string) string {
    s = strings.ToLower(strings.TrimSpace(s))
    b := make([]rune, 0, len(s))
    for _, r := range s {
        if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
            b = append(b, r)
        }
    }
    out := strings.Join(strings.Fields(string(b)), " ")
    out = strings.ReplaceAll(out, " day", "")
    out = strings.ReplaceAll(out, " days", "")
    out = strings.ReplaceAll(out, " cookie", "")
    out = strings.ReplaceAll(out, " attack", "")
    return out
}

func extractDigits(s string) string {
    var b []rune
    for _, r := range s {
        if r >= '0' && r <= '9' { b = append(b, r) }
    }
    return string(b)
}

func synonymsFor(norm string) []string {
    m := map[string][]string{
        "google cloud": {"gcp", "google cloud platform"},
        "denial of service": {"dos", "ddos", "denial of service attack"},
        "computer misuse act 1990": {"uk computer misuse act", "computer misuse act"},
        "ico": {"information commissioners office", "information commissioner's office"},
        "session": {"session cookie"},
        "no": {"not allowed", "forbidden"},
        "yes": {"allowed", "permitted"},
        "13": {"thirteen"},
        "14": {"fourteen"},
    }
    if v, ok := m[norm]; ok { return v }
    return nil
}
func levenshtein(a, b string) int {
    ra := []rune(a)
    rb := []rune(b)
    na := len(ra)
    nb := len(rb)
    if na == 0 { return nb }
    if nb == 0 { return na }
    prev := make([]int, nb+1)
    curr := make([]int, nb+1)
    for j := 0; j <= nb; j++ { prev[j] = j }
    for i := 1; i <= na; i++ {
        curr[0] = i
        for j := 1; j <= nb; j++ {
            cost := 0
            if ra[i-1] != rb[j-1] { cost = 1 }
            del := prev[j] + 1
            ins := curr[j-1] + 1
            sub := prev[j-1] + cost
            curr[j] = min3(del, ins, sub)
        }
        copy(prev, curr)
    }
    return prev[nb]
}

func min3(a, b, c int) int {
    if a < b { if a < c { return a } ; return c }
    if b < c { return b }
    return c
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
func (h *Handler) AdminExportUserContainer(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    if !user.IsAdmin { c.JSON(http.StatusForbidden, gin.H{"error": "admin required"}); return }
    idStr := c.Param("id")
    targetUserID, err := strconv.Atoi(idStr)
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"}); return }

    var req struct{ TTLDays int `json:"ttl_days"` }
    if err := c.ShouldBindJSON(&req); err != nil || req.TTLDays <= 0 || req.TTLDays > 365 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ttl_days"}); return
    }

    var containerID, nodeHostname string
    err = h.db.QueryRow(`SELECT c.id, n.hostname FROM users u JOIN containers c ON u.container_id = c.id JOIN nodes n ON c.node_id = n.id WHERE u.id = $1`, targetUserID).Scan(&containerID, &nodeHostname)
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "user has no container"}); return }

    expiresAt := time.Now().Add(time.Duration(req.TTLDays) * 24 * time.Hour)

    var exportID int
    objectKey := fmt.Sprintf("exports/%s/%d/%d.tar.zst", containerID, targetUserID, time.Now().Unix())
    err = h.db.QueryRow(`INSERT INTO exports (user_id, container_id, object_key, status, expires_at, requested_by) VALUES ($1,$2,$3,'pending',$4,$5) RETURNING id`, targetUserID, containerID, objectKey, expiresAt, user.ID).Scan(&exportID)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"}); return }

    r2, err := storage.NewR2ClientFromEnv()
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "storage not configured"}); return }
    putURL, err := r2.PresignedPut(c, objectKey, 2*time.Hour)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload url"}); return }

    _, _ = h.db.Exec(`UPDATE exports SET status='uploading', updated_at=NOW() WHERE id=$1`, exportID)

    slaveURL := fmt.Sprintf("http://%s:8081/api/export", nodeHostname)
    payload := map[string]string{"container_id": containerID, "put_url": putURL}
    body, _ := json.Marshal(payload)
    resp, err := http.Post(slaveURL, "application/json", bytes.NewBuffer(body))
    if err != nil {
        _, _ = h.db.Exec(`UPDATE exports SET status='failed', error=$2, updated_at=NOW() WHERE id=$1`, exportID, err.Error())
        c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"}); return
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        _, _ = h.db.Exec(`UPDATE exports SET status='failed', error=$2, updated_at=NOW() WHERE id=$1`, exportID, string(b))
        c.JSON(http.StatusBadGateway, gin.H{"error": "export failed: "+string(b)}); return
    }

    getURL, err := r2.PresignedGet(c, objectKey, time.Duration(req.TTLDays)*24*time.Hour)
    if err != nil {
        _, _ = h.db.Exec(`UPDATE exports SET status='failed', error=$2, updated_at=NOW() WHERE id=$1`, exportID, err.Error())
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create download url"}); return
    }
    _, _ = h.db.Exec(`UPDATE exports SET status='complete', updated_at=NOW() WHERE id=$1`, exportID)
    c.JSON(http.StatusOK, gin.H{"export_id": exportID, "download_url": getURL, "expires_at": expiresAt})
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
		var allocatedPorts pq.Int64Array
		err := h.db.QueryRow(`
			SELECT id, user_id, node_id, name, status, ip_address, ssh_port,
				   memory_mb, cpu_cores, storage_gb, allocated_ports, created_at, updated_at
			FROM containers WHERE id = $1
		`, *user.ContainerID).Scan(
			&container.ID, &container.UserID, &container.NodeID, &container.Name,
			&container.Status, &container.IPAddress, &container.SSHPort,
			&container.MemoryMB, &container.CPUCores, &container.StorageGB,
			&allocatedPorts, &container.CreatedAt, &container.UpdatedAt,
		)
		if err != nil {
			fmt.Printf("error loading container for user %d: %v\n", user.ID, err)
			container = nil
		} else {
			// i am a dumbassssssssss
			container.AllocatedPorts = make([]int, len(allocatedPorts))
			for i, port := range allocatedPorts {
				container.AllocatedPorts[i] = int(port)
			}
		}
	}
	rows, err := h.db.Query(`
		SELECT id, subdomain, target_port, subdomain_type, is_active, created_at
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
			&subdomain.SubdomainType, &subdomain.IsActive, &subdomain.CreatedAt)
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

func (h *Handler) GetNewPort(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    if user.ContainerID == nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "no container"})
        return
    }
    var nodeHostname string
    if err := h.db.QueryRow(`SELECT n.hostname FROM nodes n JOIN containers c ON c.node_id = n.id WHERE c.id = $1`, *user.ContainerID).Scan(&nodeHostname); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "node lookup failed"})
        return
    }
    slaveURL := fmt.Sprintf("http://%s:8081", nodeHostname)
    payload := map[string]string{"container_id": *user.ContainerID}
    body, _ := json.Marshal(payload)
    resp, err := http.Post(slaveURL+"/api/ports/new", "application/json", bytes.NewBuffer(body))
    if err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"})
        return
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        c.JSON(http.StatusBadGateway, gin.H{"error": string(b)})
        return
    }
    var res struct{ Port int `json:"port"` }
    if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "invalid node response"})
        return
    }
    _, err = h.db.Exec(`UPDATE containers SET allocated_ports = array_append(allocated_ports, $1), updated_at = NOW() WHERE id = $2`, res.Port, *user.ContainerID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist port"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"port": res.Port})
}

func (h *Handler) CreateContainer(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
    requestID := c.GetString("request_id")
	
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
        log.Printf("rid=%s CreateContainer: no available nodes: %v", requestID, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no available nodes"})
		return
	}
	slaveURL := fmt.Sprintf("http://%s:8081", nodeHostname)
    log.Printf("rid=%s CreateContainer: selected node id=%d host=%s url=%s for user id=%d username=%s", requestID, nodeID, nodeHostname, slaveURL, user.ID, user.Username)
	payload := map[string]interface{}{
		"user_id":   user.ID,
		"username":  user.Username,
	}
	
	data, err := json.Marshal(payload)
	if err != nil {
        log.Printf("rid=%s CreateContainer: marshal payload error: %v", requestID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal request"})
		return
	}
	
	resp, err := http.Post(slaveURL+"/api/containers", "application/json", bytes.NewBuffer(data))
	if err != nil {
        log.Printf("rid=%s CreateContainer: POST to slave failed: %v", requestID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to communicate with slave node"})
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        log.Printf("rid=%s CreateContainer: slave returned status=%d body=%s", requestID, resp.StatusCode, string(body))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "slave node failed to create container"})
		return
	}
	
	var containerInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&containerInfo); err != nil {
        log.Printf("rid=%s CreateContainer: decode container info error: %v", requestID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decode container info"})
		return
	}
	containerID := containerInfo["ID"].(string)
    var allocatedPorts []int
	
    _, err = h.db.Exec(`
        INSERT INTO containers (id, user_id, node_id, name, status, ip_address, ssh_port, memory_mb, cpu_cores, storage_gb, allocated_ports)
        VALUES ($1, $2, $3, $4, 'running', $5, $6, $7, $8, $9, $10)
	`, containerID, user.ID, nodeID, containerInfo["Name"], 
		containerInfo["IP"], containerInfo["SSHPort"], 4096, 4, 15, pq.Array(allocatedPorts))
	if err != nil {
        log.Printf("rid=%s CreateContainer: DB insert error for container %s: %v", requestID, containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store container info"})
		return
	}
    _, err = h.db.Exec("UPDATE users SET container_id = $1 WHERE id = $2", containerID, user.ID)
	if err != nil {
        log.Printf("rid=%s CreateContainer: DB update user %d with container %s failed: %v", requestID, user.ID, containerID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}
	
    log.Printf("rid=%s CreateContainer: success container_id=%s ip=%v ssh_port=%v ports=%v", requestID, containerID, containerInfo["IP"], containerInfo["SSHPort"], allocatedPorts)
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
		Subdomain     string `json:"subdomain" binding:"required"`
		TargetPort    int    `json:"target_port" binding:"required"`
		SubdomainType string `json:"subdomain_type,omitempty"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.SubdomainType == "" {
		req.SubdomainType = "project"
	}
	if req.SubdomainType != "username" && req.SubdomainType != "project" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subdomain_type must be 'username' or 'project'"})
		return
	}
	if req.SubdomainType == "username" && req.Subdomain != user.Username {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username subdomain must match your username"})
		return
	}
	
	if err := h.dns.ValidateSubdomain(req.Subdomain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var allocatedPorts []int
	if user.ContainerID != nil {
		var allocatedPortsArray pq.Int64Array
		err := h.db.QueryRow("SELECT allocated_ports FROM containers WHERE id = $1", *user.ContainerID).Scan(&allocatedPortsArray)
		if err != nil {
			fmt.Printf("error getting allocated ports for user %d: %v\n", user.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get allocated ports"})
			return
		}
		// ok but like this is go's fault not mine
		allocatedPorts = make([]int, len(allocatedPortsArray))
		for i, port := range allocatedPortsArray {
			allocatedPorts[i] = int(port)
		}
	}
	if err := h.dns.ValidateUserPort(req.TargetPort, allocatedPorts); err != nil {
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
	var nodeIP string
	if user.ContainerID != nil {
		err = h.db.QueryRow(`
			SELECT COALESCE(n.public_hostname, n.hostname) 
			FROM containers c 
			JOIN nodes n ON c.node_id = n.id 
			WHERE c.id = $1
		`, *user.ContainerID).Scan(&nodeIP)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "container node not found"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no container available"})
		return
	}
	var subdomainID int
	err = h.db.QueryRow(`
		INSERT INTO subdomains (user_id, subdomain, target_port, subdomain_type, is_active)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id
	`, user.ID, req.Subdomain, req.TargetPort, req.SubdomainType).Scan(&subdomainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create subdomain"})
		return
	}
    if err := h.dns.CreateDNSRecord(req.Subdomain, user.Username, req.SubdomainType, nodeIP, req.TargetPort); err != nil {
		h.db.Exec("DELETE FROM subdomains WHERE id = $1", subdomainID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create DNS record"})
		return
	}
	var fullSubdomain string
	if req.SubdomainType == "username" {
		fullSubdomain = req.Subdomain + ".hack.kim"
	} else {
		fullSubdomain = req.Subdomain + "." + user.Username + ".hack.kim"
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "subdomain created successfully",
		"subdomain":     fullSubdomain,
		"target_port":   req.TargetPort,
		"subdomain_type": req.SubdomainType,
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
    var subdomainType string
    _ = h.db.QueryRow("SELECT subdomain_type FROM subdomains WHERE id = $1", subdomainID).Scan(&subdomainType)
    if err := h.dns.DeleteDNSRecord(subdomain, user.Username, subdomainType); err != nil {
		fmt.Printf("failed to delete DNS record: %v\n", err)

	}

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

func (h *Handler) AdminDeleteUserContainer(c *gin.Context) {
    requestID := c.GetString("request_id")
    userID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
        return
    }

    var containerID, nodeHostname, username string
    err = h.db.QueryRow(`
        SELECT c.id, n.hostname, u.username
        FROM users u
        JOIN containers c ON u.container_id = c.id
        JOIN nodes n ON c.node_id = n.id
        WHERE u.id = $1
    `, userID).Scan(&containerID, &nodeHostname, &username)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user has no container"})
        return
    }

    slaveURL := fmt.Sprintf("http://%s:8081", nodeHostname)
    req, _ := http.NewRequest(http.MethodDelete, slaveURL+"/api/containers/"+containerID, nil)
    client := &http.Client{Timeout: 30 * time.Second}
    resp, derr := client.Do(req)
    if derr != nil {
        log.Printf("rid=%s AdminDeleteUserContainer: slave delete request failed: %v", requestID, derr)
        c.JSON(http.StatusBadGateway, gin.H{"error": "failed to communicate with node"})
        return
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        log.Printf("rid=%s AdminDeleteUserContainer: slave delete returned %d: %s", requestID, resp.StatusCode, string(body))
        c.JSON(http.StatusBadGateway, gin.H{"error": "node failed to delete container"})
        return
    }
    if rows, qerr := h.db.Query(`SELECT subdomain, subdomain_type FROM subdomains WHERE user_id = $1`, userID); qerr == nil {
        defer rows.Close()
        for rows.Next() {
            var sub, subType string
            if err := rows.Scan(&sub, &subType); err == nil {
                if derr := h.dns.DeleteDNSRecord(sub, username, subType); derr != nil {
                    log.Printf("rid=%s AdminDeleteUserContainer: DNS/Caddy cleanup failed for %s type=%s: %v", requestID, sub, subType, derr)
                }
            }
        }
        _, _ = h.db.Exec("DELETE FROM subdomains WHERE user_id = $1", userID)
    }

    _, _ = h.db.Exec("DELETE FROM containers WHERE id = $1", containerID)
    _, _ = h.db.Exec("UPDATE users SET container_id = NULL, updated_at = NOW() WHERE id = $1", userID)

    c.JSON(http.StatusOK, gin.H{"message": "container deleted"})
}

func (h *Handler) DeleteUser(c *gin.Context) {
    requestID := c.GetString("request_id")
    userID, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
        return
    }

    var username string
    var containerID *string
    err = h.db.QueryRow("SELECT username, container_id FROM users WHERE id = $1", userID).Scan(&username, &containerID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }
    rows, err := h.db.Query(`SELECT subdomain, subdomain_type FROM subdomains WHERE user_id = $1`, userID)
    if err == nil {
        defer rows.Close()
        for rows.Next() {
            var sub string
            var subType string
            if err := rows.Scan(&sub, &subType); err == nil {
                if derr := h.dns.DeleteDNSRecord(sub, username, subType); derr != nil {
                    log.Printf("rid=%s DeleteUser: failed DNS/Caddy cleanup for %s type=%s: %v", requestID, sub, subType, derr)
                }
            }
        }
    }
    if containerID != nil && *containerID != "" {
        var nodeHostname string
        err := h.db.QueryRow(`
            SELECT n.hostname FROM nodes n
            JOIN containers c ON c.node_id = n.id
            WHERE c.id = $1
        `, *containerID).Scan(&nodeHostname)
        if err != nil {
            log.Printf("rid=%s DeleteUser: could not find node for container %s: %v", requestID, *containerID, err)
        } else {
            slaveURL := fmt.Sprintf("http://%s:8081", nodeHostname)
            req, _ := http.NewRequest(http.MethodDelete, slaveURL+"/api/containers/"+*containerID, nil)
            client := &http.Client{Timeout: 30 * time.Second}
            resp, derr := client.Do(req)
            if derr != nil {
                log.Printf("rid=%s DeleteUser: slave delete request failed: %v", requestID, derr)
                c.JSON(http.StatusBadGateway, gin.H{"error": "failed to delete container on node"})
                return
            }
            defer resp.Body.Close()
            if resp.StatusCode != http.StatusOK {
                body, _ := io.ReadAll(resp.Body)
                log.Printf("rid=%s DeleteUser: slave delete returned %d: %s", requestID, resp.StatusCode, string(body))
                c.JSON(http.StatusBadGateway, gin.H{"error": "node failed to delete container"})
                return
            }
            _, _ = h.db.Exec("DELETE FROM containers WHERE id = $1", *containerID)
            _, _ = h.db.Exec("UPDATE users SET container_id = NULL WHERE id = $1", userID)
        }
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

func (h *Handler) updateTraefikConfig() error {
	return nil
}

func (h *Handler) GetUserSubdomains(c *gin.Context) {
	user := c.MustGet("user").(*models.User)

	rows, err := h.db.Query(`
		SELECT id, subdomain, target_port, subdomain_type, is_active, created_at
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
			&subdomain.SubdomainType, &subdomain.IsActive, &subdomain.CreatedAt); err != nil {
			continue
		}
		subdomains = append(subdomains, subdomain)
	}

	c.JSON(http.StatusOK, gin.H{"subdomains": subdomains})
}

func (h *Handler) APIDeleteContainer(c *gin.Context) {
	containerID := c.Param("id")
    var nodeHostname, username string
    err := h.db.QueryRow(`
        SELECT n.hostname, u.username
        FROM containers c
        JOIN nodes n ON c.node_id = n.id
        JOIN users u ON u.id = c.user_id
        WHERE c.id = $1
    `, containerID).Scan(&nodeHostname, &username)
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
    if rows, qerr := h.db.Query(`SELECT subdomain, subdomain_type, user_id FROM subdomains WHERE user_id = (SELECT user_id FROM containers WHERE id = $1)`, containerID); qerr == nil {
        defer rows.Close()
        var sub, subType string
        var uid int
        for rows.Next() {
            if err := rows.Scan(&sub, &subType, &uid); err == nil {
                if derr := h.dns.DeleteDNSRecord(sub, username, subType); derr != nil {
                    fmt.Printf("failed to delete DNS record for %s: %v\n", sub, derr)
                }
            }
        }
        _, _ = h.db.Exec("DELETE FROM subdomains WHERE user_id = (SELECT user_id FROM containers WHERE id = $1)", containerID)
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
