package netguard

import (
	"testing"
)

func TestValidateNodeTargetValid(t *testing.T) {
	opts := ValidateNodeTargetOptions{
		AllowPrivate:     true,
		AllowedProtocols: []string{"http", "https"},
		Resolve:          false,
	}

	res, err := ValidateNodeTarget("127.0.0.1", 8081, "http", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.BaseURL() != "http://127.0.0.1:8081" {
		t.Fatalf("unexpected BaseURL: %s", res.BaseURL())
	}

	// IPv6 host
	res, err = ValidateNodeTarget("::1", 8081, "https", opts)
	if err != nil {
		t.Fatalf("unexpected error on IPv6: %v", err)
	}
	if res.BaseURL() != "https://[::1]:8081" {
		t.Fatalf("unexpected BaseURL on IPv6: %s", res.BaseURL())
	}
}

func TestValidateNodeTargetBlockedPorts(t *testing.T) {
	opts := ValidateNodeTargetOptions{
		AllowPrivate:     true,
		AllowedProtocols: []string{"http", "https"},
		Resolve:          false,
	}

	blocked := []int{22, 23, 25, 445, 3306, 5432, 6379, 27017}
	for _, port := range blocked {
		_, err := ValidateNodeTarget("127.0.0.1", port, "http", opts)
		if err == nil {
			t.Fatalf("expected port %d to be blocked", port)
		}
	}
}

func TestValidateNodeTargetInvalidProtocols(t *testing.T) {
	opts := ValidateNodeTargetOptions{
		AllowPrivate:     true,
		AllowedProtocols: []string{"http", "https"},
		Resolve:          false,
	}

	invalid := []string{"ftp", "gopher", "file", "ssh", ""}
	for _, proto := range invalid {
		_, err := ValidateNodeTarget("127.0.0.1", 8081, proto, opts)
		if err == nil {
			t.Fatalf("expected protocol %q to be rejected", proto)
		}
	}
}

func TestValidateNodeTargetPrivateIPs(t *testing.T) {
	optsDisallow := ValidateNodeTargetOptions{
		AllowPrivate:     false,
		AllowedProtocols: []string{"http", "https"},
		Resolve:          true,
	}

	// Private / Loopback IPs should be rejected when AllowPrivate = false
	for _, ip := range []string{"127.0.0.1", "10.0.0.1", "192.168.1.1", "172.16.0.1", "169.254.169.254"} {
		_, err := ValidateNodeTarget(ip, 8081, "http", optsDisallow)
		if err == nil {
			t.Fatalf("expected private IP %s to be rejected when AllowPrivate=false", ip)
		}
	}
}
