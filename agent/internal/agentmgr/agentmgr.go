// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30 (re-audited 2026-06-06; security audit 2026-06-11)
// Update 2026-06-20 (owner-approved): InteractionsHandler +method DELETE = CLEAR
//   history (§6.1 anti-anchor) via PruneInteractions (param older_days; default=semua).
//   Soft-delete chat-log di workspace PERSONAL — twin/graph/brain di store beda, aman.
//   Additive. Re-locked.
// Update 2026-06-11 (owner-approved security audit, unfreeze→refreeze): (1) the
//   loopback-secret check in ToolRunHandler now uses subtle.ConstantTimeCompare
//   instead of ==; (2) extractFile caps per-entry extraction with a 64MB
//   LimitReader (zip-bomb guard, matches pack_extract.go / plugin_handler.go).
// Update 2026-06-06 (owner-approved AI-Agent audit): ConfigHandler POST now calls
//   reconcileMaskedSecrets() before Save — GET masks secrets to ••••<last4>, the GUI
//   posts the whole form back, and Save full-replaces the secrets table, so an
//   untouched secret would otherwise be clobbered by its own mask (agent loses its
//   bot token / API keys). Fixed + unit-tested (secret_reconcile_test.go).
// Reason: Agent CRUD HTTP handlers (CRITICAL). 21 handlers, all share pattern:
//   - reID regex validation untuk semua `?id=` param (anti path traversal via id)
//   - agentFolder(id) deterministic resolve (filepath.Join, no concat)
//   - UploadHandler: multipart cap 32MB, body cap 64MB, .zip ext only, manifest
//     reID check, **path traversal guard** filepath.Rel + ".." prefix (line 134-137)
//   - DownloadHandler: zip output dengan id-namespaced entries
//   - RemoveHandler: os.RemoveAll dengan agentFolder(id) only — bukan user path
//   - DBResetHandler: os.Remove dbPath + recreate (idempotent)
//   - Tool/Slash: delegate ke tools/slashcmd registries, hooks chain enforce caps
//   - No direct SQL — semua via agentdb.Store
//   - File modes: 0o755 dir, 0o644 file (POSIX, Windows ignore)
//   - HTTP method check di setiap handler (anti CSRF via wrong method)

package agentmgr

import (
	"archive/zip"
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/httpx"
	"flowork-gui/internal/kernel/loader"
	"flowork-gui/internal/slashcmd"
	"flowork-gui/internal/tools"
	"flowork-gui/internal/toolsidecar"
)

// reID — sinkron sama loader.manifest.go reID.
var reID = regexp.MustCompile(`^[a-z][a-z0-9-]{2,31}$`)

// Reload — kernelhost daftarkan callback di main.go. Dipanggil setelah
// config save / db reset supaya live instance pickup config baru (Unload
// + LoadInstance + AutoBoot daemon → env baru kebawa).
var Reload func(agentID string) error

// agentFolder return absolute path ke folder agent.
func agentFolder(id string) string {
	return filepath.Join(loader.AgentsDir(), id+".fwagent")
}

// UploadHandler — POST /api/agents/upload. multipart `file` berisi
// .fwagent.zip. Manifest minimal divalidasi (id, kind, entry) sebelum
// extract.
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "parse form: " + err.Error()})
		return
	}
	file, hdr, err := r.FormFile("file")
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "missing file field"})
		return
	}
	defer file.Close()

	lower := strings.ToLower(hdr.Filename)
	if !strings.HasSuffix(lower, ".zip") {
		httpx.WriteJSON(w, map[string]any{"error": "expected .fwagent.zip filename"})
		return
	}

	raw, err := io.ReadAll(io.LimitReader(file, 64<<20))
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "read upload: " + err.Error()})
		return
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "not a valid zip: " + err.Error()})
		return
	}

	manifestEntry, rootPrefix, err := findManifest(zr)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	manifestBody, err := readZipEntry(manifestEntry)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "read manifest: " + err.Error()})
		return
	}
	var manifest struct {
		ID    string `json:"id"`
		Kind  string `json:"kind"`
		Entry string `json:"entry"`
	}
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "manifest parse: " + err.Error()})
		return
	}
	if !reID.MatchString(manifest.ID) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid manifest.id"})
		return
	}

	targetDir := agentFolder(manifest.ID)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "mkdir target: " + err.Error()})
		return
	}
	written := 0
	for _, f := range zr.File {
		rel := f.Name
		if rootPrefix != "" {
			if !strings.HasPrefix(rel, rootPrefix) {
				continue
			}
			rel = strings.TrimPrefix(rel, rootPrefix)
		}
		if rel == "" || strings.HasSuffix(rel, "/") {
			continue
		}
		dest := filepath.Join(targetDir, filepath.FromSlash(rel))
		clean, err := filepath.Rel(targetDir, dest)
		if err != nil || strings.HasPrefix(clean, "..") {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "mkdir: " + err.Error()})
			return
		}
		if err := extractFile(f, dest); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "extract " + rel + ": " + err.Error()})
			return
		}
		written++
	}
	httpx.WriteJSON(w, map[string]any{
		"ok":            true,
		"agent_id":      manifest.ID,
		"kind":          manifest.Kind,
		"entry":         manifest.Entry,
		"files_written": written,
		"target_dir":    targetDir,
		"next":          "kernel hot-reload pickup otomatis",
	})
}

// DownloadHandler — GET /api/agents/download?id=<id>.
//
// Bundle SELURUH folder agent jadi .fwagent.zip — termasuk:
//   - manifest.json
//   - agent.wasm (kalau di-stage) atau main.go (source)
//   - workspace/ + state.db (SQLite per-agent berisi semua setting)
//   - ui/, i18n/, sub-folder lain
//
// Authoritative source: `<project>/agents/<id>/` kalau ada (preserve
// source code + state lengkap). Fallback ke staged
// `~/.flowork/agents/<id>.fwagent/` (untuk agent yang di-install dari
// zip tanpa source).
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	// Resolve source dulu (authoritative + state lengkap).
	srcDir := ""
	if cwd, err := os.Getwd(); err == nil {
		cand := filepath.Join(cwd, "agents", id)
		if stat, err := os.Stat(cand); err == nil && stat.IsDir() {
			srcDir = cand
		}
	}
	if srcDir == "" {
		srcDir = agentFolder(id) // fallback ke staged
	}
	if _, err := os.Stat(srcDir); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "agent not found: " + id})
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.fwagent.zip"`, id))
	w.Header().Set("Cache-Control", "no-store")

	zw := zip.NewWriter(w)
	defer zw.Close()

	_ = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil || rel == "." {
			return nil
		}
		// Skip artifact build (.git, node_modules, dst). Workspace +
		// state.db sengaja diikutin biar agent fully portable.
		base := filepath.Base(rel)
		if base == ".git" || base == "node_modules" || base == ".reload-trigger" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		zipPath := filepath.ToSlash(rel)
		if info.IsDir() {
			// Bikin entry dir biar zip preserve folder kosong (mis. workspace/cache/).
			_, _ = zw.Create(zipPath + "/")
			return nil
		}
		f, err := zw.Create(zipPath)
		if err != nil {
			return nil
		}
		src, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer src.Close()
		_, _ = io.Copy(f, src)
		return nil
	})
}

// ToggleHandler — POST /api/agents/toggle?id=<id>&disabled=<0|1>. Flip
// enable/disable di DB (meta.disabled) lalu reload agent. Switch off →
// kernel unload + daemon stop. Switch on → instantiate + auto-boot.
func ToggleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	// Source-aware (fix bug.md #1): cek source repo dulu, baru staged.
	dir, ok := resolveAgentDir(id)
	if !ok {
		httpx.WriteJSON(w, map[string]any{"error": "agent not found"})
		return
	}
	disabledFlag := strings.TrimSpace(r.URL.Query().Get("disabled"))
	disabled := disabledFlag == "1" || strings.EqualFold(disabledFlag, "true")

	dbPath := agentdb.Resolve(id, dir)
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	if err := store.SetDisabled(disabled); err != nil {
		store.Close()
		httpx.WriteJSON(w, map[string]any{"error": "set: " + err.Error()})
		return
	}
	store.Close()

	reloadErr := ""
	if Reload != nil {
		if err := Reload(id); err != nil {
			reloadErr = err.Error()
			log.Printf("toggle: reload %s: %v", id, err)
		}
	}
	httpx.WriteJSON(w, map[string]any{
		"ok":         true,
		"id":         id,
		"enabled":    !disabled,
		"reload_err": reloadErr,
	})
}

// DBResetHandler — POST /api/agents/db/reset?id=<id>. Hapus state.db
// agent → kernel hot-reload bakal touch ulang file kosong. Workspace
// folder lain (selain state.db) tetap utuh.
func DBResetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			httpx.WriteJSON(w, map[string]any{"ok": true, "note": "db belum ada, ngga ada yang di-reset"})
			return
		}
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	if err := os.Remove(dbPath); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "remove: " + err.Error()})
		return
	}
	// Touch ulang biar agent yang lagi jalan ngga error saat open.
	if f, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0o644); err == nil {
		_ = f.Close()
	}
	httpx.WriteJSON(w, map[string]any{"ok": true, "path": dbPath})
}

// WorkspaceRebuildIndex — kernelhost daftarkan callback. Resolve agent shared
// workspace path + invoke RebuildIndexFromDir. Nil-safe.
var WorkspaceRebuildIndex func(agentID string) (any, error)

// WorkspaceMetaHandler — multi-method endpoint /api/agents/workspace-meta?id=<id>
//
//	GET    list (?category=&limit=)
//	POST   trigger rebuild index (?action=rebuild)
//
// Roadmap section 6.
//
// ⚠️ NO over-prompt — workspace meta untuk inventory/discovery, JANGAN
// auto-inject ke system prompt.
func WorkspaceMetaHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		if r.Method == http.MethodGet {
			httpx.WriteJSON(w, map[string]any{"items": []any{}, "note": "db belum ada"})
			return
		}
		httpx.WriteJSON(w, map[string]any{"error": "db belum ada — boot agent dulu"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		store, err := agentdb.Open(dbPath)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
			return
		}
		defer store.Close()
		category := strings.TrimSpace(r.URL.Query().Get("category"))
		limit := 100
		if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
			if n, perr := strconv.Atoi(s); perr == nil && n > 0 && n <= 500 {
				limit = n
			}
		}
		items, err := store.ListMeta(category, limit)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "list: " + err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"items": items, "count": len(items)})

	case http.MethodPost:
		action := strings.TrimSpace(r.URL.Query().Get("action"))
		if action != "rebuild" {
			httpx.WriteJSON(w, map[string]any{"error": "unsupported action (only ?action=rebuild)"})
			return
		}
		if WorkspaceRebuildIndex == nil {
			httpx.WriteJSON(w, map[string]any{"error": "rebuild index not wired"})
			return
		}
		report, err := WorkspaceRebuildIndex(id)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"ok": true, "report": report})

	default:
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
	}
}

// KarmaHandler — GET /api/agents/karma?id=<id>[&key=<metric_key>]
// Tanpa key → list semua metric (cap 100 di DB layer).
// Dengan key → single metric (zero Karma + key kalau ngga ada — bukan error).
// Roadmap section 5.
//
// ⚠️ JANGAN auto-inject ke prompt (over-prompt risk). Dashboard / per-line
// teaser yang 1-baris OK.
func KarmaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (use GET)"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		httpx.WriteJSON(w, map[string]any{"items": []any{}, "note": "db belum ada"})
		return
	}
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	defer store.Close()

	key := strings.TrimSpace(r.URL.Query().Get("key"))
	if key != "" {
		k, err := store.GetKarma(key)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "get: " + err.Error()})
			return
		}
		httpx.WriteJSON(w, k)
		return
	}
	items, err := store.ListKarma()
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "list: " + err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "count": len(items)})
}

// DeathLetterHandler — multi-method endpoint /api/agents/death-letter?id=<id>
//
//	GET    list (?recipient=&sealed=1&limit=N)
//	POST   write new letter (body: {letter_type, recipient?, subject, body})
//	PUT    update unsealed letter (?letter_id=N body: {subject, body})
//	PATCH  seal letter (?letter_id=N&action=seal)
//
// Roadmap section 4. Sealed letter immutable — refuse update.
//
// ⚠️ Sensitif legacy content — JANGAN auto-inject ke prompt (over-prompt).
func DeathLetterHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		if r.Method == http.MethodGet {
			httpx.WriteJSON(w, map[string]any{"items": []any{}, "note": "db belum ada"})
			return
		}
		httpx.WriteJSON(w, map[string]any{"error": "db belum ada — boot agent dulu"})
		return
	}
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	defer store.Close()

	switch r.Method {
	case http.MethodGet:
		recipient := strings.TrimSpace(r.URL.Query().Get("recipient"))
		sealedOnly := r.URL.Query().Get("sealed") == "1"
		limit := 50
		if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
			if n, perr := strconv.Atoi(s); perr == nil && n > 0 && n <= 500 {
				limit = n
			}
		}
		items, err := store.ReadLetters(recipient, sealedOnly, limit)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "read: " + err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"items": items, "count": len(items)})

	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
		var body struct {
			LetterType string `json:"letter_type"`
			Recipient  string `json:"recipient"`
			Subject    string `json:"subject"`
			Body       string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "decode: " + err.Error()})
			return
		}
		newID, err := store.WriteLetter(body.LetterType, body.Recipient, body.Subject, body.Body)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"ok": true, "id": newID})

	case http.MethodPut:
		letterID, perr := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("letter_id")), 10, 64)
		if perr != nil || letterID <= 0 {
			httpx.WriteJSON(w, map[string]any{"error": "letter_id required"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
		var body struct {
			Subject string `json:"subject"`
			Body    string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "decode: " + err.Error()})
			return
		}
		if err := store.UpdateUnsealedLetter(letterID, body.Subject, body.Body); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"ok": true, "id": letterID})

	case http.MethodPatch:
		action := strings.TrimSpace(r.URL.Query().Get("action"))
		if action != "seal" {
			httpx.WriteJSON(w, map[string]any{"error": "unsupported action (only ?action=seal)"})
			return
		}
		letterID, perr := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("letter_id")), 10, 64)
		if perr != nil || letterID <= 0 {
			httpx.WriteJSON(w, map[string]any{"error": "letter_id required"})
			return
		}
		if err := store.SealLetter(letterID); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"ok": true, "sealed": letterID})

	default:
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
	}
}

// SlashRunHandler — POST /api/agents/slash/run?id=<agent_id>
// Body: {text, caller?}
//
// Parse + dispatch + log invocation.
//
// ⚠️ Phase 1: no rate limit / permission enforcement. Defer phase 2.
func SlashRunHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (use POST)"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "db belum ada — boot agent dulu"})
		return
	}
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	defer store.Close()

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var body struct {
		Text   string `json:"text"`
		Caller string `json:"caller"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "decode: " + err.Error()})
		return
	}
	text := strings.TrimSpace(body.Text)
	if text == "" {
		httpx.WriteJSON(w, map[string]any{"error": "text required"})
		return
	}
	caller := strings.TrimSpace(body.Caller)
	if caller == "" {
		caller = "http-admin"
	}

	// Section 15: inject store + caller + agent ke ctx supaya productive
	// commands bisa akses lewat slashcmd.FromStore.
	ctx := slashcmd.WithStore(r.Context(), store)
	ctx = slashcmd.WithCaller(ctx, caller)
	ctx = slashcmd.WithAgent(ctx, id)

	t0 := time.Now()
	result, cmdName, runErr := slashcmd.Dispatch(ctx, text)
	elapsedMs := time.Since(t0).Milliseconds()

	// Log invocation. cmdName mungkin kosong kalau parse fail di awal.
	argsRaw := ""
	if idx := strings.IndexAny(text, " \t"); idx >= 0 {
		argsRaw = strings.TrimSpace(text[idx+1:])
	}
	errText := ""
	if runErr != nil {
		errText = runErr.Error()
	}
	if _, lerr := store.LogSlashInvocation(cmdName, argsRaw, caller, result.Text, errText, elapsedMs); lerr != nil {
		log.Printf("agentmgr: log slash invocation failed: %v", lerr)
	}

	if runErr != nil {
		httpx.WriteJSON(w, map[string]any{
			"error":       runErr.Error(),
			"command":     cmdName,
			"duration_ms": elapsedMs,
		})
		return
	}
	httpx.WriteJSON(w, map[string]any{
		"ok":          true,
		"command":     cmdName,
		"result":      result,
		"duration_ms": elapsedMs,
	})
}

// SlashRegistryHandler — GET /api/agents/slash/registry
func SlashRegistryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (use GET)"})
		return
	}
	items := slashcmd.ListSummaries()
	httpx.WriteJSON(w, map[string]any{
		"items":        items,
		"count":        len(items),
		"algo_version": slashcmd.AlgoVersion,
	})
}

// SlashInvocationsHandler — GET /api/agents/slash-invocations?id=&command=&caller=&limit=
func SlashInvocationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (use GET)"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		httpx.WriteJSON(w, map[string]any{"items": []any{}, "note": "db belum ada"})
		return
	}
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	defer store.Close()

	command := strings.TrimSpace(r.URL.Query().Get("command"))
	caller := strings.TrimSpace(r.URL.Query().Get("caller"))
	limit := 50
	if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
		if n, perr := strconv.Atoi(s); perr == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	items, err := store.ListSlashInvocations(command, caller, limit)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "list: " + err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "count": len(items)})
}

// ToolRunHandler — POST /api/agents/tools/run?id=<agent_id>
// Body: {tool_name, args, caller?}
//
// Lookup tool dari registry, attach store ke ctx, dispatch Run, log
// invocation. Return Result atau error.
//
// ⚠️ Phase 1: no broker permission gate. Tool.Capability() declared tapi
// belum di-enforce. Defer phase 2.
func ToolRunHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (use POST)"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	// A2 isolation: a request carrying the host-injected loopback secret is a
	// guest-agent call; its X-Flowork-Caller (set by the kernel from the VERIFIED
	// pluginID, un-forgeable by the guest) is the authoritative identity. This
	// stops one agent running tools under another agent's id via ?id=. External
	// callers (GUI/scripts, no secret) keep using ?id (loopback-gated as before).
	if secret := strings.TrimSpace(os.Getenv("FLOWORK_LOOPBACK_SECRET")); secret != "" &&
		subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Flowork-Secret")), []byte(secret)) == 1 {
		if caller := strings.TrimSpace(r.Header.Get("X-Flowork-Caller")); caller != "" {
			if id != "" && id != caller {
				httpx.WriteJSON(w, map[string]any{"error": "agent identity mismatch (caller-bound execution)"})
				return
			}
			id = caller
		}
	}
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "db belum ada — boot agent dulu"})
		return
	}
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	defer store.Close()

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var body struct {
		ToolName string         `json:"tool_name"`
		Args     map[string]any `json:"args"`
		Caller   string         `json:"caller"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "decode: " + err.Error()})
		return
	}
	toolName := strings.TrimSpace(body.ToolName)
	if toolName == "" {
		httpx.WriteJSON(w, map[string]any{"error": "tool_name required"})
		return
	}
	caller := strings.TrimSpace(body.Caller)
	if caller == "" {
		caller = "http-admin"
	}

	t, ok := tools.Lookup(toolName)
	if !ok {
		// FASE 3 self-evolving: rekomendasi tool sepadan (drawer/semantic) → kalau bener ga ada,
		// ERROR-EDUKASI ngajarin tool_create. Detail: tool_notfound_edu.go.
		httpx.WriteJSON(w, map[string]any{"error": toolNotFoundEducation(toolName)})
		return
	}

	// Tier gate (enforcement chokepoint): tool primary-only (brain_search_shared
	// = korpus 5jt router) CUMA buat agent primary. Extension brain-nya di folder
	// sendiri (state.db lokal via brain_search) — ga nyolok 5jt. `id` di sini
	// authoritative (caller-bound, ga bisa di-spoof). brain.go locked → gate di sini.
	if IsPrimaryOnlyTool(toolName) && !IsPrimaryAgent(id) {
		httpx.WriteJSON(w, map[string]any{"error": "tool '" + toolName + "' khusus agent primary — extension pakai brain_search (brain lokal folder sendiri)"})
		return
	}
	// Run-guard tool PRIVAT (self-evolving, owner 2026-06-23): tool sidecar buatan-agent yg masih
	// PRIVAT cuma boleh dijalanin PEMBUATNYA (sampai lolos Dewan review → shared). `id` authoritative.
	if toolsidecar.IsPrivate(toolName) && toolsidecar.Owner(toolName) != id {
		httpx.WriteJSON(w, map[string]any{"error": "tool '" + toolName + "' masih PRIVAT punya agent lain — belum lolos review jadi shared"})
		return
	}

	// Inject store + caller + agent + shared dir into ctx, then dispatch.
	ctx := tools.WithStore(r.Context(), store)
	ctx = tools.WithCaller(ctx, caller)
	ctx = tools.WithAgent(ctx, id)
	if SharedDirForAgent != nil {
		if shared, derr := SharedDirForAgent(id); derr == nil && shared != "" {
			ctx = tools.WithSharedDir(ctx, shared)
		}
	}
	// Section 12: inject capability checker dari broker (kalau wired).
	if CapsCheckerForAgent != nil {
		if check := CapsCheckerForAgent(id); check != nil {
			ctx = tools.WithCapsChecker(ctx, check)
		}
	}

	t0 := time.Now()
	// Section 12 phase 3: SandboxRunV3 = approval queue + tool_audit append
	// + interceptor chain + 3 gate sandbox (cap/disabled/rate).
	result, runErr := tools.SandboxRunV3(ctx, t, body.Args, tools.SandboxOpts{})
	elapsedMs := time.Since(t0).Milliseconds()

	// Log invocation (best-effort — kalau log gagal, ngga blocking response).
	errText := ""
	if runErr != nil {
		errText = runErr.Error()
	}
	if _, lerr := store.LogToolInvocation(
		toolName,
		tools.MarshalArgs(body.Args),
		tools.MarshalResult(result),
		errText,
		caller,
		elapsedMs,
	); lerr != nil {
		log.Printf("agentmgr: log invocation failed for %s: %v", toolName, lerr)
	}

	if runErr != nil {
		httpx.WriteJSON(w, map[string]any{
			"error":      runErr.Error(),
			"tool_name":  toolName,
			"latency_ms": elapsedMs,
		})
		return
	}
	// #2C deferred-activate: tool_lookup{name} sukses → tandain tool itu "active" biar
	// ToolSpecsHandler kirim schema PENUH-nya (masuk array → grammar llama bisa manggil).
	// Kebijakan via resolveDeferPolicy (no-op kalau defer off utk agent ini). main.go re-fetch abis lookup.
	if defOn, _ := resolveDeferPolicy(id, IsPrimaryAgent(id)); defOn && toolName == "tool_lookup" {
		if ln, _ := body.Args["name"].(string); strings.TrimSpace(ln) != "" {
			if lt, ok := tools.Lookup(strings.TrimSpace(ln)); ok {
				activateDeferred(id, lt.Name())
			}
		}
	}
	httpx.WriteJSON(w, map[string]any{
		"ok":         true,
		"tool_name":  toolName,
		"result":     result,
		"latency_ms": elapsedMs,
	})
}

// ToolRegistryHandler — GET /api/agents/tools/registry
// Return list of registered tools (built-in dari binary, plug-and-play via
// init()). Phase 1 likely empty — Tier 1 tools di-register Section 11.
//
// ⚠️ Anti over-prompt: summary only (name + description + capability).
// Full schema fetch via /tools/get?name= future endpoint.
func ToolRegistryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (use GET)"})
		return
	}
	items := tools.ListSummaries()
	httpx.WriteJSON(w, map[string]any{
		"items":        items,
		"count":        len(items),
		"algo_version": tools.AlgoVersion,
	})
}

// ToolInvocationsHandler — GET /api/agents/tool-invocations?id=<id>&tool_name=&caller=&limit=
// Browse invocation log per-warga.
func ToolInvocationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (use GET)"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		httpx.WriteJSON(w, map[string]any{"items": []any{}, "note": "db belum ada"})
		return
	}
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	defer store.Close()

	toolName := strings.TrimSpace(r.URL.Query().Get("tool_name"))
	caller := strings.TrimSpace(r.URL.Query().Get("caller"))
	limit := 50
	if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
		if n, perr := strconv.Atoi(s); perr == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	items, err := store.ListToolInvocations(toolName, caller, limit)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "list: " + err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "count": len(items)})
}

// EduErrorsHandler — multi-method endpoint /api/agents/edu-errors?id=<id>
//
//	GET    list (?category=&limit=) atau single (?code=<code>)
//	POST   upsert single (body: {code, category, title, explanation, remediation})
//
// Roadmap Section 9. Cache populated via Router sync (defer phase 2) atau
// admin manual via POST.
func EduErrorsHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		if r.Method == http.MethodGet {
			httpx.WriteJSON(w, map[string]any{"items": []any{}, "note": "db belum ada"})
			return
		}
		httpx.WriteJSON(w, map[string]any{"error": "db belum ada — boot agent dulu"})
		return
	}
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	defer store.Close()

	switch r.Method {
	case http.MethodGet:
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if code != "" {
			e, err := store.LookupEduError(code)
			if err != nil {
				httpx.WriteJSON(w, map[string]any{"error": err.Error()})
				return
			}
			httpx.WriteJSON(w, e)
			return
		}
		category := strings.TrimSpace(r.URL.Query().Get("category"))
		limit := 50
		if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
			if n, perr := strconv.Atoi(s); perr == nil && n > 0 && n <= 500 {
				limit = n
			}
		}
		items, err := store.ListEduErrors(category, limit)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "list: " + err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"items": items, "count": len(items)})

	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
		var body agentdb.EduError
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "decode: " + err.Error()})
			return
		}
		if err := store.UpsertEduError(body); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"ok": true, "code": body.Code})

	default:
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
	}
}

// SharedDirForAgent — kernelhost daftarkan callback. Resolve agent shared
// workspace path (`<root>/workspace/<agent_id>/`). Nil-safe.
var SharedDirForAgent func(agentID string) (string, error)

// CapsCheckerForAgent — kernelhost daftarkan callback. Return closure
// untuk check capability per-agent via broker IsApproved. Nil-safe —
// sandbox skip cap gate kalau callback nil (Section 12 default-allow).
var CapsCheckerForAgent func(agentID string) func(capability string) bool

// PromoteRun — kernelhost daftarkan callback. Resolve agent + push
// mistakes eligible ke Router /api/mistakes/submit. Nil-safe.
var PromoteRun func(agentID string) (any, error)

// PromoteRunHandler — POST /api/agents/promote/run?id=<id>
// Trigger manual: list mistakes lokal eligible (tier='raw' + hit_count ≥ 3)
// → submit ke Router brain → mark tier='promoted' di lokal.
// Roadmap Section 7 phase 1.
func PromoteRunHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (use POST)"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	if PromoteRun == nil {
		httpx.WriteJSON(w, map[string]any{"error": "promote not wired"})
		return
	}
	report, err := PromoteRun(id)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"ok": true, "report": report})
}

// RetentionSweep — kernelhost daftarkan callback supaya admin endpoint
// bisa trigger manual sweep. Nil-safe: kalau ngga di-set, endpoint return
// error "not wired".
var RetentionSweep func(agentID string) (any, error)

// RetentionSweepHandler — POST /api/agents/retention/sweep?id=<id>
// Manual trigger retention sweep untuk satu agent. Cron 24h jalan otomatis;
// endpoint ini buat admin force-run (testing atau immediate cleanup).
// Roadmap section 8.
//
// ⚠️ DESTRUCTIVE: hard-delete row soft-deleted > grace period. Soft-delete
// reversible via backup; hard-delete final. Cron jalan default — manual
// trigger jarang perlu.
func RetentionSweepHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (use POST)"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	if RetentionSweep == nil {
		httpx.WriteJSON(w, map[string]any{"error": "retention sweep not wired"})
		return
	}
	report, err := RetentionSweep(id)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"ok": true, "report": report})
}

// MistakesHandler — GET /api/agents/mistakes?id=<id>&tier=&limit=
// POST /api/agents/mistakes?id=<id> body {category, title, content, context_origin?}
// List + admin-add mistakes journal per-warga. Roadmap section 2.
//
// ⚠️ Endpoint ini buat dashboard / admin manual add — JANGAN
// di-auto-inject ke system prompt (over-prompt). Max 500 row per call.
func MistakesHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		if r.Method == http.MethodGet {
			httpx.WriteJSON(w, map[string]any{"items": []any{}, "note": "db belum ada"})
			return
		}
		httpx.WriteJSON(w, map[string]any{"error": "db belum ada — boot agent dulu"})
		return
	}
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	defer store.Close()

	switch r.Method {
	case http.MethodGet:
		tier := strings.TrimSpace(r.URL.Query().Get("tier"))
		limit := 50
		if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
			if n, perr := strconv.Atoi(s); perr == nil {
				limit = n
			}
		}
		items, err := store.ListMistakes(tier, limit)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "list: " + err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"items": items, "count": len(items)})

	case http.MethodPost:
		// Hard cap body 64KB — anti accidental giant payload.
		r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
		var body struct {
			Category      string `json:"category"`
			Title         string `json:"title"`
			Content       string `json:"content"`
			ContextOrigin string `json:"context_origin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "decode body: " + err.Error()})
			return
		}
		idNew, addedNew, err := store.AddMistake(body.Category, body.Title, body.Content, body.ContextOrigin)
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "add: " + err.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"id": idNew, "added": addedNew})

	default:
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed (use GET or POST)"})
	}
}

// DecisionsHandler — GET /api/agents/decisions?id=<id>&type=&limit=
// List decisions log dari state.db agent. Roadmap section 3.
//
// ⚠️ Endpoint ini buat dashboard / audit / manual recall — JANGAN
// di-auto-inject ke system prompt (over-prompt risk). Max 500 row per
// call, default 50.
func DecisionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		httpx.WriteJSON(w, map[string]any{"items": []any{}, "note": "db belum ada"})
		return
	}
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	defer store.Close()

	typeFilter := strings.TrimSpace(r.URL.Query().Get("type"))
	limit := 50
	if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
		if n, perr := strconv.Atoi(s); perr == nil {
			limit = n
		}
	}
	items, err := store.ListDecisions(typeFilter, limit)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "list: " + err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "count": len(items)})
}

// InteractionsHandler — GET /api/agents/interactions?id=<id>&channel=&actor=&limit=
// List episodic interaction log dari state.db agent. Roadmap section 1.
//
// ⚠️ Endpoint ini buat dashboard / audit / manual recall — JANGAN
// di-auto-inject ke system prompt (over-prompt risk, lihat standar
// section 11). Max 500 row per call, default 50.
func InteractionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	dbPath := agentdb.Resolve(id, agentFolder(id))
	if _, err := os.Stat(dbPath); err != nil {
		httpx.WriteJSON(w, map[string]any{"items": []any{}, "note": "db belum ada"})
		return
	}
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	defer store.Close()

	// DELETE = CLEAR history (anti-anchor §6.1): soft-delete interactions. Param
	// `older_days` (default/0 = SEMUA). AMAN: interactions ada di workspace PERSONAL
	// (state.db privat) — twin/cognitive-graph/brain di tabel/store BEDA, ga kesentuh.
	// Cuma chat-log yang kehapus → bunuh history-anchoring tanpa ilangin knowledge.
	if r.Method == http.MethodDelete {
		olderDays := 0
		if s := strings.TrimSpace(r.URL.Query().Get("older_days")); s != "" {
			if n, perr := strconv.Atoi(s); perr == nil && n > 0 {
				olderDays = n
			}
		}
		n, derr := store.PruneInteractions(time.Duration(olderDays) * 24 * time.Hour)
		if derr != nil {
			httpx.WriteJSON(w, map[string]any{"error": "clear: " + derr.Error()})
			return
		}
		httpx.WriteJSON(w, map[string]any{"ok": true, "cleared": n, "older_days": olderDays})
		return
	}

	channel := strings.TrimSpace(r.URL.Query().Get("channel"))
	actor := strings.TrimSpace(r.URL.Query().Get("actor"))
	limit := 50
	if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
		if n, perr := strconv.Atoi(s); perr == nil {
			limit = n
		}
	}
	items, err := store.ListInteractions(channel, actor, limit)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "list: " + err.Error()})
		return
	}
	httpx.WriteJSON(w, map[string]any{"items": items, "count": len(items)})
}

// RemoveHandler — DELETE /api/agents/remove?id=<id>.
//
// Section 4 integration: SEBELUM hapus folder, auto-seal semua death_letter
// unsealed warga ini. Pastikan legacy preserved (folder akan ikut zip
// download future via DownloadHandler enhancement).
func RemoveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	// Source-agent (di repo agents/<id>/) ngga boleh dihapus via API — itu
	// di-manage lewat repo/git. API cuma uninstall agent STAGED (hasil upload).
	// (fix bug.md #1: gate source-aware + cegah nuke source repo.)
	if src := agentSourceDir(id); src != "" {
		httpx.WriteJSON(w, map[string]any{"error": "agent '" + id + "' adalah source-agent di repo (agents/" + id + "/) — hapus via repo/git, bukan API. API cuma uninstall agent staged."})
		return
	}
	dir := agentFolder(id)
	if _, err := os.Stat(dir); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "agent not found"})
		return
	}

	// Section 4: auto-seal unsealed letters sebelum remove. Best-effort —
	// kalau state.db ngga ada / corrupt, log saja + lanjut remove.
	// Plus log decision audit trail (Section 3 doctrine) supaya
	// kepergian warga ke-track walau folder hilang.
	sealedCount := int64(0)
	dbPath := agentdb.Resolve(id, dir)
	if _, err := os.Stat(dbPath); err == nil {
		if store, sopen := agentdb.Open(dbPath); sopen == nil {
			if n, serr := store.SealAllUnsealed(); serr == nil {
				sealedCount = n
				if sealedCount > 0 {
					_, _ = store.LogDecision("agent_retire",
						fmt.Sprintf("Auto-sealed %d unsealed death letters sebelum agent remove.", sealedCount),
						"success",
						map[string]any{
							"agent_id":       id,
							"sealed_letters": sealedCount,
						},
						0)
				}
			} else {
				log.Printf("agentmgr: auto-seal failed for %s: %v", id, serr)
			}
			_ = store.Close()
		}
	}

	if err := os.RemoveAll(dir); err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "remove: " + err.Error()})
		return
	}
	resp := map[string]any{"ok": true, "removed": id}
	if sealedCount > 0 {
		resp["auto_sealed_letters"] = sealedCount
	}
	httpx.WriteJSON(w, resp)
}

// ConfigHandler — GET/POST /api/agents/config?id=<id>.
//
// Standar agent (7 section, semua terisolasi per agent):
//
//   1. prompt      — system prompt / persona
//   2. schedule    — cron jobs ([{id, cron, task}])
//   3. tools       — capability flags ([telegram, router, kv, fs, net])
//   4. skills      — reusable behaviors ([{id, trigger, instructions}])
//   5. workspace   — host-side folder, di-mount kernel ke guest /workspace
//   6. settings    — router endpoint + model + arbitrary API credentials
//   7. database    — SQLite per-agent di workspace/state.db (file isolated)
//
// Section 5 & 7 host-side: kernel kelola di disk, ngga ada di config.json.
//
// Schema config.json:
//
//	{
//	  "prompt":   "system prompt string...",
//	  "schedule": [{"id":"morning-news","cron":"0 7 * * *","task":"..."}],
//	  "tools":    ["telegram", "kv", ...],
//	  "skills":   [{"id":"summarize","trigger":"/sum","instructions":"..."}],
//	  "router":   {"url":"...","model":"..."},
//	  "secrets":  {"TELEGRAM_BOT_TOKEN":"...","GOOGLE_API_KEY":"...", ...}
//	}
//
// File ini dibaca kernel saat boot dan di-inject ke agent via env:
//   FLOWORK_AGENT_CONFIG  — JSON utuh
//   FLOWORK_AGENT_ID      — id agent
//   FLOWORK_AGENT_WORKSPACE = /workspace
//   FLOWORK_AGENT_DB       = /workspace/state.db
//   FLOWORK_SHARED_WORKSPACE = /shared (kalau cap fs:shared)
//   <KEY>=<value>          — tiap secrets.* di-expand jadi env (telegram/google/dll)

// maskSecretValue redacts a secret to its last 4 chars for safe display.
func maskSecretValue(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return secretMaskPrefix
	}
	return secretMaskPrefix + v[len(v)-4:]
}

// secretMaskPrefix is the sentinel every masked secret starts with (4 bullets).
// A real token/key is ASCII alphanumeric, so it can never collide with this.
const secretMaskPrefix = "••••"

// reconcileMaskedSecrets restores UNCHANGED secret values that the GUI posted back
// as their ••••<last4> mask, using the stored originals — so a Save (full-replace of
// the secrets table) can't clobber real secrets with their mask. Any incoming value
// that still looks like a mask means "unchanged": swap in the stored plaintext, or
// drop it if there's no stored original (never persist a mask). Mutates `incoming`.
func reconcileMaskedSecrets(incoming map[string]any, existing map[string]string) {
	for k, v := range incoming {
		if !strings.HasPrefix(fmt.Sprint(v), secretMaskPrefix) {
			continue
		}
		if real, found := existing[k]; found {
			incoming[k] = real
		} else {
			delete(incoming, k)
		}
	}
}

func ConfigHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if !reID.MatchString(id) {
		httpx.WriteJSON(w, map[string]any{"error": "invalid id"})
		return
	}
	// Source-aware (fix bug.md #1): cek source repo dulu, baru staged.
	dir, ok := resolveAgentDir(id)
	if !ok {
		httpx.WriteJSON(w, map[string]any{"error": "agent not found"})
		return
	}
	// Resolve & open DB. HARDCODED di `<source-or-staged>/workspace/state.db`.
	dbPath := agentdb.Resolve(id, dir)
	store, err := agentdb.Open(dbPath)
	if err != nil {
		httpx.WriteJSON(w, map[string]any{"error": "open db: " + err.Error()})
		return
	}
	defer store.Close()
	// Migrate config.json (kalau masih ada — first GET/POST after upgrade).
	_ = store.MigrateFromJSON(dir)

	switch r.Method {
	case http.MethodGet:
		cfg, err := store.Load()
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "load: " + err.Error()})
			return
		}
		// SECURITY: never return secret VALUES (bot tokens, API keys) in cleartext
		// over HTTP — a stolen session cookie or GUI XSS would exfil every agent's
		// credentials. Mask to last-4 (writes still accept cleartext via POST; the
		// kernel reads secrets straight from state.db at boot, so the GUI never
		// needs the plaintext back).
		if secs, ok := cfg["secrets"].(map[string]any); ok {
			masked := make(map[string]any, len(secs))
			for k, v := range secs {
				masked[k] = maskSecretValue(fmt.Sprint(v))
			}
			cfg["secrets"] = masked
		}
		httpx.WriteJSON(w, map[string]any{"config": cfg, "exists": true, "db": dbPath})

	case http.MethodPost, http.MethodPut:
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "read body: " + err.Error()})
			return
		}
		var cfg map[string]any
		if err := json.Unmarshal(body, &cfg); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "json parse: " + err.Error()})
			return
		}
		// SECURITY/INTEGRITY: the GET above masks secret values to ••••<last4>. The
		// GUI posts the WHOLE settings form back on Save, so any UNTOUCHED secret
		// arrives here as its mask. Save() does a full-replace of the secrets table —
		// so without this reconcile a plain "edit prompt + Save" would overwrite every
		// real secret with its mask (the agent loses its bot token / API keys).
		if incoming, ok := cfg["secrets"].(map[string]any); ok {
			existing, _ := store.Secrets()
			reconcileMaskedSecrets(incoming, existing)
		}
		if err := store.Save(cfg); err != nil {
			httpx.WriteJSON(w, map[string]any{"error": "save: " + err.Error()})
			return
		}
		// Trigger kernel reload: callback registered by main.go pointing
		// ke host.ReloadAgent. inotify ngga recurse ke subfolder, jadi
		// file-system trigger ngga reliable — callback langsung lebih
		// deterministik.
		reloadErr := ""
		if Reload != nil {
			if err := Reload(id); err != nil {
				reloadErr = err.Error()
				log.Printf("config: reload %s: %v", id, err)
			}
		}
		httpx.WriteJSON(w, map[string]any{"ok": true, "db": dbPath, "reload_err": reloadErr})

	default:
		httpx.WriteJSON(w, map[string]any{"error": "method not allowed"})
	}
}

// ── zip helpers ────────────────────────────────────────────────────────────

func findManifest(zr *zip.Reader) (*zip.File, string, error) {
	for _, f := range zr.File {
		if filepath.Base(f.Name) != "manifest.json" {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(f.Name))
		if dir == "." || dir == "" {
			return f, "", nil
		}
		return f, dir + "/", nil
	}
	return nil, "", errors.New("manifest.json not found in zip")
}

func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, 1<<20))
}

func extractFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	// Cap per-entry extraction (zip-bomb guard): a small compressed entry can
	// expand without bound and fill the disk. 64MB matches the upload body cap
	// and the LimitReader already used in pack_extract.go / plugin_handler.go.
	_, err = io.Copy(out, io.LimitReader(rc, 64<<20))
	return err
}
