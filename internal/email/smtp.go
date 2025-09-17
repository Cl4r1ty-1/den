package email

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

type Client struct {
	Host       string
	Port       int
	Username   string
	Password   string
	From       string
	FromName   string
	UseSTARTTLS bool
}

func NewFromEnv() (*Client, error) {
	host := os.Getenv("SMTP_HOST")
	portStr := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USERNAME")
	pass := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")
	fromName := os.Getenv("SMTP_FROM_NAME")
	if host == "" || portStr == "" || from == "" {
		return nil, fmt.Errorf("smtp not configured")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil { return nil, err }
	useSTARTTLS := strings.ToLower(os.Getenv("SMTP_STARTTLS")) != "false"
	return &Client{Host: host, Port: port, Username: user, Password: pass, From: from, FromName: fromName, UseSTARTTLS: useSTARTTLS}, nil
}

func (c *Client) Send(to []string, subject, htmlBody, textBody string) error {
	if len(to) == 0 { return fmt.Errorf("no recipients") }
	fromHeader := c.From
	if c.FromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", c.FromName, c.From)
	}
	boundary := "den-boundary-9f2b3c7a"
	headers := []string{
		fmt.Sprintf("From: %s", fromHeader),
		fmt.Sprintf("To: %s", strings.Join(to, ", ")),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		fmt.Sprintf("Content-Type: multipart/alternative; boundary=%q", boundary),
	}
	if textBody == "" { textBody = stripHTML(htmlBody) }
	msg := strings.Join(headers, "\r\n") + "\r\n\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" + textBody + "\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n\r\n" + htmlBody + "\r\n" +
		"--" + boundary + "--\r\n"

	auth := smtp.PlainAuth("", c.Username, c.Password, c.Host)
	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)

	if c.UseSTARTTLS {
		conn, err := net.Dial("tcp", addr)
		if err != nil { return err }
		client, err := smtp.NewClient(conn, c.Host)
		if err != nil { return err }
		defer client.Close()
		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: c.Host}
			if err := client.StartTLS(config); err != nil { return err }
		}
		if c.Username != "" {
			if err := client.Auth(auth); err != nil { return err }
		}
		if err := client.Mail(c.From); err != nil { return err }
		for _, rcpt := range to {
			if err := client.Rcpt(rcpt); err != nil { return err }
		}
		w, err := client.Data()
		if err != nil { return err }
		if _, err := w.Write([]byte(msg)); err != nil { _ = w.Close(); return err }
		_ = w.Close()
		return client.Quit()
	}
	tlsCfg := &tls.Config{ServerName: c.Host}
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil { return err }
	client, err := smtp.NewClient(conn, c.Host)
	if err != nil { return err }
	defer client.Close()
	if c.Username != "" {
		if err := client.Auth(auth); err != nil { return err }
	}
	if err := client.Mail(c.From); err != nil { return err }
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil { return err }
	}
	w, err := client.Data()
	if err != nil { return err }
	if _, err := w.Write([]byte(msg)); err != nil { _ = w.Close(); return err }
	_ = w.Close()
	return client.Quit()
}

func stripHTML(s string) string {
	out := s
	out = strings.ReplaceAll(out, "<br>", "\n")
	out = strings.ReplaceAll(out, "<br/>", "\n")
	out = strings.ReplaceAll(out, "<br />", "\n")
	for strings.Contains(out, "<") {
		start := strings.Index(out, "<")
		end := strings.Index(out, ">")
		if start == -1 || end == -1 || end < start { break }
		out = out[:start] + out[end+1:]
	}
	return strings.TrimSpace(out)
}

func RenderNeobrutalismEmail(title, subtitle, bodyHTML string) string {
	return "" +
		"<html><head><meta charset=\"utf-8\"><style>" +
		"body{background:#0a0a0a;color:#f5f5f5;font-family:Inter,system-ui,-apple-system,Segoe UI,Roboto,Ubuntu,\"Helvetica Neue\",Arial,sans-serif;padding:24px;}"+
		".card{background:#111;border:4px solid #333;box-shadow:6px 6px 0 #000;padding:20px;max-width:680px;margin:0 auto;}"+
		".title{font-weight:800;font-size:22px;margin:0 0 8px 0;}"+
		".subtitle{opacity:.8;margin:0 0 16px 0;font-size:14px;}"+
		".btn{display:inline-block;background:#22c55e;color:#0a0a0a;font-weight:800;border:3px solid #0a0a0a;padding:10px 14px;text-decoration:none;box-shadow:4px 4px 0 #000}"+
		"a{color:#22c55e}"+
		"</style></head><body>"+
		"<div class=\"card\">"+
		fmt.Sprintf("<div class=\"title\">%s</div>", htmlEscape(title))+
		fmt.Sprintf("<div class=\"subtitle\">%s</div>", htmlEscape(subtitle))+
		bodyHTML+
		"</div></body></html>"
}

func htmlEscape(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#39;")
	return replacer.Replace(s)
}


