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
    "os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/den/internal/container"
	"github.com/joho/godotenv"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	MasterURL      string `json:"master_url"`
	NodeToken      string `json:"node_token"`
	NodeID         string `json:"node_id"`
	PublicHostname string `json:"public_hostname"`
	MaxMemoryMB    int    `json:"max_memory_mb"`
	MaxCPUCores    int    `json:"max_cpu_cores"`
	MaxStorage     int    `json:"max_storage_gb"`
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
	manager, err := container.NewManager(config.PublicHostname)
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
    containerGauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: "den_slave_containers", Help: "Number of containers on this node"})
    prometheus.MustRegister(containerGauge)

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if err := s.updateContainerStatus(); err != nil {
				log.Printf("container monitoring failed: %v", err)
			}
            if list, err := s.manager.ListContainers(); err == nil { containerGauge.Set(float64(len(list))) }
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
	mux := http.NewServeMux()
    mux.Handle("/metrics", promhttp.Handler())
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request){ w.WriteHeader(http.StatusOK); w.Write([]byte("ok")) })
	
	// fuck this shit i'm out
	mux.HandleFunc("/api/containers", s.handleCreateContainer)
    mux.HandleFunc("/api/containers/", s.handleContainerOperations)
    mux.HandleFunc("/api/containers-stats/", s.handleContainerStats)
    mux.HandleFunc("/api/control/containers/", s.handleControlContainer)
    mux.HandleFunc("/api/export", s.handleExportContainer)
	mux.HandleFunc("/api/ports", s.handlePortMapping)
    mux.HandleFunc("/api/ports/new", s.handleAllocateNewPort)
	mux.HandleFunc("/api/ssh", s.handleSSHSetup)
	
	server := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}
	
    log.Println("slave api server listening on :8081")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("slave api server failed: %v", err)
	}
}
func (s *Slave) handleExportContainer(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
    var req struct {
        ContainerID string `json:"container_id"`
        PutURL      string `json:"put_url"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, "invalid request", http.StatusBadRequest); return }
    if req.ContainerID == "" || req.PutURL == "" { http.Error(w, "missing fields", http.StatusBadRequest); return }

    sanitized := strings.ReplaceAll(req.ContainerID, "/", "-")
    ts := time.Now().Unix()
    tmpPath := fmt.Sprintf("/tmp/%s-%d.tar.gz", sanitized, ts)
    exportCmd := exec.Command("lxc", "export", req.ContainerID, tmpPath)
    var exportOut bytes.Buffer
    exportCmd.Stdout = &exportOut
    exportCmd.Stderr = &exportOut
    if err := exportCmd.Run(); err != nil {
        http.Error(w, "export failed: "+exportOut.String(), http.StatusInternalServerError); return
    }
    workDir := fmt.Sprintf("/tmp/export-%s-%d", sanitized, ts)
    _ = os.RemoveAll(workDir)
    if err := os.MkdirAll(workDir, 0o755); err != nil {
        http.Error(w, "prep failed", http.StatusInternalServerError); return
    }
    untar := exec.Command("tar", "-xzf", tmpPath, "-C", workDir)
    var untarOut bytes.Buffer
    untar.Stdout = &untarOut
    untar.Stderr = &untarOut
    if err := untar.Run(); err != nil {
        os.RemoveAll(workDir); os.Remove(tmpPath)
        http.Error(w, "unpack failed: "+untarOut.String(), http.StatusInternalServerError); return
    }
    _ = exec.Command("chmod", "-R", "a+rX", workDir).Run()
    readableTar := fmt.Sprintf("/tmp/%s-%d.readable.tar.gz", sanitized, ts)
    retar := exec.Command("bash", "-lc", fmt.Sprintf("tar -czf '%s' -C '%s' .", readableTar, workDir))
    var retarOut bytes.Buffer
    retar.Stdout = &retarOut
    retar.Stderr = &retarOut
    if err := retar.Run(); err != nil {
        os.RemoveAll(workDir); os.Remove(tmpPath)
        http.Error(w, "repack failed: "+retarOut.String(), http.StatusInternalServerError); return
    }
    defer os.Remove(tmpPath)
    defer os.RemoveAll(workDir)
    defer os.Remove(readableTar)

    uploadPath := readableTar

    curl := exec.Command("curl", "-sS", "--fail", "-X", "PUT", "-H", "Content-Type: application/octet-stream", "--upload-file", uploadPath, req.PutURL)
    var curlOut bytes.Buffer
    curl.Stdout = &curlOut
    curl.Stderr = &curlOut
    if err := curl.Run(); err != nil {
        http.Error(w, "upload failed: "+curlOut.String(), http.StatusBadGateway); return
    }
    fi, _ := os.Stat(uploadPath)
    _ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "size": func() int64 { if fi!=nil { return fi.Size() } ; return 0 }()})
}

func (s *Slave) handleCreateContainer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		UserID   int    `json:"user_id"`
		Username string `json:"username"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	
	log.Printf("creating container for user %d (%s)", req.UserID, req.Username)
	container, err := s.manager.CreateContainer(req.UserID, req.Username)
	if err != nil {
		log.Printf("container creation failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("container created successfully: %+v", container)
	
	json.NewEncoder(w).Encode(container)
}

func (s *Slave) handleContainerOperations(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	
	containerID := parts[3]
	
	switch r.Method {
	case http.MethodGet:
		status, err := s.manager.GetContainerStatus(containerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": status})
		
	case http.MethodDelete:
		if err := s.manager.DeleteContainer(containerID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Slave) handleContainerStats(w http.ResponseWriter, r *http.Request) {
    parts := strings.Split(r.URL.Path, "/")
    if len(parts) < 4 {
        http.Error(w, "invalid path", http.StatusBadRequest)
        return
    }
    containerID := parts[3]
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    stats, err := s.manager.GetContainerStats(containerID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}
func (s *Slave) handleControlContainer(w http.ResponseWriter, r *http.Request) {
    parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
    if len(parts) < 5 {
        http.Error(w, "invalid path", http.StatusBadRequest)
        return
    }
    containerID := parts[4]
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    var req struct { Action string `json:"action"` }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    switch strings.ToLower(req.Action) {
    case "stop", "pause":
        if err := s.manager.StopContainer(containerID); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    case "start", "resume":
        if err := s.manager.StartContainer(containerID); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    default:
        http.Error(w, "unknown action", http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusOK)
}

func (s *Slave) handlePortMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		ContainerID string `json:"container_id"`
		InternalPort int   `json:"internal_port"`
		ExternalPort int   `json:"external_port"`
		Protocol    string `json:"protocol"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	
	if err := s.manager.MapPort(req.ContainerID, req.InternalPort, req.ExternalPort, req.Protocol); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
}

func (s *Slave) handleAllocateNewPort(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }
    var req struct {
        ContainerID string `json:"container_id"`
        Protocol    string `json:"protocol"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    if req.Protocol == "" { req.Protocol = "tcp" }
    port, err := s.manager.FindAvailablePort()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    if err := s.manager.MapPort(req.ContainerID, port, port, req.Protocol); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(map[string]int{"port": port})
}

func (s *Slave) handleSSHSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		ContainerID string `json:"container_id"`
		Username    string `json:"username"`
		PublicKey   string `json:"public_key"`
		Password    string `json:"password"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	
	var err error
	if req.PublicKey != "" {
		log.Printf("setting up ssh key access for user %s in container %s", req.Username, req.ContainerID)
		err = s.manager.SetupSSHAccess(req.ContainerID, req.Username, req.PublicKey)
	} else if req.Password != "" {
		log.Printf("setting up ssh password access for user %s in container %s", req.Username, req.ContainerID)
		err = s.manager.SetupSSHPassword(req.ContainerID, req.Username, req.Password)
	} else {
		log.Printf("ssh setup request missing both public key and password")
		http.Error(w, "either public_key or password must be provided", http.StatusBadRequest)
		return
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusOK)
}
