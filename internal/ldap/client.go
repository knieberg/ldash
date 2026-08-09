package ldap

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"

	"github.com/knieberg/ldash/internal/config"
)

// Client wraps an LDAP connection for admin operations.
type Client struct {
	cfg    *config.Config
	conn   *ldap.Conn
	bound  bool
}

type PingResult struct {
	OK       bool
	Message  string
	BaseDN   string
	Server   string
	Duration time.Duration
}

func NewClient(cfg *config.Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Connect(bindPassword string) error {
	url := c.cfg.Server.URL
	var conn *ldap.Conn
	var err error

	switch strings.ToLower(c.cfg.Server.TLSMode) {
	case "ldaps":
		conn, err = ldap.DialURL(normalizeLDAPS(url))
	case "starttls":
		conn, err = ldap.DialURL(normalizeLDAP(url))
		if err == nil {
			err = conn.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12})
		}
	default:
		conn, err = ldap.DialURL(normalizeLDAP(url))
	}
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	if err := conn.Bind(c.cfg.BindDN, bindPassword); err != nil {
		conn.Close()
		return fmt.Errorf("bind: %w", err)
	}

	c.conn = conn
	c.bound = true
	return nil
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		c.bound = false
	}
}

// Conn exposes the underlying connection for advanced callers (tests).
func (c *Client) Connected() bool {
	return c.bound && c.conn != nil
}

func (c *Client) Ping(bindPassword string) PingResult {
	start := time.Now()
	if err := c.Connect(bindPassword); err != nil {
		return PingResult{
			OK:       false,
			Message:  err.Error(),
			BaseDN:   c.cfg.BaseDN,
			Server:   c.cfg.Server.URL,
			Duration: time.Since(start),
		}
	}
	defer c.Close()

	req := ldap.NewSearchRequest(
		c.cfg.BaseDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		1, 0, false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)
	if _, err := c.conn.Search(req); err != nil {
		return PingResult{
			OK:       false,
			Message:  fmt.Sprintf("search base: %v", err),
			BaseDN:   c.cfg.BaseDN,
			Server:   c.cfg.Server.URL,
			Duration: time.Since(start),
		}
	}

	return PingResult{
		OK:       true,
		Message:  "connected",
		BaseDN:   c.cfg.BaseDN,
		Server:   c.cfg.Server.URL,
		Duration: time.Since(start),
	}
}

func normalizeLDAP(url string) string {
	if strings.HasPrefix(url, "ldap://") || strings.HasPrefix(url, "ldaps://") {
		return url
	}
	return "ldap://" + url
}

func normalizeLDAPS(url string) string {
	if strings.HasPrefix(url, "ldaps://") {
		return url
	}
	if strings.HasPrefix(url, "ldap://") {
		return "ldaps://" + strings.TrimPrefix(url, "ldap://")
	}
	return "ldaps://" + url
}
