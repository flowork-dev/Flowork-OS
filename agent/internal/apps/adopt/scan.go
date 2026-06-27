// scan.go — PRE-FLIGHT SCANNER repo untrusted (ROADMAP_REPO_TO_APP F6). Sebelum jalanin kode
// eksternal pertama kali, scan pola berbahaya (destruktif / exfil / reverse-shell / SSRF-metadata).
// Bukan anti-malware lengkap — lapis kewaspadaan: owner liat red-flag sebelum approve (consent sadar).
//
// Reusable juga buat AI Studio verifier (gerbang kapabilitas baru). Murni logika, no side-effect.
package adopt

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding — 1 pola berbahaya ketemu.
type Finding struct {
	File     string `json:"file"`     // path relatif ke repo
	Line     int    `json:"line"`     // nomor baris
	Severity string `json:"severity"` // critical | warn
	Pattern  string `json:"pattern"`  // label pola
	Evidence string `json:"evidence"` // potongan baris (dipangkas)
}

// ScanReport — hasil scan 1 repo.
type ScanReport struct {
	Findings []Finding `json:"findings"`
	Critical int       `json:"critical"`
	Warn     int       `json:"warn"`
	Scanned  int       `json:"scanned_files"`
}

type rule struct {
	label string
	sev   string
	re    *regexp.Regexp
}

// dangerRules — pola high-signal buat kode untrusted. Sengaja ketat (false-positive lebih baik
// daripada lolos). Owner tetap bisa approve sadar (consent).
var dangerRules = []rule{
	// rm -rf yg nyasar ke ROOT/HOME/wildcard-telanjang (bahaya). Sengaja TIDAK match subdir spesifik
	// (mis. `rm -rf /var/lib/apt/lists/*` = cleanup apt standar Docker) biar ga false-positive.
	{"rm-rf-destruktif", "critical", regexp.MustCompile(`(?i)\brm\s+-[a-z]*r[a-z]*\s+(--no-preserve-root\s+)?(/([\s*;&|]|$)|~([\s/]|$)|\$HOME|\*(\s|$))`)},
	{"fork-bomb", "critical", regexp.MustCompile(`:\(\)\s*\{\s*:\s*\|\s*:`)},
	{"mkfs/dd-disk", "critical", regexp.MustCompile(`(?i)\b(mkfs\b|dd\s+if=|>\s*/dev/sd[a-z])`)},
	{"pipe-ke-shell", "critical", regexp.MustCompile(`(?i)\b(curl|wget)\b[^|\n]*\|\s*(sudo\s+)?(sh|bash|zsh)\b`)},
	{"reverse-shell", "critical", regexp.MustCompile(`(?i)(bash\s+-i\s*>&?\s*/dev/tcp|nc\s+-e\b|/dev/tcp/\d)`)},
	{"cloud-metadata-ssrf", "critical", regexp.MustCompile(`169\.254\.169\.254`)},
	{"eval-obfuscated", "critical", regexp.MustCompile(`(?i)eval\s*\(\s*(atob|base64|fromCharCode)`)},
	{"baca-tulis-credstore", "warn", regexp.MustCompile(`(?i)/etc/(passwd|shadow)\b`)},
	{"setuid/chmod-s", "warn", regexp.MustCompile(`(?i)\b(setuid|chmod\s+[ug]?\+?s)\b`)},
	{"base64-decode-exec", "warn", regexp.MustCompile(`(?i)base64\s+(-d|--decode)\b[^|\n]*\|\s*(sh|bash)`)},
	{"crontab-persistence", "warn", regexp.MustCompile(`(?i)\bcrontab\s+-`)},
}

// extraRules — SWITCH/seam (POLA A): pola scan BARU didaftarin lewat sibling _ext.go tanpa nyentuh
// file ini (frozen). Default kosong = perilaku persis built-in. (Rule #7: nambah aturan ga buka freeze.)
var extraRules []rule

// RegisterScanRule — daftarin pola berbahaya BARU (mis. ransomware-marker, miner) via sibling.
// severity selain "critical" → dianggap "warn". Pattern invalid / label kosong → di-skip (aman).
func RegisterScanRule(label, severity, pattern string) {
	re, err := regexp.Compile(pattern)
	if err != nil || strings.TrimSpace(label) == "" {
		return
	}
	if severity != "critical" {
		severity = "warn"
	}
	extraRules = append(extraRules, rule{label, severity, re})
}

// skipDir — folder yang dilewati (dep/vcs, bukan kode repo asli).
var skipDir = map[string]bool{
	".git": true, "node_modules": true, ".venv": true, "venv": true,
	"target": true, "dist": true, "build": true, "__pycache__": true,
	".tox": true, "vendor": true,
}

// scanExt — ekstensi text/source yang di-scan (skip biner/aset gede).
var scanExt = map[string]bool{
	".sh": true, ".bash": true, ".zsh": true, ".py": true, ".js": true, ".ts": true,
	".mjs": true, ".cjs": true, ".rb": true, ".pl": true, ".php": true, ".go": true,
	".rs": true, ".java": true, ".lua": true, ".ps1": true, ".bat": true, ".cmd": true,
	".yml": true, ".yaml": true, ".toml": true, ".cfg": true, ".ini": true, ".env": true,
	".dockerfile": true, ".mk": true, ".makefile": true, "": true, // "" = file tanpa ext (Makefile, Dockerfile)
}

const maxScanBytes = 512 * 1024

// ScanRepo — walk repo, scan file text buat pola berbahaya. Ga pernah error (best-effort).
func ScanRepo(repoDir string) ScanReport {
	var rep ScanReport
	_ = filepath.Walk(repoDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			if skipDir[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > maxScanBytes {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		base := strings.ToLower(info.Name())
		if !scanExt[ext] && !knownNoExt(base) {
			return nil
		}
		rel, _ := filepath.Rel(repoDir, p)
		scanFile(p, rel, &rep)
		return nil
	})
	return rep
}

func knownNoExt(base string) bool {
	switch base {
	case "makefile", "dockerfile", "rakefile", "gemfile", "procfile":
		return true
	}
	return false
}

func scanFile(path, rel string, rep *ScanReport) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	rep.Scanned++
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	ln := 0
	for sc.Scan() {
		ln++
		line := sc.Text()
		for _, rs := range [][]rule{dangerRules, extraRules} {
			for _, r := range rs {
				if r.re.MatchString(line) {
					rep.Findings = append(rep.Findings, Finding{
						File: rel, Line: ln, Severity: r.sev, Pattern: r.label,
						Evidence: trimEvidence(line),
					})
					if r.sev == "critical" {
						rep.Critical++
					} else {
						rep.Warn++
					}
				}
			}
		}
	}
}

func trimEvidence(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 160 {
		return s[:160] + "…"
	}
	return s
}
