package slave

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/den/internal/container"
	"github.com/joho/godotenv"
)

type Config struct {
	MasterURL   string `json:"master_url"`
	NodeToken   string `json:"node_token"`
	NodeID      string `json:"node_id"`
	MaxMemoryMB int    `json:"max_memory_mb"`
	MaxCPUCores int    `json:"max_cpu_cores"`
	MaxStorage  int    `json:"max_storage_gb"`
}

type Slave struct {
	config    *Config
	manager   *container.Manager
	client    *http.Client
	ctx       context.Context
	cancel    context.CancelFunc
}

func Run() error {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found")
	}
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	manager, err := container.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize container manager: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	slave := &Slave{
		config:  config,
		manager: manager,
		client:  &http.Client{Timeout: 30 * time.Second},
		ctx:     ctx,
		cancel:  cancel,
	}

	if err := slave.registerWithMaster(); err != nil {
		return fmt.Errorf("failed to register with master: %w", err)
	}

	go slave.heartbeat()

	go slave.monitorContainers()

	go slave.startAPIServer()

	log.Printf("den slave started (node id: %s)", config.NodeID)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down slave...")

	cancel()
	return nil
}

func loadConfig() (*Config, error) {
	configPath := os.Getenv("DEN_SLAVE_CONFIG")
	if configPath == "" {
		configPath = "/etc/den/slave.json"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.MaxMemoryMB == 0 {
		config.MaxMemoryMB = 4096
	}
	if config.MaxCPUCores == 0 {
		config.MaxCPUCores = 4
	}
	if config.MaxStorage == 0 {
		config.MaxStorage = 15
	}

	return &config, nil
}

func (s *Slave) registerWithMaster() error {
	payload := map[string]interface{}{
		"node_id":        s.config.NodeID,
		"node_token":     s.config.NodeToken,
		"max_memory_mb":  s.config.MaxMemoryMB,
		"max_cpu_cores":  s.config.MaxCPUCores,
		"max_storage_gb": s.config.MaxStorage,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := s.client.Post(
		s.config.MasterURL+"/api/nodes/register",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status: %d", resp.StatusCode)
	}

	log.Println("successfully registered with master")
	return nil
}

func (s *Slave) heartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.sendHeartbeat(); err != nil {
				log.Printf("Heartbeat failed: %v", err)
			}
		}}}
func (s *Slave) sendHeartbeat() error {
	containers, err := s.manager.ListContainers()
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"node_id":    s.config.NodeID,
		"node_token": s.config.NodeToken,
		"containers": containers,
		"timestamp":  time.Now().Unix(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := s.client.Post(
		s.config.MasterURL+"/api/nodes/heartbeat",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
func (s *Slave) monitorContainers() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.updateContainerStatus(); err != nil {
				log.Printf("container monitoring failed: %v", err)
			}
		}
	}
}

func (s *Slave) updateContainerStatus() error {
	containers, err := s.manager.ListContainers()
	if err != nil {
		return err
	}
	for _, container := range containers {
		status, err := s.manager.GetContainerStatus(container.ID)
		if err != nil {
			log.Printf("failed to get status for container %s: %v", container.ID, err)
			continue
		}
		payload := map[string]interface{}{
			"node_token":   s.config.NodeToken,
			"container_id": container.ID,
			"status":       status,
			"timestamp":    time.Now().Unix(),
		}

		data, err := json.Marshal(payload)
		if err != nil {
			continue
		}

		resp, err := s.client.Post(
			s.config.MasterURL+"/api/containers/"+container.ID+"/status",
			"application/json",
			bytes.NewBuffer(data),
		)
		if err != nil {
			log.Printf("failed to report status for container %s: %v", container.ID, err)
			continue
		}
		resp.Body.Close()
	}

	return nil
}

func (s *Slave) startAPIServer() {
	log.Println("boooo i haven't implemented this yet")
}
