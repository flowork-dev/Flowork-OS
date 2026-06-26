package main

import "testing"

// TestTunnelRegistry buktiin provider tunnel BARU bisa didaftar via registry (pola sibling
// init()) → muncul di list, bisa di-get — TANPA edit file frozen. Built-in tailscale (sibling)
// juga harus terdaftar.
func TestTunnelRegistry(t *testing.T) {
	if _, ok := getTunnelProvider("tailscale"); !ok {
		t.Fatal("tailscale (sibling) harus terdaftar di registry")
	}
	called := false
	RegisterTunnelProvider(TunnelProvider{
		Name:   "dummy-ngrok",
		Detect: func() bool { return true },
		Enable: func(target string) (map[string]any, error) {
			called = true
			return map[string]any{"enabled": true, "target": target}, nil
		},
	})
	p, ok := getTunnelProvider("dummy-ngrok")
	if !ok {
		t.Fatal("provider baru via Register harus muncul")
	}
	if _, err := p.Enable("http://x"); err != nil || !called {
		t.Fatal("Enable provider baru harus jalan")
	}
	found := false
	for _, n := range tunnelProviderNames() {
		if n == "dummy-ngrok" {
			found = true
		}
	}
	if !found {
		t.Fatal("provider baru harus muncul di tunnelProviderNames()")
	}
}
