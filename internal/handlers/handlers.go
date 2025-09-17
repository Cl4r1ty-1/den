package handlers

import (
	"bytes"
	"database/sql"
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
	"os"
	"html"
	
	"github.com/lib/pq"
	denemail "github.com/den/internal/email"

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

func (h *Handler) inertia(c *gin.Context, component string, props gin.H) {
	page := gin.H{
		"component": component,
		"props":     props,
		"url":       c.Request.URL.Path,
		"version":   "",
	}
	if c.GetHeader("X-Inertia") == "true" {
		c.Header("Vary", "Accept")
		c.Header("X-Inertia", "true")
		c.JSON(http.StatusOK, page)
		return
	}
	data, err := os.ReadFile("webapp/dist/index.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "frontend not built")
		return
	}
	b, _ := json.Marshal(page)
	escaped := html.EscapeString(string(b))
	replacement := []byte(`<div id="app" data-page="` + escaped + `"></div>`)
	out := bytes.Replace(data, []byte(`<div id="app"></div>`), replacement, 1)
	c.Data(http.StatusOK, "text/html; charset=utf-8", out)
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
            if c.FullPath() != "/user/aup" && c.FullPath() != "/user/aup/accept" {
                c.Redirect(http.StatusFound, "/user/aup")
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
    h.inertia(c, "AUP", gin.H{
        "user":  user,
        "quiz_questions": questions,
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

    var req struct{ TTLDays int `json:"ttl_days"`; EmailUser bool `json:"email_user"` }
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

    // enqueue async export job
    // include requester and target email info for completion emails
    var targetEmail, targetUsername string
    _ = h.db.QueryRow(`SELECT email, username FROM users WHERE id=$1`, targetUserID).Scan(&targetEmail, &targetUsername)
    payload := map[string]interface{}{
        "export_id": exportID,
        "user_id": targetUserID,
        "container_id": containerID,
        "node_hostname": nodeHostname,
        "object_key": objectKey,
        "ttl_days": req.TTLDays,
        "email_user": req.EmailUser,
        "requester_email": user.Email,
        "target_email": targetEmail,
        "target_username": targetUsername,
    }
    jb, _ := json.Marshal(payload)
    if _, err := h.db.Exec(`INSERT INTO jobs (type, status, payload) VALUES ('export_container','queued',$1)`, string(jb)); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue job"}); return
    }
    c.JSON(http.StatusOK, gin.H{"export_id": exportID, "queued": true, "expires_at": expiresAt})
}
func (h *Handler) UserExportContainer(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    var req struct{ TTLDays int `json:"ttl_days"` }
    _ = c.ShouldBindJSON(&req)
    if req.TTLDays <= 0 || req.TTLDays > 365 { req.TTLDays = 7 }

    var containerID, nodeHostname string
    err := h.db.QueryRow(`SELECT c.id, n.hostname FROM containers c JOIN nodes n ON c.node_id = n.id WHERE c.user_id = $1`, user.ID).Scan(&containerID, &nodeHostname)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "you have no container"}); return
    }

    expiresAt := time.Now().Add(time.Duration(req.TTLDays) * 24 * time.Hour)
    var exportID int
    objectKey := fmt.Sprintf("exports/%s/%d/%d.tar.zst", containerID, user.ID, time.Now().Unix())
    err = h.db.QueryRow(`INSERT INTO exports (user_id, container_id, object_key, status, expires_at, requested_by) VALUES ($1,$2,$3,'pending',$4,$5) RETURNING id`, user.ID, containerID, objectKey, expiresAt, user.ID).Scan(&exportID)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"}); return }

    payload := map[string]interface{}{
        "export_id":     exportID,
        "user_id":       user.ID,
        "container_id":  containerID,
        "node_hostname": nodeHostname,
        "object_key":    objectKey,
        "ttl_days":      req.TTLDays,
    }
    jb, _ := json.Marshal(payload)
    if _, err := h.db.Exec(`INSERT INTO jobs (type, status, payload) VALUES ('export_container','queued',$1)`, string(jb)); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue job"}); return
    }
    go func(u *models.User, contID string, expires time.Time) {
        if u.Email == "" { return }
        client, err := denemail.NewFromEnv(); if err != nil { return }
        html := denemail.RenderNeobrutalismEmail(
            "Your export has been queued",
            "We'll email you again when it's ready",
            fmt.Sprintf("<p>Container <b>%s</b> export has been queued. Link will expire on <b>%s</b>.</p>", contID, expires.Format(time.RFC1123)),
        )
        _ = client.Send([]string{u.Email}, "den: export queued", html, "")
    }(user, containerID, expiresAt)
    c.JSON(http.StatusOK, gin.H{"export_id": exportID, "queued": true, "expires_at": expiresAt})
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
func (h *Handler) RequireCLIAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if strings.HasPrefix(strings.ToLower(token), "bearer ") {
            token = strings.TrimSpace(token[7:])
        }
        if token == "" {
            token = c.GetHeader("Den-Container-Token")
        }
        if token == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "missing container token"})
            c.Abort(); return
        }
        var containerID string
        var userID int
        var nodeHostname string
        err := h.db.QueryRow(`SELECT c.id, c.user_id, n.hostname FROM containers c JOIN nodes n ON c.node_id = n.id WHERE c.container_token = $1`, token).Scan(&containerID, &userID, &nodeHostname)
        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid container token"})
            c.Abort(); return
        }
        c.Set("cli_container_id", containerID)
        c.Set("cli_user_id", userID)
        c.Set("cli_node_hostname", nodeHostname)
        c.Next()
    }
}
func (h *Handler) CLIMe(c *gin.Context) {
    userID := c.GetInt("cli_user_id")
    containerID := c.GetString("cli_container_id")
    var user models.User
    var email sql.NullString
    var displayName sql.NullString
    err := h.db.QueryRow(`SELECT id, username, email, display_name, agreed_to_tos, agreed_to_privacy, is_admin FROM users WHERE id = $1`, userID).Scan(&user.ID, &user.Username, &email, &displayName, &user.AgreedToTOS, &user.AgreedToPrivacy, &user.IsAdmin)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"}); return }
    user.Email = email.String
    user.DisplayName = displayName.String
    var cont models.Container
    var ip sql.NullString
    var ports pq.Int64Array
    err = h.db.QueryRow(`SELECT id, user_id, node_id, name, status, ip_address, ssh_port, memory_mb, cpu_cores, storage_gb, allocated_ports, created_at, updated_at FROM containers WHERE id = $1`, containerID).Scan(&cont.ID, &cont.UserID, &cont.NodeID, &cont.Name, &cont.Status, &ip, &cont.SSHPort, &cont.MemoryMB, &cont.CPUCores, &cont.StorageGB, &ports, &cont.CreatedAt, &cont.UpdatedAt)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"}); return }
    if ip.Valid { s := ip.String; cont.IPAddress = &s } else { cont.IPAddress = nil }
    if len(ports) > 0 { cont.AllocatedPorts = make([]int, len(ports)); for i, p := range ports { cont.AllocatedPorts[i] = int(p) } }
    c.JSON(http.StatusOK, gin.H{"user": user, "container": cont})
}

func (h *Handler) CLIContainerStats(c *gin.Context) {
    containerID := c.GetString("cli_container_id")
    nodeHostname := c.GetString("cli_node_hostname")
    slaveURL := fmt.Sprintf("http://%s:8081", nodeHostname)
    resp, err := http.Get(slaveURL+"/api/containers-stats/"+containerID)
    if err != nil { c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"}); return }
    defer resp.Body.Close()
    b, _ := io.ReadAll(resp.Body)
    c.Data(resp.StatusCode, "application/json", b)
}

func (h *Handler) CLIContainerControl(c *gin.Context) {
    action := c.Param("action")
    if action != "start" && action != "stop" && action != "restart" { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"}); return }
    containerID := c.GetString("cli_container_id")
    nodeHostname := c.GetString("cli_node_hostname")
    slaveURL := fmt.Sprintf("http://%s:8081/api/control/containers/%s", nodeHostname, containerID)
    if action == "restart" {
        sb, _ := json.Marshal(map[string]interface{}{"action": "stop"})
        _, _ = http.Post(slaveURL, "application/json", bytes.NewBuffer(sb))
        time.Sleep(1 * time.Second)
        action = "start"
    }
    body, _ := json.Marshal(map[string]interface{}{"action": action})
    resp, err := http.Post(slaveURL, "application/json", bytes.NewBuffer(body))
    if err != nil { c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()}); return }
    defer resp.Body.Close()
    if resp.StatusCode >= 200 && resp.StatusCode < 300 { c.JSON(http.StatusOK, gin.H{"ok": true}); return }
    b, _ := io.ReadAll(resp.Body)
    c.Data(resp.StatusCode, "application/json", b)
}

func (h *Handler) CLIContainerPorts(c *gin.Context) {
    containerID := c.GetString("cli_container_id")
    var ports pq.Int64Array
    if err := h.db.QueryRow(`SELECT allocated_ports FROM containers WHERE id=$1`, containerID).Scan(&ports); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"}); return
    }
    out := make([]int, len(ports))
    for i, p := range ports { out[i] = int(p) }
    c.JSON(http.StatusOK, gin.H{"ports": out})
}

func (h *Handler) CLIContainerNewPort(c *gin.Context) {
    containerID := c.GetString("cli_container_id")
    nodeHostname := c.GetString("cli_node_hostname")
    slaveURL := fmt.Sprintf("http://%s:8081", nodeHostname)
    payload := map[string]string{"container_id": containerID}
    body, _ := json.Marshal(payload)
    resp, err := http.Post(slaveURL+"/api/ports/new", "application/json", bytes.NewBuffer(body))
    if err != nil { c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"}); return }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        c.JSON(http.StatusBadGateway, gin.H{"error": string(b)})
        return
    }
    var res struct{ Port int `json:"port"` }
    if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "invalid node response"}); return
    }
    if _, err := h.db.Exec(`UPDATE containers SET allocated_ports = array_append(allocated_ports, $1), updated_at = NOW() WHERE id = $2`, res.Port, containerID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist port"}); return
    }
    c.JSON(http.StatusOK, gin.H{"port": res.Port})
}
func (h *Handler) Home(c *gin.Context) {
	props := gin.H{}
	if sessionID, err := c.Cookie("session"); err == nil {
		if user, err := h.auth.GetUserBySession(sessionID); err == nil && user != nil {
			props["user"] = user
		}
	}
	h.inertia(c, "Home", props)
}

func (h *Handler) LoginPage(c *gin.Context) {
	h.inertia(c, "Login", gin.H{})
}

func (h *Handler) LegalPage(c *gin.Context) {
	h.inertia(c, "Legal", gin.H{})
}

func (h *Handler) Logout(c *gin.Context) {
	if sessionID, err := c.Cookie("session"); err == nil && sessionID != "" {
		_ = h.auth.DeleteSession(sessionID)
	}
	c.SetCookie("session", "", -1, "/", "", false, true)
	c.SetCookie("oauth_state", "", -1, "/", "", false, true)
	h.inertia(c, "Logout", gin.H{"message": "you've been successfully logged out"})
}

func (h *Handler) GitHubAuth(c *gin.Context) {
	state := generateState()
	c.SetCookie("oauth_state", state, 300, "/", "", false, true)
	authURL := h.auth.GetAuthURL(state)
	c.Redirect(http.StatusFound, authURL)
}

func (h *Handler) GitHubCallback(c *gin.Context) {
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
	if strings.ToLower(strings.TrimSpace(user.ApprovalStatus)) != "approved" {
		h.inertia(c, "PendingApproval", gin.H{"user": user})
		return
	}
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
			container.AllocatedPorts = make([]int, len(allocatedPorts))
			for i, port := range allocatedPorts { container.AllocatedPorts[i] = int(port) }
		}
	}
	rows, err := h.db.Query(`
		SELECT id, subdomain, target_port, subdomain_type, is_active, created_at
		FROM subdomains WHERE user_id = $1
		ORDER BY created_at DESC
	`, user.ID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"}); return }
	defer rows.Close()
	var subdomains []models.Subdomain
	for rows.Next() {
		var subdomain models.Subdomain
		if err := rows.Scan(&subdomain.ID, &subdomain.Subdomain, &subdomain.TargetPort, &subdomain.SubdomainType, &subdomain.IsActive, &subdomain.CreatedAt); err == nil {
			subdomain.UserID = user.ID
			subdomains = append(subdomains, subdomain)
		}
	}
	h.inertia(c, "Dashboard", gin.H{"user": user, "container": container, "subdomains": subdomains})
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
func (h *Handler) ContainerToken(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    if user.ContainerID == nil || *user.ContainerID == "" {
        c.JSON(http.StatusNotFound, gin.H{"error": "no container"})
        return
    }
    var token string
    if err := h.db.QueryRow(`SELECT container_token FROM containers WHERE id = $1`, *user.ContainerID).Scan(&token); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"container_token": token})
}

func (h *Handler) RotateContainerToken(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    if user.ContainerID == nil || *user.ContainerID == "" {
        c.JSON(http.StatusNotFound, gin.H{"error": "no container"})
        return
    }
    b := make([]byte, 24)
    if _, err := rand.Read(b); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "token gen failed"}); return }
    newTok := hex.EncodeToString(b)
    if _, err := h.db.Exec(`UPDATE containers SET container_token=$1, updated_at=NOW() WHERE id=$2`, newTok, *user.ContainerID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"}); return
    }
    var nodeHostname string
    if err := h.db.QueryRow(`SELECT n.hostname FROM nodes n JOIN containers c ON c.node_id=n.id WHERE c.id=$1`, *user.ContainerID).Scan(&nodeHostname); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "node lookup failed"}); return
    }
    slaveURL := fmt.Sprintf("http://%s:8081/api/cli/token", nodeHostname)
    body := map[string]string{"container_id": *user.ContainerID, "token": newTok, "username": user.Username}
    bb, _ := json.Marshal(body)
    resp, err := http.Post(slaveURL, "application/json", bytes.NewBuffer(bb))
    if err != nil { c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"}); return }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        c.JSON(http.StatusBadGateway, gin.H{"error": "node rejected token write"}); return
    }
    c.JSON(http.StatusOK, gin.H{"container_token": newTok})
}
func (h *Handler) ContainerStats(c *gin.Context) {
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
    resp, err := http.Get(slaveURL+"/api/containers-stats/"+*user.ContainerID)
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
    var stats map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
        c.JSON(http.StatusBadGateway, gin.H{"error": "invalid node response"})
        return
    }
    c.JSON(http.StatusOK, stats)
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
	if strings.ToLower(strings.TrimSpace(user.ApprovalStatus)) != "approved" {
		c.JSON(http.StatusForbidden, gin.H{"error": "account not approved"})
		return
	}
	
	if user.ContainerID != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container already exists"})
		return
	}
	
	payload := map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
	}
	jb, _ := json.Marshal(payload)
	if _, err := h.db.Exec(`INSERT INTO jobs (type, status, payload) VALUES ('create_container','queued',$1)`, string(jb)); err != nil {
		log.Printf("rid=%s CreateContainer: enqueue failed: %v", requestID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue job"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"queued": true})
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
			container = nil
		} else {
			container.AllocatedPorts = make([]int, len(allocatedPorts))
			for i, port := range allocatedPorts { container.AllocatedPorts[i] = int(port) }
		}
	}
	
	rows, err := h.db.Query(`
		SELECT id, subdomain, target_port, subdomain_type, is_active, created_at
		FROM subdomains WHERE user_id = $1
		ORDER BY created_at DESC
	`, user.ID)
	if err != nil { 
		h.inertia(c, "Subdomains", gin.H{"user": user, "container": container, "subdomains": []models.Subdomain{}})
		return 
	}
	defer rows.Close()
	var subdomains []models.Subdomain
	for rows.Next() {
		var subdomain models.Subdomain
		if err := rows.Scan(&subdomain.ID, &subdomain.Subdomain, &subdomain.TargetPort, &subdomain.SubdomainType, &subdomain.IsActive, &subdomain.CreatedAt); err == nil {
			subdomain.UserID = user.ID
			subdomains = append(subdomains, subdomain)
		}
	}
	
	h.inertia(c, "Subdomains", gin.H{"user": user, "container": container, "subdomains": subdomains})
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
	
	h.inertia(c, "SSHSetup", gin.H{
		"user": user,
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
	h.inertia(c, "Admin", gin.H{"user_count": userCount, "node_count": nodeCount, "container_count": containerCount})
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
		SELECT id, username, email, display_name, is_admin, container_id, created_at,
		       approval_status, approved_by, approved_at, rejection_reason
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
			&user.IsAdmin, &user.ContainerID, &user.CreatedAt,
			&user.ApprovalStatus, &user.ApprovedBy, &user.ApprovedAt, &user.RejectionReason)
		if err != nil {
			continue
		}
		users = append(users, user)
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *Handler) AdminApproveUser(c *gin.Context) {
	admin := c.MustGet("user").(*models.User)
	if !admin.IsAdmin { c.JSON(http.StatusForbidden, gin.H{"error": "admin required"}); return }
	idStr := c.Param("id")
	userID, err := strconv.Atoi(idStr)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"}); return }
    var email, username string
    _ = h.db.QueryRow(`SELECT email, username FROM users WHERE id=$1`, userID).Scan(&email, &username)
	_, err = h.db.Exec(`UPDATE users SET approval_status = 'approved', approved_by = $1, approved_at = NOW(), rejection_reason = NULL, updated_at = NOW() WHERE id = $2`, admin.ID, userID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve user"}); return }
    if strings.TrimSpace(email) != "" {
        go func(to, uname string) {
            client, err := denemail.NewFromEnv(); if err != nil { return }
            html := denemail.RenderNeobrutalismEmail(
                "You're approved!",
                "Your den environment can now be created",
                fmt.Sprintf("<p>Hi <b>%s</b>, your account has been approved. You can now create and use your container from the dashboard.</p>", html.EscapeString(uname)),
            )
            _ = client.Send([]string{to}, "den: account approved", html, "")
        }(email, username)
    }
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminRejectUser(c *gin.Context) {
	admin := c.MustGet("user").(*models.User)
	if !admin.IsAdmin { c.JSON(http.StatusForbidden, gin.H{"error": "admin required"}); return }
	idStr := c.Param("id")
	userID, err := strconv.Atoi(idStr)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"}); return }
	var req struct { Reason string `json:"reason"` }
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"}); return }
	var reasonPtr *string
	if strings.TrimSpace(req.Reason) != "" { r := strings.TrimSpace(req.Reason); reasonPtr = &r }
    var email, username string
    _ = h.db.QueryRow(`SELECT email, username FROM users WHERE id=$1`, userID).Scan(&email, &username)
	_, err = h.db.Exec(`UPDATE users SET approval_status = 'rejected', approved_by = $1, approved_at = NOW(), rejection_reason = $2, updated_at = NOW() WHERE id = $3`, admin.ID, reasonPtr, userID)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject user"}); return }
    if strings.TrimSpace(email) != "" {
        go func(to, uname, reason string) {
            client, err := denemail.NewFromEnv(); if err != nil { return }
            body := "<p>Hi <b>" + html.EscapeString(uname) + "</b>, unfortunately your account was not approved at this time.</p>"
            if strings.TrimSpace(reason) != "" {
                body += "<p><b>Reason:</b> " + html.EscapeString(reason) + "</p>"
            }
            htmlMsg := denemail.RenderNeobrutalismEmail(
                "Account not approved",
                "You can reply to this email if you believe this is an error",
                body,
            )
            _ = client.Send([]string{to}, "den: account not approved", htmlMsg, "")
        }(email, username, req.Reason)
    }
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminDeleteUserContainer(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    if !user.IsAdmin { c.JSON(http.StatusForbidden, gin.H{"error": "admin required"}); return }
    userID, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"}); return }

    var containerID, nodeHostname, username string
    err = h.db.QueryRow(`
        SELECT c.id, n.hostname, u.username
        FROM users u
        JOIN containers c ON u.container_id = c.id
        JOIN nodes n ON c.node_id = n.id
        WHERE u.id = $1
    `, userID).Scan(&containerID, &nodeHostname, &username)
    if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "user has no container"}); return }

    payload := map[string]interface{}{
        "user_id": userID,
        "container_id": containerID,
        "node_hostname": nodeHostname,
        "username": username,
    }
    jb, _ := json.Marshal(payload)
    var jobID int
    if err := h.db.QueryRow(`INSERT INTO jobs (type, status, payload) VALUES ('delete_container','queued',$1) RETURNING id`, string(jb)).Scan(&jobID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue job"}); return
    }
    c.JSON(http.StatusOK, gin.H{"queued": true, "job_id": jobID})
}

func (h *Handler) AdminRotateUserContainerToken(c *gin.Context) {
    admin := c.MustGet("user").(*models.User)
    if !admin.IsAdmin { c.JSON(http.StatusForbidden, gin.H{"error": "admin required"}); return }
    userID, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"}); return }
    var containerID, nodeHostname string
    if err := h.db.QueryRow(`SELECT c.id, n.hostname FROM users u JOIN containers c ON u.container_id=c.id JOIN nodes n ON c.node_id=n.id WHERE u.id=$1`, userID).Scan(&containerID, &nodeHostname); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "user has no container"}); return
    }
    b := make([]byte, 24)
    if _, err := rand.Read(b); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "token gen failed"}); return }
    newTok := hex.EncodeToString(b)
    if _, err := h.db.Exec(`UPDATE containers SET container_token=$1, updated_at=NOW() WHERE id=$2`, newTok, containerID); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"}); return }
    slaveURL := fmt.Sprintf("http://%s:8081/api/cli/token", nodeHostname)
    var username string
    _ = h.db.QueryRow(`SELECT username FROM users WHERE id = $1`, userID).Scan(&username)
    body := map[string]string{"container_id": containerID, "token": newTok, "username": username}
    bb, _ := json.Marshal(body)
    resp, err := http.Post(slaveURL, "application/json", bytes.NewBuffer(bb))
    if err != nil { c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"}); return }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 { c.JSON(http.StatusBadGateway, gin.H{"error": "node rejected token write"}); return }
    c.JSON(http.StatusOK, gin.H{"ok": true, "container_token": newTok})
}

func (h *Handler) AdminReinstallUserCLI(c *gin.Context) {
    admin := c.MustGet("user").(*models.User)
    if !admin.IsAdmin { c.JSON(http.StatusForbidden, gin.H{"error": "admin required"}); return }
    userID, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"}); return }
    var containerID, nodeHostname string
    if err := h.db.QueryRow(`SELECT c.id, n.hostname FROM users u JOIN containers c ON u.container_id=c.id JOIN nodes n ON c.node_id=n.id WHERE u.id=$1`, userID).Scan(&containerID, &nodeHostname); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "user has no container"}); return
    }
    slaveURL := fmt.Sprintf("http://%s:8081/api/cli/install", nodeHostname)
    body := map[string]string{"container_id": containerID}
    bb, _ := json.Marshal(body)
    resp, err := http.Post(slaveURL, "application/json", bytes.NewBuffer(bb))
    if err != nil { c.JSON(http.StatusBadGateway, gin.H{"error": "node unreachable"}); return }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 { b, _ := io.ReadAll(resp.Body); c.JSON(resp.StatusCode, gin.H{"error": strings.TrimSpace(string(b))}); return }
    c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) AdminListJobs(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    if !user.IsAdmin { c.JSON(http.StatusForbidden, gin.H{"error": "admin required"}); return }
    limit := 50
    if l := c.Query("limit"); l != "" {
        if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 { limit = v }
    }
    rows, err := h.db.Query(`SELECT id, type, status, error, created_at, updated_at FROM jobs ORDER BY id DESC LIMIT $1`, limit)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"}); return }
        defer rows.Close()
    type jobRow struct {
        ID int `json:"id"`
        Type string `json:"type"`
        Status string `json:"status"`
        Error *string `json:"error"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
    }
    var out []jobRow
        for rows.Next() {
        var j jobRow
        if err := rows.Scan(&j.ID, &j.Type, &j.Status, &j.Error, &j.CreatedAt, &j.UpdatedAt); err == nil {
            out = append(out, j)
        }
    }
    c.JSON(http.StatusOK, gin.H{"jobs": out})
}

func (h *Handler) AdminGetJob(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    if !user.IsAdmin { c.JSON(http.StatusForbidden, gin.H{"error": "admin required"}); return }
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"}); return }
    var j struct {
        ID int `json:"id"`
        Type string `json:"type"`
        Status string `json:"status"`
        Error *string `json:"error"`
        Result *string `json:"result"`
        CreatedAt time.Time `json:"created_at"`
        UpdatedAt time.Time `json:"updated_at"`
    }
    var resultBytes, errStr *string
    err = h.db.QueryRow(`SELECT id, type, status, result, error, created_at, updated_at FROM jobs WHERE id=$1`, id).Scan(&j.ID, &j.Type, &j.Status, &resultBytes, &errStr, &j.CreatedAt, &j.UpdatedAt)
    if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "job not found"}); return }
    j.Result = resultBytes
    j.Error = errStr
    c.JSON(http.StatusOK, j)
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
	_, _ = h.db.Exec("DELETE FROM subdomains WHERE user_id = $1", userID)

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
				// proceed anyway
			} else {
            defer resp.Body.Close()
				if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
                body, _ := io.ReadAll(resp.Body)
                log.Printf("rid=%s DeleteUser: slave delete returned %d: %s", requestID, resp.StatusCode, string(body))
					// proceed anyway, the container is gone (probably)
            }
			}
			// this is assuming that the container was deleted by the slave, however this does need to be handled better/more gracefully
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
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"}); return
	}
	ctoken := hex.EncodeToString(tokenBytes)
	_, err = h.db.Exec(`
		INSERT INTO containers (id, user_id, node_id, name, status, ip_address, ssh_port, memory_mb, cpu_cores, storage_gb, container_token)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, containerID, req.UserID, req.NodeID, containerInfo["Name"], "RUNNING", 
		containerInfo["IP"], containerInfo["SSHPort"], 4096, 4, 15, ctoken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store container info"})
		return
	}
	_, err = h.db.Exec("UPDATE users SET container_id = $1 WHERE id = $2", containerID, req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}
	if err := h.db.QueryRow("SELECT hostname FROM nodes WHERE id = $1", req.NodeID).Scan(&nodeHostname); err == nil {
		slaveURL := fmt.Sprintf("http://%s:8081", nodeHostname)
		payload := map[string]string{"container_id": containerID, "token": ctoken}
		bb, _ := json.Marshal(payload)
		go http.Post(slaveURL+"/api/cli/token", "application/json", bytes.NewBuffer(bb))
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
		NodeToken   string `json:"node_token" binding:"required"`
		ContainerID string `json:"container_id" binding:"required"`
		Status      string `json:"status" binding:"required"`
		Timestamp   int64  `json:"timestamp"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	var nodeID int
	err := h.db.QueryRow("SELECT id FROM nodes WHERE token = $1", req.NodeToken).Scan(&nodeID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid node token"})
		return
	}
	
	_, err = h.db.Exec("UPDATE containers SET status = $1, updated_at = NOW() WHERE id = $2", req.Status, containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update container status"})
		return
	}
	
	log.Printf("Updated container %s status to %s", containerID, req.Status)
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
    // Update online + container count
    // containers may be []object; we just count length
    count := 0
    switch v := req.Containers.(type) {
    case []interface{}:
        count = len(v)
    }
    _, err := h.db.Exec(`
        UPDATE nodes SET is_online = true, last_seen = NOW(), updated_at = NOW() WHERE token = $1
    `, req.NodeToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid node token"})
		return
	}
	
    c.JSON(http.StatusOK, gin.H{"message": "heartbeat received", "containers": count})
}

func (h *Handler) GetContainerShell(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	if user.ContainerID == nil || *user.ContainerID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no container"})
		return
	}
	var nodeHostname string
	err := h.db.QueryRow(`SELECT n.hostname FROM nodes n JOIN containers c ON c.node_id=n.id WHERE c.id=$1`, *user.ContainerID).Scan(&nodeHostname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "node lookup failed"})
		return
	}
	slaveURL := fmt.Sprintf("http://%s:8081/api/control/containers/%s", nodeHostname, *user.ContainerID)
	body := map[string]interface{}{"action": "get_shell", "username": user.Username}
	bb, _ := json.Marshal(body)
	resp, err := http.Post(slaveURL, "application/json", bytes.NewBuffer(bb))
	if err != nil { c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()}); return }
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", b)
}

func (h *Handler) SetContainerShell(c *gin.Context) {
	user := c.MustGet("user").(*models.User)
	if user.ContainerID == nil || *user.ContainerID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no container"})
		return
	}
	var req struct{ Shell string `json:"shell" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if req.Shell != "bash" && req.Shell != "zsh" && req.Shell != "fish" { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shell"}); return }
	var nodeHostname string
	err := h.db.QueryRow(`SELECT n.hostname FROM nodes n JOIN containers c ON c.node_id=n.id WHERE c.id=$1`, *user.ContainerID).Scan(&nodeHostname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "node lookup failed"})
		return
	}
	slaveURL := fmt.Sprintf("http://%s:8081/api/control/containers/%s", nodeHostname, *user.ContainerID)
	body := map[string]interface{}{"action": "set_shell", "shell": req.Shell, "username": user.Username}
	bb, _ := json.Marshal(body)
	resp, err := http.Post(slaveURL, "application/json", bytes.NewBuffer(bb))
	if err != nil { c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()}); return }
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", b)
}

func (h *Handler) ContainerStart(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    if user.ContainerID == nil || *user.ContainerID == "" {
        c.JSON(http.StatusNotFound, gin.H{"error": "no container"})
        return
    }
    var nodeHostname string
    if err := h.db.QueryRow(`SELECT n.hostname FROM nodes n JOIN containers c ON c.node_id=n.id WHERE c.id=$1`, *user.ContainerID).Scan(&nodeHostname); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "node lookup failed"})
        return
    }
    slaveURL := fmt.Sprintf("http://%s:8081/api/control/containers/%s", nodeHostname, *user.ContainerID)
    body := map[string]interface{}{"action": "start"}
    bb, _ := json.Marshal(body)
    resp, err := http.Post(slaveURL, "application/json", bytes.NewBuffer(bb))
    if err != nil { c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()}); return }
    defer resp.Body.Close()
    b, _ := io.ReadAll(resp.Body)
    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        c.JSON(http.StatusOK, gin.H{"ok": true})
        return
    }
    c.Data(resp.StatusCode, "application/json", b)
}

func (h *Handler) ContainerStop(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    if user.ContainerID == nil || *user.ContainerID == "" {
        c.JSON(http.StatusNotFound, gin.H{"error": "no container"})
        return
    }
    var nodeHostname string
    if err := h.db.QueryRow(`SELECT n.hostname FROM nodes n JOIN containers c ON c.node_id=n.id WHERE c.id=$1`, *user.ContainerID).Scan(&nodeHostname); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "node lookup failed"})
        return
    }
    slaveURL := fmt.Sprintf("http://%s:8081/api/control/containers/%s", nodeHostname, *user.ContainerID)
    body := map[string]interface{}{"action": "stop"}
    bb, _ := json.Marshal(body)
    resp, err := http.Post(slaveURL, "application/json", bytes.NewBuffer(bb))
    if err != nil { c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()}); return }
    defer resp.Body.Close()
    b, _ := io.ReadAll(resp.Body)
    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        c.JSON(http.StatusOK, gin.H{"ok": true})
        return
    }
    c.Data(resp.StatusCode, "application/json", b)
}

func (h *Handler) ContainerRestart(c *gin.Context) {
    user := c.MustGet("user").(*models.User)
    if user.ContainerID == nil || *user.ContainerID == "" {
        c.JSON(http.StatusNotFound, gin.H{"error": "no container"})
        return
    }
    var nodeHostname string
    if err := h.db.QueryRow(`SELECT n.hostname FROM nodes n JOIN containers c ON c.node_id=n.id WHERE c.id=$1`, *user.ContainerID).Scan(&nodeHostname); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "node lookup failed"})
        return
    }
    slaveURL := fmt.Sprintf("http://%s:8081/api/control/containers/%s", nodeHostname, *user.ContainerID)
    // stop
    sb, _ := json.Marshal(map[string]interface{}{"action": "stop"})
    _, _ = http.Post(slaveURL, "application/json", bytes.NewBuffer(sb))
    time.Sleep(1 * time.Second)
    // start (this should be pretty easy to tell though)
    rb, _ := json.Marshal(map[string]interface{}{"action": "start"})
    resp, err := http.Post(slaveURL, "application/json", bytes.NewBuffer(rb))
    if err != nil { c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()}); return }
    defer resp.Body.Close()
    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        c.JSON(http.StatusOK, gin.H{"ok": true})
        return
    }
    b, _ := io.ReadAll(resp.Body)
    c.Data(resp.StatusCode, "application/json", b)
}

func (h *Handler) NotFound(c *gin.Context) {
	var user *models.User
	if sessionID, err := c.Cookie("session"); err == nil {
		if u, err := h.auth.GetUserBySession(sessionID); err == nil && u != nil {
			user = u
		}
	}
	
	c.Status(http.StatusNotFound)
	h.inertia(c, "NotFound", gin.H{
		"user": user,
		"path": c.Request.URL.Path,
	})
}
