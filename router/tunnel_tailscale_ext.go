// tunnel_tailscale_ext.go — provider tunnel "tailscale" via registry (NON-frozen, deletable).
// Reuse helper di handlers_tunnel.go (runShort/extractTailscaleURL/extractAuthURL). Hapus file
// ini → tailscale ilang dari /api/tunnel/providers, build TETAP OK (registry 1 entri kurang).
// Endpoint khusus /api/tunnel/tailscale-* di handler frozen tetap jalan (jalur lama).
//
// Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
package main

import (
	"os/exec"
	"strings"
)

func init() {
	RegisterTunnelProvider(TunnelProvider{
		Name: "tailscale",
		Detect: func() bool {
			_, err := exec.LookPath("tailscale")
			return err == nil
		},
		Status: func() map[string]any {
			out := map[string]any{"installed": false}
			if _, err := exec.LookPath("tailscale"); err != nil {
				return out
			}
			out["installed"] = true
			if s, err := runShort("tailscale", "status", "--json"); err == nil {
				out["enabled"] = strings.Contains(s, `"BackendState":"Running"`)
				if u := extractTailscaleURL(s); u != "" {
					out["url"] = u
				}
			}
			return out
		},
		Enable: func(target string) (map[string]any, error) {
			out, err := runShort("tailscale", "up", "--accept-routes", "--accept-dns=true")
			res := map[string]any{"output": out}
			if u := extractAuthURL(out); u != "" {
				res["authUrl"] = u
			}
			if err == nil {
				res["enabled"] = true
			}
			return res, err
		},
		Disable: func() (map[string]any, error) {
			out, err := runShort("tailscale", "down")
			res := map[string]any{"output": out}
			if err == nil {
				res["disabled"] = true
			}
			return res, err
		},
	})
}
