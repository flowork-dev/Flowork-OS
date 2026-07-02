// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// ⚠️ FROZEN — jangan edit file ini. 📄 Dok: lock/tools.md
// Nambah aksi power TANPA buka frozen: pakai SEAM tools.Register() di file sibling non-frozen.
// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// 📄 Dok: lock/tool-manager.md — TOOL PLUG-IN (NON-frozen, self-register).
// Manager (registry/sandbox/interceptor) yg BEKU; tool ini BEBAS diedit/hapus/tambah
// TANPA buka freeze. Daftar ke papan-colokan (registry beku) via init() di bawah.

package builtins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/tools"
)

// SELF-REGISTER ke manager (papan-colokan beku). Hapus file ini → tool ilang, inti tetep berdiri.
func init() { tools.Register(&systemPowerTool{}) }

type systemPowerTool struct{}

func (systemPowerTool) Name() string       { return "system_power" }
func (systemPowerTool) Capability() string { return "exec:power" }
func (systemPowerTool) Schema() tools.Schema {
	return tools.Schema{
		Description: "Kontrol power HOST: shutdown|reboot|suspend|lock|logout|cancel. Butuh cap exec:power (operator tepercaya), tiap aksi di-audit. Eksekusi NYATA cuma kalau ARMED (env FLOWORK_POWER_ARMED), else dry-run. WAJIB konfirmasi user sebelum shutdown/reboot.",
		Params: []tools.Param{
			{Name: "action", Type: tools.ParamString, Description: "shutdown | reboot | suspend | lock | logout | cancel", Required: true},
			{Name: "delay_seconds", Type: tools.ParamInt, Description: "jeda sebelum eksekusi (window cancel); default 10, max 3600"},
			{Name: "reason", Type: tools.ParamString, Description: "alasan singkat, masuk audit log"},
		},
		Returns: "{status, action, delay_seconds, armed, command, message}",
	}
}

var validPowerActions = map[string]bool{
	"shutdown": true,
	"reboot":   true,
	"suspend":  true,
	"lock":     true,
	"logout":   true,
	"cancel":   true,
}

var (
	powerMu      sync.Mutex
	powerTimer   *time.Timer
	powerPending string
)

func powerArmed() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("FLOWORK_POWER_ARMED"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func (systemPowerTool) Run(ctx context.Context, args map[string]any) (tools.Result, error) {
	action, _ := args["action"].(string)
	action = strings.ToLower(strings.TrimSpace(action))
	if !validPowerActions[action] {
		return tools.Result{}, fmt.Errorf("system_power: invalid action %q (want shutdown|reboot|suspend|lock|logout|cancel)", action)
	}

	store, _ := tools.FromStore(ctx)
	caller := tools.FromCaller(ctx)
	reason, _ := args["reason"].(string)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "(no reason given)"
	}

	if action == "cancel" {
		powerMu.Lock()
		had := powerTimer != nil
		desc := powerPending
		if powerTimer != nil {
			powerTimer.Stop()
			powerTimer = nil
			powerPending = ""
		}
		powerMu.Unlock()
		writePowerAudit(store, caller, action, reason, "(cancel)", powerArmed(), agentdb.AuditSevInfo)
		if had {
			return tools.Result{Output: map[string]any{
				"status":  "cancelled",
				"action":  "cancel",
				"message": "Pending power action cancelled: " + desc,
			}}, nil
		}
		return tools.Result{Output: map[string]any{
			"status":  "nothing_pending",
			"action":  "cancel",
			"message": "No pending power action to cancel.",
		}}, nil
	}

	delay := 10
	if v, ok := args["delay_seconds"].(float64); ok {
		delay = int(v)
	}
	if v, ok := args["delay_seconds"].(int); ok {
		delay = v
	}
	if delay < 0 {
		delay = 0
	}
	if delay > 3600 {
		delay = 3600
	}

	argv, err := resolvePowerCmd(action)
	if err != nil {
		return tools.Result{}, err
	}
	cmdStr := strings.Join(argv, " ")
	armed := powerArmed()

	sev := agentdb.AuditSevWarning
	if action == "lock" || action == "logout" {
		sev = agentdb.AuditSevInfo
	}
	writePowerAudit(store, caller, action, reason, cmdStr, armed, sev)

	if !armed {
		return tools.Result{
			Output: map[string]any{
				"status":        "dry_run",
				"action":        action,
				"delay_seconds": delay,
				"armed":         false,
				"command":       cmdStr,
				"message":       fmt.Sprintf("DRY-RUN: would run %q in %ds. Host is NOT armed — set FLOWORK_POWER_ARMED=1 to enable real power control.", cmdStr, delay),
			},
			Note: "host not armed (FLOWORK_POWER_ARMED unset) — no action taken",
		}, nil
	}

	desc := fmt.Sprintf("%s (in %ds) — %s", action, delay, reason)
	powerMu.Lock()
	if powerTimer != nil {
		powerTimer.Stop()
	}
	powerPending = desc
	powerTimer = time.AfterFunc(time.Duration(delay)*time.Second, func() {
		defer func() {
			if r := recover(); r != nil {
				writePowerAudit(store, caller, action, fmt.Sprintf("panic during exec: %v", r), cmdStr, true, agentdb.AuditSevError)
			}
		}()
		powerMu.Lock()
		powerTimer = nil
		powerPending = ""
		powerMu.Unlock()
		c := exec.Command(argv[0], argv[1:]...)
		c.Env = os.Environ()
		if runErr := c.Run(); runErr != nil {

			writePowerAudit(store, caller, action, "exec failed: "+runErr.Error(), cmdStr, true, agentdb.AuditSevError)
		}
	})
	powerMu.Unlock()

	return tools.Result{Output: map[string]any{
		"status":        "scheduled",
		"action":        action,
		"delay_seconds": delay,
		"armed":         true,
		"command":       cmdStr,
		"message":       fmt.Sprintf("Power action %q scheduled in %ds. Call system_power with action=cancel within the window to abort.", action, delay),
	}}, nil
}

func resolvePowerCmd(action string) ([]string, error) {
	return resolvePowerCmdFor(runtime.GOOS, action)
}

func resolvePowerCmdFor(goos, action string) ([]string, error) {
	switch goos {
	case "linux":
		switch action {
		case "shutdown":
			return []string{"systemctl", "poweroff"}, nil
		case "reboot":
			return []string{"systemctl", "reboot"}, nil
		case "suspend":
			return []string{"systemctl", "suspend"}, nil
		case "lock":
			return []string{"loginctl", "lock-session"}, nil
		case "logout":
			user := os.Getenv("USER")
			if user == "" {
				return nil, fmt.Errorf("system_power: $USER kosong jadi belum bisa logout. Petunjuk: jalankan Flowork di sesi login user (bukan service tanpa env), atau set env USER")
			}
			return []string{"loginctl", "terminate-user", user}, nil
		}
	case "darwin":
		switch action {
		case "shutdown":
			return []string{"osascript", "-e", `tell app "System Events" to shut down`}, nil
		case "reboot":
			return []string{"osascript", "-e", `tell app "System Events" to restart`}, nil
		case "suspend":
			return []string{"pmset", "sleepnow"}, nil
		case "lock":
			return []string{"pmset", "displaysleepnow"}, nil
		case "logout":
			return []string{"osascript", "-e", `tell app "System Events" to log out`}, nil
		}
	case "windows":
		switch action {
		case "shutdown":
			return []string{"shutdown.exe", "/s", "/t", "0"}, nil
		case "reboot":
			return []string{"shutdown.exe", "/r", "/t", "0"}, nil
		case "suspend":
			return []string{"rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0"}, nil
		case "lock":
			return []string{"rundll32.exe", "user32.dll,LockWorkStation"}, nil
		case "logout":
			return []string{"shutdown.exe", "/l"}, nil
		}
	case "android":

		return nil, fmt.Errorf("system_power di Android dibedakan: shutdown/reboot butuh ROOT (`svc power shutdown` / `reboot -p`). Petunjuk: di Android pakai fitur lain (notifikasi/akun), kontrol daya OS diserahkan ke user — ini normal, bukan kesalahan lo")
	}
	return nil, fmt.Errorf("system_power: aksi %q belum dipetakan untuk OS %q. Petunjuk: OS yang didukung penuh = linux (termasuk RasPi/STB), darwin (macOS), windows. Kalau ini OS baru, perlu tambah cabang di resolvePowerCmdFor", action, goos)
}

func writePowerAudit(store *agentdb.Store, caller, action, reason, cmdStr string, armed bool, sev string) {
	if store == nil {
		return
	}
	detail, _ := json.Marshal(map[string]any{
		"tool":    "system_power",
		"action":  action,
		"reason":  reason,
		"command": cmdStr,
		"armed":   armed,
	})
	_, _ = store.AppendAudit(agentdb.AuditEntry{
		EventType:  agentdb.EventToolCall,
		Severity:   sev,
		Actor:      caller,
		DetailJSON: string(detail),
	})
}
