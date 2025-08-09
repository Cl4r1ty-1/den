package models

import (
	"time"
)

type User struct {
	ID           int       `json:"id" db:"id"`
	SlackID      string    `json:"slack_id" db:"slack_id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	DisplayName  string    `json:"display_name" db:"display_name"`
	IsAdmin      bool      `json:"is_admin" db:"is_admin"`
	ContainerID  *string   `json:"container_id" db:"container_id"`
	SSHPassword  *string   `json:"-" db:"ssh_password"`
	SSHPublicKey *string   `json:"ssh_public_key" db:"ssh_public_key"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
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
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type Subdomain struct {
	ID          int       `json:"id" db:"id"`
	UserID      int       `json:"user_id" db:"user_id"`
	Subdomain   string    `json:"subdomain" db:"subdomain"`
	TargetPort  int       `json:"target_port" db:"target_port"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
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
