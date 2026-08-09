package ldap

import (
	"crypto/tls"
	"testing"
)

func TestHostnameFromServerURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "ldap with port", raw: "ldap://ldap.example.com:389", want: "ldap.example.com"},
		{name: "ldaps with port", raw: "ldaps://ldap.example.com:636", want: "ldap.example.com"},
		{name: "host only", raw: "ldap.example.com", want: "ldap.example.com"},
		{name: "host with port no scheme", raw: "ldap.example.com:389", want: "ldap.example.com"},
		{name: "empty", raw: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hostnameFromServerURL(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("hostnameFromServerURL: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func TestTLSConfigFromServerURL(t *testing.T) {
	cfg, err := tlsConfigFromServerURL("ldap://ldap.example.com:389")
	if err != nil {
		t.Fatalf("tlsConfigFromServerURL: %v", err)
	}
	if cfg.ServerName != "ldap.example.com" {
		t.Fatalf("ServerName=%q", cfg.ServerName)
	}
	if cfg.MinVersion != tls.VersionTLS12 {
		t.Fatalf("MinVersion=%v want TLS1.2", cfg.MinVersion)
	}
	if cfg.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify must be false")
	}
}
