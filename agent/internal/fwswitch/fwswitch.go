// fwswitch — GROWTH-POINT (NON-frozen). Plug-and-play settings (Rule 6): switch fitur
// FLOWORK_* dikelola dari GUI Setting, BUKAN file env edit-tangan (flowork.local.env yg
// invisible buat user install fresh). Kembaran router/internal/fwswitch (modul terpisah →
// duplikat sengaja; core identik). Lihat lock/fwswitch.md.
//
// File LINTAS-PROSES ~/.flowork/flowork_settings.json (pola agent_brain_config.json) di-APPLY
// ke os.Setenv di STARTUP tiap proses + watcher mtime → SEMUA os.Getenv("FLOWORK_*") yg udah
// ada (frozen sekalipun) baca nilai GUI. ZERO refactor call-site.
//
// PRESEDENSI (owner 2026-06-26): GUI menang → file overwrite ENV. Key dihapus dari file →
// restore ENV asli (snapshot). Key ga ada di file → ENV asli utuh → call-site pakai default.
package fwswitch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const fileName = "flowork_settings.json"

var (
	mu      sync.Mutex
	managed = map[string]string{} // key → ENV asli sebelum di-override (buat restore)
	lastMod time.Time
	booted  bool
	hadFile bool
)

// SettingsPath — ~/.flowork/flowork_settings.json (kosong kalau home ga ketemu).
func SettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".flowork", fileName)
}

// ReadFile → map[string]string (tolerir nilai non-string). nil kalau ga ada/rusak.
func ReadFile() map[string]string {
	p := SettingsPath()
	if p == "" {
		return nil
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var m map[string]string
	if json.Unmarshal(raw, &m) == nil {
		return m
	}
	var ma map[string]any
	if json.Unmarshal(raw, &ma) == nil {
		m = make(map[string]string, len(ma))
		for k, v := range ma {
			m[k] = strings.TrimSpace(fmt.Sprint(v))
		}
		return m
	}
	return nil
}

// WriteValues — merge nilai (set/hapus) ke file lalu Apply lokal (host). Nilai "" = hapus key
// (revert ke ENV/default). Router nyusul lewat watcher mtime. Atomic-ish (tmp+rename).
func WriteValues(vals map[string]string) error {
	mu.Lock()
	cur := ReadFile()
	if cur == nil {
		cur = map[string]string{}
	}
	for k, v := range vals {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if strings.TrimSpace(v) == "" {
			delete(cur, k)
		} else {
			cur[k] = strings.TrimSpace(v)
		}
	}
	p := SettingsPath()
	if p == "" {
		mu.Unlock()
		return fmt.Errorf("home dir tidak ketemu")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		mu.Unlock()
		return err
	}
	b, _ := json.MarshalIndent(cur, "", "  ")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		mu.Unlock()
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		mu.Unlock()
		return err
	}
	n := applyLocked()
	mu.Unlock()
	_ = n
	return nil
}

// Apply — sinkronkan os.Setenv dgn isi file (file menang atas ENV). Idempotent.
func Apply() int {
	mu.Lock()
	defer mu.Unlock()
	return applyLocked()
}

func applyLocked() int {
	file := ReadFile()
	active := 0
	for k, v := range file {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || !strings.HasPrefix(k, "FLOWORK_") || v == "" {
			continue
		}
		if _, ok := managed[k]; !ok {
			managed[k] = os.Getenv(k)
		}
		_ = os.Setenv(k, v)
		active++
	}
	for k, orig := range managed {
		if fv, ok := file[k]; ok && strings.TrimSpace(fv) != "" {
			continue
		}
		if orig == "" {
			_ = os.Unsetenv(k)
		} else {
			_ = os.Setenv(k, orig)
		}
		delete(managed, k)
	}
	return active
}

// Boot — apply sekali + watcher mtime (live lintas-proses tanpa restart). Idempotent.
func Boot() {
	mu.Lock()
	if booted {
		mu.Unlock()
		return
	}
	booted = true
	mu.Unlock()
	Apply()
	go watch()
}

func watch() {
	for {
		time.Sleep(3 * time.Second)
		p := SettingsPath()
		if p == "" {
			continue
		}
		fi, err := os.Stat(p)
		mu.Lock()
		switch {
		case err != nil:
			if hadFile {
				applyLocked()
				hadFile = false
			}
		case !fi.ModTime().Equal(lastMod):
			lastMod = fi.ModTime()
			hadFile = true
			applyLocked()
		}
		mu.Unlock()
	}
}
