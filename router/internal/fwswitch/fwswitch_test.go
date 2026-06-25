package fwswitch

import (
	"os"
	"path/filepath"
	"testing"
)

// reset state global antar-test (paket pakai var paket-level).
func reset() {
	mu.Lock()
	managed = map[string]string{}
	mu.Unlock()
}

func writeSettings(t *testing.T, home, body string) {
	t.Helper()
	dir := filepath.Join(home, ".flowork")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// GUI menang: file override ENV.
func TestApply_FileBeatsEnv(t *testing.T) {
	reset()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FLOWORK_SEARCH_MINSCORE", "0.10") // ENV
	writeSettings(t, home, `{"FLOWORK_SEARCH_MINSCORE":"0.90"}`)
	Apply()
	if got := os.Getenv("FLOWORK_SEARCH_MINSCORE"); got != "0.90" {
		t.Errorf("file harus menang atas ENV, mau 0.90 dapet %q", got)
	}
}

// Key ga ada di file → ENV asli utuh (call-site pakai default-nya sendiri).
func TestApply_NoFileKeyKeepsEnv(t *testing.T) {
	reset()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FLOWORK_INSTINCT_SCOPED", "1")
	writeSettings(t, home, `{"FLOWORK_SEARCH_MINSCORE":"0.5"}`) // beda key
	Apply()
	if got := os.Getenv("FLOWORK_INSTINCT_SCOPED"); got != "1" {
		t.Errorf("ENV asli harus utuh, mau 1 dapet %q", got)
	}
}

// Key DIHAPUS dari file (sesudah pernah di-manage) → restore ke ENV asli.
func TestApply_RemoveRestoresEnv(t *testing.T) {
	reset()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FLOWORK_TOOLCALL_RECOVER", "1") // ENV asli
	writeSettings(t, home, `{"FLOWORK_TOOLCALL_RECOVER":"0"}`)
	Apply()
	if got := os.Getenv("FLOWORK_TOOLCALL_RECOVER"); got != "0" {
		t.Fatalf("file harus override jadi 0, dapet %q", got)
	}
	writeSettings(t, home, `{}`) // hapus
	Apply()
	if got := os.Getenv("FLOWORK_TOOLCALL_RECOVER"); got != "1" {
		t.Errorf("dihapus dari file → restore ENV asli 1, dapet %q", got)
	}
}

// Nilai non-FLOWORK_ / kosong diabaikan (cuma kelola FLOWORK_* non-kosong).
func TestApply_IgnoresNonFloworkAndEmpty(t *testing.T) {
	reset()
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeSettings(t, home, `{"PATH":"/evil","FLOWORK_X":"","FLOWORK_DEFER_TOOLS":"1"}`)
	Apply()
	if os.Getenv("FLOWORK_DEFER_TOOLS") != "1" {
		t.Error("FLOWORK_DEFER_TOOLS harus ke-set")
	}
	if os.Getenv("PATH") == "/evil" {
		t.Error("non-FLOWORK_ key TIDAK boleh disentuh (PATH)")
	}
}
