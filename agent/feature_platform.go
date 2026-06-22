// feature_platform.go — FASE-B: platform/infra routes (system, groups, pages, settings,
// kernel+loket, connections, mcp). Self-register (PhaseRoute). Local servePage + loketSvc
// di-build di sini (pola: feature build local-nya sendiri pakai Deps).
package main

import (
	"context"
	"io/fs"
	"net/http"
	"time"

	"flowork-gui/internal/connections"
	"flowork-gui/internal/mcphub"
	"flowork-gui/internal/settingsapi"
)

func init() {
	RegisterFeature(Feature{Name: "platform", Phase: PhaseRoute, Apply: func(d *Deps) {
		d.Mux.HandleFunc("/api/system/health", systemHealth)

		// Groups (§F2).
		d.Mux.HandleFunc("/api/groups", d.GroupsAPI.ListHandler)
		d.Mux.HandleFunc("/api/groups/config", d.GroupsAPI.ConfigHandler)
		d.Mux.HandleFunc("/api/groups/create", d.GroupsAPI.CreateHandler)
		d.Mux.HandleFunc("/api/groups/delete", d.GroupsAPI.DeleteHandler)
		d.Mux.HandleFunc("/api/groups/toggle", d.GroupsAPI.ToggleHandler)
		d.Mux.HandleFunc("/api/groups/reset", d.GroupsAPI.ResetHandler)

		// Page routes — serve embedded HTML (FileServer cuma map exact filename).
		servePage := func(name string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				data, rerr := fs.ReadFile(d.StaticFS, name)
				if rerr != nil {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(data)
			}
		}
		d.Mux.HandleFunc("/login", servePage("login.html"))
		d.Mux.HandleFunc("/register", servePage("register.html"))

		// Settings (owner-level, flowork.db global).
		d.Mux.HandleFunc("/api/settings/keys", d.SettingsAPI.KeysHandler)
		d.Mux.HandleFunc("/api/settings/router-default", d.SettingsAPI.RouterDefaultHandler)
		d.Mux.HandleFunc("/api/settings/notify", d.SettingsAPI.NotifyHandler)
		settingsapi.TestNotifyFunc = notifyOwnerTelegram
		d.Mux.HandleFunc("/api/notify", notifyHubHandler)
		d.Mux.HandleFunc("/api/settings/youtube", d.SettingsAPI.YouTubeStatusHandler)
		d.Mux.HandleFunc("/api/settings/youtube/credentials", d.SettingsAPI.YouTubeCredentialsHandler)
		d.Mux.HandleFunc("/api/settings/youtube/connect", d.SettingsAPI.YouTubeConnectHandler)
		d.Mux.HandleFunc("/api/settings/youtube/disconnect", d.SettingsAPI.YouTubeDisconnectHandler)
		d.Mux.HandleFunc("/api/settings/youtube/config", d.SettingsAPI.YouTubeConfigHandler)

		// Kernel introspection + loket (single-endpoint microkernel).
		d.Mux.HandleFunc("/api/kernel/status", d.Host.StatusHandler)
		d.Mux.HandleFunc("/api/kernel/agents", d.Host.AgentsHandler)
		d.Mux.HandleFunc("/api/kernel/rpc", d.Host.RPCHandler)
		d.Mux.HandleFunc("/api/agents/ui-schema", d.Host.UISchemaHandler)
		loketSvc := wireLoket(d.Host)
		d.Mux.HandleFunc("/api/kernel/call", loketSvc.CallHandler)
		d.Mux.HandleFunc("/api/kernel/gui", loketSvc.GUIHandler)
		d.Mux.HandleFunc("/api/kernel/webhook/", loketSvc.WebhookHandler)

		// Connections — universal connector registry.
		d.Mux.HandleFunc("/api/connections", connections.ListHandler)
		d.Mux.HandleFunc("/api/connections/toggle", connections.ToggleHandler)
		d.Mux.HandleFunc("/api/connections/config", connections.ConfigHandler)
		d.Mux.HandleFunc("/api/connections/uninstall", connections.UninstallHandler)

		// MCP connectors (external MCP servers as agent tool-sources).
		d.Mux.HandleFunc("/api/mcp", mcphub.ListHandler)
		d.Mux.HandleFunc("/api/mcp/install", mcphub.InstallHandler)
		d.Mux.HandleFunc("/api/mcp/enable", mcphub.EnableHandler)
		d.Mux.HandleFunc("/api/mcp/disable", mcphub.DisableHandler)
		d.Mux.HandleFunc("/api/mcp/uninstall", mcphub.UninstallHandler)
		// Auto-start installed MCP connectors (best-effort, goroutine — jangan delay boot).
		go func() {
			ec, ecancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer ecancel()
			mcphub.Default.EnableAll(ec)
		}()
	}})
}
