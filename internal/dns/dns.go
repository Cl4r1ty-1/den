package dns

import (
	"fmt"
	"net"
	"strings"
)

type Service struct {
	domain string
}

func NewService() *Service {
	return &Service{
		domain: "hack.kim",
	}
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

// im too lazy to implement cf support
func (s *Service) CreateDNSRecord(subdomain, targetIP string, port int) error {
	fullDomain := fmt.Sprintf("%s.%s", subdomain, s.domain)
	fmt.Printf("would create DNS record: %s -> %s:%d\n", fullDomain, targetIP, port)
	return s.updateProxyConfig(subdomain, targetIP, port)
}
func (s *Service) DeleteDNSRecord(subdomain string) error {
	fullDomain := fmt.Sprintf("%s.%s", subdomain, s.domain)
	
	fmt.Printf("wuld delete DNS record: %s\n", fullDomain)
	
	return s.removeProxyConfig(subdomain)
}

// idk wether to use like caddy or nginx or traefik for reverse proxy
func (s *Service) updateProxyConfig(subdomain, targetIP string, port int) error {
// so for now i guess nginx?
	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.%s", subdomain, s.domain)
	config := fmt.Sprintf(`
server {
    listen 80;
    server_name %s.%s;
    
    location / {
        proxy_pass http://%s:%d;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
`, subdomain, s.domain, targetIP, port)

	fmt.Printf("write nginx config to %s:\n%s\n", configPath, config)
	return nil
}
func (s *Service) removeProxyConfig(subdomain string) error {
	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s.%s", subdomain, s.domain)
	symlinkPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.%s", subdomain, s.domain)
	
	fmt.Printf("Would remove nginx config: %s and %s\n", configPath, symlinkPath)	
	return nil
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
