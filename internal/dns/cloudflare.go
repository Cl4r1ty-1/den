package dns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type CloudflareService struct {
	apiToken string
	zoneID   string
	baseURL  string
}

type CloudflareRecord struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type CloudflareResponse struct {
	Success bool                `json:"success"`
	Errors  []CloudflareError   `json:"errors"`
    Result  []CloudflareRecord  `json:"result"`
}

type CloudflareError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Cloudflare responses vary per endpoint. Create/Update return a single object,
// list/search return an array, and delete returns a small result payload.
type CloudflareListResponse struct {
    Success bool               `json:"success"`
    Errors  []CloudflareError  `json:"errors"`
    Result  []CloudflareRecord `json:"result"`
}

type CloudflareSingleResponse struct {
    Success bool              `json:"success"`
    Errors  []CloudflareError `json:"errors"`
    Result  CloudflareRecord  `json:"result"`
}

type CloudflareDeleteResponse struct {
    Success bool              `json:"success"`
    Errors  []CloudflareError `json:"errors"`
    Result  struct {
        ID string `json:"id"`
    } `json:"result"`
}

func NewCloudflareService() *CloudflareService {
	return &CloudflareService{
		apiToken: os.Getenv("CLOUDFLARE_API_TOKEN"),
		zoneID:   os.Getenv("CLOUDFLARE_ZONE_ID"),
		baseURL:  "https://api.cloudflare.com/client/v4",
	}
}

func (c *CloudflareService) CreateRecord(subdomain, targetIP string) error {
	if c.apiToken == "" || c.zoneID == "" {
		return fmt.Errorf("cloudflare API token or zone ID not configured")
	}
	
	fmt.Printf("cloudflare config: zoneID=%s, hasToken=%t\n", c.zoneID, c.apiToken != "")

	record := CloudflareRecord{
		Type:    "A",
		Name:    subdomain,
		Content: targetIP,
		TTL:     300,
		Proxied: false,
	}

	jsonData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records", c.baseURL, c.zoneID)
	fmt.Printf("cloudflare API URL: %s\n", url)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

    var cfResp CloudflareSingleResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			return fmt.Errorf("cloudflare API error: %s", cfResp.Errors[0].Message)
		}
		return fmt.Errorf("cloudflare API request failed")
	}

	fmt.Printf("created DNS record: %s -> %s\n", subdomain, targetIP)
	return nil
}

func (c *CloudflareService) DeleteRecord(subdomain string) error {
	if c.apiToken == "" || c.zoneID == "" {
		return fmt.Errorf("cloudflare API token or zone ID not configured")
	}

	recordID, err := c.findRecordID(subdomain)
	if err != nil {
		return fmt.Errorf("failed to find record: %w", err)
	}

	if recordID == "" {
		fmt.Printf("DNS record %s not found (may have been already deleted)\n", subdomain)
		return nil
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.baseURL, c.zoneID, recordID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make delete request: %w", err)
	}
	defer resp.Body.Close()

    var cfResp CloudflareDeleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return fmt.Errorf("failed to decode delete response: %w", err)
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			return fmt.Errorf("cloudflare delete error: %s", cfResp.Errors[0].Message)
		}
		return fmt.Errorf("cloudflare delete request failed")
	}

	fmt.Printf("deleted DNS record: %s\n", subdomain)
	return nil
}

func (c *CloudflareService) findRecordID(subdomain string) (string, error) {
	url := fmt.Sprintf("%s/zones/%s/dns_records?name=%s&type=A", c.baseURL, c.zoneID, subdomain)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create search request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to search for record: %w", err)
	}
	defer resp.Body.Close()

    var cfResp CloudflareListResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return "", fmt.Errorf("failed to decode search response: %w", err)
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			return "", fmt.Errorf("cloudflare search error: %s", cfResp.Errors[0].Message)
		}
		return "", fmt.Errorf("cloudflare search request failed")
	}

	if len(cfResp.Result) > 0 {
		return cfResp.Result[0].ID, nil
	}

	return "", nil
}

func (c *CloudflareService) UpdateRecord(subdomain, newTargetIP string) error {
	recordID, err := c.findRecordID(subdomain)
	if err != nil {
		return fmt.Errorf("failed to find record for update: %w", err)
	}

	if recordID == "" {
		return c.CreateRecord(subdomain, newTargetIP)
	}

	record := CloudflareRecord{
		Type:    "A",
		Name:    subdomain,
		Content: newTargetIP,
		TTL:     300,
		Proxied: false,
	}

	jsonData, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal update record: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records/%s", c.baseURL, c.zoneID, recordID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make update request: %w", err)
	}
	defer resp.Body.Close()

    var cfResp CloudflareSingleResponse
	if err := json.NewDecoder(resp.Body).Decode(&cfResp); err != nil {
		return fmt.Errorf("failed to decode update response: %w", err)
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			return fmt.Errorf("cloudflare update error: %s", cfResp.Errors[0].Message)
		}
		return fmt.Errorf("cloudflare update request failed")
	}

	fmt.Printf("updated dns record: %s -> %s\n", subdomain, newTargetIP)
	return nil
}
