package models

import (
	"time"
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type User struct {
	ID              int        `json:"id" db:"id"`
	GitHubID        string     `json:"github_id" db:"github_id"`
	Username        string     `json:"username" db:"username"`
	Email           string     `json:"email" db:"email"`
	DisplayName     string     `json:"display_name" db:"display_name"`
	IsAdmin         bool       `json:"is_admin" db:"is_admin"`
	ContainerID     *string    `json:"container_id" db:"container_id"`
	SSHPassword     *string    `json:"-" db:"ssh_password"`
	SSHPublicKey    *string    `json:"ssh_public_key" db:"ssh_public_key"`
	AgreedToTOS     bool       `json:"agreed_to_tos" db:"agreed_to_tos"`
	AgreedToPrivacy bool       `json:"agreed_to_privacy" db:"agreed_to_privacy"`
	TOSQuestions    []int      `json:"tos_questions" db:"tos_questions"`
	ApprovalStatus  string     `json:"approval_status" db:"approval_status"`
	ApprovedBy      *int       `json:"approved_by" db:"approved_by"`
	ApprovedAt      *time.Time `json:"approved_at" db:"approved_at"`
	RejectionReason *string    `json:"rejection_reason" db:"rejection_reason"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type Question struct {
    ID            int       `json:"id" db:"id"`
    Prompt        string    `json:"prompt" db:"prompt"`
    CorrectAnswer string    `json:"-" db:"correct_answer"`
    IsActive      bool      `json:"is_active" db:"is_active"`
    CreatedAt     time.Time `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type Node struct {
	ID             int       `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Hostname       string    `json:"hostname" db:"hostname"`
	PublicHostname *string   `json:"public_hostname" db:"public_hostname"`
	Token          string    `json:"token" db:"token"`
	MaxMemoryMB    int       `json:"max_memory_mb" db:"max_memory_mb"`
	MaxCPUCores    int       `json:"max_cpu_cores" db:"max_cpu_cores"`
	MaxStorageGB   int       `json:"max_storage_gb" db:"max_storage_gb"`
	IsOnline       bool      `json:"is_online" db:"is_online"`
	LastSeen       *time.Time `json:"last_seen" db:"last_seen"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type Container struct {
	ID             string    `json:"id" db:"id"`
	UserID         int       `json:"user_id" db:"user_id"`
	NodeID         int       `json:"node_id" db:"node_id"`
	Name           string    `json:"name" db:"name"`
	Status         string    `json:"status" db:"status"`
	IPAddress      *string   `json:"ip_address" db:"ip_address"`
	SSHPort        int       `json:"ssh_port" db:"ssh_port"`
	MemoryMB       int       `json:"memory_mb" db:"memory_mb"`
	CPUCores       int       `json:"cpu_cores" db:"cpu_cores"`
	StorageGB      int       `json:"storage_gb" db:"storage_gb"`
	AllocatedPorts []int     `json:"allocated_ports" db:"allocated_ports"`
	ContainerToken string    `json:"container_token" db:"container_token"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type Subdomain struct {
	ID            int       `json:"id" db:"id"`
	UserID        int       `json:"user_id" db:"user_id"`
	Subdomain     string    `json:"subdomain" db:"subdomain"`
	TargetPort    int       `json:"target_port" db:"target_port"`
	SubdomainType string    `json:"subdomain_type" db:"subdomain_type"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type PortMapping struct {
	ID          int       `json:"id" db:"id"`
	ContainerID string    `json:"container_id" db:"container_id"`
	InternalPort int      `json:"internal_port" db:"internal_port"`
	ExternalPort int      `json:"external_port" db:"external_port"`
	Protocol    string    `json:"protocol" db:"protocol"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Session struct {
	ID        string    `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Export struct {
    ID          int       `json:"id" db:"id"`
    UserID      int       `json:"user_id" db:"user_id"`
    ContainerID string    `json:"container_id" db:"container_id"`
    ObjectKey   string    `json:"object_key" db:"object_key"`
    Status      string    `json:"status" db:"status"`
    SizeBytes   *int64    `json:"size_bytes" db:"size_bytes"`
    ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
    RequestedBy *int      `json:"requested_by" db:"requested_by"`
    Error       *string   `json:"error" db:"error"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Job struct {
    ID          int         `json:"id" db:"id"`
    Type        string      `json:"type" db:"type"`
    Status      string      `json:"status" db:"status"`
    Payload     []byte      `json:"payload" db:"payload"`
    Result      []byte      `json:"result" db:"result"`
    Error       *string     `json:"error" db:"error"`
    RunAfter    time.Time   `json:"run_after" db:"run_after"`
    Attempts    int         `json:"attempts" db:"attempts"`
    MaxAttempts int         `json:"max_attempts" db:"max_attempts"`
    CreatedAt   time.Time   `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	
	return json.Unmarshal(bytes, j)
}

type VerificationSession struct {
    ID            int       `json:"id" db:"id"`
    UserID        int       `json:"user_id" db:"user_id"`
    SessionID     string    `json:"session_id" db:"session_id"`
    SessionNumber *int      `json:"session_number" db:"session_number"`
    SessionToken  *string   `json:"session_token" db:"session_token"`
    VendorData    *string   `json:"vendor_data" db:"vendor_data"`
    WorkflowID    string    `json:"workflow_id" db:"workflow_id"`
    VerificationURL *string `json:"verification_url" db:"verification_url"`
    Status        string    `json:"status" db:"status"`
    Decision      JSONB     `json:"decision" db:"decision"`
    CreatedAt     time.Time `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}
