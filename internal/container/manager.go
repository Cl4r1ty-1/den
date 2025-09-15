//go:build slave
// +build slave

package container

import (
	"fmt"
	"math/rand"
	"net"
	"encoding/json"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Manager struct {
	defaultMemoryMB  int
	defaultCPUCores  int
	defaultStorageGB int
	publicHostname   string
}

type ContainerInfo struct {
	ID             string
	Name           string
	Status         string
	IP             string
	SSHPort        int
	AllocatedPorts []int
}

type ContainerStats struct {
    CPUUsageNanoseconds   uint64 `json:"cpu_usage_ns"`
    MemoryUsageBytes      uint64 `json:"memory_usage_bytes"`
    MemoryTotalBytes      uint64 `json:"memory_total_bytes"`
    DiskUsageBytes        uint64 `json:"disk_usage_bytes"`
    NetworkRXBytes        uint64 `json:"network_rx_bytes"`
    NetworkTXBytes        uint64 `json:"network_tx_bytes"`
}

func NewManager(publicHostname string) (*Manager, error) {
	return &Manager{
		defaultMemoryMB:  4096,
		defaultCPUCores:  4,
		defaultStorageGB: 15,
		publicHostname:   publicHostname,
	}, nil
}
func (m *Manager) allocateRandomPorts() ([]int, error) {
	const minPort = 20000
	const maxPort = 65535
	const numPorts = 10
	
	allocatedPorts := make([]int, 0, numPorts)
	maxAttempts := 1000
	
	for len(allocatedPorts) < numPorts && maxAttempts > 0 {
		port := rand.Intn(maxPort-minPort+1) + minPort
		if m.portInSlice(port, allocatedPorts) {
			maxAttempts--
			continue
		}
		if m.isPortAvailable(port) {
			allocatedPorts = append(allocatedPorts, port)
		}
		maxAttempts--
	}
	
	if len(allocatedPorts) < numPorts {
		return nil, fmt.Errorf("could not allocate %d ports, only found %d available ports", numPorts, len(allocatedPorts))
	}
	
	return allocatedPorts, nil
}
func (m *Manager) isPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}
func (m *Manager) portInSlice(port int, ports []int) bool {
	for _, p := range ports {
		if p == port {
			return true
		}
	}
	return false
}

func (m *Manager) CreateContainer(userID int, username string) (*ContainerInfo, error) {
	containerName := fmt.Sprintf("den-%s", username)
	cmd := exec.Command("lxc", "launch", "ubuntu:22.04", containerName)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	
	if err := m.configureContainer(containerName); err != nil {
		exec.Command("lxc", "delete", containerName, "--force").Run()
		return nil, fmt.Errorf("failed to configure container: %w", err)
	}
	
	if err := m.waitForContainer(containerName); err != nil {
		exec.Command("lxc", "delete", containerName, "--force").Run()
		return nil, fmt.Errorf("container failed to start: %w", err)
	}
	
	if err := m.setupUserInContainer(containerName, username); err != nil {
		exec.Command("lxc", "delete", containerName, "--force").Run()
		return nil, fmt.Errorf("failed to setup user: %w", err)
	}
	
	info, err := m.getContainerInfo(containerName)
	if err != nil {
		exec.Command("lxc", "delete", containerName, "--force").Run()
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}
	
    info.AllocatedPorts = []int{}
	
	return info, nil
}

func (m *Manager) configureContainer(name string) error {
	configs := [][]string{
		{"lxc", "config", "set", name, "limits.memory", fmt.Sprintf("%dMB", m.defaultMemoryMB)},
		{"lxc", "config", "set", name, "limits.cpu", strconv.Itoa(m.defaultCPUCores)},
		{"lxc", "config", "set", name, "security.nesting", "true"},
		{"lxc", "config", "set", name, "security.privileged", "false"},
	}

	for _, config := range configs {
		cmd := exec.Command(config[0], config[1:]...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run config command %v: %w", config, err)
		}
	}

	return nil
}

func (m *Manager) waitForContainer(name string) error {
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		cmd := exec.Command("lxc", "exec", name, "--", "echo", "ready")
		if err := cmd.Run(); err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("container did not become ready in time")
}

func (m *Manager) setupUserInContainer(containerName, username string) error {
	commands := [][]string{
		{"lxc", "exec", containerName, "--", "useradd", "-m", "-s", "/bin/bash", username},
		{"lxc", "exec", containerName, "--", "usermod", "-aG", "sudo", username},
		{"lxc", "exec", containerName, "--", "mkdir", "-p", fmt.Sprintf("/home/%s/.ssh", username)},
		{"lxc", "exec", containerName, "--", "chown", fmt.Sprintf("%s:%s", username, username), fmt.Sprintf("/home/%s/.ssh", username)},
		{"lxc", "exec", containerName, "--", "chmod", "700", fmt.Sprintf("/home/%s/.ssh", username)},
		{"lxc", "exec", containerName, "--", "apt-get", "update"},
		{"lxc", "exec", containerName, "--", "apt-get", "install", "-y", "openssh-server", "sudo", "curl", "git", "vim", "htop", "nano", "zsh", "fish"},
		{"lxc", "exec", containerName, "--", "systemctl", "enable", "ssh"},
		{"lxc", "exec", containerName, "--", "systemctl", "start", "ssh"},
		{"lxc", "exec", containerName, "--", "bash", "-c", fmt.Sprintf("echo '%s ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/%s", username, username)},
		{"lxc", "exec", containerName, "--", "chmod", "440", fmt.Sprintf("/etc/sudoers.d/%s", username)},
		{"lxc", "exec", containerName, "--", "bash", "-c", fmt.Sprintf("echo -e 'welcome to den! i hope you enjoy your stay here!\\n\\nyour container is running on: %s\\nfor direct port access, use this hostname\\n\\n~ a fuzzy little dog' > /etc/motd", m.getDisplayHostname())},
	}

	for _, cmd := range commands {
		execCmd := exec.Command(cmd[0], cmd[1:]...)
		if err := execCmd.Run(); err != nil {
			return fmt.Errorf("failed to run setup command %v: %w", cmd, err)
		}
	}

	return nil
}

func (m *Manager) getContainerInfo(name string) (*ContainerInfo, error) {
	cmd := exec.Command("lxc", "list", name, "-c", "4", "--format", "csv")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get container IP: %w", err)
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return nil, fmt.Errorf("no IP address found for container")
	}

	re := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(ip)
	if len(matches) < 2 {
		return nil, fmt.Errorf("could not parse IP address: %s", ip)
	}
	ip = matches[1]

	return &ContainerInfo{
		ID:      name,
		Name:    name,
		Status:  "RUNNING",
		IP:      ip,
		SSHPort: 22,
	}, nil
}
func (m *Manager) ListContainers() ([]*ContainerInfo, error) {
	cmd := exec.Command("lxc", "list", "--format", "csv", "-c", "n,s,4")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var containers []*ContainerInfo

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		name := parts[0]
		status := parts[1]
		ip := parts[2]
		if !strings.HasPrefix(name, "den-") {
			continue
		}
		if ip != "" {
			re := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)`)
			matches := re.FindStringSubmatch(ip)
			if len(matches) >= 2 {
				ip = matches[1]
			}
		}
		containers = append(containers, &ContainerInfo{
			ID:      name,
			Name:    name,
			Status:  status,
			IP:      ip,
			SSHPort: 22,
		})
	}

	return containers, nil
}

func (m *Manager) GetContainerStatus(containerID string) (string, error) {
	cmd := exec.Command("lxc", "info", containerID)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get container status: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Status:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "unknown", nil
}

func (m *Manager) GetContainerStats(containerID string) (*ContainerStats, error) {
    cmd := exec.Command("lxc", "query", fmt.Sprintf("/1.0/instances/%s/state", containerID))
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("failed to query container state: %w", err)
    }
    var state struct {
        CPU struct { Usage uint64 `json:"usage"` } `json:"cpu"`
        Memory struct { Usage uint64 `json:"usage"`; UsagePeak uint64 `json:"usage_peak"`; SwapUsage uint64 `json:"swap_usage"`; SwapUsagePeak uint64 `json:"swap_usage_peak"`; Total uint64 `json:"total"` } `json:"memory"`
        Disk map[string]struct{ Usage uint64 `json:"usage"` } `json:"disk"`
        Network map[string]struct{ Counters struct{ BytesReceived uint64 `json:"bytes_received"`; BytesSent uint64 `json:"bytes_sent"` } `json:"counters"` } `json:"network"`
    }
    if err := json.Unmarshal(output, &state); err != nil {
        return nil, fmt.Errorf("failed to parse state: %w", err)
    }
    var diskUsage uint64
    for _, d := range state.Disk {
        diskUsage += d.Usage
    }
    var rx, tx uint64
    for name, n := range state.Network {
        // skip loopback
        if strings.HasPrefix(name, "lo") { continue }
        rx += n.Counters.BytesReceived
        tx += n.Counters.BytesSent
    }
    stats := &ContainerStats{
        CPUUsageNanoseconds: state.CPU.Usage,
        MemoryUsageBytes:    state.Memory.Usage,
        MemoryTotalBytes:    state.Memory.Total,
        DiskUsageBytes:      diskUsage,
        NetworkRXBytes:      rx,
        NetworkTXBytes:      tx,
    }
    return stats, nil
}

func (m *Manager) DeleteContainer(containerID string) error {
	stopCmd := exec.Command("lxc", "stop", containerID)
	stopCmd.Run()
	deleteCmd := exec.Command("lxc", "delete", containerID)
	if err := deleteCmd.Run(); err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}

	return nil
}

func (m *Manager) StopContainer(containerID string) error {
    stopCmd := exec.Command("lxc", "stop", containerID)
    if err := stopCmd.Run(); err != nil {
        return fmt.Errorf("failed to stop container: %w", err)
    }
    return nil
}

func (m *Manager) StartContainer(containerID string) error {
    startCmd := exec.Command("lxc", "start", containerID)
    if err := startCmd.Run(); err != nil {
        return fmt.Errorf("failed to start container: %w", err)
    }
    return nil
}

func (m *Manager) SetupSSHAccess(containerName, username, publicKey string) error {
	authorizedKeysPath := fmt.Sprintf("/home/%s/.ssh/authorized_keys", username)
	cmd := exec.Command("lxc", "exec", containerName, "--", "bash", "-c", 
		fmt.Sprintf("echo '%s' > %s", publicKey, authorizedKeysPath))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}
	chownCmd := exec.Command("lxc", "exec", containerName, "--", "chown", 
		fmt.Sprintf("%s:%s", username, username), authorizedKeysPath)
	if err := chownCmd.Run(); err != nil {
		return fmt.Errorf("failed to set key ownership: %w", err)
	}

	chmodCmd := exec.Command("lxc", "exec", containerName, "--", "chmod", "600", authorizedKeysPath)
	if err := chmodCmd.Run(); err != nil {
		return fmt.Errorf("failed to set key permissions: %w", err)
	}

	return nil
}

func (m *Manager) SetupSSHPassword(containerName, username, password string) error {
	cmd := exec.Command("lxc", "exec", containerName, "--", "bash", "-c", 
		fmt.Sprintf("echo '%s:%s' | chpasswd", username, password))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}
	
	sshConfigCommands := [][]string{
		{"lxc", "exec", containerName, "--", "sed", "-i", "s/#PasswordAuthentication yes/PasswordAuthentication yes/g", "/etc/ssh/sshd_config"},
		{"lxc", "exec", containerName, "--", "sed", "-i", "s/PasswordAuthentication no/PasswordAuthentication yes/g", "/etc/ssh/sshd_config"},
		{"lxc", "exec", containerName, "--", "sed", "-i", "s/PasswordAuthentication no/PasswordAuthentication yes/g", "/etc/ssh/sshd_config.d/60-cloudimg-settings.conf"},
		{"lxc", "exec", containerName, "--", "sed", "-i", "s/KbdInteractiveAuthentication no/KbdInteractiveAuthentication yes/g", "/etc/ssh/sshd_config"},
		{"lxc", "exec", containerName, "--", "sed", "-i", "s/#PubkeyAuthentication yes/PubkeyAuthentication yes/g", "/etc/ssh/sshd_config"},
		{"lxc", "exec", containerName, "--", "systemctl", "restart", "ssh"},
	}
	
	for _, cmd := range sshConfigCommands {
		execCmd := exec.Command(cmd[0], cmd[1:]...)
		if err := execCmd.Run(); err != nil {
			return fmt.Errorf("failed to configure SSH: %w", err)
		}
	}

	return nil
}

func (m *Manager) SetDefaultShell(containerName, username, shell string) (string, error) {
	var shellPath string
	switch shell {
	case "bash":
		shellPath = "/bin/bash"
	case "zsh":
		shellPath = "/usr/bin/zsh"
	case "fish":
		shellPath = "/usr/bin/fish"
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
	cmds := [][]string{
		{"lxc", "exec", containerName, "--", "bash", "-lc", fmt.Sprintf("grep -qx '%s' /etc/shells || echo '%s' >> /etc/shells", shellPath, shellPath)},
	}
	for _, c := range cmds {
		x := exec.Command(c[0], c[1:]...)
		if out, err := x.CombinedOutput(); err != nil {
			return string(out), fmt.Errorf("failed to set shell prereqs: %w", err)
		}
	}
	x := exec.Command("lxc", "exec", containerName, "--", "chsh", "-s", shellPath, username)
	out, err := x.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("failed to set shell: %w", err)
	}
	return string(out), nil
}

func (m *Manager) GetDefaultShell(containerName, username string) (string, error) {
	cmd := exec.Command("lxc", "exec", containerName, "--", "bash", "-lc", fmt.Sprintf("getent passwd %s | cut -d: -f7", username))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get shell: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (m *Manager) setupPortForwarding(containerName string, allocatedPorts []int) error {
	for _, port := range allocatedPorts {
		if err := m.MapPort(containerName, port, port, "tcp"); err != nil {
			return fmt.Errorf("failed to map port %d: %w", port, err)
		}
	}
	return nil
}

func (m *Manager) FindAvailablePort() (int, error) {
    const minPort = 20000
    const maxPort = 65535
    attempts := 2000
    for attempts > 0 {
        port := rand.Intn(maxPort-minPort+1) + minPort
        if m.isPortAvailable(port) {
            return port, nil
        }
        attempts--
    }
    return 0, fmt.Errorf("no available port found")
}
func (m *Manager) MapPort(containerID string, internalPort, externalPort int, protocol string) error {
	if protocol == "" {
		protocol = "tcp"
	}
	containerIP, err := m.getContainerIP(containerID)
	if err != nil {
		return fmt.Errorf("failed to get container IP: %w", err)
	}
	dnatCmd := exec.Command("iptables", "-t", "nat", "-A", "PREROUTING", 
		"-p", protocol, "--dport", strconv.Itoa(externalPort),
		"-j", "DNAT", "--to-destination", fmt.Sprintf("%s:%d", containerIP, internalPort))
	if err := dnatCmd.Run(); err != nil {
		return fmt.Errorf("failed to add DNAT rule: %w", err)
	}
	
	snatCmd := exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", 
		"-s", containerIP, "-j", "MASQUERADE")
	if err := snatCmd.Run(); err != nil {
		exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", 
			"-p", protocol, "--dport", strconv.Itoa(externalPort),
			"-j", "DNAT", "--to-destination", fmt.Sprintf("%s:%d", containerIP, internalPort)).Run()
		return fmt.Errorf("failed to add SNAT rule: %w", err)
	}
	allowCmd := exec.Command("iptables", "-A", "INPUT", 
		"-p", protocol, "--dport", strconv.Itoa(externalPort), "-j", "ACCEPT")
	if err := allowCmd.Run(); err != nil {
		exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", 
			"-p", protocol, "--dport", strconv.Itoa(externalPort),
			"-j", "DNAT", "--to-destination", fmt.Sprintf("%s:%d", containerIP, internalPort)).Run()
		return fmt.Errorf("failed to add INPUT rule: %w", err)
	}
	
	return nil
}

func (m *Manager) UnmapPort(containerID string, externalPort int, protocol string) error {
	if protocol == "" {
		protocol = "tcp"
	}
	
	containerIP, err := m.getContainerIP(containerID)
	if err != nil {
		return fmt.Errorf("failed to get container IP: %w", err)
	}
	

	dnatCmd := exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", 
		"-p", protocol, "--dport", strconv.Itoa(externalPort),
		"-j", "DNAT", "--to-destination", fmt.Sprintf("%s:%d", containerIP, externalPort))
	dnatCmd.Run()
	
	allowCmd := exec.Command("iptables", "-D", "INPUT", 
		"-p", protocol, "--dport", strconv.Itoa(externalPort), "-j", "ACCEPT")
	allowCmd.Run()
	
	return nil
}

func (m *Manager) getContainerIP(containerID string) (string, error) {
	cmd := exec.Command("lxc", "list", containerID, "-c", "4", "--format", "csv")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get container IP: %w", err)
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "", fmt.Errorf("no IP address found for container")
	}

	re := regexp.MustCompile(`(\d+\.\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(ip)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not parse IP address: %s", ip)
	}
	ip = matches[1]

	return ip, nil
}

func (m *Manager) GetRandomPort() (int, error) {
	minPort := 20000
	maxPort := 65535
	
	port := minPort + int(time.Now().UnixNano() % int64(maxPort-minPort))
	return port, nil
}

func (m *Manager) getDisplayHostname() string {
	if m.publicHostname != "" {
		return m.publicHostname
	}
	if hostname, err := exec.Command("hostname", "-f").Output(); err == nil {
		return strings.TrimSpace(string(hostname))
	}
	return "this node"
}