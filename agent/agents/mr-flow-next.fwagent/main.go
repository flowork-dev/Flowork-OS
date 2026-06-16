// ⚠️ NEW AGENT? READ doc/handbook/menu-ai-agent.md FIRST — enforced rules: secrets→Settings·API Keys, prompt→GUI (kv.prompt), two-tier brain (router+per-agent), bus over fs:shared, extend the frozen kernel via hooks (never unlock). Breaking one is a bug, not a shortcut.
// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Locked at: 2026-06-12 — mature core agent, stable.
//
// Package main is Mr.Flow, loket-native — the first mature agent migrated to the
// "papan kosong" (blank board) microkernel. It reaches EVERY capability through
// the single kernel counter call(cap, args): its own brain, the real clock, and
// the LLM service. It owns no privileged code of its own.
//
// Built ALONGSIDE the legacy mr-flow (non-destructive migration): the old agent
// stays live until this one is proven, then we swap. Persona (prompt.md) and the
// sacred doctrine (doktrin.md) live as transparent, editable files in this
// agent's OWN folder and travel with it — nothing the owner cannot read + change.
//
// Phase A scope: an HONEST chat core — recall brain, ground on the real clock,
// obey the doctrine, answer in Mr.Flow's voice. No fake tools and no daemon yet:
// tools (Phase C) and the Telegram channel (Phase D) arrive as separate modules,
// so the agent never claims a capability it does not actually have (anti-halu).
//
// Build: GOOS=wasip1 GOARCH=wasm go build -o agent.wasm .
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

//go:wasmimport flowork host_net_fetch
func hostNetFetch(reqPtr, reqLen, outPtr, outMax uint32) uint32

// outBuf receives the host's response. 256KB is plenty for a loket Result.
var outBuf [262144]byte

func bytesPtr(b []byte) uint32 {
	if len(b) == 0 {
		return 0
	}
	return uint32(uintptr(unsafe.Pointer(&b[0])))
}

func emit(v any) {
	b, _ := json.Marshal(v)
	fmt.Println(string(b))
}

// selfID is this agent's id, injected by the host via FLOWORK_AGENT_ID.
func selfID() string {
	if id := strings.TrimSpace(os.Getenv("FLOWORK_AGENT_ID")); id != "" {
		return id
	}
	return "mr-flow-next"
}

// readWS reads a file from this agent's OWN folder (mounted at /workspace). The
// persona (prompt.md) and the sacred doctrine (doktrin.md) live there as plain,
// transparent files that travel with the folder. Returns "" if absent.
func readWS(name string) string {
	b, err := os.ReadFile("/workspace/" + name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

const loketURL = "http://127.0.0.1:1987/api/kernel/call"

// loketCall is the agent's ONLY door to the world: ask the kernel for a
// capability by name. The host stamps our verified id + the loopback secret on
// the outbound request, so the kernel always knows it is us — un-forgeable.
// Returns the raw "result" on success, or an error if the kernel refused.
func loketCall(capName string, args any) (json.RawMessage, error) {
	argsJSON, _ := json.Marshal(args)
	body, _ := json.Marshal(map[string]any{"cap": capName, "args": json.RawMessage(argsJSON)})

	reqJSON, _ := json.Marshal(map[string]any{
		"method": "POST",
		"url":    loketURL,
		// 240s (was 120s) — match the kernel loket CallHandler deadline (also 240s). The
		// old 120s made the orchestrator give up on a slow LOCAL-model sub-call (e.g.
		// ask_group / a multi-round tool loop) while the kernel was still willing to wait,
		// turning a slow-but-fine call into a premature failure (OPS-1, owner-approved 2026-06-16).
		"timeout_ms":     240000,
		"max_resp_bytes": 4 << 20,
		"headers":        map[string]string{"Content-Type": "application/json"},
		"body_base64":    base64.StdEncoding.EncodeToString(body),
	})
	n := hostNetFetch(bytesPtr(reqJSON), uint32(len(reqJSON)), bytesPtr(outBuf[:]), uint32(len(outBuf)))
	if n == 0 {
		return nil, fmt.Errorf("host_net_fetch: 0 bytes")
	}
	var host struct {
		Status  int    `json:"status"`
		BodyB64 string `json:"body_base64"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(outBuf[:n], &host); err != nil {
		return nil, fmt.Errorf("host decode: %w", err)
	}
	if host.Error != "" {
		return nil, fmt.Errorf("host: %s", host.Error)
	}
	raw, _ := base64.StdEncoding.DecodeString(host.BodyB64)
	var res struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("loket decode: %w (body=%s)", err, string(raw))
	}
	if !res.OK {
		return nil, fmt.Errorf("loket refused: %s", res.Error)
	}
	return res.Result, nil
}

// defaultModel — a SMALL model by default (the ant ethos: sovereign, runs local).
// Mr.Flow is the commander, but Phase A is plain chat; flip to a larger model in
// config when the orchestration tools land. Overridable via FLOWORK_AGENT_CONFIG.
const defaultModel = "claude-haiku-4-5"

// model resolves which model to use: config override, else the small default.
func model() string {
	raw := strings.TrimSpace(os.Getenv("FLOWORK_AGENT_CONFIG"))
	if raw == "" {
		return defaultModel
	}
	var c struct {
		Router struct {
			Model string `json:"model"`
		} `json:"router"`
		Model string `json:"model"`
	}
	if json.Unmarshal([]byte(raw), &c) == nil {
		if c.Router.Model != "" {
			return c.Router.Model
		}
		if c.Model != "" {
			return c.Model
		}
	}
	return defaultModel
}

func main() {
	if len(os.Args) < 2 {
		emit(map[string]string{"error": "missing function"})
		return
	}
	fn := os.Args[1]
	args := "{}"
	if len(os.Args) > 2 && os.Args[2] != "" {
		args = os.Args[2]
	}
	switch fn {
	case "handle_message":
		// Direct/RPC invocation: args is the payload itself, e.g. {"text":"..."}.
		handleMessage(args)
	case "handle":
		// Loket-bus invocation: args is a loket Message — unwrap its payload so a
		// channel or a group can route work to this agent the same way a chat does.
		var msg struct {
			Payload json.RawMessage `json:"payload"`
		}
		_ = json.Unmarshal([]byte(args), &msg)
		if len(msg.Payload) == 0 {
			msg.Payload = json.RawMessage(args)
		}
		handleMessage(string(msg.Payload))
	case "boot":
		// No daemon yet — Telegram arrives via a separate channel module (Phase D).
		emit(map[string]any{"ok": true})
	case "on_load":
		// Lifecycle (§8.A): init. Record our load time in our own store.
		if ts := nowUTC(); ts != "" {
			_, _ = loketCall("store.kv.set", map[string]any{"k": "last_load", "v": ts})
		}
		emit(map[string]any{"ok": true})
	case "on_stop":
		// Lifecycle (§8.A): death-letter — leave a wasiat in our OWN brain before we go.
		_, _ = loketCall("store.brain.add", map[string]any{
			"content": "[death-letter] mr-flow-next stopped at " + nowUTC(),
			"wing":    "experience",
			"room":    "lifecycle",
		})
		emit(map[string]any{"ok": true})
	default:
		emit(map[string]any{"error": "unknown function: " + fn})
	}
}

// nowUTC asks the kernel for the real clock (time.now). Grounding the model on
// the true time is part of anti-halu: never let it guess "today".
func nowUTC() string {
	r, err := loketCall("time.now", map[string]any{})
	if err != nil {
		return ""
	}
	var s struct {
		TS string `json:"ts"`
	}
	_ = json.Unmarshal(r, &s)
	return s.TS
}

// recall searches our OWN isolated brain for memories related to the message, so
// the answer is grounded in what we actually remember — not invented. Returns the
// joined memory block and the hit count (the count feeds the debug affordance).
func recall(text string) (string, int) {
	r, err := loketCall("store.brain.search", map[string]any{"query": text, "k": 3})
	if err != nil {
		return "", 0
	}
	var s struct {
		Hits []struct {
			Content string `json:"content"`
		} `json:"hits"`
	}
	if json.Unmarshal(r, &s) != nil || len(s.Hits) == 0 {
		return "", 0
	}
	var b strings.Builder
	n := 0
	for _, h := range s.Hits {
		if c := strings.TrimSpace(h.Content); c != "" {
			b.WriteString("- ")
			b.WriteString(c)
			b.WriteString("\n")
			n++
		}
	}
	return strings.TrimSpace(b.String()), n
}

// searchShared queries the 5M shared corpus (brain.shared.search) — a PRIMARY-tier
// privilege. The kernel REFUSES it for extension agents (the tier gate); this same
// code then simply returns "" and the agent grounds on its local brain only. So one
// wasm stays correct at either tier, with the KERNEL — not the agent — enforcing.
func searchShared(query string) (text string, n int, status string) {
	r, err := loketCall("brain.shared.search", map[string]any{"query": query, "k": 3})
	if err != nil {
		// A refusal here is the tier gate doing its job for an extension agent.
		return "", 0, "refused/err: " + err.Error()
	}
	var s struct {
		Hits []struct {
			Content string `json:"content"`
		} `json:"hits"`
	}
	if json.Unmarshal(r, &s) != nil {
		return "", 0, "decode-err"
	}
	var b strings.Builder
	for _, h := range s.Hits {
		c := strings.TrimSpace(h.Content)
		if c == "" {
			continue
		}
		if len(c) > 600 {
			c = c[:600] + "…"
		}
		b.WriteString("- ")
		b.WriteString(c)
		b.WriteString("\n")
		n++
	}
	return strings.TrimSpace(b.String()), n, "ok"
}

// handleMessage is Mr.Flow's chat core. Every step goes through the loket:
// recall brain → ground on the real clock → obey the doctrine → answer in
// Mr.Flow's voice → remember the exchange.
func handleMessage(argsJSON string) {
	var in struct {
		Text   string `json:"text"`
		User   string `json:"user"`
		ChatID int64  `json:"chat_id"`
		Debug  bool   `json:"debug"` // owner diagnostic: report grounding sources used
	}
	_ = json.Unmarshal([]byte(argsJSON), &in)
	in.Text = strings.TrimSpace(in.Text)
	if in.Text == "" {
		emit(map[string]any{"reply": "(pesan kosong)", "agent": selfID()})
		return
	}

	// Slash commands (/cmd …) are dispatched by the engine slash registry, NOT the
	// LLM — deterministic, reliable, independent of the model.
	// Internal: telegram-channel fetches the group→command list here (over the bus)
	// to auto-sync the Telegram slash menu. Not user-facing.
	if strings.TrimSpace(in.Text) == "/__groupcmds__" {
		emit(map[string]any{"reply": groupCommandsJSON(), "agent": selfID()})
		return
	}
	// GROUP SLASH command (/<cmd> <problem>) — discoverable in Telegram's menu, for ANY
	// owner-listed group. Checked before the generic slash handler so it isn't swallowed.
	if gid, subj, ok := stripGroupSlash(in.Text); ok {
		if subj == "" {
			c := in.Text
			if i := strings.IndexAny(c, " \n"); i >= 0 {
				c = c[:i]
			}
			emit(map[string]any{"reply": "Kasih gw masalahnya, bro 🙂\nContoh:\n" + c + " gimana caranya naikin omzet 2x dalam 3 bulan tanpa modal?", "agent": selfID()})
			return
		}
		handleGroupChat(gid, subj, in.ChatID)
		return
	}
	if strings.HasPrefix(in.Text, "/") {
		emit(map[string]any{"reply": slashRun(in.Text), "agent": selfID()})
		return
	}

	// Computer-control PRE-ROUTER (deterministic, before the LLM): a clear power/
	// app command MUST act, not be second-guessed by the model. If the text looks
	// like one, route it straight to the operasi-komputer-grup GROUP (whose member
	// is the operator executor) and return its reply — no LLM hedging on shutdown.
	if isComputerCommand(in.Text) {
		res := askGroup(json.RawMessage(`{"group":"operasi-komputer-grup","subject":` + jsonStr(in.Text) + `}`))
		reply := in.Text
		var gr struct {
			GroupResult string `json:"group_result"`
			Error       string `json:"error"`
		}
		if json.Unmarshal([]byte(res), &gr) == nil {
			if gr.GroupResult != "" {
				reply = gr.GroupResult
			} else if gr.Error != "" {
				reply = "kontrol komputer error: " + gr.Error
			}
		}
		// The group (no synth) labels member sections "### <id> …"; for a single
		// executor that's noise — strip the leading "### <id>" token (the section
		// separator may be a newline OR a space, since jsonEsc flattens newlines).
		if strings.HasPrefix(reply, "### ") {
			rest := reply[len("### "):]
			if i := strings.IndexAny(rest, " \n"); i >= 0 {
				reply = strings.TrimSpace(rest[i+1:])
			}
		}
		emit(map[string]any{"reply": reply, "agent": selfID()})
		return
	}

	// Thinking PRE-ROUTER (deterministic, before the LLM): when the owner explicitly
	// asks to think a problem through ("pikirin pake tim thinking", "cara berpikir",
	// "pikir mateng"), route straight to the `thinking` GROUP and return its SYNTH
	// answer verbatim — no LLM hedging (so it reliably delegates) and no final LLM
	// wrap (so the long multi-lens pipeline never trips mr-flow's response deadline).
	if isThinkingCommand(in.Text) {
		handleGroupChat("thinking", in.Text, in.ChatID)
		return
	}

	// Saham/investasi PRE-ROUTER (deterministik) → ASYNC. Tim investment (6 agent) di model LOKAL
	// >4 menit → kalau sync, channel timeout ("loket: no response", OPS-1). Solusi: delegasi ASYNC
	// via task_run (category "saham" = mirror group investment) — balik ACK CEPET, crew jalan di
	// background, hasil dikabarin balik ke chat pas kelar (notifyTelegram). Bot gak pernah timeout.
	if in.ChatID != 0 && isInvestmentCommand(in.Text) {
		args := json.RawMessage(`{"category":"saham","subject":` + jsonStr(in.Text) +
			`,"notify_chat_id":` + jsonStr(strconv.FormatInt(in.ChatID, 10)) + `}`)
		_ = toolRun("task_run", args) // async: run jalan di belakang, notify Telegram pas kelar
		emit(map[string]any{"reply": "⏳ Oke bro, gw lempar ke Tim Saham — lagi dianalisa (harga, fundamental, teknikal, risiko). Beberapa menit ya, hasilnya gw kabarin di chat ini pas kelar 📈", "agent": selfID()})
		return
	}

	// Doctrine is SACRED and injected FIRST — the always-on anti-halu layer.
	doktrin := readWS("doktrin.md")
	persona := readWS("prompt.md")
	if persona == "" {
		persona = "Lo Mr.Flow, AI agent Flowork buat Mr.Dev. Jawab santai Bahasa Indonesia, jujur, anti-halu."
	}
	personaBlock := persona
	if ts := nowUTC(); ts != "" {
		personaBlock += "\n[WAKTU_UTC: " + ts + "]"
	}

	msgs := []any{}
	if doktrin != "" {
		msgs = append(msgs, map[string]any{"role": "system", "content": doktrin})
	}
	msgs = append(msgs, map[string]any{"role": "system", "content": personaBlock})
	mem, localN := recall(in.Text)
	if mem != "" {
		msgs = append(msgs, map[string]any{
			"role":    "system",
			"content": "[INGATAN terkait dari brain lo — pakai kalau relevan, jangan ngarang di luar ini]:\n" + mem,
		})
	}
	// PRIMARY privilege: pull grounding from the 5M shared corpus. Refused (and
	// skipped) automatically if this agent is ever run at extension tier.
	shared, sharedN, sharedStatus := searchShared(in.Text)
	if shared != "" {
		msgs = append(msgs, map[string]any{
			"role":    "system",
			"content": "[REFERENSI dari korpus bersama 5jt — bahan grounding, WAJIB verifikasi sebelum diklaim sebagai fakta, jangan telan mentah]:\n" + shared,
		})
	}
	// Multi-turn: replay the recent conversation turns so the agent is NOT stateless
	// — it remembers what was just said, not only FTS-relevant brain hits.
	histKey := histKeyFor(in.User, in.ChatID)
	hist := histLoad(histKey)
	for _, turn := range hist {
		msgs = append(msgs, map[string]any{"role": turn.Role, "content": turn.Content})
	}
	msgs = append(msgs, map[string]any{"role": "user", "content": in.Text})

	// Tools: offer the engine-selected set to the model and run a tool-calling loop
	// — every hop (specs, llm, run) through the single loket counter.
	specs := toolSpecs()
	reply, toolsUsed := runToolLoop(msgs, specs)

	// Remember the exchange in our own isolated brain so it survives across turns.
	if reply != "" {
		_, _ = loketCall("store.brain.add", map[string]any{
			"content": "User: " + in.Text + "\nMr.Flow: " + reply,
			"wing":    "experience",
		})
		// Append this turn to the rolling conversation buffer (recent context).
		histSave(histKey, append(hist,
			histTurn{Role: "user", Content: in.Text},
			histTurn{Role: "assistant", Content: reply}))
	}

	out := map[string]any{"reply": reply, "agent": selfID()}
	if in.Debug {
		// Transparent grounding report (off by default). Lets the owner see exactly
		// which sources the answer was grounded on — and confirms the shared-corpus
		// tier grant is live for this primary agent.
		out["_debug"] = map[string]any{
			"local_hits":    localN,
			"shared_hits":   sharedN,
			"shared_status": sharedStatus,
			"model":         model(),
			"tools_exposed": len(specs),
			"tool_calls":    toolsUsed,
		}
	}
	emit(out)
}

// ── multi-turn conversation buffer (recent turns, per user/chat) ──────────────

type histTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

const maxHistTurns = 6 // keep the last 6 exchanges (12 messages)

// histKeyFor derives a stable conversation key from the user or chat id.
func histKeyFor(user string, chatID int64) string {
	k := strings.TrimSpace(user)
	if k == "" && chatID != 0 {
		k = fmt.Sprintf("%d", chatID)
	}
	if k == "" {
		k = "default"
	}
	return "hist:" + k
}

func histLoad(key string) []histTurn {
	r, err := loketCall("store.kv.get", map[string]any{"k": key})
	if err != nil {
		return nil
	}
	var s struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(r, &s) != nil || s.Value == "" {
		return nil
	}
	var turns []histTurn
	_ = json.Unmarshal([]byte(s.Value), &turns)
	return turns
}

func histSave(key string, turns []histTurn) {
	if len(turns) > maxHistTurns*2 {
		turns = turns[len(turns)-maxHistTurns*2:]
	}
	b, _ := json.Marshal(turns)
	_, _ = loketCall("store.kv.set", map[string]any{"k": key, "v": string(b)})
}

// slashRun dispatches a /command via the loket (slash.run) and returns its text.
func slashRun(text string) string {
	r, err := loketCall("slash.run", map[string]any{"text": text})
	if err != nil {
		return "[slash] " + err.Error()
	}
	var s struct {
		Result map[string]any `json:"result"`
		Error  string         `json:"error"`
	}
	if json.Unmarshal(r, &s) == nil {
		if s.Error != "" {
			return "[slash] " + s.Error
		}
		for _, k := range []string{"text", "Text"} {
			if v, ok := s.Result[k].(string); ok && v != "" {
				return v
			}
		}
	}
	return string(r)
}

// maxToolIters bounds the tool-calling loop. Tools run sequentially (one per
// turn), so a handful of rounds covers most tasks; the bound stops a runaway.
const maxToolIters = 10

// toolSpecs fetches the OpenAI function schemas the engine exposes to us
// (tool.specs). The engine picks the set (core, anti-over-prompt); we just offer
// them. Empty on any error → the agent still answers from its own knowledge.
func toolSpecs() []json.RawMessage {
	specs := []json.RawMessage{}
	if r, err := loketCall("tool.specs", map[string]any{}); err == nil {
		var out struct {
			Tools []json.RawMessage `json:"tools"`
		}
		if json.Unmarshal(r, &out) == nil {
			specs = out.Tools
		}
	}
	// Append the synthetic `ask_group` ORCHESTRATION tool. It is loket-native
	// (handled locally in runToolLoop, NOT via the engine tool bridge): it lets
	// Mr.Flow hand a deep-analysis job to a GROUP (a colony of ants) and weave the
	// group's synthesized answer into its reply. Offered ONLY when the owner has
	// listed groups in config (store.kv "groups") — so this is config-driven and
	// Mr.Flow can reach only the groups the owner allowed (isolation).
	if gs := availableGroups(); len(gs) > 0 {
		lines := make([]string, 0, len(gs))
		for _, g := range gs {
			lines = append(lines, "- "+g.ID+": "+g.Desc)
		}
		spec := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        "ask_group",
				"description": "Delegasikan analisa MENDALAM ke sebuah GROUP (koloni agent yang ngerjain banyak sudut pandang lalu digabung synthesizer). Pakai kalau user minta analisa yang cocok salah satu group ini:\n" + strings.Join(lines, "\n"),
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"group":   map[string]any{"type": "string", "description": "id group tujuan (salah satu dari daftar)"},
						"subject": map[string]any{"type": "string", "description": "subjek / pertanyaan yang mau dianalisa group"},
					},
					"required": []string{"group", "subject"},
				},
			},
		}
		if b, err := json.Marshal(spec); err == nil {
			specs = append(specs, b)
		}
	}
	return specs
}

// groupRef is one delegable GROUP: its id + a short description for the LLM.
type groupRef struct {
	ID, Command, Desc string
	Memory            bool // conversational memory on by default; "nomem" groups (executors) skip it
}

// deriveCommand turns a group id into a clean Telegram slash command (a-z0-9_, the
// first segment) when the allowlist entry doesn't declare one explicitly.
func deriveCommand(id string) string {
	seg := id
	if i := strings.Index(seg, "-"); i > 0 {
		seg = seg[:i]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(seg) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	cmd := b.String()
	if cmd == "" {
		cmd = "group"
	}
	if len(cmd) > 32 {
		cmd = cmd[:32]
	}
	return cmd
}

// availableGroups reads which GROUPs this orchestrator may delegate to from its OWN
// config (store.kv "groups"). Each entry is "id|command|desc" (or legacy "id:desc",
// where the command is derived from the id). Config-driven: adding a group is owner
// config, never a code change, and Mr.Flow reaches ONLY the groups the owner listed.
func availableGroups() []groupRef {
	r, err := loketCall("store.kv.get", map[string]any{"k": "groups"})
	if err != nil {
		return nil
	}
	var s struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(r, &s) != nil || strings.TrimSpace(s.Value) == "" {
		return nil
	}
	var out []groupRef
	for _, part := range strings.Split(s.Value, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		g := groupRef{Memory: true}
		if strings.Contains(part, "|") {
			f := strings.SplitN(part, "|", 4)
			g.ID = strings.TrimSpace(f[0])
			if len(f) > 1 {
				g.Command = strings.TrimSpace(f[1])
			}
			if len(f) > 2 {
				g.Desc = strings.TrimSpace(f[2])
			}
			if len(f) > 3 {
				if flag := strings.ToLower(strings.TrimSpace(f[3])); flag == "nomem" || flag == "raw" {
					g.Memory = false
				}
			}
		} else { // legacy "id:desc"
			g.ID = part
			if i := strings.Index(part, ":"); i > 0 {
				g.ID = strings.TrimSpace(part[:i])
				g.Desc = strings.TrimSpace(part[i+1:])
			}
		}
		if g.ID == "" {
			continue
		}
		if g.Command == "" {
			g.Command = deriveCommand(g.ID)
		}
		out = append(out, g)
	}
	// A3 (karma-aware routing): order by track-record — a group that consistently
	// DELIVERS floats to the top of the ask_group menu / slash list; a group that keeps
	// failing sinks. Soft signal only: EVERY group stays reachable (the model still picks
	// by relevance), and a new/unknown group starts NEUTRAL so it's never buried (fair-chance).
	if len(out) > 1 {
		km := loadGroupKarma()
		sort.SliceStable(out, func(i, j int) bool {
			return groupKarmaScore(km, out[i].ID) > groupKarmaScore(km, out[j].ID)
		})
	}
	return out
}

// ── A3: karma-aware group routing (per-group track-record) ───────────────────
// Kept in OUR OWN kv as a single JSON map "group_karma" (id→score) — one read per
// availableGroups, no cross-agent writes (isolation holds). Bumped on each delegation
// outcome in askGroup. Clamped so one streak can't permanently exalt/bury a group.
const groupKarmaNeutral = 1.0

func loadGroupKarma() map[string]float64 {
	m := map[string]float64{}
	r, err := loketCall("store.kv.get", map[string]any{"k": "group_karma"})
	if err != nil {
		return m
	}
	var s struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(r, &s) == nil && strings.TrimSpace(s.Value) != "" {
		_ = json.Unmarshal([]byte(s.Value), &m)
	}
	return m
}

// groupKarmaScore — score for id; unknown = neutral (fair start, not buried).
func groupKarmaScore(km map[string]float64, id string) float64 {
	if v, ok := km[id]; ok {
		return v
	}
	return groupKarmaNeutral
}

// bumpGroupKarma — +1 on a delivered result, -1 on failure. Clamped [-5, 20].
func bumpGroupKarma(id string, ok bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	m := loadGroupKarma()
	cur := groupKarmaScore(m, id)
	if ok {
		cur += 1.0
	} else {
		cur -= 1.0
	}
	if cur > 20 {
		cur = 20
	}
	if cur < -5 {
		cur = -5
	}
	m[id] = cur
	b, _ := json.Marshal(m)
	_, _ = loketCall("store.kv.set", map[string]any{"k": "group_karma", "v": string(b)})
}

// askGroup is the orchestrator hop: mr-flow → GROUP → member ants → synthesizer →
// back. It hands the subject to the group over the loket bus (bus.request) and
// returns the group's synthesized answer as the tool result, which the LLM then
// weaves into its final reply. Only owner-listed groups are reachable.
func askGroup(argsRaw json.RawMessage) string {
	var a struct {
		Group   string `json:"group"`
		Subject string `json:"subject"`
	}
	_ = json.Unmarshal(argsRaw, &a)
	a.Group = strings.TrimSpace(a.Group)
	a.Subject = strings.TrimSpace(a.Subject)
	if a.Group == "" || a.Subject == "" {
		return `{"error":"ask_group: group dan subject wajib"}`
	}
	ok := false
	for _, g := range availableGroups() {
		if g.ID == a.Group {
			ok = true
			break
		}
	}
	if !ok {
		return `{"error":"group tidak terdaftar di config (minta owner daftarin di kv groups)"}`
	}
	r, err := loketCall("bus.request", map[string]any{
		"to":      a.Group,
		"type":    "task",
		"payload": map[string]any{"text": a.Subject},
	})
	if err != nil {
		bumpGroupKarma(a.Group, false) // A3: delegation failed → lower this group's track-record
		return `{"error":"group error: ` + jsonEsc(err.Error()) + `"}`
	}
	// bus.request → {reply:<group emit>}; the group emit → {reply:"…", …}. Unwrap.
	var outer struct {
		Reply json.RawMessage `json:"reply"`
	}
	_ = json.Unmarshal(r, &outer)
	var inner struct {
		Reply string `json:"reply"`
	}
	if json.Unmarshal(outer.Reply, &inner) == nil && inner.Reply != "" {
		bumpGroupKarma(a.Group, true) // A3: delivered → raise this group's track-record
		// jsonStr (proper JSON marshal) preserves newlines as \n — using jsonEsc here
		// flattened them to spaces, turning a formatted answer into one wall of text.
		return `{"group_result":` + jsonStr(inner.Reply) + `}`
	}
	return string(r)
}

// runToolLoop is Mr.Flow's tool-calling loop, every hop through the loket: offer
// tools → the model asks for one → tool.run executes it → feed the result back →
// repeat until the model answers in plain text. Returns the final reply + how
// many tools ran.
//
// It mirrors the legacy agent's hard-won rules: process only the FIRST tool_call
// per turn (the router mistranslates parallel tool_results into a 400), keep the
// assistant content non-empty (some providers reject empty content alongside
// tool_calls), and pair each tool_call id with exactly one tool result.
func runToolLoop(msgs []any, specs []json.RawMessage) (string, int) {
	toolsUsed := 0
	rescued := false
	for iter := 0; iter < maxToolIters; iter++ {
		llmArgs := map[string]any{"model": model(), "messages": msgs}
		if len(specs) > 0 {
			llmArgs["tools"] = specs
			llmArgs["parallel_tool_calls"] = false
		}
		r, err := loketCall("llm.complete", llmArgs)
		if err != nil {
			return "[mr-flow-next] LLM error: " + err.Error(), toolsUsed
		}
		var resp struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		}
		_ = json.Unmarshal(r, &resp)
		if len(resp.ToolCalls) == 0 {
			final := strings.TrimSpace(resp.Content)
			// LOCAL-model rescue: gemma sometimes WRITES the delegation as prose
			// ("ask_group(group=…, subject=…)") instead of emitting a structured
			// tool_call — which skips the crew entirely and leaks raw syntax to the
			// user. If the "answer" is really an unexecuted ask_group, run it for real
			// (once). Strictly better: the un-rescued path shows the user pure garbage.
			if !rescued {
				if g, subj, ok := parseTextAskGroup(final); ok {
					rescued = true
					res := askGroup(json.RawMessage(`{"group":` + jsonStr(g) + `,"subject":` + jsonStr(subj) + `}`))
					var gr struct {
						GroupResult string `json:"group_result"`
					}
					if json.Unmarshal([]byte(res), &gr) == nil && strings.TrimSpace(gr.GroupResult) != "" {
						return plainify(strings.TrimSpace(gr.GroupResult)), toolsUsed + 1
					}
					return "Maaf bro, gw mau lempar ini ke tim " + g + " tapi gagal nyambung. Coba lagi bentar ya 🙏", toolsUsed
				}
			}
			return final, toolsUsed // final text answer
		}
		tc := resp.ToolCalls[0] // serialize: one tool per turn
		id := fmt.Sprintf("call_%d", iter)
		content := resp.Content
		if strings.TrimSpace(content) == "" {
			content = "(memanggil tool)"
		}
		msgs = append(msgs, map[string]any{
			"role": "assistant", "content": content,
			"tool_calls": []any{map[string]any{
				"id": id, "type": "function",
				"function": map[string]any{"name": tc.Function.Name, "arguments": tc.Function.Arguments},
			}},
		})
		argsRaw := json.RawMessage("{}")
		if strings.TrimSpace(tc.Function.Arguments) != "" {
			argsRaw = json.RawMessage(tc.Function.Arguments)
		}
		// ask_group is loket-native (orchestration), handled locally; every other
		// tool goes through the engine bridge (tool.run).
		if tc.Function.Name == "ask_group" {
			result := askGroup(argsRaw)
			toolsUsed++
			// ask_group is TERMINAL delegation: the group (with its OWN synthesizer) already
			// produced the user's answer, and the persona says "jawaban tim langsung sampe ke
			// user, gak perlu nulis ulang". So on a real group_result, RETURN IT VERBATIM
			// (plainified, like handleGroupChat) instead of burning ANOTHER slow LOCAL-model
			// LLM round just to re-wrap it — that extra round-trip on the heaviest path is a
			// big chunk of the OPS-1 latency. Only on an ERROR do we feed it back so the model
			// can recover/explain.
			var gr struct {
				GroupResult string `json:"group_result"`
			}
			if json.Unmarshal([]byte(result), &gr) == nil && strings.TrimSpace(gr.GroupResult) != "" {
				return plainify(strings.TrimSpace(gr.GroupResult)), toolsUsed
			}
			msgs = append(msgs, map[string]any{"role": "tool", "tool_call_id": id, "content": result})
			continue
		}
		result := toolRun(tc.Function.Name, argsRaw)
		toolsUsed++
		msgs = append(msgs, map[string]any{
			"role": "tool", "tool_call_id": id, "content": result,
		})
	}
	return "(batas loop tool kena — coba perjelas permintaan lo)", toolsUsed
}

// toolRun executes one tool by name via the loket bridge (tool.run) and returns
// its raw result JSON, fed straight back to the model as the tool reply.
func toolRun(name string, argsRaw json.RawMessage) string {
	r, err := loketCall("tool.run", map[string]any{"name": name, "args": argsRaw})
	if err != nil {
		return `{"error":"tool.run refused/failed: ` + jsonEsc(err.Error()) + `"}`
	}
	return string(r)
}

// jsonEsc minimally escapes a string for inlining inside a JSON error literal.
func jsonEsc(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

// jsonStr marshals a string to a JSON string literal (with quotes).
func jsonStr(s string) string { b, _ := json.Marshal(s); return string(b) }

// parseTextAskGroup rescues an ask_group call the LOCAL model wrote as PROSE
// instead of a structured tool_call (gemma does this intermittently — it narrates
// "ask_group(group=…, subject=…)" as text). It recognises the common shapes:
//
//	ask_group(group="investment", subject="analisa BBCA")
//	ask_group({"group":"investment","subject":"…"})
//
// and returns (group, subject, true) only when BOTH keys are present after the
// literal "ask_group" token — so a mere passing mention never triggers a crew.
func parseTextAskGroup(s string) (string, string, bool) {
	i := strings.Index(s, "ask_group")
	if i < 0 {
		return "", "", false
	}
	seg := s[i+len("ask_group"):]
	g := extractKV(seg, "group")
	subj := extractKV(seg, "subject")
	if g == "" || subj == "" {
		return "", "", false
	}
	return g, subj, true
}

// extractKV pulls the value of key from a loose "key=val" / "key: val" /
// "\"key\":\"val\"" snippet: it finds key, skips the separators/quotes after it,
// and reads up to the next quote, comma, or closing paren/brace. "" if absent.
func extractKV(s, key string) string {
	idx := strings.Index(s, key)
	if idx < 0 {
		return ""
	}
	r := strings.TrimLeft(s[idx+len(key):], " \t:=\"'")
	end := strings.IndexAny(r, "\"',)}\n")
	if end < 0 {
		end = len(r)
	}
	return strings.TrimSpace(r[:end])
}

// plainify strips markdown to clean chat text (Telegram renders no markdown, so
// "##" / "**" / "---" would show raw). Done in code because the model keeps using
// markdown even when told not to. Preserves newlines + simple "-" bullets.
func plainify(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	s = strings.ReplaceAll(s, "`", "")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		t := strings.TrimRight(ln, " \t")
		trimmed := strings.TrimSpace(t)
		// drop horizontal rules (---, ***, ___ of any length)
		if len(trimmed) >= 3 && (strings.Trim(trimmed, "-") == "" || strings.Trim(trimmed, "*") == "" || strings.Trim(trimmed, "_") == "") {
			out = append(out, "")
			continue
		}
		// header lines (#, ##, …) → plain line
		ls := strings.TrimLeft(t, " ")
		if strings.HasPrefix(ls, "#") {
			t = strings.TrimSpace(strings.TrimLeft(ls, "#"))
		} else if strings.HasPrefix(ls, "* ") || strings.HasPrefix(ls, "+ ") {
			indent := t[:len(t)-len(ls)]
			t = indent + "- " + ls[2:]
		}
		out = append(out, t)
	}
	res := strings.Join(out, "\n")
	for strings.Contains(res, "\n\n\n") {
		res = strings.ReplaceAll(res, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(res)
}

// tkvGet/tkvSet — small kv helpers for the thinking conversation memory (own store).
func tkvGet(k string) string {
	r, err := loketCall("store.kv.get", map[string]any{"k": k})
	if err != nil {
		return ""
	}
	var s struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(r, &s) != nil {
		return ""
	}
	return s.Value
}
func tkvSet(k, v string) { _, _ = loketCall("store.kv.set", map[string]any{"k": k, "v": v}) }

// stripGroupSlash recognizes a GROUP slash command (/<command> <problem>, with the
// optional @botname form in groups) and maps it to the group id — for ANY group the
// owner listed. So every group gets a discoverable command automatically; no per-group
// code. Returns (groupID, problem, ok).
func stripGroupSlash(text string) (groupID, subject string, ok bool) {
	t := strings.TrimSpace(text)
	if !strings.HasPrefix(t, "/") {
		return "", "", false
	}
	cmd, rest := t, ""
	if i := strings.IndexAny(t, " \n"); i >= 0 {
		cmd, rest = t[:i], strings.TrimSpace(t[i+1:])
	}
	if i := strings.Index(cmd, "@"); i >= 0 { // strip @botname (groups)
		cmd = cmd[:i]
	}
	c := strings.ToLower(strings.TrimPrefix(cmd, "/"))
	if c == "think" || c == "pikir" || c == "mikir" { // friendly aliases for thinking
		c = "thinking"
	}
	for _, g := range availableGroups() {
		if strings.ToLower(g.Command) == c {
			return g.ID, rest, true
		}
	}
	return "", "", false
}

// groupCommandsJSON returns the owner's groups as a Telegram setMyCommands payload —
// fetched by the telegram-channel (over the bus) so the slash menu auto-syncs with
// whatever groups exist. No shared store needed (respects isolation).
func groupCommandsJSON() string {
	type cmd struct {
		Command     string `json:"command"`
		Description string `json:"description"`
	}
	cs := []cmd{}
	for _, g := range availableGroups() {
		d := strings.TrimSpace(g.Desc)
		if d == "" {
			d = "Group " + g.ID
		}
		if len(d) > 240 {
			d = d[:240]
		}
		cs = append(cs, cmd{Command: g.Command, Description: d})
	}
	b, _ := json.Marshal(map[string]any{"commands": cs})
	return string(b)
}

// handleThinking runs the thinking colony for one user message, with per-chat
// rolling memory so it can continue a multi-turn diagnosis. Used by BOTH the slash
// command and the legacy keyword trigger.
// handleGroupChat — entry buat slash-group + thinking pre-router. Dari Telegram (chatID != 0)
// group BERAT (multi-agent / debate) bisa >4 mnt → kalau SYNC, channel timeout ("loket: no
// response"). Generalisasi pola saham (OPS-1): delegasi ASYNC via task_run(group=…) — ACK cepet,
// crew (member+synth+debate) jalan di belakang, hasil di-notify balik ke chat pas kelar. Bot gak
// pernah timeout. Trade-off: turn async gak nyimpen memory percakapan (konsisten sama saham async).
// chatID==0 (internal/non-Telegram) tetap SYNC (handleGroupChatSync) — perilaku lama, NOL regresi.
func handleGroupChat(groupID, userMsg string, chatID int64) {
	userMsg = strings.TrimSpace(userMsg)
	if chatID != 0 {
		args := json.RawMessage(`{"group":` + jsonStr(groupID) + `,"subject":` + jsonStr(userMsg) +
			`,"notify_chat_id":` + jsonStr(strconv.FormatInt(chatID, 10)) + `}`)
		out := toolRun("task_run", args)
		if !strings.Contains(out, `"error"`) { // dispatch sukses → ACK, hasil nyusul lewat notify
			emit(map[string]any{"reply": groupAckMsg(groupID), "agent": selfID()})
			return
		}
		// dispatch GAGAL (validasi cepet — group ga kebaca / bukan group=1 / no member; BUKAN
		// timeout) → fallback SYNC biar user tetap dapet jawaban/error yang jelas, bukan diem.
	}
	handleGroupChatSync(groupID, userMsg, chatID)
}

// groupAckMsg — ACK ramah pas delegasi group async; sebut deskripsi group kalau ada.
func groupAckMsg(groupID string) string {
	label := groupID
	for _, g := range availableGroups() {
		if g.ID == groupID {
			if d := strings.TrimSpace(g.Desc); d != "" {
				label = d
			}
			break
		}
	}
	return "⏳ Oke bro, gw lempar ke tim " + label + " — lagi digarap. Beberapa menit ya, hasilnya gw kabarin di chat ini pas kelar 🙌"
}

func handleGroupChatSync(groupID, userMsg string, chatID int64) {
	userMsg = strings.TrimSpace(userMsg)
	mem := true
	for _, g := range availableGroups() {
		if g.ID == groupID {
			mem = g.Memory
			break
		}
	}
	histKey := "ghist:" + groupID + ":" + strconv.FormatInt(chatID, 10)
	hist := ""
	if mem {
		if low := strings.ToLower(userMsg); strings.Contains(low, "topik baru") || strings.Contains(low, "mulai baru") || strings.Contains(low, "reset") {
			tkvSet(histKey, "") // explicit fresh start
		}
		hist = tkvGet(histKey)
	}
	subject := userMsg
	if mem && strings.TrimSpace(hist) != "" {
		subject = "Konteks percakapan sebelumnya (LANJUTKAN dari sini, jangan ulang dari nol):\n" +
			hist + "=== Pesan terbaru user ===\n" + userMsg
	}
	res := askGroup(json.RawMessage(`{"group":` + jsonStr(groupID) + `,"subject":` + jsonStr(subject) + `}`))
	reply := ""
	var gr struct {
		GroupResult string `json:"group_result"`
		Error       string `json:"error"`
	}
	if json.Unmarshal([]byte(res), &gr) == nil {
		if gr.GroupResult != "" {
			reply = gr.GroupResult
		} else if gr.Error != "" {
			reply = "grup error: " + gr.Error
		}
	}
	if strings.TrimSpace(reply) == "" {
		reply = "grup ga ngasih jawaban."
	} else {
		reply = plainify(reply)
	}
	if mem && !strings.HasPrefix(reply, "grup error") && !strings.HasPrefix(reply, "grup ga") {
		ans := reply
		if len(ans) > 700 {
			ans = ans[:700] + " …"
		}
		newHist := hist + "User: " + userMsg + "\nTim: " + ans + "\n\n"
		if len(newHist) > 2400 {
			newHist = newHist[len(newHist)-2400:]
		}
		tkvSet(histKey, newHist)
	}
	emit(map[string]any{"reply": reply, "agent": selfID()})
}

// isComputerCommand deterministically detects a host power/app control request, so
// mr-flow routes it straight to the operasi-komputer-grup GROUP instead of letting
// the LLM second-guess a shutdown. Keep it specific (avoid hijacking normal chat).
func isComputerCommand(text string) bool {
	s := strings.ToLower(text)
	kw := []string{
		"matiin pc", "matikan pc", "matiin komputer", "matikan komputer", "shutdown", "shut down",
		"restart pc", "restart komputer", "reboot", "mulai ulang",
		"suspend", "sleep pc", "tidurin pc", "hibernate",
		"kunci layar", "lock screen", "lock pc",
		"logout", "log out",
		"buka chrome", "buka vscode", "buka vs code", "buka code", "open chrome", "open vscode",
		"batal matiin", "batal shutdown", "cancel shutdown",
	}
	for _, k := range kw {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

// isThinkingCommand deterministically detects an explicit request to reason a
// problem through with the `thinking` GROUP, so mr-flow delegates reliably instead
// of the LLM choosing to answer (or ask back) on its own. Kept specific so it never
// hijacks ordinary chat.
// isInvestmentCommand — deteksi niat analisa SAHAM/investasi (owner's persona: SEMUA saham WAJIB
// ke tim investment). Dipakai pre-router ASYNC (di bawah) — tim investment 6-agent di model LOKAL
// >4 menit, jadi gak bisa sync (timeout). Keyword sengaja sempit (saham/emiten/ihsg/bursa) biar
// gak salah-trigger chat biasa.
func isInvestmentCommand(text string) bool {
	s := strings.ToLower(text)
	kw := []string{"saham", "emiten", "ihsg", "bursa efek", "bursa saham"}
	for _, k := range kw {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func isThinkingCommand(text string) bool {
	s := strings.ToLower(text)
	kw := []string{
		"tim thinking", "grup thinking", "group thinking", "thinking group",
		"pake thinking", "pakai thinking", "lewat thinking", "minta thinking",
		"cara berpikir", "pikirin mateng", "pikirin mateng", "pikir mateng",
		"pikirkan matang", "pikir matang", "pikirin mendalam",
	}
	for _, k := range kw {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}
