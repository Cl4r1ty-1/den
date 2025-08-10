package dns

import (
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	
	"github.com/den/internal/proxy"
)

type Service struct {
	domain     string
	caddy      *proxy.CaddyService
	cloudflare *CloudflareService
}

func NewService() *Service {
	return &Service{
		domain:     "hack.kim",
		caddy:      proxy.NewCaddyService(""),
		cloudflare: NewCloudflareService(),
	}
}

func (s *Service) RebuildRoutesFromDatabase(db *sql.DB) error {
	fmt.Println("rebuilding caddy routes from database on startup...")
	return s.caddy.RebuildAllRoutes(db)
}

func (s *Service) ValidateSubdomain(subdomain string) error {
	if len(subdomain) < 1 || len(subdomain) > 63 {
		return fmt.Errorf("subdomain must be between 1 and 63 characters")
		}
	for _, char := range subdomain {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '-') {
			return fmt.Errorf("subdomain can only contain alphanumeric characters and hyphens")
		}
	}
	if strings.HasPrefix(subdomain, "-") || strings.HasSuffix(subdomain, "-") {
		return fmt.Errorf("subdomain cannot start or end with a hyphen")
	}
	reserved := []string{
		"www", "mail", "ftp", "admin", "api", "app", "blog", "dev", "test",
		"staging", "demo", "support", "help", "docs", "forum", "shop", "store",
		"cdn", "static", "media", "assets", "img", "images", "files", "download",
		"upload", "secure", "ssl", "vpn", "proxy", "gateway", "router", "firewall",
		"ns", "ns1", "ns2", "dns", "mx", "smtp", "pop", "imap", "webmail",
	}

	lowerSubdomain := strings.ToLower(subdomain)
	for _, r := range reserved {
		if lowerSubdomain == r {
			return fmt.Errorf("subdomain '%s' is reserved", subdomain)
		}
	}

	return nil
}

func (s *Service) CreateDNSRecord(subdomain, nodeIP string, externalPort int) error {
	fullDomain := fmt.Sprintf("%s.%s", subdomain, s.domain)
	fmt.Printf("creating dns record: %s -> %s:%d\n", fullDomain, nodeIP, externalPort)
	
	publicIP := s.getPublicIP()
	if err := s.cloudflare.CreateRecord(fullDomain, publicIP); err != nil {
		return fmt.Errorf("failed to create cloudflare record: %w", err)
	}
	
	if err := s.caddy.AddSubdomain(fullDomain, nodeIP, externalPort); err != nil {
		s.cloudflare.DeleteRecord(fullDomain)
		return fmt.Errorf("failed to add caddy route: %w", err)
	}
	
	return nil
}
func (s *Service) DeleteDNSRecord(subdomain string) error {
	fullDomain := fmt.Sprintf("%s.%s", subdomain, s.domain)
	
	fmt.Printf("deleting dns record: %s\n", fullDomain)
	
	if err := s.caddy.RemoveSubdomain(fullDomain); err != nil {
		fmt.Printf("failed to remove caddy route: %v\n", err)
	}

	if err := s.cloudflare.DeleteRecord(fullDomain); err != nil {
		return fmt.Errorf("failed to delete cloudflare record: %w", err)
	}
	
	return nil
}

func (s *Service) ValidateUserPort(port int, allocatedPorts []int) error {
	for _, allocatedPort := range allocatedPorts {
		if port == allocatedPort {
			return nil
		}
	}
	return fmt.Errorf("port %d is not in your allocated ports", port)
}

func (s *Service) GetAvailablePort() (int, error) {
	for port := 20000; port <= 30000; port++ {
		if s.isPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range 20000-30000")
}

func (s *Service) isPortAvailable(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

func (s *Service) ValidatePort(port int) error {
	if port < 1024 || port > 65535 {
		return fmt.Errorf("port must be between 1024 and 65535")
	}
	if port >= 20000 && port <= 30000 {
		return fmt.Errorf("port range 20000-30000 is reserved for external access")
	}
	
	return nil
}

func (s *Service) getPublicIP() string {
	if publicIP := os.Getenv("PUBLIC_IP"); publicIP != "" {
		return publicIP
	}
	
	if detectedIP := s.detectPublicIP(); detectedIP != "" {
		return detectedIP
	}
	
	fmt.Println("could not detect public IP, using fallback. set the PUBLIC_IP environment variable ya ding dong")
	return "127.0.0.1"
}

func (s *Service) detectPublicIP() string {
	services := []string{
		"https://api.ipify.org",
		"https://checkip.amazonaws.com",
		"https://icanhazip.com",
	}
	
	client := &http.Client{}
	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			
			ip := strings.TrimSpace(string(body))
			if net.ParseIP(ip) != nil {
				fmt.Printf("detected public IP: %s\n", ip)
				return ip
			}
		}
	}
	
	return ""
}
