// Package main — TEMPLATE AGENT FLOWORK (worker autonomus, "pasukan semut").
//
// Ini CETAKAN agent baru sesuai STANDAR Flowork (lihat AGENT_STANDARD.md +
// ../../agents/readme.md). Prinsip: "agent bodoh, engine pinter" — wasm SAMA jadi
// agent beda cukup ganti manifest.id + config (persona DB), TANPA edit kode.
//
// AUTONOMY (sama kayak mr-flow — owner 2026-06-20 "biar bisa looping, wait, awake"):
//   - TOOL-LOOP: panggil tool berkali-kali (chain) sampe jawab — bukan 1 call doang.
//   - GHOST-GUARD: narasi "mau ngapain" tanpa manggil tool → DIPAKSA bertindak.
//   - TIME-BOUND (bukan cap-angka): loop jalan terus dalam budget waktu turn.
//   - AUTO-CONTINUE (wait→awake): budget abis & belum kelar → HARNESS jadwalin
//     ScheduleWakeup sendiri (deterministik) → nyambung lintas-turn sampe SELESAI =
//     unbounded over time. Counter #N (di marker prompt) = anti-runaway (max 50).
//
// Persona DB-BASED (FLOWORK_AGENT_CONFIG) + DNA/konstitusi sacred (di-render host via
// /api/agents/self-prompt/render — incl anti-halu/sync-honest/autonomy-mode). Tools
// dari subscription agent (/api/agents/tools/specs). NO file .md.
//
// Build: GOWORK=off GOOS=wasip1 GOARCH=wasm go build -o agent.wasm .
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"
)

//go:wasmimport flowork host_net_fetch
func hostNetFetch(reqPtr, reqLen, outPtr, outMax uint32) uint32

//go:wasmimport flowork host_time_now_ms
func hostTimeNowMs() uint64

var outBuf [1 << 20]byte // 1MB: muat tool-specs + response LLM

func bytesPtr(b []byte) uint32 {
	if len(b) == 0 {
		return 0
	}
	return uint32(uintptr(unsafe.Pointer(&b[0])))
}

func emit(v any)     { b, _ := json.Marshal(v); fmt.Println(string(b)) }
func selfID() string { return os.Getenv("FLOWORK_AGENT_ID") }

const (
	routerURL = "http://127.0.0.1:2402/v1/chat/completions"
	specsURL  = "http://127.0.0.1:1987/api/agents/tools/specs?id="
	toolRun   = "http://127.0.0.1:1987/api/agents/tools/run?id="
	selfPromp = "http://127.0.0.1:1987/api/agents/self-prompt/render?id="

	maxToolIters           = 100    // backstop anti pure-infinite; batas NYATA = loopBudgetMs
	loopBudgetMs    uint64 = 200000 // budget waktu in-turn (~200s); turn-timeout 290s = backstop keras
	maxGhostNudges         = 6      // ghost-guard: max paksa-lanjut pas narasi-tanpa-tool (bounded)
	maxAutoContinue        = 50     // anti-runaway: max chunk auto-continue (≈2.7 jam) — BUKAN batas kerja
	autoContDelim          = "\n===TUGAS===\n"
)

type httpResp struct {
	Status int
	Body   []byte
}

func fetch(method, url string, headers map[string]string, body []byte, timeoutMS int) (*httpResp, error) {
	req := map[string]any{"method": method, "url": url, "timeout_ms": timeoutMS, "max_resp_bytes": 4 << 20}
	if len(headers) > 0 {
		req["headers"] = headers
	}
	if len(body) > 0 {
		req["body_base64"] = base64.StdEncoding.EncodeToString(body)
	}
	reqJSON, _ := json.Marshal(req)
	n := hostNetFetch(bytesPtr(reqJSON), uint32(len(reqJSON)), bytesPtr(outBuf[:]), uint32(len(outBuf)))
	if n == 0 {
		return nil, fmt.Errorf("host_net_fetch 0 bytes")
	}
	var hr struct {
		Status  int    `json:"status"`
		BodyB64 string `json:"body_base64"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(outBuf[:n], &hr); err != nil {
		return nil, err
	}
	if hr.Error != "" {
		return nil, fmt.Errorf("host: %s", hr.Error)
	}
	b, _ := base64.StdEncoding.DecodeString(hr.BodyB64)
	return &httpResp{Status: hr.Status, Body: b}, nil
}

func main() {
	if len(os.Args) < 2 {
		return
	}
	args := "{}"
	if len(os.Args) > 2 && os.Args[2] != "" {
		args = os.Args[2]
	}
	switch os.Args[1] {
	case "handle_message":
		handleMessage(args)
	case "handle": // loket-bus (group route): unwrap payload
		var msg struct {
			Payload json.RawMessage `json:"payload"`
		}
		_ = json.Unmarshal([]byte(args), &msg)
		if len(msg.Payload) == 0 {
			msg.Payload = json.RawMessage(args)
		}
		handleMessage(string(msg.Payload))
	case "boot":
		emit(map[string]any{"ok": true}) // worker = no daemon
	default:
		emit(map[string]any{"error": "unknown function: " + os.Args[1]})
	}
}

type agentConfig struct {
	Prompt string
	Model  string
}

// LOCKED (soft, owner-approved 2026-06-20 model-from-gui): baca router.model dari GUI.
// JANGAN balik ke baca top-level "model" (bug lama → semua agent jatuh ke flowork-brain).
func loadConfig() agentConfig {
	c := agentConfig{
		Prompt: "Lo agent spesialis Flowork. Kerjain tugas dengan jelas + jujur (anti-halu). Ganti persona ini di GUI/config.",
		Model:  "flowork-brain", // last-resort ONLY kalau GUI belum set model
	}
	if raw := os.Getenv("FLOWORK_AGENT_CONFIG"); raw != "" {
		// GUI = SUMBER KEBENARAN (owner 2026-06-20). store.Load() taruh model di router.model
		// (nested), BUKAN top-level "model". Bug lama: template baca "model" → selalu kosong →
		// jatuh ke flowork-brain (lokal) walau owner set Opus di Settings. Sekarang baca router.model.
		var p struct {
			Prompt string `json:"prompt"`
			Router struct {
				Model string `json:"model"`
			} `json:"router"`
			Model string `json:"model"` // back-compat: hormati top-level "model" kalau ada
		}
		if json.Unmarshal([]byte(raw), &p) == nil {
			if p.Prompt != "" {
				c.Prompt = p.Prompt
			}
			if p.Router.Model != "" {
				c.Model = p.Router.Model
			} else if p.Model != "" {
				c.Model = p.Model
			}
		}
	}
	return c
}

func handleMessage(argsJSON string) {
	var in struct {
		Text string `json:"text"`
		User string `json:"user"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &in)
	if strings.TrimSpace(in.Text) == "" {
		emit(map[string]any{"reply": "(pesan kosong)"})
		return
	}
	emit(map[string]any{"reply": callLLM(loadConfig(), in.Text), "agent": selfID()})
}

// callLLM — TOOL-LOOP autonomus (inti standar Flowork). Persona DB + DNA (self-prompt
// render) jadi system; tools dari subscription. Loop: LLM → tool → feed → ulang, dengan
// ghost-guard + time-bound + auto-continue. Balik teks final (atau konfirmasi lanjutan).
// callRouterRetry — ITEM 4 resilience: panggil router LLM dgn RETRY exp-backoff buat error TRANSIENT
// (jaringan/timeout/5xx/429/408), TRANSPARAN ke loop. Fatal (4xx selain 429/408) → balik langsung, NO
// retry. Akar fix "router error → loop muter". Prinsip Claude (withRetry.ts). Switch: FLOWORK_ROUTER_RETRY
// (default 5). DI TEMPLATE = SEMUA ant baru lahir tahan-banting.
func callRouterRetry(reqJSON []byte) (*httpResp, error) {
	maxAtt := 5
	if v := strings.TrimSpace(os.Getenv("FLOWORK_ROUTER_RETRY")); v != "" {
		if n := atoiSafe(v); n >= 1 {
			maxAtt = n
		}
	}
	hdr := map[string]string{"Content-Type": "application/json"}
	var lastErr error
	for att := 1; att <= maxAtt; att++ {
		resp, err := fetch("POST", routerURL, hdr, reqJSON, 240000)
		if err == nil && resp != nil && resp.Status < 400 {
			return resp, nil
		}
		transient := err != nil || resp == nil || resp.Status >= 500 || resp.Status == 429 || resp.Status == 408
		if !transient {
			return resp, fmt.Errorf("router fatal HTTP %d (ga di-retry)", resp.Status)
		}
		lastErr = err
		if att >= maxAtt {
			break
		}
		d := 500 << uint(att-1)
		if d > 30000 {
			d = 30000
		}
		jitter := int(hostTimeNowMs() % uint64(d/4+1))
		time.Sleep(time.Duration(d+jitter) * time.Millisecond)
	}
	if lastErr != nil {
		return nil, fmt.Errorf("router ga nyambung setelah %d retry: %v", maxAtt, lastErr)
	}
	return nil, fmt.Errorf("router error (5xx) setelah %d retry", maxAtt)
}

// atoiSafe — parse int sederhana (wasip1-safe). Balik -1 kalau invalid.
func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func callLLM(cfg agentConfig, userText string) string {
	sys := cfg.Prompt
	if dna := fetchSelfPrompt(); dna != "" { // konstitusi sacred + doktrin (anti-halu/sync-honest/autonomy-mode)
		sys = sys + "\n\n" + dna
	}
	msgs := []map[string]any{
		{"role": "system", "content": sys},
		{"role": "user", "content": userText},
	}
	toolSpecs := fetchToolSpecs()
	ghostNudges := 0
	loopStartMs := hostTimeNowMs()
	budgetNudged := false

	for iter := 0; iter < maxToolIters; iter++ {
		// AUTO-CONTINUE deterministik: budget abis & belum kelar → harness jadwalin
		// lanjutan sendiri (ScheduleWakeup), nyambung lintas-turn = unbounded over time.
		if !budgetNudged && hostTimeNowMs()-loopStartMs > loopBudgetMs {
			budgetNudged = true
			base, cont := parseAutoCont(userText)
			next := cont + 1
			if next > maxAutoContinue {
				return fmt.Sprintf("⏳ Udah %d kali nyambung otomatis tapi belum kelar — kemungkinan tugasnya kegedean/muter. Gw STOP biar ga infinite. Pecah jadi lebih kecil ya.", cont)
			}
			contPrompt := fmt.Sprintf("[LANJUTAN OTOMATIS #%d] Lo lagi di tengah ngerjain tugas ini & BELUM kelar. LANJUTIN dari progres terakhir (cek memori/hasil sebelumnya — JANGAN ulang dari nol, JANGAN ngaku kelar kalau belum). Pas beneran kelar, jawab hasil FINAL + tutup 'SELESAI'.%s%s", next, autoContDelim, base)
			_ = runTool("ScheduleWakeup", map[string]any{"delaySeconds": 5, "reason": "auto-continue tugas panjang", "prompt": contPrompt})
			return fmt.Sprintf("⏳ Tugasnya panjang — chunk %d kelar, gw jadwalin lanjutan OTOMATIS (nyambung sendiri sampe SELESAI).", next)
		}

		reqMap := map[string]any{"model": cfg.Model, "messages": msgs}
		if len(toolSpecs) > 0 {
			reqMap["tools"] = toolSpecs
			reqMap["parallel_tool_calls"] = false // 1 tool/iter → router aman (anti 400 multi tool_result)
		}
		reqJSON, _ := json.Marshal(reqMap)
		resp, err := callRouterRetry(reqJSON) // ITEM 2/4: retry-backoff transient (transparan ke loop)
		if err != nil || resp == nil {
			return "Maaf, router LLM lagi ga stabil — udah gw coba ulang beberapa kali tapi belum nyambung. Coba lagi sebentar ya."
		}
		var o struct {
			Choices []struct {
				Message struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"message"`
			} `json:"choices"`
		}
		if json.Unmarshal(resp.Body, &o) != nil || len(o.Choices) == 0 {
			return "(no response)"
		}
		m := o.Choices[0].Message
		if len(m.ToolCalls) == 0 {
			// GHOST-GUARD: narasi niat-aksi tanpa tool → paksa 1 putaran (bounded).
			if ghostNudges < maxGhostNudges && looksLikeGhostPromise(m.Content) {
				ghostNudges++
				c := m.Content
				if strings.TrimSpace(c) == "" {
					c = "(niat tanpa tool)"
				}
				msgs = append(msgs, map[string]any{"role": "assistant", "content": c})
				msgs = append(msgs, map[string]any{"role": "user", "content": ghostNudgeMsg})
				continue
			}
			return m.Content // jawaban final
		}
		// SERIALIZE: proses tool_call PERTAMA aja per iterasi (router aman).
		tc := m.ToolCalls[0]
		id := fmt.Sprintf("call_%d", iter)
		content := m.Content
		if strings.TrimSpace(content) == "" {
			content = "(memanggil tool)"
		}
		msgs = append(msgs, map[string]any{"role": "assistant", "content": content,
			"tool_calls": []any{map[string]any{"id": id, "type": "function",
				"function": map[string]any{"name": tc.Function.Name, "arguments": tc.Function.Arguments}}}})
		var targs map[string]any
		if tc.Function.Arguments != "" {
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &targs)
		}
		result := runTool(tc.Function.Name, targs)
		msgs = append(msgs, map[string]any{"role": "tool", "tool_call_id": id, "content": result})
	}
	return "(tool loop limit — coba lagi/perjelas)"
}

func fetchSelfPrompt() string {
	resp, err := fetch("GET", selfPromp+selfID(), nil, nil, 2500)
	if err != nil || resp == nil || resp.Status >= 400 {
		return ""
	}
	var out struct {
		Rendered string `json:"rendered"` // self-prompt render (directive + DNA) — field "rendered"
	}
	if json.Unmarshal(resp.Body, &out) != nil {
		return ""
	}
	return out.Rendered
}

func fetchToolSpecs() []json.RawMessage {
	resp, err := fetch("GET", specsURL+selfID(), nil, nil, 2500)
	if err != nil || resp == nil || resp.Status >= 400 {
		return nil
	}
	var out struct {
		Tools []json.RawMessage `json:"tools"`
	}
	if json.Unmarshal(resp.Body, &out) != nil {
		return nil
	}
	return out.Tools
}

func runTool(name string, args map[string]any) string {
	reqBody, _ := json.Marshal(map[string]any{"tool_name": name, "args": args, "caller": selfID() + "-loop"})
	resp, err := fetch("POST", toolRun+selfID(), map[string]string{"Content-Type": "application/json"}, reqBody, 30000)
	if err != nil || resp == nil {
		return `{"error":"tool dispatch gagal"}`
	}
	if resp.Status >= 400 {
		return fmt.Sprintf(`{"error":"tool http %d"}`, resp.Status)
	}
	out := string(resp.Body)
	if len(out) > 8*1024 {
		out = out[:8*1024] + " …[truncated]"
	}
	return out
}

// parseAutoCont — pesan LANJUTAN ("[LANJUTAN OTOMATIS #N] ...===TUGAS===<task>") →
// (task asli, N). Fresh (tanpa marker) → (s, 0). Counter ride di prompt (stateless).
func parseAutoCont(s string) (base string, count int) {
	const pfx = "[LANJUTAN OTOMATIS #"
	if !strings.HasPrefix(s, pfx) {
		return s, 0
	}
	end := strings.IndexByte(s, ']')
	if end < 0 || end <= len(pfx) {
		return s, 0
	}
	for _, r := range s[len(pfx):end] {
		if r >= '0' && r <= '9' {
			count = count*10 + int(r-'0')
		}
	}
	if k := strings.Index(s, autoContDelim); k >= 0 {
		return strings.TrimSpace(s[k+len(autoContDelim):]), count
	}
	return s, count
}

const ghostNudgeMsg = "⚠️ Lo barusan bilang mau ngelakuin sesuatu (cek/list/scan/cari/tunggu) TAPI GA manggil tool. Itu ghosting — DILARANG. LAKUIN SEKARANG: panggil tool yang lo maksud. Kalau harus nunggu kerja lama, panggil ScheduleWakeup(delaySeconds, reason, prompt) biar kebangun otomatis & lanjut. JANGAN jawab teks doang."

// ghostPhrases — sinyal niat-aksi/kelanjutan khas ghosting (substring match, no regexp).
var ghostPhrases = []string{
	"tunggu bentar", "tunggu sebentar", "tunggu ya", "bentar gw", "bentar ya",
	"gw cek dulu", "cek dulu ya", "gw lihat dulu", "gw list dulu", "gw scan dulu",
	"gw cari dulu", "gw proses dulu", "lagi gw cek", "hasilnya nyusul", "nyusul ya",
	"stay tuned", "nanti gw kabarin", "nanti gw lapor",
	"lanjut ke huruf", "mulai ke huruf", "lanjut ke pencarian", "berikutnya...",
	"berikutnya…", "scan berikutnya", "lanjut scan", "mulai scanning", "iterasi berikutnya",
	"lanjut ke tahap berikutnya", "lanjut ke iterasi",
}

func looksLikeGhostPromise(s string) bool {
	low := strings.ToLower(s)
	for _, p := range ghostPhrases {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}
