//go:build slave
// +build slave

package container

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ManagerLXD struct {
	defaultMemoryMB  int
	defaultCPUCores  int
	defaultStorageGB int
}

func NewManagerLXD() (*ManagerLXD, error) {
	return &ManagerLXD{
		defaultMemoryMB:  4096,
		defaultCPUCores:  4,
		defaultStorageGB: 15,
	}, nil
}

func (m *ManagerLXD) CreateContainer(userID int, username string) (*ContainerInfo, error) {
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
	
	return info, nil
}

func (m *ManagerLXD) configureContainer(name string) error {
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

func (m *ManagerLXD) waitForContainer(name string) error {
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

func (m *ManagerLXD) setupUserInContainer(containerName, username string) error {
	commands := [][]string{
		{"lxc", "exec", containerName, "--", "useradd", "-m", "-s", "/bin/bash", username},
		{"lxc", "exec", containerName, "--", "usermod", "-aG", "sudo", username},
		{"lxc", "exec", containerName, "--", "mkdir", "-p", fmt.Sprintf("/home/%s/.ssh", username)},
		{"lxc", "exec", containerName, "--", "chown", fmt.Sprintf("%s:%s", username, username), fmt.Sprintf("/home/%s/.ssh", username)},
		{"lxc", "exec", containerName, "--", "chmod", "700", fmt.Sprintf("/home/%s/.ssh", username)},
		{"lxc", "exec", containerName, "--", "apt-get", "update"},
		{"lxc", "exec", containerName, "--", "apt-get", "install", "-y", "openssh-server", "sudo", "curl", "git", "vim", "htop"},
		{"lxc", "exec", containerName, "--", "systemctl", "enable", "ssh"},
		{"lxc", "exec", containerName, "--", "systemctl", "start", "ssh"},
	}

	for _, cmd := range commands {
		execCmd := exec.Command(cmd[0], cmd[1:]...)
		if err := execCmd.Run(); err != nil {
			return fmt.Errorf("failed to run setup command %v: %w", cmd, err)
		}
	}

	return nil
}

func (m *ManagerLXD) getContainerInfo(name string) (*ContainerInfo, error) {
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
		Status:  "running",
		IP:      ip,
		SSHPort: 22,
	}, nil
}

func (m *ManagerLXD) ListContainers() ([]*ContainerInfo, error) {
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

func (m *ManagerLXD) GetContainerStatus(containerID string) (string, error) {
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

func (m *ManagerLXD) DeleteContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	stopCmd := exec.CommandContext(ctx, "lxc", "stop", containerID, "--timeout", "20")
	_ = stopCmd.Run()

	deleteCmd := exec.CommandContext(ctx, "lxc", "delete", containerID, "--force")
	if err := deleteCmd.Run(); err != nil {
		return fmt.Errorf("failed to delete container: %w", err)
	}

	return nil
}

func (m *ManagerLXD) SetupSSHAccess(containerName, username, publicKey string) error {
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

func (m *ManagerLXD) SetupSSHPassword(containerName, username, password string) error {
	cmd := exec.Command("lxc", "exec", containerName, "--", "bash", "-c", 
		fmt.Sprintf("echo '%s:%s' | chpasswd", username, password))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	return nil
}

func (m *ManagerLXD) MapPort(containerID string, internalPort, externalPort int, protocol string) error {
	return fmt.Errorf("port mapping not implemented yet")
}

func (m *ManagerLXD) UnmapPort(containerID string, externalPort int, protocol string) error {
	return fmt.Errorf("port unmapping not implemented yet")
}

func (m *ManagerLXD) GetRandomPort() (int, error) {
	minPort := 20000
	maxPort := 65535
	
	port := minPort + int(time.Now().UnixNano() % int64(maxPort-minPort))
	return port, nil
}