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
type CaddyConfig map[string]interface{}
type CaddyRoute map[string]interface{}

type CaddyService struct {
	adminURL string
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

	config, err := c.getCurrentConfig()
	if err != nil {
		return fmt.Errorf("failed to get current config: %w", err)
	}

	routes, srv0, err := extractRoutesFromConfig(config)
	if err != nil {
		return err
	}

	routeID := fmt.Sprintf("managed-subdomain-%s", subdomain)
	newRoute := CaddyRoute{
		"@id": routeID,
		"match": []map[string]interface{}{
			{"host": []string{subdomain}},
		},
		"handle": []map[string]interface{}{
			{
				"handler": "reverse_proxy",
				"upstreams": []map[string]interface{}{
					{"dial": fmt.Sprintf("%s:%d", targetHost, targetPort)},
				},
				"headers": map[string]interface{}{
					"request": map[string]interface{}{
						"set": map[string][]string{
							"Host":              {"{http.request.host}"},
							"X-Real-IP":         {"{http.request.remote.host}"},
							"X-Forwarded-For":   {"{http.request.header.X-Forwarded-For},{http.request.remote.host}"},
							"X-Forwarded-Proto": {"{http.request.scheme}"},
						},
					},
				},
			},
		},
	}

	var finalRoutes []CaddyRoute
	for _, route := range routes {
		if id, ok := route["@id"].(string); ok && id == routeID {
			continue
		}
		finalRoutes = append(finalRoutes, route)
	}
	wildcardIndex := findWildcardIndex(finalRoutes)

	if wildcardIndex != -1 {
		finalRoutes = append(finalRoutes[:wildcardIndex], append([]CaddyRoute{newRoute}, finalRoutes[wildcardIndex:]...)...)
	} else {
		finalRoutes = append(finalRoutes, newRoute)
	}
	srv0["routes"] = finalRoutes
	return c.loadConfig(config)
}

func (c *CaddyService) RemoveSubdomain(subdomain string) error {
	routeID := fmt.Sprintf("managed-subdomain-%s", subdomain)
	url := fmt.Sprintf("%s/id/%s", c.adminURL, routeID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy API delete error: %s - %s", resp.Status, string(body))
	}

	fmt.Printf("route for %s removed successfully\n", subdomain)
	return nil
}

func (c *CaddyService) RebuildAllRoutes(db *sql.DB) error {
	fmt.Println("rebuilding all caddy routes from database...")

	config, err := c.getCurrentConfig()
	if err != nil {
		return fmt.Errorf("failed to get current config for rebuild: %w", err)
	}

	currentRoutes, srv0, err := extractRoutesFromConfig(config)
	if err != nil {
		return err
	}
	
	var baseRoutes []CaddyRoute
	for _, route := range currentRoutes {
		if id, ok := route["@id"].(string); ok && strings.HasPrefix(id, "managed-subdomain-") {
			continue
		}
		baseRoutes = append(baseRoutes, route)
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

	var managedRoutes []CaddyRoute
	for rows.Next() {
		var subdomain, subdomainType, username, nodeIP string
		var targetPort int
		if err := rows.Scan(&subdomain, &targetPort, &subdomainType, &username, &nodeIP); err != nil {
			fmt.Printf("error scanning row: %v\n", err)
			continue
		}
		
		fullSubdomain := ""
		if subdomainType == "username" {
			fullSubdomain = fmt.Sprintf("%s.hack.kim", subdomain)
		} else {
			fullSubdomain = fmt.Sprintf("%s.%s.hack.kim", subdomain, username)
		}
		
		route := CaddyRoute{
			"@id":   fmt.Sprintf("managed-subdomain-%s", fullSubdomain),
			"match": []map[string]interface{}{{"host": []string{fullSubdomain}}},
			"handle": []map[string]interface{}{
				{
					"handler":   "reverse_proxy",
					"upstreams": []map[string]interface{}{{"dial": fmt.Sprintf("%s:%d", nodeIP, targetPort)}},
				},
			},
		}
		managedRoutes = append(managedRoutes, route)
	}

	wildcardIndex := findWildcardIndex(baseRoutes)
	var finalRoutes []CaddyRoute
	if wildcardIndex != -1 {
		finalRoutes = append(finalRoutes, baseRoutes[:wildcardIndex]...)
		finalRoutes = append(finalRoutes, managedRoutes...)
		finalRoutes = append(finalRoutes, baseRoutes[wildcardIndex:]...)
	} else {
		finalRoutes = append(baseRoutes, managedRoutes...)
	}
	
	srv0["routes"] = finalRoutes

	if err := c.loadConfig(config); err != nil {
		return fmt.Errorf("failed to apply rebuilt routes: %w", err)
	}
	
	fmt.Printf("successfully rebuilt %d routes from database\n", len(managedRoutes))
	return nil
}

func (c *CaddyService) getCurrentConfig() (CaddyConfig, error) {
	resp, err := http.Get(fmt.Sprintf("%s/config/", c.adminURL))
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("caddy API error on GET /config: %s", resp.Status)
	}
	
	var config CaddyConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config JSON: %w", err)
	}

	return config, nil
}

func (c *CaddyService) loadConfig(config CaddyConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config for loading: %w", err)
	}

	url := fmt.Sprintf("%s/load", c.adminURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(configJSON))
	if err != nil {
		return fmt.Errorf("failed to POST to /load endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("caddy API error on /load: %s - %s", resp.Status, string(bodyBytes))
	}
	
	fmt.Println("successfully loaded new configuration into caddy")
	return nil
}

func extractRoutesFromConfig(config CaddyConfig) ([]CaddyRoute, map[string]interface{}, error) {
	apps, ok := config["apps"].(map[string]interface{})
	if !ok { return nil, nil, fmt.Errorf("config missing 'apps' key") }
	
	http, ok := apps["http"].(map[string]interface{})
	if !ok { return nil, nil, fmt.Errorf("config missing 'http' app") }

	servers, ok := http["servers"].(map[string]interface{})
	if !ok { return nil, nil, fmt.Errorf("http app missing 'servers' key") }

	srv0, ok := servers["srv0"].(map[string]interface{})
	if !ok { return nil, nil, fmt.Errorf("http servers missing 'srv0' server") }
	
	routesRaw, ok := srv0["routes"].([]interface{})
	if !ok { return nil, nil, fmt.Errorf("srv0 missing 'routes' key") }
	
	var routes []CaddyRoute
	for _, r := range routesRaw {
		if routeMap, ok := r.(map[string]interface{}); ok {
			routes = append(routes, routeMap)
		}
	}
	
	return routes, srv0, nil
}

func findWildcardIndex(routes []CaddyRoute) int {
	for i, route := range routes {
		if match, ok := route["match"].([]interface{}); ok && len(match) > 0 {
			if matchCond, ok := match[0].(map[string]interface{}); ok {
				if hosts, ok := matchCond["host"].([]interface{}); ok && len(hosts) > 0 {
					if host, ok := hosts[0].(string); ok && host == "*.hack.kim" {
						return i
					}
				}
			}
		}
	}
	return -1
}