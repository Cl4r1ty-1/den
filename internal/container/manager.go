package container

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/lxc/go-lxc"
)

type Manager struct {
	// limits
	defaultMemoryMB  int
	defaultCPUCores  int
	defaultStorageGB int
}

type ContainerInfo struct {
	ID       string
	Name     string
	Status   string
	IP       string
	SSHPort  int
}

func NewManager() (*Manager, error) {
	return &Manager{
		defaultMemoryMB:  4096,
		defaultCPUCores:  4,
		defaultStorageGB: 15,
	}, nil
}

func (m *Manager) CreateContainer(userID int, username string) (*ContainerInfo, error) {
	containerName := fmt.Sprintf("den-%s", username)
	
	// lxc can be a piece of shit istg
	container, err := lxc.NewContainer(containerName, lxc.DefaultConfigPath())
	if err != nil {
		return nil, fmt.Errorf("failed to create container object: %w", err)
	}
	defer container.Release()
	
	if err := m.configureContainer(container); err != nil {
		return nil, fmt.Errorf("failed to configure container: %w", err)
	}
	
	if err := container.Create(lxc.TemplateOptions{
		Template: "ubuntu",
		Release:  "22.04",
	}); err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}
	
	if err := container.Start(); err != nil {
		container.Destroy()
		return nil, fmt.Errorf("failed to start container: %w", err)
	}
	
	if err := m.waitForContainer(container); err != nil {
		container.Stop()
		container.Destroy()
		return nil, fmt.Errorf("container failed to start: %w", err)
	}
	if err := m.setupUserInContainer(container, username); err != nil {
		container.Stop()
		container.Destroy()
		return nil, fmt.Errorf("failed to setup user: %w", err)
	}
	info, err := m.getContainerInfo(container)
	if err != nil {
		container.Stop()
		container.Destroy()
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}
	
	return info, nil
}

func (m *Manager) configureContainer(container *lxc.Container) error {
	// i
	// hate
	// lxc
	// so
	// so
	// muchhhhh
	if err := container.SetConfigItem("lxc.limits.memory", fmt.Sprintf("%dMB", m.defaultMemoryMB)); err != nil {
		return fmt.Errorf("failed to set memory limit: %w", err)
	}
	if err := container.SetConfigItem("lxc.limits.cpu", strconv.Itoa(m.defaultCPUCores)); err != nil {
		return fmt.Errorf("failed to set CPU limit: %w", err)
	}
	if err := container.SetConfigItem("lxc.apparmor.profile", "unconfined"); err != nil {
		return fmt.Errorf("failed to set apparmor profile: %w", err)
	}
	
	if err := container.SetConfigItem("lxc.security.nesting", "true"); err != nil {
		return fmt.Errorf("failed to set security nesting: %w", err)
	}
	
	if err := container.SetConfigItem("lxc.security.privileged", "false"); err != nil {
		return fmt.Errorf("failed to set security privileged: %w", err)
	}
	
	if err := container.SetConfigItem("lxc.net.0.type", "veth"); err != nil {
		return fmt.Errorf("failed to set network type: %w", err)
	}
	
	if err := container.SetConfigItem("lxc.net.0.link", "lxdbr0"); err != nil {
		return fmt.Errorf("failed to set network link: %w", err)
	}
	
	if err := container.SetConfigItem("lxc.net.0.flags", "up"); err != nil {
		return fmt.Errorf("failed to set network flags: %w", err)
	}
	
	return nil
}

func (m *Manager) waitForContainer(container *lxc.Container) error {
	maxAttempts := 30
	for i := 0; i < maxAttempts; i++ {
		if container.State() == lxc.RUNNING {
			_, err := container.RunCommand([]string{"echo", "ready"}, lxc.DefaultAttachOptions)
			if err == nil {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("container did not become ready in time")
}

func (m *Manager) setupUserInContainer(container *lxc.Container, username string) error {
	commands := [][]string{
		{"useradd", "-m", "-s", "/bin/bash", username},
		{"usermod", "-aG", "sudo", username},
		{"mkdir", "-p", fmt.Sprintf("/home/%s/.ssh", username)},
		{"chown", fmt.Sprintf("%s:%s", username, username), fmt.Sprintf("/home/%s/.ssh", username)},
		{"chmod", "700", fmt.Sprintf("/home/%s/.ssh", username)},
		{"apt-get", "update"},
		{"apt-get", "install", "-y", "openssh-server", "sudo", "curl", "git", "vim", "htop"},
		{"systemctl", "enable", "ssh"},
		{"systemctl", "start", "ssh"},
	}

	for _, cmd := range commands {
		_, err := container.RunCommand(cmd, lxc.DefaultAttachOptions)
		if err != nil {
			return fmt.Errorf("failed to run setup command %v: %w", cmd, err)
		}
	}

	return nil
}

func (m *Manager) getContainerInfo(container *lxc.Container) (*ContainerInfo, error) {
	ips, err := container.IPAddresses()
	if err != nil {
		return nil, fmt.Errorf("failed to get container IP addresses: %w", err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP address found for container")
	}
	
	var ip string
	for _, addr := range ips {
		if strings.Contains(addr, ".") { 
			ip = addr
			break
		}
	}
	
	if ip == "" {
		return nil, fmt.Errorf("no ip address found for container")
	}

	return &ContainerInfo{
		ID:      container.Name(),
		Name:    container.Name(),
		Status:  "running",
		IP:      ip,
		SSHPort: 22,
	}, nil
}

func (m *Manager) ListContainers() ([]*ContainerInfo, error) {
	allContainers := lxc.Containers(lxc.DefaultConfigPath())
	
	var containers []*ContainerInfo
	
	for _, container := range allContainers {
		defer container.Release()
		
		name := container.Name()
		if !strings.HasPrefix(name, "den-") {
			continue
		}
		state := container.State()
		var status string
		switch state {
		case lxc.RUNNING:
			status = "running"
		case lxc.STOPPED:
			status = "stopped"
		case lxc.STARTING:
			status = "starting"
		case lxc.STOPPING:
			status = "stopping"
		default:
			status = "unknown"
		}
		ips, err := container.IPAddresses()
		if err != nil {
			continue
		}
		var ip string
		for _, addr := range ips {
			if strings.Contains(addr, ".") { 
				ip = addr
				break
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
	container, err := lxc.NewContainer(containerID, lxc.DefaultConfigPath())
	if err != nil {
		return "", fmt.Errorf("failed to get container: %w", err)
	}
	defer container.Release()
	
	state := container.State()
	switch state {
	case lxc.STOPPED:
		return "stopped", nil
	case lxc.STARTING:
		return "starting", nil
	case lxc.RUNNING:
		return "running", nil
	case lxc.STOPPING:
		return "stopping", nil
	case lxc.ABORTING:
		return "aborting", nil
	case lxc.FREEZING:
		return "freezing", nil
	case lxc.FROZEN:
		return "frozen", nil
	case lxc.THAWED:
		return "thawed", nil
	default:
		return "unknown", nil
	}
}

func (m *Manager) DeleteContainer(containerID string) error {
	container, err := lxc.NewContainer(containerID, lxc.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}
	defer container.Release()
	if container.State() == lxc.RUNNING {
		if err := container.Stop(); err != nil {
			return fmt.Errorf("failed to stop container: %w", err)
		}
	}
	if err := container.Destroy(); err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}
	
	return nil
}

func (m *Manager) SetupSSHAccess(containerName, username, publicKey string) error {
	container, err := lxc.NewContainer(containerName, lxc.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}
	defer container.Release()
	
	authorizedKeysPath := fmt.Sprintf("/home/%s/.ssh/authorized_keys", username)
	_, err = container.RunCommand([]string{"bash", "-c", fmt.Sprintf("echo '%s' > %s", publicKey, authorizedKeysPath)}, lxc.DefaultAttachOptions)
	if err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}
	
	_, err = container.RunCommand([]string{"chown", fmt.Sprintf("%s:%s", username, username), authorizedKeysPath}, lxc.DefaultAttachOptions)
	if err != nil {
		return fmt.Errorf("failed to set key ownership: %w", err)
	}
	
	// Set permissions
	_, err = container.RunCommand([]string{"chmod", "600", authorizedKeysPath}, lxc.DefaultAttachOptions)
	if err != nil {
		return fmt.Errorf("failed to set key permissions: %w", err)
	}
	
	return nil
}

func (m *Manager) SetupSSHPassword(containerName, username, password string) error {
	container, err := lxc.NewContainer(containerName, lxc.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("failed to get container: %w", err)
	}
	defer container.Release()
	
	_, err = container.RunCommand([]string{"bash", "-c", fmt.Sprintf("echo '%s:%s' | chpasswd", username, password)}, lxc.DefaultAttachOptions)
	if err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}
	
	return nil
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
		return fmt.Errorf("failed to add SNAT rule: %w", err)
	}
	allowCmd := exec.Command("iptables", "-A", "INPUT", 
		"-p", protocol, "--dport", strconv.Itoa(externalPort), "-j", "ACCEPT")
	if err := allowCmd.Run(); err != nil {
		return fmt.Errorf("failed to add INPUT rule: %w", err)
	}
	
	return nil
}

func (m *Manager) UnmapPort(containerID string, externalPort int, protocol string) error {
	if protocol == "" {
		protocol = "tcp"
	}
	dnatCmd := exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", 
		"-p", protocol, "--dport", strconv.Itoa(externalPort),
		"-j", "DNAT", "--to-destination", fmt.Sprintf("%s:%d", containerID, externalPort))
	dnatCmd.Run()
	
	allowCmd := exec.Command("iptables", "-D", "INPUT", 
		"-p", protocol, "--dport", strconv.Itoa(externalPort), "-j", "ACCEPT")
	allowCmd.Run()
	
	return nil
}

func (m *Manager) getContainerIP(containerID string) (string, error) {
	container, err := lxc.NewContainer(containerID, lxc.DefaultConfigPath())
	if err != nil {
		return "", fmt.Errorf("failed to get container: %w", err)
	}
	defer container.Release()
	
	ips, err := container.IPAddresses()
	if err != nil {
		return "", fmt.Errorf("failed to get container IP addresses: %w", err)
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("container %s has no IP address", containerID)
	}
	
	for _, addr := range ips {
		if strings.Contains(addr, ".") { 
			return addr, nil
		}
	}
	
	return "", fmt.Errorf("container %s has no IPv4 address", containerID)
}

func (m *Manager) GetRandomPort() (int, error) {
	minPort := 20000
	maxPort := 65535
	
	port := minPort + int(time.Now().UnixNano() % int64(maxPort-minPort))
	return port, nil
}
