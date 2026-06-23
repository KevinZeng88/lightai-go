package agentclient

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultTimeout   = 30 * time.Second
	FileScanTimeout  = 120 * time.Second
	MaxResponseBytes = 100 * 1024 * 1024 // 100MB
)

var deniedCIDRs = []*net.IPNet{
	parseCIDR("169.254.0.0/16"),
	parseCIDR("0.0.0.0/32"),
	parseCIDR("::/128"),
	parseCIDR("224.0.0.0/4"),
	parseCIDR("ff00::/8"),
}

type Client struct {
	httpClient *http.Client
	agentToken string
}

func New(agentToken string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		agentToken: agentToken,
	}
}

func (c *Client) GetJSON(ctx context.Context, addr string, port int, path string, params url.Values) ([]byte, int, error) {
	if err := ValidateAgentAddress(addr); err != nil {
		return nil, 0, fmt.Errorf("agent address rejected: %w", err)
	}
	u := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(addr, fmt.Sprintf("%d", port)),
		Path:   path,
	}
	if params != nil {
		u.RawQuery = params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	if c.agentToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.agentToken)
	}
	return c.doRequest(req)
}

func (c *Client) PostJSON(ctx context.Context, addr string, port int, path string, body io.Reader, params url.Values) ([]byte, int, error) {
	if err := ValidateAgentAddress(addr); err != nil {
		return nil, 0, fmt.Errorf("agent address rejected: %w", err)
	}
	u := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(addr, fmt.Sprintf("%d", port)),
		Path:   path,
	}
	if params != nil {
		u.RawQuery = params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.agentToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.agentToken)
	}
	return c.doRequest(req)
}

func (c *Client) doRequest(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, MaxResponseBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if int64(len(body)) > MaxResponseBytes {
		return nil, resp.StatusCode, fmt.Errorf("agent response too large (> %d bytes)", MaxResponseBytes)
	}
	if resp.StatusCode >= 400 {
		return nil, resp.StatusCode, fmt.Errorf("agent returned %d: %s", resp.StatusCode, string(body))
	}
	return body, resp.StatusCode, nil
}

func ValidateAgentAddress(addr string) error {
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil
	}
	for _, cidr := range deniedCIDRs {
		if cidr.Contains(ip) {
			return fmt.Errorf("address %s is in denied range %s", addr, cidr)
		}
	}
	return nil
}

func parseCIDR(s string) *net.IPNet {
	_, n, _ := net.ParseCIDR(s)
	return n
}
