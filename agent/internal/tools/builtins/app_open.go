// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package builtins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"flowork-gui/internal/tools"
)

// SELF-REGISTER (plug-in NON-frozen, 📄 lock/tool-manager.md). Edit/hapus bebas tanpa unfreeze.
func init() { tools.Register(&appOpenTool{}) }

var appDefaults = map[string]map[string][]string{
	"chrome": {
		"linux":   {"google-chrome", "google-chrome-stable", "chromium", "chromium-browser"},
		"windows": {"chrome"},
		"darwin":  {"Google Chrome"},
	},
	"code": {
		"linux":   {"code", "codium"},
		"windows": {"code.cmd", "code"},
		"darwin":  {"Visual Studio Code"},
	},
}

var appAliases = map[string]string{
	"chrome": "chrome", "google chrome": "chrome", "google-chrome": "chrome", "browser": "chrome",
	"code": "code", "vscode": "code", "vs code": "code", "visual studio code": "code", "vsc": "code",
}

func loadAppAllow() map[string]map[string][]string {
	out := map[string]map[string][]string{}
	for k, v := range appDefaults {
		out[k] = v
	}
	if home, err := os.UserHomeDir(); err == nil {
		if raw, err := os.ReadFile(filepath.Join(home, ".flowork", "operator-apps.json")); err == nil {
			var extra map[string]map[string][]string
			if json.Unmarshal(raw, &extra) == nil {
				for k, v := range extra {
					out[strings.ToLower(strings.TrimSpace(k))] = v
				}
			}
		}
	}
	return out
}

type appOpenTool struct{}

func (appOpenTool) Name() string       { return "app_open" }
func (appOpenTool) Capability() string { return "exec:app" }
func (appOpenTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Launch a whitelisted desktop app on the host computer (e.g. chrome, code). Safe: only apps in the owner's allowlist can be opened — the request can never run an arbitrary command. Requires exec:app (operator agent). Use for 'open/buka <app>' requests.",
		Params: []tools.Param{
			{Name: "app", Type: tools.ParamString, Description: "app to open: chrome | code (aliases: vscode, browser, …)", Required: true},
		},
		Returns: "{opened, app, command} — or {opened:false, error, allowed:[…]} if not in the allowlist / not installed",
	}
}

func (appOpenTool) Run(_ context.Context, args map[string]any) (tools.Result, error) {
	raw, _ := args["app"].(string)
	want := strings.ToLower(strings.TrimSpace(raw))
	if want == "" {
		return tools.Result{}, fmt.Errorf("app required")
	}
	allow := loadAppAllow()
	key, ok := appAliases[want]
	if !ok {
		if _, direct := allow[want]; direct {
			key = want
		}
	}
	allowedKeys := make([]string, 0, len(allow))
	for k := range allow {
		allowedKeys = append(allowedKeys, k)
	}
	perOS, known := allow[key]
	if key == "" || !known {
		return tools.Result{Output: map[string]any{"opened": false, "error": "app not in allowlist", "requested": raw, "allowed": allowedKeys}}, nil
	}
	cands := perOS[runtime.GOOS]
	if len(cands) == 0 {
		return tools.Result{Output: map[string]any{"opened": false, "error": "app not configured for this OS (" + runtime.GOOS + ")", "app": key}}, nil
	}

	var cmd *exec.Cmd
	var resolved string
	switch runtime.GOOS {
	case "darwin":
		resolved = cands[0]
		cmd = exec.Command("open", "-a", resolved)
	case "windows":
		for _, c := range cands {
			if p, err := exec.LookPath(c); err == nil {
				resolved = p
				break
			}
		}
		if resolved == "" {
			resolved = cands[0]
		}

		cmd = exec.Command("cmd", "/c", "start", "", resolved)
	default:
		for _, c := range cands {
			if p, err := exec.LookPath(c); err == nil {
				resolved = p
				break
			}
		}
		if resolved == "" {
			return tools.Result{Output: map[string]any{"opened": false, "error": "not installed (none of the candidates found on PATH)", "app": key, "candidates": cands}}, nil
		}
		cmd = exec.Command(resolved)
	}

	if err := cmd.Start(); err != nil {
		return tools.Result{Output: map[string]any{"opened": false, "error": err.Error(), "app": key, "command": resolved}}, nil
	}
	_ = cmd.Process.Release()
	return tools.Result{
		Output: map[string]any{"opened": true, "app": key, "command": resolved},
		Note:   "launched " + key,
	}, nil
}
