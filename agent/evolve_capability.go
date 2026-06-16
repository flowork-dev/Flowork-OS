// evolve_capability.go — R7 CAPABILITY FLOOR. Owner-approved 2026-06-15 (keputusan
// didelegasikan ke AI). "Buktikan, jangan asumsi." Ganti cek-nama dangkal (model=claude).
//
// PENTING (temuan empiris 2026-06-15): coding-puzzle BUKAN diskriminator "≥Opus 4.7" —
// haiku-4.5 (di bawah 4.7) lolos 3/3 termasuk wildcard-hard. Jadi eval ini = **FLOOR
// "bisa-ngoding"** doang: nyaring model RUSAK/LOKAL (Gemma gak bisa compile Go → gagal).
// Karena eval jalan lawan model yang BENERAN dilayani router, fallback-ke-lokal = gagal
// floor = GUARD ANTI-LOKAL otomatis. NO HARDCODE NAMA → future-proof (DeepSeek/Mythos ok).
//
// Bar "≥4.7" SEJATI = KARMA (rekam-jejak perubahan nyata yang lolos) + per-change TEST-GATE
// + owner colok model kuat + eksekusi re-probe model asli sebelum commit. Bukan eval ini.
// Hasil di-cache per-model (eval ~90s).

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// capResult — hasil eval kapabilitas 1 model.
type capResult struct {
	Model  string
	Passed bool
	Score  int
	Total  int
	Detail string
	At     int64
}

var capCache sync.Map // model(string) → capResult

// capTask — 1 tugas eval: prompt + harness Go yang manggil fungsi model + cetak PASS/FAIL.
type capTask struct {
	name    string
	prompt  string
	harness string // package main lengkap KECUALI fungsi model (di-prepend)
}

// capTasks — suite kalibrasi kelas Opus-4.7 (non-trivial; model lemah gagal ≥1).
var capTasks = []capTask{
	{
		name:   "twoSum",
		prompt: "Write a Go function with EXACT signature `func twoSum(nums []int, target int) []int` returning the two indices (i<j) whose values sum to target. Reply ONLY the function body code (the func), no package, no prose, no markdown fences.",
		harness: `
func eq(a, b []int) bool { if len(a)!=len(b){return false}; for i:=range a{if a[i]!=b[i]{return false}}; return true }
func main() {
  if eq(twoSum([]int{2,7,11,15},9),[]int{0,1}) && eq(twoSum([]int{3,2,4},6),[]int{1,2}) && eq(twoSum([]int{3,3},6),[]int{0,1}) {
    println("PASS")
  } else { println("FAIL") }
}`,
	},
	{
		name:   "lengthOfLongestSubstring",
		prompt: "Write a Go function with EXACT signature `func lengthOfLongestSubstring(s string) int` returning the length of the longest substring without repeating characters. Reply ONLY the function code, no package, no prose, no markdown fences.",
		harness: `
func main() {
  if lengthOfLongestSubstring("abcabcbb")==3 && lengthOfLongestSubstring("bbbbb")==1 && lengthOfLongestSubstring("pwwkew")==3 && lengthOfLongestSubstring("")==0 {
    println("PASS")
  } else { println("FAIL") }
}`,
	},
	{
		// HARD (kalibrasi ≥4.7): wildcard matching — model lemah sering gagal edge-case '*'.
		name:   "isMatch",
		prompt: "Write a Go function with EXACT signature `func isMatch(s string, p string) bool` implementing wildcard pattern matching where '?' matches any single char and '*' matches any sequence (including empty). The whole string s must match pattern p. Reply ONLY the function code, no package, no prose, no markdown fences.",
		harness: `
func main() {
  ok := !isMatch("aa","a") && isMatch("aa","*") && !isMatch("cb","?a") && isMatch("adceb","*a*b") && !isMatch("acdcb","a*c?b") && isMatch("","*") && isMatch("","") && !isMatch("a","")
  if ok { println("PASS") } else { println("FAIL") }
}`,
	},
}

// extractGoCode — buang fence ```go ... ``` / prosa, ambil kode mentah.
func extractGoCode(s string) string {
	if i := strings.Index(s, "```"); i >= 0 {
		rest := s[i+3:]
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			rest = rest[nl+1:]
		}
		if j := strings.Index(rest, "```"); j >= 0 {
			return strings.TrimSpace(rest[:j])
		}
	}
	return strings.TrimSpace(s)
}

// runOneCapTask — minta model nulis fungsi → compile+run harness di temp → true kalau "PASS".
func runOneCapTask(ctx context.Context, model string, t capTask) bool {
	res, err := routerChat(ctx, model, []map[string]any{
		{"role": "system", "content": "You are a precise Go programmer. Output ONLY compilable Go code, no prose."},
		{"role": "user", "content": t.prompt},
	}, nil, 1200)
	if err != nil {
		return false
	}
	fn := extractGoCode(res.Content)
	if fn == "" || !strings.Contains(fn, "func ") {
		return false
	}
	dir, err := os.MkdirTemp("", "flowork-capeval-*")
	if err != nil {
		return false
	}
	defer os.RemoveAll(dir)
	src := "package main\n\n" + fn + "\n" + t.harness + "\n"
	if werr := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600); werr != nil {
		return false
	}
	runCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "go", "run", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GO111MODULE=off") // pure func, no deps
	out, _ := cmd.CombinedOutput()
	return strings.Contains(string(out), "PASS")
}

// runCapabilityEval — jalanin SELURUH suite (semua wajib lolos = bar tinggi kelas 4.7).
func runCapabilityEval(ctx context.Context, model string) capResult {
	r := capResult{Model: model, Total: len(capTasks)}
	for _, t := range capTasks {
		if runOneCapTask(ctx, model, t) {
			r.Score++
		}
	}
	r.Passed = r.Score == r.Total
	if r.Passed {
		r.Detail = "lolos semua tugas eval (≥ kelas Opus 4.7)"
	} else {
		r.Detail = "gagal eval kapabilitas (di bawah bar self-modify)"
	}
	return r
}

// capabilityMeetsBar — GANTI model_strong dangkal (dipakai sbg ModelStrong di gate).
// CACHE-ONLY (non-blocking) biar status GUI gak nge-hang nunggu eval 90s. Kalau model
// belum dievaluasi → false + minta owner klik "Evaluate". Eval beneran lewat /api/evolve/eval.
func capabilityMeetsBar() (bool, string) {
	model := coderModel("")
	if c, ok := capCache.Load(model); ok {
		r := c.(capResult)
		return r.Passed, model + ": " + r.Detail + " (" + itoaSmall(r.Score) + "/" + itoaSmall(r.Total) + ")"
	}
	return false, model + ": belum dievaluasi — klik Evaluate"
}

// evolveEvalAndCache — jalanin eval kapabilitas model aktif (BLOCKING ~90s) + cache.
// Dipanggil on-demand dari /api/evolve/eval (tombol GUI), bukan tiap status-poll.
func evolveEvalAndCache() capResult {
	model := coderModel("")
	// 300s (was 120s, owner-approved 2026-06-16): suite = 3 tugas SEKUENSIAL (tiap tugas = 1 LLM-call
	// 1200-token + compile `go run` sampai 25s). 120s total kependekan → tugas ke-3 SELALU ke-cut →
	// max 2/3 → floor MUSTAHIL lolos, model sekuat apapun (Opus pun cuma 2/3). 300s kasih ~100s/tugas.
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	r := runCapabilityEval(ctx, model)
	capCache.Store(model, r)
	return r
}

func itoaSmall(n int) string {
	if n < 0 {
		n = 0
	}
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
