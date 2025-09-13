//go:build !slave
// +build !slave

package container

import (
	"fmt"
)

type Manager struct {
	defaultMemoryMB  int
	defaultCPUCores  int
	defaultStorageGB int
	publicHostname   string
}

type ContainerInfo struct {
	ID       string
	Name     string
	Status   string
	IP       string
	SSHPort  int
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

func (m *Manager) CreateContainer(userID int, username string) (*ContainerInfo, error) {
	return nil, fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) ListContainers() ([]*ContainerInfo, error) {
	return nil, fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) GetContainerStatus(containerID string) (string, error) {
	return "", fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) GetContainerStats(containerID string) (*ContainerStats, error) {
	return nil, fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) DeleteContainer(containerID string) error {
	return fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) StopContainer(containerID string) error {
    return fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) StartContainer(containerID string) error {
    return fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) SetupSSHAccess(containerName, username, publicKey string) error {
	return fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) SetupSSHPassword(containerName, username, password string) error {
	return fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) MapPort(containerID string, internalPort, externalPort int, protocol string) error {
	return fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) UnmapPort(containerID string, externalPort int, protocol string) error {
	return fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) GetRandomPort() (int, error) {
	return 0, fmt.Errorf("container operations not supported on master node")
}

func (m *Manager) FindAvailablePort() (int, error) {
    return 0, fmt.Errorf("container operations not supported on master node")
}