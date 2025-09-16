package main

import (
    "bytes"
    "crypto/sha256"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "runtime"
    "strings"
    "time"
)

const defaultBaseURL = "https://hack.kim"

type httpClient interface {
    Do(req *http.Request) (*http.Response, error)
}

func main() {
    tokenFlag := flag.String("token", "", "container token (overrides env/files)")
    baseURLFlag := flag.String("url", "", "base URL of den master (e.g. http://master:8080)")
    flag.Parse()

    if flag.NArg() == 0 {
        usage()
        os.Exit(2)
    }

    baseURL := resolveBaseURL(*baseURLFlag)
    client := &http.Client{Timeout: 15 * time.Second}

    cmd := flag.Arg(0)
    switch cmd {
    case "me":
        token, err := resolveToken(*tokenFlag)
        if err != nil { fail(err) }
        if err := cmdMe(client, baseURL, token); err != nil { fail(err) }
    case "stats":
        token, err := resolveToken(*tokenFlag)
        if err != nil { fail(err) }
        if err := cmdStats(client, baseURL, token); err != nil { fail(err) }
    case "start", "stop", "restart":
        token, err := resolveToken(*tokenFlag)
        if err != nil { fail(err) }
        if err := cmdControl(client, baseURL, token, cmd); err != nil { fail(err) }
    case "ports":
        token, err := resolveToken(*tokenFlag)
        if err != nil { fail(err) }
        if err := cmdPorts(client, baseURL, token); err != nil { fail(err) }
    case "get_port":
        token, err := resolveToken(*tokenFlag)
        if err != nil { fail(err) }
        if err := cmdGetPort(client, baseURL, token); err != nil { fail(err) }
    case "update":
        if err := cmdUpdate(client, baseURL); err != nil { fail(err) }
    default:
        usage()
        os.Exit(2)
    }
}

func usage() {
    fmt.Println("den - control your environment from inside the container")
    fmt.Println()
    fmt.Println("Usage:")
    fmt.Println("  den [--token TOKEN] [--url BASE_URL] me")
    fmt.Println("  den [--token TOKEN] [--url BASE_URL] stats")
    fmt.Println("  den [--token TOKEN] [--url BASE_URL] start|stop|restart")
    fmt.Println("  den [--token TOKEN] [--url BASE_URL] ports")
    fmt.Println("  den [--token TOKEN] [--url BASE_URL] get_port")
    fmt.Println("  den [--url BASE_URL] update")
    fmt.Println()
    fmt.Println("Token resolution order: --token, DEN_CONTAINER_TOKEN, /etc/den/container_token, $HOME/.config/den/token")
}

func fail(err error) {
    fmt.Fprintln(os.Stderr, "error:", err)
    os.Exit(1)
}

func resolveBaseURL(flagVal string) string {
    if strings.TrimSpace(flagVal) != "" {
        return strings.TrimRight(flagVal, "/")
    }
    if v := os.Getenv("DEN_MASTER_URL"); strings.TrimSpace(v) != "" {
        return strings.TrimRight(v, "/")
    }
    return defaultBaseURL
}

func resolveToken(flagVal string) (string, error) {
    if strings.TrimSpace(flagVal) != "" { return strings.TrimSpace(flagVal), nil }
    if v := os.Getenv("DEN_CONTAINER_TOKEN"); strings.TrimSpace(v) != "" { return strings.TrimSpace(v), nil }
    candidates := []string{
        "/etc/den/container_token",
        filepath.Join(os.Getenv("HOME"), ".config/den/token"),
    }
    for _, p := range candidates {
        if b, err := os.ReadFile(p); err == nil {
            s := strings.TrimSpace(string(b))
            if s != "" { return s, nil }
        }
    }
    return "", errors.New("container token not found; set --token or DEN_CONTAINER_TOKEN")
}

func newRequest(method, url, token string, body io.Reader) (*http.Request, error) {
    req, err := http.NewRequest(method, url, body)
    if err != nil { return nil, err }
    req.Header.Set("Authorization", "Bearer "+token)
    return req, nil
}

func cmdMe(client httpClient, baseURL, token string) error {
    req, err := newRequest(http.MethodGet, baseURL+"/cli/me", token, nil)
    if err != nil { return err }
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("%s", strings.TrimSpace(string(b)))
    }
    var out map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return err }
    pretty, _ := json.MarshalIndent(out, "", "  ")
    fmt.Println(string(pretty))
    return nil
}

func cmdStats(client httpClient, baseURL, token string) error {
    req, err := newRequest(http.MethodGet, baseURL+"/cli/container/stats", token, nil)
    if err != nil { return err }
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("%s", strings.TrimSpace(string(b)))
    }
    var out map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return err }
    pretty, _ := json.MarshalIndent(out, "", "  ")
    fmt.Println(string(pretty))
    return nil
}

func cmdControl(client httpClient, baseURL, token, action string) error {
    body, _ := json.Marshal(map[string]string{"action": action})
    req, err := newRequest(http.MethodPost, baseURL+"/cli/container/"+action, token, bytes.NewBuffer(body))
    if err != nil { return err }
    req.Header.Set("Content-Type", "application/json")
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("%s", strings.TrimSpace(string(b)))
    }
    fmt.Println("ok")
    return nil
}

func cmdPorts(client httpClient, baseURL, token string) error {
    req, err := newRequest(http.MethodGet, baseURL+"/cli/container/ports", token, nil)
    if err != nil { return err }
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("%s", strings.TrimSpace(string(b)))
    }
    var out map[string][]int
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return err }
    ports := out["ports"]
    for _, p := range ports { fmt.Println(p) }
    return nil
}

func cmdGetPort(client httpClient, baseURL, token string) error {
    reqBody := bytes.NewBufferString("{}")
    req, err := newRequest(http.MethodPost, baseURL+"/cli/container/ports/new", token, reqBody)
    if err != nil { return err }
    req.Header.Set("Content-Type", "application/json")
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("%s", strings.TrimSpace(string(b)))
    }
    var out map[string]int
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil { return err }
    fmt.Println(out["port"])
    return nil
}

func cmdUpdate(client httpClient, baseURL string) error {
    arch := runtime.GOARCH
    if arch != "amd64" && arch != "arm64" {
        return fmt.Errorf("unsupported architecture: %s", arch)
    }
    binName := "den-linux-" + arch
    binURL := baseURL + "/downloads/cli/" + binName
    sumsURL := baseURL + "/downloads/cli/SHA256SUMS"

    b, err := httpGetBytes(client, binURL)
    if err != nil { return fmt.Errorf("download failed: %w", err) }
    if len(b) == 0 { return errors.New("empty binary downloaded") }

    if sums, err := httpGetBytes(client, sumsURL); err == nil {
        want := parseSHA256Sum(string(sums), binName)
        if want != "" {
            got := fmt.Sprintf("%x", sha256.Sum256(b))
            if !strings.EqualFold(want, got) {
                return fmt.Errorf("checksum mismatch: got %s want %s", got, want)
            }
        }
    }

    exePath, err := os.Executable()
    if err != nil { return fmt.Errorf("cannot resolve current executable: %w", err) }
    dir := filepath.Dir(exePath)
    tmp := filepath.Join(dir, ".den.tmp")
    if err := os.WriteFile(tmp, b, 0755); err != nil {
        return fmt.Errorf("write temp failed (need permissions?): %w", err)
    }
    if err := os.Rename(tmp, exePath); err != nil {
        return fmt.Errorf("replace failed (need permissions?): %w", err)
    }
    fmt.Println("updated")
    return nil
}

func httpGetBytes(client httpClient, url string) ([]byte, error) {
    req, err := http.NewRequest(http.MethodGet, url, nil)
    if err != nil { return nil, err }
    resp, err := client.Do(req)
    if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK { return nil, fmt.Errorf("%s", resp.Status) }
    return io.ReadAll(resp.Body)
}

func parseSHA256Sum(sums, file string) string {
    lines := strings.Split(sums, "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" { continue }
        parts := strings.Fields(line)
        if len(parts) >= 2 {
            name := parts[len(parts)-1]
            if name == file { return parts[0] }
        }
    }
    return ""
}


