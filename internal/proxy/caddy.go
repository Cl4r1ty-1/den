package proxy

import (
    "bytes"
    "database/sql"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strings"
)

type CaddyService struct {
	adminURL string
}

type CaddyRoute struct {
	Match []map[string]interface{} `json:"match"`
	Handle []map[string]interface{} `json:"handle"`
}

func NewCaddyService(adminURL string) *CaddyService {
	if adminURL == "" {
		adminURL = "http://localhost:2019"
	}
	return &CaddyService{
		adminURL: adminURL,
	}
}

func (c *CaddyService) AddSubdomain(subdomain, targetHost string, targetPort int) error {
	fmt.Printf("adding route via caddy api: %s -> %s:%d\n", subdomain, targetHost, targetPort)
	
	route := CaddyRoute{
		Match: []map[string]interface{}{
			{
				"host": []string{subdomain},
			},
		},
		Handle: []map[string]interface{}{
			{
				"handler": "reverse_proxy",
				"upstreams": []map[string]interface{}{
					{
						"dial": fmt.Sprintf("%s:%d", targetHost, targetPort),
					},
				},
				"headers": map[string]interface{}{
					"request": map[string]interface{}{
						"set": map[string]string{
							"Host":              "{http.request.host}",
							"X-Real-IP":         "{http.request.remote.host}",
							"X-Forwarded-For":   "{http.request.header.X-Forwarded-For},{http.request.remote.host}",
							"X-Forwarded-Proto": "{http.request.scheme}",
						},
					},
				},
			},
		},
	}

	routeJSON, err := json.Marshal(route)
	if err != nil {
		return fmt.Errorf("failed to marshal route: %w", err)
	}

	fmt.Printf("making api call to caddy: POST %s/config/apps/http/servers/srv0/routes\n", c.adminURL)
	
	url := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes", c.adminURL)
    resp, err := http.Post(url, "application/json", bytes.NewBuffer(routeJSON))
	if err != nil {
		return fmt.Errorf("failed to add route: %w", err)
	}
	defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        var bodyBytes []byte
        if resp.Body != nil {
            bodyBytes, _ = io.ReadAll(resp.Body)
        }
        return fmt.Errorf("caddy API error: %s - %s", resp.Status, string(bodyBytes))
	}

	fmt.Printf("route added successfully! %s is now live\n", subdomain)
	return nil
}

func (c *CaddyService) RemoveSubdomain(subdomain string) error {
	resp, err := http.Get(fmt.Sprintf("%s/config/apps/http/servers/srv0/routes", c.adminURL))
	if err != nil {
		return fmt.Errorf("failed to get current routes: %w", err)
	}
	defer resp.Body.Close()

	var routes []CaddyRoute
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return fmt.Errorf("failed to decode routes: %w", err)
	}

	for i, route := range routes {
		for _, match := range route.Match {
			if hosts, ok := match["host"].([]interface{}); ok {
				for _, host := range hosts {
					if hostStr, ok := host.(string); ok && hostStr == subdomain {
						deleteURL := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes/%d", c.adminURL, i)
						req, err := http.NewRequest("DELETE", deleteURL, nil)
						if err != nil {
							return fmt.Errorf("failed to create delete request: %w", err)
						}

						client := &http.Client{}
						resp, err := client.Do(req)
						if err != nil {
							return fmt.Errorf("failed to delete route: %w", err)
						}
						resp.Body.Close()

						if resp.StatusCode >= 400 {
							return fmt.Errorf("caddy API delete error: %s", resp.Status)
						}
						return nil
					}
				}
			}
		}
	}

	return fmt.Errorf("subdomain %s not found", subdomain)
}

func (c *CaddyService) UpdateConfig() error {
	url := fmt.Sprintf("%s/load", c.adminURL)
	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("caddy reload error: %s", resp.Status)
	}

	return nil
}

func (c *CaddyService) GetCurrentConfig() (map[string]interface{}, error) {
	resp, err := http.Get(fmt.Sprintf("%s/config/", c.adminURL))
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return config, nil
}

func (c *CaddyService) ListActiveRoutes() ([]CaddyRoute, error) {
	resp, err := http.Get(fmt.Sprintf("%s/config/apps/http/servers/srv0/routes", c.adminURL))
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}
	defer resp.Body.Close()

	var routes []CaddyRoute
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return nil, fmt.Errorf("failed to decode routes: %w", err)
	}

	return routes, nil
}
func (c *CaddyService) RebuildAllRoutes(db *sql.DB) error {
	fmt.Println("rebuilding all caddy routes from database...")
	
	if err := c.clearAllRoutes(); err != nil {
		return fmt.Errorf("failed to clear existing routes: %w", err)
	}
	rows, err := db.Query(`
		SELECT s.subdomain, s.target_port, s.subdomain_type, u.username, 
		       COALESCE(n.public_hostname, n.hostname) as node_ip
		FROM subdomains s
		JOIN users u ON s.user_id = u.id
		JOIN containers c ON u.container_id = c.id
		JOIN nodes n ON c.node_id = n.id
		WHERE s.is_active = true
	`)
	if err != nil {
		return fmt.Errorf("failed to query subdomains: %w", err)
	}
	defer rows.Close()
	
	routesAdded := 0
	for rows.Next() {
		var subdomain, subdomainType, username string
		var targetPort int
		var nodeIP *string
		
		if err := rows.Scan(&subdomain, &targetPort, &subdomainType, &username, &nodeIP); err != nil {
			fmt.Printf("error scanning row: %v\n", err)
			continue
		}
		
		if nodeIP == nil {
			fmt.Printf("skipping %s - no node IP address\n", subdomain)
			continue
		}
		
		var fullSubdomain string
		if subdomainType == "username" {
			fullSubdomain = fmt.Sprintf("%s.hack.kim", subdomain)
		} else {
			fullSubdomain = fmt.Sprintf("%s.%s.hack.kim", subdomain, username)
		}
		
		if err := c.AddSubdomain(fullSubdomain, *nodeIP, targetPort); err != nil {
			fmt.Printf("failed to add route for %s: %v\n", fullSubdomain, err)
			continue
		}
		
		routesAdded++
	}
	
	fmt.Printf("successfully rebuilt %d routes from database\n", routesAdded)
	return nil
}

func (c *CaddyService) clearAllRoutes() error {
	routes, err := c.ListActiveRoutes()
	if err != nil {
		return fmt.Errorf("failed to get current routes: %w", err)
	}
	
	for i := len(routes) - 1; i >= 0; i-- {
		deleteURL := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes/%d", c.adminURL, i)
		req, err := http.NewRequest("DELETE", deleteURL, nil)
		if err != nil {
			continue
		}
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
	}
	
	return nil
}
