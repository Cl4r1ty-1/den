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

type CaddyRoute map[string]interface{}

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

    routes, err := c.ListActiveRoutes()
    if err != nil {
        return fmt.Errorf("failed to get current routes for adding: %w", err)
    }

    routeID := fmt.Sprintf("managed-subdomain-%s", subdomain)
    newRoute := CaddyRoute{
        "@id": routeID,
        "match": []map[string]interface{}{
            {
                "host": []string{subdomain},
            },
        },
        "handle": []map[string]interface{}{
            {
                "handler": "reverse_proxy",
                "upstreams": []map[string]interface{}{
                    {
                        "dial": fmt.Sprintf("%s:%d", targetHost, targetPort),
                    },
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

    wildcardIndex := -1
    for i, route := range routes {
        if id, ok := route["@id"].(string); ok && id == routeID {
            routes = append(routes[:i], routes[i+1:]...)
            break
        }

        match, ok := route["match"].([]interface{})
        if !ok || len(match) == 0 {
            continue
        }
        matchCond, ok := match[0].(map[string]interface{})
        if !ok {
            continue
        }
        hosts, ok := matchCond["host"].([]interface{})
        if !ok || len(hosts) == 0 {
            continue
        }
        if host, ok := hosts[0].(string); ok && host == "*.hack.kim" {
            wildcardIndex = i
        }
    }

    if wildcardIndex != -1 {
        routes = append(routes[:wildcardIndex], append([]CaddyRoute{newRoute}, routes[wildcardIndex:]...)...)
    } else {
        routes = append(routes, newRoute)
    }

    return c.updateRoutes(routes)
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
        return fmt.Errorf("caddy API delete error: %s", resp.Status)
    }

    fmt.Printf("route for %s removed successfully\n", subdomain)
    return nil
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
        if err == io.EOF {
            return []CaddyRoute{}, nil
        }
        return nil, fmt.Errorf("failed to decode routes: %w", err)
    }

    return routes, nil
}

func (c *CaddyService) RebuildAllRoutes(db *sql.DB) error {
    fmt.Println("rebuilding all caddy routes from database...")

    currentRoutes, err := c.ListActiveRoutes()
    if err != nil {
        return fmt.Errorf("failed to get current routes for rebuild: %w", err)
    }

    finalRoutes := make([]CaddyRoute, 0)
    for _, route := range currentRoutes {
        if id, ok := route["@id"].(string); ok && strings.HasPrefix(id, "managed-subdomain-") {
            continue
        }
        finalRoutes = append(finalRoutes, route)
    }

    wildcardIndex := -1
    for i, route := range finalRoutes {
        match, _ := route["match"].([]interface{})
        if len(match) > 0 {
            if matchCond, _ := match[0].(map[string]interface{}); matchCond != nil {
                if hosts, _ := matchCond["host"].([]interface{}); len(hosts) > 0 {
                    if host, _ := hosts[0].(string); host == "*.hack.kim" {
                        wildcardIndex = i
                        break
                    }
                }
            }
        }
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

        newRoute := CaddyRoute{
            "@id":   fmt.Sprintf("managed-subdomain-%s", fullSubdomain),
            "match": []map[string]interface{}{{"host": []string{fullSubdomain}}},
            "handle": []map[string]interface{}{
                {
                    "handler":   "reverse_proxy",
                    "upstreams": []map[string]interface{}{{"dial": fmt.Sprintf("%s:%d", nodeIP, targetPort)}},
                },
            },
        }
        managedRoutes = append(managedRoutes, newRoute)
    }

    if wildcardIndex != -1 {
        finalRoutes = append(finalRoutes[:wildcardIndex], append(managedRoutes, finalRoutes[wildcardIndex:]...)...)
    } else {
        finalRoutes = append(finalRoutes, managedRoutes...)
    }

    if err := c.updateRoutes(finalRoutes); err != nil {
        return fmt.Errorf("failed to apply rebuilt routes: %w", err)
    }

    fmt.Printf("successfully rebuilt %d routes from database\n", len(managedRoutes))
    return nil
}

func (c *CaddyService) updateRoutes(routes []CaddyRoute) error {
    routesJSON, err := json.Marshal(routes)
    if err != nil {
        return fmt.Errorf("failed to marshal routes for update: %w", err)
    }

    url := fmt.Sprintf("%s/config/apps/http/servers/srv0/routes", c.adminURL)
    req, err := http.NewRequest("PUT", url, bytes.NewBuffer(routesJSON))
    if err != nil {
        return fmt.Errorf("failed to create update request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("failed to update routes in caddy: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("caddy API error on update: %s - %s", resp.Status, string(bodyBytes))
    }
    return nil
}
