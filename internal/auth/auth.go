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
	"time"

	"github.com/den/internal/database"
	"github.com/den/internal/models"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	db             *database.DB
	slackClientID     string
	slackClientSecret string
	baseURL           string
}

type SlackUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	Email    string `json:"email"`
}

type SlackAuthResponse struct {
	OK     bool      `json:"ok"`
	User   SlackUser `json:"user"`
	TeamID string    `json:"team_id"`
}

func NewService(db *database.DB) *Service {
	return &Service{
		db:                db,
		slackClientID:     os.Getenv("SLACK_CLIENT_ID"),
		slackClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
		baseURL:           getEnvDefault("BASE_URL", "http://localhost:8080"),
	}
}
func (s *Service) GetAuthURL(state string) string {
	params := url.Values{}
	params.Add("client_id", s.slackClientID)
	params.Add("scope", "identity.basic,identity.email")
	params.Add("redirect_uri", s.baseURL+"/auth/callback")
	params.Add("state", state)
	
	return "https://slack.com/oauth/authorize?" + params.Encode()
}

func (s *Service) HandleCallback(code string) (*models.User, error) {
	token, err := s.exchangeCode(code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	slackUser, err := s.getSlackUser(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	user, err := s.createOrUpdateUser(slackUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create/update user: %w", err)
	}
	return user, nil
}

func (s *Service) exchangeCode(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", s.slackClientID)
	data.Set("client_secret", s.slackClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", s.baseURL+"/auth/callback")

	resp, err := http.PostForm("https://slack.com/api/oauth.access", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if !result["ok"].(bool) {
		return "", fmt.Errorf("slack auth failed: %v", result["error"])
	}

	return result["access_token"].(string), nil
}

func (s *Service) getSlackUser(token string) (*SlackUser, error) {
	req, err := http.NewRequest("GET", "https://slack.com/api/users.identity", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result SlackAuthResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("failed to get user identity")
	}

	return &result.User, nil
}

func (s *Service) createOrUpdateUser(slackUser *SlackUser) (*models.User, error) {
	var user models.User
    err := s.db.QueryRow(`
        SELECT id, slack_id, username, email, display_name, is_admin, container_id, 
               ssh_public_key, agreed_to_tos, agreed_to_privacy, tos_questions, created_at, updated_at
		FROM users WHERE slack_id = $1
	`, slackUser.ID).Scan(
        &user.ID, &user.SlackID, &user.Username, &user.Email, &user.DisplayName,
        &user.IsAdmin, &user.ContainerID, &user.SSHPublicKey, &user.AgreedToTOS, &user.AgreedToPrivacy, pq.Array(&user.TOSQuestions), &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		username := generateUsername(slackUser.Name)
		
        err = s.db.QueryRow(`
            INSERT INTO users (slack_id, username, email, display_name)
            VALUES ($1, $2, $3, $4)
            RETURNING id, slack_id, username, email, display_name, is_admin, container_id,
                      ssh_public_key, agreed_to_tos, agreed_to_privacy, tos_questions, created_at, updated_at
        `, slackUser.ID, username, slackUser.Email, slackUser.RealName).Scan(
            &user.ID, &user.SlackID, &user.Username, &user.Email, &user.DisplayName,
            &user.IsAdmin, &user.ContainerID, &user.SSHPublicKey, &user.AgreedToTOS, &user.AgreedToPrivacy, pq.Array(&user.TOSQuestions), &user.CreatedAt, &user.UpdatedAt,
        )
		
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else {
		_, err = s.db.Exec(`
			UPDATE users SET email = $1, display_name = $2, updated_at = NOW()
			WHERE id = $3
		`, slackUser.Email, slackUser.RealName, user.ID)
		
		if err != nil {
			return nil, err
		}
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
    err := s.db.QueryRow(`
        SELECT u.id, u.slack_id, u.username, u.email, u.display_name, u.is_admin, 
               u.container_id, u.ssh_public_key, u.agreed_to_tos, u.agreed_to_privacy, u.tos_questions, u.created_at, u.updated_at
		FROM users u
		JOIN sessions s ON u.id = s.user_id
		WHERE s.id = $1 AND s.expires_at > NOW()
	`, sessionID).Scan(
        &user.ID, &user.SlackID, &user.Username, &user.Email, &user.DisplayName,
        &user.IsAdmin, &user.ContainerID, &user.SSHPublicKey, &user.AgreedToTOS, &user.AgreedToPrivacy, pq.Array(&user.TOSQuestions), &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

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

func generateUsername(name string) string {
	// just fucking yoink it from slack
	username := ""
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			username += string(char)
		} else if char >= 'A' && char <= 'Z' {
			username += string(char + 32) // lowercase this bish
		}
	}
	
	if username == "" {
		username = "user"
	}
	
	// people probably have the same user so woooo suffix
	suffix := make([]byte, 4)
	rand.Read(suffix)
	username += hex.EncodeToString(suffix)
	
	return username
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
