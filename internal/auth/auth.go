package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/den/internal/database"
	"github.com/den/internal/models"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db                 *database.DB
	githubClientID     string
	githubClientSecret string
	baseURL             string
}

type GitHubUser struct {
	ID    int     `json:"id"`
	Login string  `json:"login"`
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

type githubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func NewService(db *database.DB) *Service {
	return &Service{
		db:                db,
		githubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		githubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		baseURL:           getEnvDefault("BASE_URL", "http://localhost:8080"),
	}
}
func (s *Service) GetAuthURL(state string) string {
	params := url.Values{}
	params.Add("client_id", s.githubClientID)
	params.Add("scope", "read:user user:email")
	params.Add("redirect_uri", s.baseURL+"/auth/callback")
	params.Add("state", state)
	params.Add("allow_signup", "true")
	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

func (s *Service) HandleCallback(code string) (*models.User, error) {
	token, err := s.exchangeCode(code)
	if err != nil { return nil, fmt.Errorf("failed to exchange code: %w", err) }
	ghUser, err := s.getGitHubUser(token)
	if err != nil { return nil, fmt.Errorf("failed to get user info: %w", err) }
	user, err := s.createOrUpdateUser(ghUser)
	if err != nil { return nil, fmt.Errorf("failed to create/update user: %w", err) }
	return user, nil
}

func (s *Service) exchangeCode(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", s.githubClientID)
	data.Set("client_secret", s.githubClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", s.baseURL+"/auth/callback")

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil { return "", err }
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil { return "", err }
	var tr githubTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil { return "", err }
	if tr.AccessToken == "" { return "", fmt.Errorf("github auth failed: empty access token") }
	return tr.AccessToken, nil
}

func (s *Service) getGitHubUser(token string) (*GitHubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil { return nil, err }
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil { return nil, err }
	var gh GitHubUser
	if err := json.Unmarshal(body, &gh); err != nil { return nil, err }
	if gh.Email == nil || *gh.Email == "" {
		req2, err := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
		if err == nil {
			req2.Header.Set("Authorization", "Bearer "+token)
			req2.Header.Set("Accept", "application/vnd.github+json")
			resp2, err2 := client.Do(req2)
			if err2 == nil {
				defer resp2.Body.Close()
				eb, _ := io.ReadAll(resp2.Body)
				var emails []githubEmail
				if err := json.Unmarshal(eb, &emails); err == nil {
					for _, e := range emails {
						if e.Primary && e.Verified { em := e.Email; gh.Email = &em; break }
					}
					if gh.Email == nil && len(emails) > 0 { em := emails[0].Email; gh.Email = &em }
				}
			}
		}
	}
	return &gh, nil
}

func (s *Service) createOrUpdateUser(ghUser *GitHubUser) (*models.User, error) {
	var user models.User
	var tosQuestions pq.Int64Array
	err := s.db.QueryRow(`
		SELECT id, github_id, username, email, display_name, is_admin, container_id,
		       ssh_public_key, agreed_to_tos, agreed_to_privacy, tos_questions, created_at, updated_at
		FROM users WHERE github_id = $1
	`, fmt.Sprintf("%d", ghUser.ID)).Scan(
		&user.ID, &user.GitHubID, &user.Username, &user.Email, &user.DisplayName,
		&user.IsAdmin, &user.ContainerID, &user.SSHPublicKey, &user.AgreedToTOS, &user.AgreedToPrivacy, &tosQuestions, &user.CreatedAt, &user.UpdatedAt,
	)
	user.TOSQuestions = make([]int, len(tosQuestions))
	for i, v := range tosQuestions { user.TOSQuestions[i] = int(v) }
	if err == sql.ErrNoRows {
		username := ghUser.Login
		if username == "" { username = "user" }
		
		displayName := ""
		if ghUser.Name != nil { displayName = *ghUser.Name }
		if displayName == "" { displayName = username }
		
		email := ""
		if ghUser.Email != nil { email = *ghUser.Email }
		var tos pq.Int64Array
		err = s.db.QueryRow(`
			INSERT INTO users (github_id, username, email, display_name)
			VALUES ($1, $2, $3, $4)
			RETURNING id, github_id, username, email, display_name, is_admin, container_id,
			          ssh_public_key, agreed_to_tos, agreed_to_privacy, tos_questions, created_at, updated_at
		`, fmt.Sprintf("%d", ghUser.ID), username, email, displayName).Scan(
			&user.ID, &user.GitHubID, &user.Username, &user.Email, &user.DisplayName,
			&user.IsAdmin, &user.ContainerID, &user.SSHPublicKey, &user.AgreedToTOS, &user.AgreedToPrivacy, &tos, &user.CreatedAt, &user.UpdatedAt,
		)
		user.TOSQuestions = make([]int, len(tos))
		for i, v := range tos { user.TOSQuestions[i] = int(v) }
		if err != nil { return nil, err }
	} else if err != nil {
		return nil, err
	} else {
		displayName := ""
		if ghUser.Name != nil { displayName = *ghUser.Name }
		if displayName == "" { displayName = ghUser.Login }
		email := ""
		if ghUser.Email != nil { email = *ghUser.Email }
		_, err = s.db.Exec(`
			UPDATE users SET email = $1, display_name = $2, updated_at = NOW()
			WHERE id = $3
		`, email, displayName, user.ID)
		if err != nil { return nil, err }
		user.Email = email
		user.DisplayName = displayName
	}
	return &user, nil
}

func (s *Service) CreateSession(userID int) (string, error) {
	sessionID := generateSessionID()
	expiresAt := time.Now().Add(24 * time.Hour * 7)
	_, err := s.db.Exec(`
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, sessionID, userID, expiresAt)

	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func (s *Service) GetUserBySession(sessionID string) (*models.User, error) {
	var user models.User
	var tosQ pq.Int64Array
	err := s.db.QueryRow(`
		SELECT u.id, u.github_id, u.username, u.email, u.display_name, u.is_admin,
		       u.container_id, u.ssh_public_key, u.agreed_to_tos, u.agreed_to_privacy, u.tos_questions, u.created_at, u.updated_at
		FROM users u
		JOIN sessions s ON u.id = s.user_id
		WHERE s.id = $1 AND s.expires_at > NOW()
	`, sessionID).Scan(
		&user.ID, &user.GitHubID, &user.Username, &user.Email, &user.DisplayName,
		&user.IsAdmin, &user.ContainerID, &user.SSHPublicKey, &user.AgreedToTOS, &user.AgreedToPrivacy, &tosQ, &user.CreatedAt, &user.UpdatedAt,
	)
	user.TOSQuestions = make([]int, len(tosQ))
	for i, v := range tosQ { user.TOSQuestions[i] = int(v) }
	if err != nil { return nil, err }
	return &user, nil
}

func (s *Service) SetSSHPassword(userID int, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		UPDATE users SET ssh_password = $1, updated_at = NOW()
		WHERE id = $2
	`, string(hashedPassword), userID)

	return err
}

func (s *Service) SetSSHPublicKey(userID int, publicKey string) error {
	_, err := s.db.Exec(`
		UPDATE users SET ssh_public_key = $1, updated_at = NOW()
		WHERE id = $2
	`, publicKey, userID)

	return err
}


func generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
