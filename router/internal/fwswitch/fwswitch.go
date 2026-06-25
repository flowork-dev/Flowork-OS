// fwswitch — ⚠️ FROZEN 2026-06-26 (core; extend lewat agent registry.go + file GUI, BUKAN file ini).
// Plug-and-play settings (Rule 6): switch fitur
// FLOWORK_* dikelola dari GUI Setting, BUKAN file env edit-tangan (flowork.local.env yg
// invisible buat user install fresh).
//
// AKAR (Rule 5): ~80 ENV FLOWORK_*; switch fitur kebawa di flowork.local.env → user fresh
// ga tau. GUI udah punya Settings tapi (a) buat secret/token, (b) os.Setenv cuma di proses
// host :1987 → router :2402 ga kebagian (beda proses).
//
// SOLUSI tanpa sentuh call-site (frozen-safe, Rule 7): file LINTAS-PROSES
// ~/.flowork/flowork_settings.json (pola sama agent_brain_config.json) di-APPLY ke os.Setenv
// di STARTUP tiap proses (router + host) + watcher mtime → SEMUA os.Getenv("FLOWORK_*") yg
// udah ada (frozen sekalipun) otomatis baca nilai GUI. ZERO refactor call-site.
//
// PRESEDENSI (owner 2026-06-26): GUI menang → file overwrite ENV. Key yg DIHAPUS dari file
// → di-restore ke ENV asli (snapshot saat pertama di-manage). Key ga ada di file → ENV asli
// utuh → call-site pakai default-kode.
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
	mu       sync.Mutex
	managed  = map[string]string{} // key → ENV asli (sebelum di-override GUI); buat restore
	lastMod  time.Time
	booted   bool
	hadFile  bool
)

// SettingsPath — ~/.flowork/flowork_settings.json (kosong kalau home ga ketemu).
func SettingsPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".flowork", fileName)
}

// readFile → map[string]string (tolerir nilai non-string). nil kalau ga ada/rusak.
func readFile() map[string]string {
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

// Apply — sinkronkan os.Setenv dgn isi file (file menang atas ENV). Idempotent, aman dipanggil
// berkali (watcher). Return jumlah key aktif dari GUI.
func Apply() int {
	mu.Lock()
	defer mu.Unlock()
	return applyLocked()
}

func applyLocked() int {
	file := readFile()
	active := 0
	// 1. terapkan/segarkan key dari file (GUI menang).
	for k, v := range file {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || !strings.HasPrefix(k, "FLOWORK_") || v == "" {
			continue // hanya FLOWORK_* non-kosong; kosong = "pakai ENV/default"
		}
		if _, ok := managed[k]; !ok {
			managed[k] = os.Getenv(k) // snapshot ENV asli SEBELUM override (buat restore)
		}
		_ = os.Setenv(k, v)
		active++
	}
	// 2. key yg DULU di-manage tapi sekarang ga ada di file → restore ENV asli, lepas dari manage.
	for k, orig := range managed {
		if fv, ok := file[k]; ok && strings.TrimSpace(fv) != "" {
			continue // masih aktif
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

// Boot — apply sekali + jalanin watcher mtime (live update lintas-proses tanpa restart).
// Idempotent (cuma sekali). Panggil dari init() package main tiap proses.
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
			if hadFile { // file dihapus → restore semua
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
