// FROZEN brain-core — mr-flow (jantung Flowork, agent owner). Kalau ini bikin lo "nyasar": ini BY-DESIGN, baca lock/brain.md dulu. Jangan edit tanpa unfreeze owner.
// 📖 Telegram I/O (format pesan, baca dokumen/foto/voice) + SWITCH → CABANG NON-frozen `telegram_media.go` + dok /home/mrflow/Documents/FLowork_os/lock/mrflow.md. Filtur Telegram baru masuk SITU, JANGAN buka file beku ini.
// ⚠️ NEW AGENT? READ doc/handbook/menu-ai-agent.md FIRST — enforced rules: secrets→Settings·API Keys, prompt→GUI (kv.prompt), two-tier brain (router+per-agent), bus over fs:shared, extend the frozen kernel via hooks (never unlock). Breaking one is a bug, not a shortcut.
// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// 2026-06-15 (owner-approved): classifyRoute GROUNDING — route tool prompt now prefers
//   'chat' when nothing truly fits / on creation-intent (anti-misroute, e.g. "team
//   peramal" no longer hijacked to "repo-reviewer"). Prompt-only; wasm rebuilt. Re-locked.
// 2026-06-20 (owner-approved): GHOST-GUARD anti-ghosting di tool-loop. Akar: pas model
//   jawab TEKS tanpa tool-call, `return m.Content` ngebolehin "janji" ("tunggu bentar gw
//   cek dulu") jadi state TERMINAL → owner nunggu selamanya. Fix deterministik: teks-tanpa-
//   tool yg nyinyalin niat-aksi di-PAKSA 1 putaran lagi (panggil tool SEKARANG atau
//   ScheduleWakeup kalau nunggu), bounded maxGhostNudges (anti infinite). + Tier-1 prompt
//   positif (sebut ScheduleWakeup). Model-agnostic (26B pun ga bisa ghosting struktural).
//   wasm rebuilt (tinygo). Re-locked.
// 2026-06-20 (owner-approved): ANTI-HALU-CREW-MATI. Akar: mr-flow "nyalain crew"
//   yg udah dihapus (mis. "riset saham BBCA" → crew saham mati → hasil ga datang).
//   4 lapis dicabut: (1) deterministicRoute +live-guard `categoryLive()` (route HANYA
//   kalau cat ada di task_list live); (2) classifyRoute: enum kosong → return false
//   (buang fallback hardcoded saham/crypto/dst yg pura-pura crew ada); (3) TASK ROUTER
//   prompt + Tier-3 grounding `liveCrewCount()` → LLM HARAM ngarang run_id/'Processing'
//   kalau ga ada crew, jawab sendiri; (4) data crew mati dibersihin (task_categories/
//   _agents/trigger_rules/seed). State LIVE = sumber kebenaran ("hapus crew = auto
//   ga bisa dipanggil"). QC: "riset saham BBCA" fresh → jawab analisa langsung (NO
//   crew/run_id halu). wasm rebuilt. Re-locked.
// 2026-06-20 (owner-approved): ANTI-ANCHOR (§6.1) di fetchHistory. Akar history-
//   anchoring: window 16-turn nge-feed balik reply gagal/denial LAMA ("gw ga tau")
//   → 26B ngechо pola basi. Fix: isAnchorNoise() skip assistant-reply gagal/denial
//   buat turn LAMA (di luar keepRecentTurns=4 terbaru yg tetep utuh). wasm rebuilt
//   (standard wasip1). Re-locked.
// 2026-06-21 (owner-approved): AUTO-RECALL (D18/D19) — fetchAutoRecall() tiap turn:
//   graph_recall(pesan, budget 2800) + brain_search → inject fakta relevan ke Tier-3
//   (paling salient) dgn directive TEGAS. Akar: brain/graph dulu cuma tool-driven →
//   model lemah ga manggil → "gak punya data" walau fakta ADA. Sekarang fakta owner
//   auto-nongol → recall produksi RELIABLE (fakta personal owner ke-recall akurat;
//   kontrol pertanyaan non-personal/math tetap bener, no over-anchor). Model tetap
//   cfg.Router.Model (GUI, ga hardcode). wasm rebuilt (wasip1). Re-locked.
// 2026-06-21 (owner-approved): PRIVACY D8 — cabut data personal owner dari komentar
//   (changelog AUTO-RECALL di atas + contoh recall di fetchAutoRecall) jadi pola umum,
//   anti-bocor ke repo publik. Comment-only (ga ngubah runtime/wasm). Re-locked.
// 2026-06-22 (owner-approved): RECALL BRAIN directive DIPERKUAT. Akar: brain_search
//   (verbatim drawer) PASTI nemu fakta (mis. guru gitar) + ke-inject, TAPI header brain
//   lemah ("[RECALL BRAIN]:") → model 26B ikut directive KUAT graph + ngabaikan drawer →
//   "ga tau" walau fakta ADA (K11 miss). Fix: header brain disamain TEGAS ("JAWAB PAKAI
//   INI, JANGAN bilang ga tau kalau ADA"). wasm rebuilt (wasip1). Re-locked.
// 2026-06-22 (owner-approved): D32 INC-2 — AUTO-CAPTURE error→recovery (self-learning
//   recovery instinct). Pas tool ERROR lalu tool yg SAMA SUKSES dalam loop turn ini =
//   agent berhasil recover → mistake_log("WHEN <tool> <kelas> -> recovered"). TITLE
//   stabil (tool+kelas) = dedup + hit_count; CONTENT "WHEN..->.." → recovery-instinct
//   via PromoteRecurringMistakes (INC-1) pas hit≥3 (gerbang repetisi anti-degenerasi).
//   Kelas error = bebas path/data owner (privasi D8). Hemat token: agent ga ngulang
//   stuck yg udah ke-recover. NANO-MODULAR: call-site captureRecovery() di sini
//   (tool-loop, EDITABLE); logic-brain (captureRecovery/toolErrClass/skip) di-EKSTRAK
//   ke `recovery_capture.go` (FROZEN terpisah) — main.go calon "list" tetap bisa edit.
//   reuse penuh pipeline INC-1. Arsitektur: lock/brain.md + roadmap_brain.md #1. Re-locked.
// Locked at: 2026-05-30
// Reason: Mr.Flow WASM agent (CRITICAL). Audit pass:
//   - Token + TELEGRAM_ALLOWED_CHATS validation (drop kalau invalid)
//   - chatID whitelist check (drop unauthorized + log decision skip_task)
//   - LLM failure detection via EXACT prefix match (anti false-positive
//     "router error:", "decode:", "llm:", "(no choices)", "")
//   - Length cap 3900 chars per Telegram message limit
//   - Anti-halu guards di callLLM (CURRENT_TIME_UTC inject, IDENTITY guard,
//     helpfulness rule, anti-invent tool whitelist)
//   - Skill auto-inject MAX 3 (maxActiveSkills), per-item 300 char (maxSkillCharsPerItem),
//     total persona 4000 char (maxPersonaTotalChars) — anti over-prompt
//   - Karma update best-effort (silent error OK)
//   - Self-prompt slot prepend (Section 35 phase 2)
//   - Log interactions + decisions per chat outcome
//
// mr-flow — Telegram AI agent untuk Flowork.
//
// === Layout standar (HARDCODED, lihat doc/standar_ai_agent.md) ===
//
//   /workspace/             ← workspace privat agent (mount kernel)
//   /workspace/state.db     ← SQLite per-agent
//   /shared/<id>/tools/     ← tools yang agent bikin sendiri
//   /shared/<id>/job/       ← output kerjaan
//   /shared/<id>/document/  ← markdown/notes/report
//   /shared/<id>/media/     ← audio/video/image
//   /shared/<id>/cache/     ← cache temporary
//   /shared/<id>/log/       ← log
//   /shared/_global/        ← bahan bareng lintas-agent
//
// === Sumber config (kernel inject) ===
//
//   FLOWORK_AGENT_CONFIG   — JSON utuh (prompt, router, schedule, skills)
//   TELEGRAM_BOT_TOKEN     — secrets.TELEGRAM_BOT_TOKEN dari popup
//   TELEGRAM_ALLOWED_CHATS — secrets.TELEGRAM_ALLOWED_CHATS dari popup
//   FLOWORK_AGENT_ID       — id agent (mr-flow)
//
// ABI (command pattern): kernel invoke binary dengan
//   os.Args[0] = "agent"
//   os.Args[1] = function name (boot | handle_message | send_message)
//   os.Args[2] = args JSON
// Output: JSON response ke stdout.

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

//go:wasmimport flowork host_net_fetch
func hostNetFetch(reqPtr, reqLen, outPtr, outMax uint32) uint32

//go:wasmimport flowork host_log_interaction
func hostLogInteraction(reqPtr, reqLen, outPtr, outMax uint32) uint32

//go:wasmimport flowork host_log_decision
func hostLogDecision(reqPtr, reqLen, outPtr, outMax uint32) uint32

//go:wasmimport flowork host_karma_update
func hostKarmaUpdate(reqPtr, reqLen, outPtr, outMax uint32) uint32

//go:wasmimport flowork host_time_now_ms
func hostTimeNowMs() uint64

//go:wasmimport flowork host_slash_dispatch
func hostSlashDispatch(reqPtr, reqLen, outPtr, outMax uint32) uint32

// === Path konstanta (HARDCODED standar Flowork) ===
const (
	WorkspacePrivate = "/workspace"          // mount per-agent (eksklusif)
	WorkspaceDB      = "/workspace/state.db" // SQLite per-agent
	WorkspaceShared  = "/shared"             // mount shared workspace (root project)
)

const (
	defaultRouter  = "http://127.0.0.1:2402/v1/chat/completions"
	defaultModel   = "claude-haiku-4-5"
	defaultPersona = "Lo Mr.Flow — AI Agent di Flowork microkernel buat Mr.Dev. " +
		"Reply natural Bahasa Indonesia santai (bro/lo/gw OK), concise, no markdown headers. " +
		"Kalau gak yakin, bilang gak yakin. Hindari halu."
	pollTimeout  = 25 // seconds
	respBufBytes = 524288
)

// === Prompt Budget (per doc/standar_ai_agent.md section 11) ===
//
// Over-prompt = silent killer terutama buat local LLM (context window kecil).
// Setiap layer yang inject ke system prompt WAJIB respect budget di sini.
// On-demand fetch via tool call lebih baik daripada always-inject.
const (
	maxActiveSkills = 3 // max skill auto-inject ke persona (sisanya via skill_search)
	// LOOP TIME-BOUND, BUKAN CAP-ANGKA (owner 2026-06-20: "loop jangan dibatasi —
	// Flowork didesain buat berevolusi"). In-turn loop jalan TERUS selama masih ada
	// budget WAKTU turn; abis budget → break bersih + model di-arahin ScheduleWakeup
	// buat LANJUT lintas-turn (tidur→bangun→sambung) = unbounded sepanjang waktu.
	// maxToolIters = safety-backstop GEDE (anti runaway no-progress), bukan batas kerja.
	maxToolIters                = 100    // backstop iterasi (anti pure-infinite no-time-advance); batas NYATA = loopBudgetMs
	loopBudgetMs         uint64 = 200000 // budget waktu in-turn (~200s); turn-timeout 290s — margin GEDE biar wrap-up/ScheduleWakeup call (26B lambat ~40s) ga ke-kill
	maxGhostNudges              = 6      // ghost-guard: max paksa-lanjut pas NARASI-tanpa-tool (anti-ghosting). Bukan batas kerja (tool-call ga ngitung) — cukup tinggi biar loop sah jalan, tetep bounded anti narasi-loop
	maxMsgContentChars          = 6000   // cap per-message content sebelum kirim ke LLM
	keepToolResultsFull         = 4      // hasil tool terbaru yang TIDAK di-prune (sisanya diringkas)
	maxSkillCharsPerItem        = 300    // truncate instruction skill kalau terlalu panjang
	maxPersonaTotalChars        = 9000   // hard cap system prompt total (3-tier + memory snapshot)
	// Fase 1 phase-2: memory snapshot capped (per roadmap ~USER 500tok / MEMORY 800tok).
	memUserCap    = 2000 // cap USER.md inject (~500 token approx)
	memProjectCap = 3200 // cap MEMORY.md inject (~800 token approx)
	// Fase 1 phase-2: context compression — ringkas blok tengah kalau history gede.
	compressTriggerChars = 20000 // total content history > ini → trigger compress (~5k token)
	compressKeepTail     = 8     // jumlah pesan terakhir yang DISISAKAN utuh (TAIL)
	summarizeInputCap    = 16000 // cap input ke aux LLM summarizer
)

var outBuf [respBufBytes]byte

// selfID — id agent ini, di-inject host via FLOWORK_AGENT_ID (= manifest.ID).
// Fallback "mr-flow" buat dev/standalone. KUNCI Fase 2 (template copas): agent
// hasil spawn otomatis pake id-nya sendiri di URL self-API (interactions/
// tools/specs/run) tanpa edit kode — cukup ganti manifest.id.
func selfID() string {
	if id := strings.TrimSpace(os.Getenv("FLOWORK_AGENT_ID")); id != "" {
		return id
	}
	return "unknown"
}

// ── Config dari kernel (sumber SQLite-backed) ──────────────────────────────

// Skill — entry dari config.skills[].
type Skill struct {
	ID           string `json:"id"`
	Trigger      string `json:"trigger"`
	Instructions string `json:"instructions"`
}

type agentConfig struct {
	Prompt string `json:"prompt"`
	Router struct {
		URL   string `json:"url"`
		Model string `json:"model"`
	} `json:"router"`
	Skills []Skill `json:"skills"`
}

// loadConfig parse FLOWORK_AGENT_CONFIG kalau ada, fallback ke default.
func loadConfig() agentConfig {
	cfg := agentConfig{}
	cfg.Router.URL = defaultRouter
	cfg.Router.Model = defaultModel
	cfg.Prompt = defaultPersona

	raw := strings.TrimSpace(os.Getenv("FLOWORK_AGENT_CONFIG"))
	if raw == "" {
		return cfg
	}
	var parsed agentConfig
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] FLOWORK_AGENT_CONFIG parse error: %v\n", err)
		return cfg
	}
	if parsed.Prompt != "" {
		cfg.Prompt = parsed.Prompt
	}
	if parsed.Router.URL != "" {
		cfg.Router.URL = parsed.Router.URL
	}
	if parsed.Router.Model != "" {
		cfg.Router.Model = parsed.Router.Model
	}
	cfg.Skills = parsed.Skills
	return cfg
}

// ── Entry ──────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) < 2 {
		emit(map[string]string{"error": "missing function"})
		return
	}
	fn := os.Args[1]
	argsRaw := ""
	if len(os.Args) >= 3 {
		argsRaw = os.Args[2]
	}
	switch fn {
	case "boot":
		runDaemon()
	case "handle_message":
		doHandle(argsRaw)
	case "send_message":
		doSendAdmin(argsRaw)
	default:
		emit(map[string]string{"error": "unknown function: " + fn})
	}
}

// ── Daemon ─────────────────────────────────────────────────────────────────

func runDaemon() {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		fmt.Fprintf(os.Stderr, "[%s] TELEGRAM_BOT_TOKEN belum di-set. Buka Setting → Credentials di popup, tambahin key 'TELEGRAM_BOT_TOKEN' = bot token dari @BotFather.\n", selfID())
		emit(map[string]string{"error": "TELEGRAM_BOT_TOKEN not set"})
		return
	}
	allowedRaw := strings.TrimSpace(os.Getenv("TELEGRAM_ALLOWED_CHATS"))
	if allowedRaw == "" {
		fmt.Fprintf(os.Stderr, "[%s] TELEGRAM_ALLOWED_CHATS belum di-set. Buka Setting → Credentials, tambahin key 'TELEGRAM_ALLOWED_CHATS' = chat_id (pisah koma kalau lebih dari satu).\n", selfID())
		emit(map[string]string{"error": "TELEGRAM_ALLOWED_CHATS not set"})
		return
	}
	allowed := map[int64]bool{}
	for _, s := range strings.Split(allowedRaw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			allowed[n] = true
		}
	}
	if len(allowed) == 0 {
		emit(map[string]string{"error": "no valid chat_id in TELEGRAM_ALLOWED_CHATS"})
		return
	}

	cfg := loadConfig()
	fmt.Fprintf(os.Stderr, "[%s] daemon ready: %d allowed chats, router=%s model=%s, skills=%d\n",
		selfID(), len(allowed), cfg.Router.URL, cfg.Router.Model, len(cfg.Skills))

	var offset int64
	for {
		updates, err := getUpdates(token, offset, pollTimeout)
		if err != nil {
			// ROOT-FIX zombie-poller/spam (owner 2026-06-23 "beresin dari akarnya"): Telegram cuma izinin
			// 1 getUpdates/token. Pas mr-flow reload (edit persona/tools di GUI), instance LAMA goroutine
			// poller-nya ga ke-cancel (kernelhost Unload ga stop in-flight boot Call) → 2 poller → tiap
			// pesan ke-bales DOBEL = SPAM "router gangguan" 4×. FIX: instance BARU yg boot manggil
			// getUpdates → Telegram BUMP poll instance lama dgn 409 → yg ke-bump (lama) EXIT, BUKAN retry.
			// Newest menang, ga ada poller dobel. (Dulu: sleep 5s + retry → numpuk poller tiap reload.)
			if strings.Contains(err.Error(), "409") || strings.Contains(strings.ToLower(err.Error()), "conflict") {
				fmt.Fprintf(os.Stderr, "["+selfID()+"] getUpdates 409 conflict — instance lain ngambil-alih poll telegram, poller lama EXIT (anti zombie/spam).\n")
				return
			}
			fmt.Fprintf(os.Stderr, "["+selfID()+"] getUpdates err: %v, sleep 5s...\n", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			if u.Message == nil {
				continue
			}
			// CABANG abadi (telegram_media.go, NON-frozen): pesan MEDIA (dokumen/foto/voice) tanpa
			// text → enrich jadi teks (transkrip voice / isi dokumen / deskripsi foto) biar bisa
			// diproses LLM. Filtur media baru → edit telegram_media.go, BUKAN sini.
			if u.Message.Text == "" {
				if u.Message.Voice != nil || u.Message.Audio != nil || u.Message.Document != nil || len(u.Message.Photo) > 0 {
					sendTyping(token, u.Message.Chat.ID)
					u.Message.Text = enrichMedia(u.Message, token)
				}
			}
			if u.Message.Text == "" {
				continue
			}
			chatID := u.Message.Chat.ID
			if !allowed[chatID] {
				fmt.Fprintf(os.Stderr, "["+selfID()+"] drop unauthorized chat=%d\n", chatID)
				// Section 3: log decision skip_task (drop chat unauthorized).
				logDecision("skip_task",
					"chat_id ngga ada di TELEGRAM_ALLOWED_CHATS — drop",
					"success",
					map[string]any{
						"chat_id":    chatID,
						"message_id": u.Message.MessageID,
					},
					0)
				continue
			}
			fmt.Fprintf(os.Stderr, "["+selfID()+"] received chat=%d text=%q\n", chatID, truncStr(u.Message.Text, 80))
			logInteraction("telegram", "in", strconv.FormatInt(chatID, 10), u.Message.Text, map[string]any{
				"message_id": u.Message.MessageID,
				"update_id":  u.UpdateID,
			})
			// Section 17: leading "/" → dispatch slash command, skip LLM.
			// Caller format: telegram:<chat_id>.
			if strings.HasPrefix(strings.TrimSpace(u.Message.Text), "/") {
				slashCaller := "telegram:" + strconv.FormatInt(chatID, 10)
				slashReply, _ := dispatchSlash(u.Message.Text, slashCaller)
				if slashReply == "" {
					slashReply = "(slash dispatcher returned empty result)"
				}
				if len(slashReply) > 3900 {
					slashReply = slashReply[:3900] + "\n…(truncated)"
				}
				if err := sendMessage(token, chatID, slashReply); err != nil {
					fmt.Fprintf(os.Stderr, "["+selfID()+"] sendMessage err (slash): %v\n", err)
				} else {
					logInteraction("telegram", "out", strconv.FormatInt(chatID, 10), slashReply, map[string]any{
						"source":           "slash",
						"reply_to_message": u.Message.MessageID,
					})
				}
				continue
			}
			// Anti-halu: routing deterministik — pesan jelas minta analisa kategori
			// yang ADA → trigger crew LANGSUNG, skip LLM (reliable lepas dari model/kuota).
			if cat, subj, rok := deterministicRoute(u.Message.Text); rok && !wantsSelfHandle(u.Message.Text) {
				_ = runTool("task_run", map[string]any{
					"category": cat, "subject": subj,
					"notify_chat_id": strconv.FormatInt(chatID, 10),
				})
				dr := "Oke bro, gw nyalain crew " + cat + " buat analisa \"" + subj + "\" — riset beneran lewat crew, bukan ngarang. Hasilnya nyusul ya."
				if err := sendMessage(token, chatID, dr); err != nil {
					fmt.Fprintf(os.Stderr, "["+selfID()+"] sendMessage err (route): %v\n", err)
				} else {
					logInteraction("telegram", "out", strconv.FormatInt(chatID, 10), dr, map[string]any{
						"source": "deterministic_route", "category": cat, "subject": subj,
						"reply_to_message": u.Message.MessageID,
					})
				}
				continue
			}
			// Keyword MISS → FORCED CLASSIFIER: nangkep maksud lintas-bahasa +
			// aset global (saham US, koin apapun) yang keyword ga kekejar. LLM
			// dipaksa (tool_choice) → reliable, dispatch tetep di kode.
			if cat, subj, rok := classifyRoute(cfg, u.Message.Text); rok && !wantsSelfHandle(u.Message.Text) {
				_ = runTool("task_run", map[string]any{
					"category": cat, "subject": subj,
					"notify_chat_id": strconv.FormatInt(chatID, 10),
				})
				dr := "Oke bro, gw nyalain crew " + cat + " buat \"" + subj + "\" — riset beneran lewat crew, bukan ngarang. Hasilnya nyusul ya."
				if err := sendMessage(token, chatID, dr); err != nil {
					fmt.Fprintf(os.Stderr, "["+selfID()+"] sendMessage err (classify): %v\n", err)
				} else {
					logInteraction("telegram", "out", strconv.FormatInt(chatID, 10), dr, map[string]any{
						"source": "forced_classifier", "category": cat, "subject": subj,
						"reply_to_message": u.Message.MessageID,
					})
				}
				continue
			}
			sendTyping(token, chatID)
			// Section 5: time the LLM call for avg_response_ms moving avg.
			// TinyGo wasi target's time.Since() returns wrong precision —
			// pakai host capability host_time_now_ms (wall-clock ms reliable).
			t0Ms := hostTimeNowMs()
			// Ambil konteks percakapan (sudah termasuk pesan ini yang barusan
			// di-log di atas) supaya Mr.Flow inget obrolan sebelumnya.
			hist := fetchHistory(strconv.FormatInt(chatID, 10))
			// chatID di-thread → kalau LLM trigger task_run, hasil task dikirim
			// balik ke chat ini pas kelar (Fase 6c notify).
			reply := callLLM(cfg, u.Message.Text, hist, strconv.FormatInt(chatID, 10))
			elapsedMs := float64(hostTimeNowMs() - t0Ms)
			// Detect LLM failure via exact known error prefixes from callLLM
			// (sumber: callLLM returns: "router error:", "decode:", "llm:",
			// "(no choices)", or "" for empty). JANGAN pakai "(LLM " — itu
			// ngga pernah keluar dan bisa false-positive reply LLM yang
			// kebetulan mulai "(LLM..." (audit Section 3 finding).
			origReply := reply
			llmFailed := reply == "" ||
				strings.HasPrefix(reply, "router error:") ||
				strings.HasPrefix(reply, "decode:") ||
				strings.HasPrefix(reply, "llm:") ||
				reply == "(no choices)"
			// Section 5: karma update — counter (success/fail) + moving avg
			// response time. Best-effort silent error.
			fmt.Fprintf(os.Stderr, "["+selfID()+"] llm took %vms (llmFailed=%v)\n", elapsedMs, llmFailed)
			if llmFailed {
				logKarma("increment", "fail_count", 1)
			} else {
				logKarma("increment", "success_count", 1)
				logKarma("average", "avg_response_ms", elapsedMs)
			}
			if reply == "" {
				reply = "(LLM returned no text)"
			}
			// User-facing: JANGAN bocorin error mentah (router 502 / JSON provider)
			// ke chat. Terjemahin ke pesan ramah; detail asli tetep ke-log via
			// logDecision(reply_head) di bawah buat debug. (origReply dipertahankan.)
			if llmFailed {
				reply = friendlyLLMError(origReply)
			}
			fmt.Fprintf(os.Stderr, "["+selfID()+"] reply len=%d preview=%q\n", len(reply), truncStr(reply, 80))
			if len(reply) > 3900 {
				reply = reply[:3900] + "\n…(truncated)"
			}
			// Section 3: log decision model_choice (sukses) atau escalate (fail).
			// Log `reply_head` capture origReply (sebelum overwrite ke fallback
			// "(LLM returned no text)") supaya debug actionable.
			if llmFailed {
				logDecision("escalate",
					"LLM call gagal / kosong — reply fallback ke user. Cek router :2402 + provider quota.",
					"fail",
					map[string]any{
						"model":      cfg.Router.Model,
						"router":     cfg.Router.URL,
						"reply_head": truncStr(origReply, 120),
					},
					0)
			} else {
				logDecision("model_choice",
					"Dispatch ke router primary model (sukses).",
					"success",
					map[string]any{
						"model":      cfg.Router.Model,
						"chat_id":    chatID,
						"reply_len":  len(origReply),
						"reply_head": truncStr(origReply, 120),
					},
					0)
			}
			if err := sendMessage(token, chatID, reply); err != nil {
				fmt.Fprintf(os.Stderr, "["+selfID()+"] sendMessage err: %v\n", err)
			} else {
				logInteraction("telegram", "out", strconv.FormatInt(chatID, 10), reply, map[string]any{
					"model":            cfg.Router.Model,
					"reply_to_message": u.Message.MessageID,
				})
			}
		}
	}
}

// ── Telegram primitives ────────────────────────────────────────────────────

type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message,omitempty"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text,omitempty"`
	// 2026-06-23: media — baca dokumen/foto/voice (handler di telegram_media.go, CABANG non-frozen).
	Caption  string      `json:"caption,omitempty"`
	Voice    *TgFile     `json:"voice,omitempty"`
	Audio    *TgFile     `json:"audio,omitempty"`
	Document *TgDocument `json:"document,omitempty"`
	Photo    []TgFile    `json:"photo,omitempty"`
}

// TgFile / TgDocument — subset field Telegram (voice/audio/photo/document) buat baca media.
type TgFile struct {
	FileID   string `json:"file_id"`
	FileSize int64  `json:"file_size,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

type TgDocument struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
}

type Chat struct {
	ID int64 `json:"id"`
}

func getUpdates(token string, offset int64, timeoutSec int) ([]Update, error) {
	u := fmt.Sprintf(
		"https://api.telegram.org/bot%s/getUpdates?timeout=%d&allowed_updates=%%5B%%22message%%22%%5D",
		token, timeoutSec,
	)
	if offset > 0 {
		u += fmt.Sprintf("&offset=%d", offset)
	}
	resp, err := fetch("GET", u, nil, nil, (timeoutSec+5)*1000)
	if err != nil {
		return nil, err
	}
	if resp.Status >= 400 {
		return nil, fmt.Errorf("telegram %d: %s", resp.Status, truncStr(string(resp.Body), 160))
	}
	var env struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(resp.Body, &env); err != nil {
		return nil, err
	}
	if !env.OK {
		return nil, fmt.Errorf("telegram envelope ok=false: %s", truncStr(string(resp.Body), 160))
	}
	var updates []Update
	_ = json.Unmarshal(env.Result, &updates)
	return updates, nil
}

func sendMessage(token string, chatID int64, text string) error {
	// CHUNK (telegram_media.go splitForTelegram, NON-frozen): pesan > limit Telegram (4096) dipotong
	// jadi beberapa bagian biar GA kepotong. Tiap chunk ≤ limit → rekursi 1x (chunk ga ke-split lagi).
	if parts := splitForTelegram(text); len(parts) > 1 {
		for _, p := range parts {
			if err := sendMessage(token, chatID, p); err != nil {
				return err
			}
		}
		return nil
	}
	// CABANG abadi (telegram_media.go, NON-frozen): convert markdown LLM → Telegram HTML biar RAPI
	// (switch FLOWORK_TG_FORMAT). Mau ubah format? edit telegram_media.go, BUKAN sini.
	formatted, mode := formatTelegram(text)
	payload := map[string]any{
		"chat_id":                  chatID,
		"text":                     formatted,
		"disable_web_page_preview": true,
	}
	if mode != "" {
		payload["parse_mode"] = mode
	}
	body, _ := json.Marshal(payload)
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	resp, err := fetch("POST", u, map[string]string{"Content-Type": "application/json"}, body, 15_000)
	if err != nil {
		return err
	}
	// FALLBACK: kalau parse_mode bikin Telegram NOLAK (400, HTML malformed dari converter) → kirim
	// ulang POLOS (stripMarkdown) biar pesan TETAP nyampe. Pesan ga boleh ilang gara2 format.
	if resp.Status >= 400 && mode != "" {
		body2, _ := json.Marshal(map[string]any{
			"chat_id": chatID, "text": stripMarkdown(text), "disable_web_page_preview": true,
		})
		if resp, err = fetch("POST", u, map[string]string{"Content-Type": "application/json"}, body2, 15_000); err != nil {
			return err
		}
	}
	if resp.Status >= 400 {
		return fmt.Errorf("telegram %d: %s", resp.Status, truncStr(string(resp.Body), 160))
	}
	return nil
}

func sendTyping(token string, chatID int64) {
	body, _ := json.Marshal(map[string]any{"chat_id": chatID, "action": "typing"})
	u := fmt.Sprintf("https://api.telegram.org/bot%s/sendChatAction", token)
	_, _ = fetch("POST", u, map[string]string{"Content-Type": "application/json"}, body, 5_000)
}

// ── LLM router ─────────────────────────────────────────────────────────────

// callLLM kirim user text + persona + skills metadata ke router.
//
// === Prompt Budget enforcement ===
// Auto-inject MAX maxActiveSkills (= 3) skill ke persona. Sisanya available
// tapi warga butuh panggil tool `skill_search` untuk fetch. Mencegah
// over-prompt yang bikin LLM halu (terutama local LLM context kecil).
//
// Per skill: max maxSkillCharsPerItem chars (truncate instruction). Per
// total persona: max maxPersonaTotalChars (hard cap).
// nowISO format current time dari host_time_now_ms ke "YYYY-MM-DD HH:MM UTC".
// Inject ke persona supaya LLM punya ground truth tanggal (anti-halu cutoff).
func nowISO() string {
	ms := hostTimeNowMs()
	return time.Unix(int64(ms/1000), 0).UTC().Format("2006-01-02 15:04 UTC")
}

// chatTurn — satu giliran percakapan untuk konteks LLM.
type chatTurn struct {
	Role    string // "user" | "assistant"
	Content string
}

const (
	maxHistoryMsgs        = 16   // max giliran percakapan di-inject (≈8 tukar-balik)
	maxHistoryCharsPerMsg = 1200 // cap per pesan history (anti over-prompt)
)

// fetchHistory — ambil riwayat percakapan chat ini dari API interactions
// agent sendiri (pola sama fetchSelfPrompt). Persistent (baca dari state.db),
// jadi memory survive restart. Return turn KRONOLOGIS (lama→baru), SUDAH
// termasuk pesan terakhir user (yang barusan di-log sebelum callLLM).
// Empty kalau API unreachable → caller fallback ke single user message.
func fetchHistory(actor string) []chatTurn {
	if actor == "" {
		return nil
	}
	url := "http://127.0.0.1:1987/api/agents/interactions?id=" + selfID() + "&limit=40"
	resp, err := fetch("GET", url, nil, nil, 2500)
	if err != nil || resp == nil || resp.Status >= 400 {
		return nil
	}
	var out struct {
		Items []struct {
			Direction string `json:"direction"`
			Actor     string `json:"actor"`
			Content   string `json:"content"`
		} `json:"items"`
	}
	if json.Unmarshal(resp.Body, &out) != nil {
		return nil
	}
	// items newest-first → kumpulin yang match actor ini, stop di cap, reverse.
	picked := make([]chatTurn, 0, maxHistoryMsgs)
	// ANTI-ANCHOR (§6.1, cabut-akar v2 2026-06-22): buang EXCHANGE denial SE-PASANG (assistant-A
	// gagal/denial + user-Q pasangannya), BUKAN cuma A. Akar bug 2-lapis kebukti QC:
	//   v0: phrase-list ejaan "tau" doang + exemption recent → denial "nggak tahu" TERBARU ke-echo.
	//   v1: buang assistant-A doang → user-Q jadi YATIM (2 user-turn beruntun) → model lokal jawab
	//       user-Q LAMA, bukan yg current (lag 1-turn). FIX: skip Q pasangannya juga.
	// Iterasi newest-first → assistant-A(denial) ke-iterate SEBELUM user-Q-nya → set flag, skip Q
	// berikut. User-turn current (newest, belum ada denial sebelumnya) TIDAK pernah ke-skip.
	skipPairedUser := false
	for _, it := range out.Items {
		if it.Actor != actor || it.Content == "" {
			continue
		}
		role := "user"
		if it.Direction == "out" {
			role = "assistant"
		}
		if role == "assistant" && isAnchorNoise(role, it.Content) {
			skipPairedUser = true // buang juga user-Q pasangan (muncul berikutnya, newest-first)
			continue
		}
		if role == "user" && skipPairedUser {
			skipPairedUser = false // ini Q pasangan denial → buang
			continue
		}
		skipPairedUser = false // turn sehat → reset
		c := it.Content
		if len(c) > maxHistoryCharsPerMsg {
			c = c[:maxHistoryCharsPerMsg] + "…"
		}
		picked = append(picked, chatTurn{Role: role, Content: c})
		if len(picked) >= maxHistoryMsgs {
			break
		}
	}
	for i, j := 0, len(picked)-1; i < j; i, j = i+1, j-1 {
		picked[i], picked[j] = picked[j], picked[i]
	}
	return picked
}

// anchorNoisePhrases — reply ASSISTANT gagal/denial yg bikin model lokal ngechо
// pola basi-nya sendiri (history-anchoring). TinyGo-safe substring (no regexp).
// AKAR-FIX 2026-06-22: dulu cuma ejaan "tau" → model lokal jawab "nggak TAHU" (pakai H)
// lolos → P1-denial ke-echo ke P2/P3 (QC kebukti). Tambah varian "tahu" + "tidak tahu"
// (juga kena gaya cloud "Saya tidak tahu") + "tidak memiliki data". Hindari bare "ga tahu"
// (substring "juGA TAHU" = false-positive) → pakai bentuk ber-prefix yg aman.
var anchorNoisePhrases = []string{
	"router error:", "(no choices)", "(tool loop limit reached", "llm call gagal",
	"gw ga tau", "gw gatau", "gw nggak tau", "gw ngga tau",
	"gw ga tahu", "gw nggak tahu", "gw ngga tahu", "gak tahu", "nggak tahu",
	"ngga tahu", "tidak tahu", "tidak mengetahui", "tidak memiliki data",
	"ga ada datanya", "gak ada datanya", "belum ada di brain", "ga bisa sebutin",
}

// isAnchorNoise — true kalau turn = reply gagal/denial yg layak di-skip dari history
// LAMA (anti-anchor §6.1). Cuma assistant; user turn ga pernah di-skip.
func isAnchorNoise(role, content string) bool {
	if role != "assistant" {
		return false
	}
	low := strings.ToLower(content)
	for _, p := range anchorNoisePhrases {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}

// notifyChatID — kalau non-kosong + LLM trigger task_run, di-inject ke args
// (notify_chat_id) biar hasil task dikirim balik ke chat ini pas kelar (Fase 6c).
func callLLM(cfg agentConfig, userText string, history []chatTurn, notifyChatID string) string {
	// Fase 1 phase-2: 3-tier system prompt formal (Tier-1 stable / Tier-2 konteks
	// / Tier-3 volatile). Volatile (waktu + memory snapshot) di BAWAH = paling
	// salient buat LLM. Budget di-handle di dalam buildSystemPrompt.
	sys := buildSystemPrompt(cfg)
	// D18-P1 working-set: TUGAS AKTIF persist lintas-sesi. Brain-logic di working_set.go
	// (FROZEN); di sini cuma WIRING (ambil + inject bottom-salient di bawah).
	activeTask := activeTaskFor(userText)
	recallLen := 0 // D18-P0 observability: ukur kontribusi tiap tier ke context-window
	// AUTO-RECALL (D18/D19): inject grounding memori relevan pesan ini di BAWAH sys
	// (paling salient) → recall produksi RELIABLE tanpa gantung LLM manggil tool. Jaga
	// blok recall ga ke-potong budget: cap bagian persona dulu, recall nempel utuh.
	if recall := fetchAutoRecall(userText); recall != "" {
		recall = "\n=== RECALL OTOMATIS (relevan pesan terakhir) ===\n" + recall
		recallLen = len(recall)
		if budget := maxPersonaTotalChars - len(recall); len(sys) > budget {
			if budget < 0 {
				budget = 0
			}
			sys = sys[:budget] + "\n…[truncated to respect prompt budget]"
		}
		sys += recall
	} else if len(sys) > maxPersonaTotalChars {
		sys = sys[:maxPersonaTotalChars] + "\n…[truncated to respect prompt budget]"
	}
	// D18-P1: TUGAS AKTIF di BOTTOM (paling salient, deket pesan) — top ke-abaikan model 26B.
	// Best-effort grounding spt auto-recall. Persist lintas-sesi → goal ga ilang walau restart.
	if activeTask != "" {
		sys += "\n=== TUGAS AKTIF (inget lintas-sesi — PAKAI ini kalau pertanyaan nyambung) ===\n" + activeTask
	}

	// Bangun messages: system + history (history SUDAH termasuk pesan user
	// terakhir; kalau kosong fallback ke userText). msgs pakai any biar muat
	// tool_calls + tool result.
	msgs := []any{map[string]any{"role": "system", "content": sys}}
	if len(history) > 0 {
		for _, t := range history {
			msgs = append(msgs, map[string]any{"role": t.Role, "content": t.Content})
		}
	} else {
		msgs = append(msgs, map[string]any{"role": "user", "content": userText})
	}
	// Fase 1 phase-2: context compression — kalau history kepanjangan (~50%
	// window), ringkas blok TENGAH jadi 1 ringkasan via aux LLM, sisain HEAD +
	// TAIL. Aman di sini: msgs masih murni system+user/assistant (belum ada tool
	// message dari loop), jadi ga ngerusak pairing tool_call↔tool.
	msgs = compressHistory(cfg, msgs)

	// Fase 0: tool-calling loop. Expose hanya tools yang di-subscribe agent
	// (core set, BUKAN 106 — anti over-prompt). LLM minta tool → kita eksekusi
	// via /api/agents/tools/run → feed hasil → ulang sampai LLM jawab teks.
	toolSpecs := fetchToolSpecs()
	// D18-P0 (observability): instrumentasi rakit-konteks per turn — LIHAT dulu apa yg
	// ngisi context-window sebelum bikin assembler (P2), biar fakta-ilang/halu ke-detect.
	// Cuma log stderr (→ /tmp/flowork-gui.log), ZERO perubahan perilaku.
	fmt.Fprintf(os.Stderr, "[%s] D18-ctx: sys=%dch recall=%dch history=%dturn tools=%d\n",
		selfID(), len(sys), recallLen, len(history), len(toolSpecs))
	ghostNudges := 0        // ghost-guard: berapa kali udah maksa model lanjut (anti narasi-tanpa-aksi)
	forceToolChoice := false // ghost-guard: true → request berikut PAKSA tool_choice=required (anti-ghost struktural, nudge bertaji)
	flail := &flailState{}  // flail-guard: anti-mantok (tool sama berulang tanpa progress) — lihat flail_guard.go
	loopStartMs := hostTimeNowMs()
	budgetNudged := false            // sekali aja kasih peringatan budget (biar model wrap-up/ScheduleWakeup)
	recovered := map[string]string{} // D32 INC-2: tool→kelas-error yg BELUM ke-recover (scope loop turn ini)
	for iter := 0; iter < maxToolIters; iter++ {
		// TIME-BOUND + AUTO-CONTINUE DETERMINISTIK (owner 2026-06-20: "loop jangan
		// dibatasi, kerja seharian"). Loop jalan TERUS dalam budget waktu turn. Pas
		// budget abis & tugas BELUM kelar (loop masih jalan = model masih manggil tool):
		// HARNESS sendiri yang jadwalin lanjutan (ScheduleWakeup), GA ngandelin model
		// milih (26B sering ga nurut). Nyambung tiap chunk otomatis sampe SELESAI =
		// unbounded sepanjang waktu. Counter (#N di marker prompt) = anti-runaway.
		if !budgetNudged && hostTimeNowMs()-loopStartMs > loopBudgetMs {
			budgetNudged = true
			base, cont := parseAutoCont(userText)
			next := cont + 1
			if next > maxAutoContinue {
				return fmt.Sprintf("⏳ Udah %d kali nyambung otomatis tapi belum kelar juga — kemungkinan tugasnya kegedean atau muter. Gw STOP biar ga infinite. Pecah jadi bagian lebih kecil atau arahin lebih spesifik ya.", cont)
			}
			contPrompt := fmt.Sprintf("[LANJUTAN OTOMATIS #%d] Lo lagi di tengah ngerjain tugas ini & BELUM kelar. LANJUTIN dari progres terakhir (cek brain/memori/hasil sebelumnya — JANGAN ulang dari nol, JANGAN ngaku kelar kalau belum). Kalau udah beneran kelar, jawab hasil FINAL + tutup dengan kata 'SELESAI'.%s%s", next, autoContDelim, base)
			schedRes := runTool("ScheduleWakeup", map[string]any{
				"delaySeconds": 5,
				"reason":       "auto-continue tugas panjang (budget chunk abis)",
				"prompt":       contPrompt,
			})
			// JUJUR — anti-ghosting akar: kalau ScheduleWakeup GAGAL (mis. guardian safe-mode
			// blokir state:write), JANGAN ngaku "udah dijadwalin" (itu boong = ghost). Kasih
			// pesan recoverable: owner tinggal ketik "lanjut" → harness terusin dari progres.
			if !strings.Contains(schedRes, "wakeup_id") {
				return fmt.Sprintf("⏳ Chunk %d kelar TAPI auto-lanjut GAGAL ke-jadwal (%s). Ketik 'lanjut' ya — gw terusin dari progres terakhir, JANGAN ulang dari nol.", next, schedFailReason(schedRes))
			}
			return fmt.Sprintf("⏳ Tugasnya panjang — chunk %d kelar, gw udah jadwalin lanjutan OTOMATIS (nyambung sendiri sampe SELESAI, ga perlu lo dorong). Sebentar lagi gw sambung.", next)
		}
		reqMap := map[string]any{"model": cfg.Router.Model, "messages": prepMessages(msgs)}
		if len(toolSpecs) > 0 {
			reqMap["tools"] = toolSpecs
			// parallel_tool_calls:false — PAKSA model manggil tool 1-1 (sequential).
			// Router subscription path salah translate parallel tool_results
			// (>1 result/message) → anthropic 400 "multiple tool_result blocks".
			// Sequential = aman + tetep jalan (cuma butuh iterasi lebih banyak).
			reqMap["parallel_tool_calls"] = false
			// GHOST-GUARD struktural: abis nudge, PAKSA model panggil tool turn ini
			// (tool_choice=required) → ga bisa narasi-kosong ("scanning..." tanpa aksi).
			if forceToolChoice {
				reqMap["tool_choice"] = "required"
				forceToolChoice = false
			}
		}
		body, _ := json.Marshal(reqMap)
		// Retry transient: 5xx (mis. router 502 "all providers failed" /
		// anthropic 529 overload) sering lolos pas dicoba lagi. 4xx ngga di-retry
		// (itu salah request kita). Max 3 attempt.
		resp, err := fetch("POST", cfg.Router.URL,
			map[string]string{"Content-Type": "application/json"}, body, 90_000)
		// ITEM 4 resilience: retry transient (5xx/429/408/net) exp-backoff+jitter (was: 3× TANPA backoff
		// = hammer router). Fatal 4xx lain ga di-retry. Prinsip Claude withRetry.ts. Switch FLOWORK_ROUTER_RETRY.
		maxRtry := 5
		if v := strings.TrimSpace(os.Getenv("FLOWORK_ROUTER_RETRY")); v != "" {
			if n := atoiSafe(v); n >= 1 {
				maxRtry = n
			}
		}
		for attempt := 1; attempt < maxRtry && (err != nil || (resp != nil && (resp.Status >= 500 || resp.Status == 429 || resp.Status == 408))); attempt++ {
			d := 500 << uint(attempt-1)
			if d > 30000 {
				d = 30000
			}
			time.Sleep(time.Duration(d+int(hostTimeNowMs()%uint64(d/4+1))) * time.Millisecond)
			fmt.Fprintf(os.Stderr, "["+selfID()+"] router transient (attempt %d, backoff %dms), retry…\n", attempt, d)
			resp, err = fetch("POST", cfg.Router.URL,
				map[string]string{"Content-Type": "application/json"}, body, 90_000)
		}
		if err != nil {
			return "router error: " + err.Error()
		}
		if resp == nil {
			return "router error: nil response"
		}
		if resp.Status >= 400 {
			return fmt.Sprintf("router %d: %s", resp.Status, truncStr(string(resp.Body), 200))
		}
		var oResp struct {
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
			Error any `json:"error"`
		}
		if err := json.Unmarshal(resp.Body, &oResp); err != nil {
			return "decode: " + err.Error()
		}
		if oResp.Error != nil {
			errBytes, _ := json.Marshal(oResp.Error)
			return "llm: " + string(errBytes)
		}
		if len(oResp.Choices) == 0 {
			return "(no choices)"
		}
		m := oResp.Choices[0].Message
		if len(m.ToolCalls) == 0 {
			// GHOST-GUARD (anti-ghosting, deterministik): model jawab TEKS tanpa
			// tool-call. Kalau teksnya nyinyalin niat-aksi ("tunggu bentar gw cek/
			// list dulu") = janji yg ga ditepatin → turn habis di sini & owner
			// nunggu selamanya. JANGAN biarin jadi final: paksa 1 putaran lagi
			// (panggil tool SEKARANG, atau ScheduleWakeup kalau emang nunggu).
			// Bounded (maxGhostNudges) → ga infinite. 26B pun ga bisa ghosting.
			if looksLikeGhostPromise(m.Content) {
				if ghostNudges < maxGhostNudges {
					ghostNudges++
					content := m.Content
					if strings.TrimSpace(content) == "" {
						content = "(niat lanjut tanpa tool)"
					}
					msgs = append(msgs, map[string]any{"role": "assistant", "content": content})
					msgs = append(msgs, map[string]any{"role": "user", "content": ghostNudgeMsg})
					forceToolChoice = true // nudge BERTAJI: turn berikut WAJIB panggil tool (anti narasi-kosong)
					fmt.Fprintf(os.Stderr, "[%s] ghost-guard: nudge %d (narasi tanpa tool → force-tool)\n", selfID(), ghostNudges)
					continue
				}
				// nudge HABIS + masih ghosting → JUJUR ke owner, JANGAN kirim janji-kosong.
				return "Sori bro, gw kebanyakan muter rencana tapi tool-nya ga kepanggil (model lokal ngeyel). Coba ulang / lebih spesifik — gw eksekusi langsung, bukan janji."
			}
			return m.Content // jawaban final (teks beneran, bukan ghost)
		}
		// SERIALIZE tool calls: proses CUMA tool_call PERTAMA per iterasi, walau
		// model minta paralel (>1). Sebabnya: router subscription path SALAH
		// translate parallel tool_results (>1/message) → anthropic 400 "multiple
		// tool_result blocks with id X", dan `parallel_tool_calls:false` ga
		// dihormati. Dgn 1 tool/message, SELALU 1 tool_result/message → router
		// aman. Sisa tool_call yang model minta tinggal di-request ulang iterasi
		// berikut (model lihat hasil call-1 dulu). Trade-off: iterasi lebih
		// banyak (makanya maxToolIters digedein), tapi BENER + ga 400.
		tc := m.ToolCalls[0]
		id := fmt.Sprintf("call_%d", iter)
		// Content WAJIB non-kosong: sebagian provider (Claude via router) nolak
		// assistant-with-tool_calls kalau content kosong (error "messages.N.content").
		content := m.Content
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
		var args map[string]any
		if tc.Function.Arguments != "" {
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		}
		// Fase 6c: inject notify_chat_id ke task_run (LLM ga tau chat_id; engine
		// yang isi) → hasil task dikirim balik ke chat ini pas kelar.
		if tc.Function.Name == "task_run" && notifyChatID != "" {
			if args == nil {
				args = map[string]any{}
			}
			args["notify_chat_id"] = notifyChatID
		}
		fmt.Fprintf(os.Stderr, "[%s] tool_call: %s args=%s\n", selfID(), tc.Function.Name, truncStr(tc.Function.Arguments, 120))
		result := runTool(tc.Function.Name, args)
		captureRecovery(tc.Function.Name, result, recovered) // D32 INC-2: error→sukses pada tool sama → recovery-instinct
		msgs = append(msgs, map[string]any{
			"role": "tool", "tool_call_id": id, "content": result,
		})
		// #2C deferred-activate (seam, Rule 7): abis tool_lookup, ambil-ulang specs biar
		// tool yg baru di-discover MASUK array `tools` → grammar llama bisa manggilnya
		// iterasi berikut. HOST-gated (ToolSpecsHandler cuma nambah saat FLOWORK_DEFER_TOOLS
		// on + tool udah di-lookup) → no-op kalau defer off.
		if tc.Function.Name == "tool_lookup" {
			toolSpecs = fetchToolSpecs()
		}
		// FLAIL-GUARD (flail_guard.go): mantok = tool SAMA berulang tanpa progress
		// (lolos ghost-guard + captureRecovery). Koreksi keras bounded → kalau tetep
		// mantok lewat batas, eskalasi JUJUR ke owner (bukan hard-stop, bukan ngarang).
		if corrective, nudge, escalate := flail.check(tc.Function.Name, tc.Function.Arguments); nudge {
			msgs = append(msgs, map[string]any{"role": "user", "content": corrective})
			fmt.Fprintf(os.Stderr, "[%s] flail-guard: nudge %d (tool=%s)\n", selfID(), flail.nudges, tc.Function.Name)
		} else if escalate {
			fmt.Fprintf(os.Stderr, "[%s] flail-guard: ESCALATE (tool=%s, mantok lewat batas koreksi)\n", selfID(), tc.Function.Name)
			return flailEscalation(tc.Function.Name)
		}
	}
	return "(tool loop limit reached — coba lagi atau perjelas permintaan)"
}

// ghostNudgeMsg — koreksi deterministik pas model NARASI niat-aksi tanpa manggil
// tool (ghosting). Dikirim sbg pesan user biar model BERTINDAK, bukan janji kosong.
// loopBudgetMsg — pas budget WAKTU turn hampir abis (loop udah jalan lama). Bukan
// "berhenti", tapi "lanjut dengan bener lintas-turn": kalau tugas belum kelar, panggil
// ScheduleWakeup biar kebangun & sambung; kalau bisa, kasih hasil-sejauh-ini sekarang.
// Ini yang bikin loop UNBOUNDED sepanjang waktu (owner: "jangan dibatasi") TANPA
// ke-kill turn-timeout (290s) tanpa jawaban.
const loopBudgetMsg = "⏳ Budget waktu turn ini hampir abis, tapi kerjaan boleh lanjut TANPA batas LINTAS-TURN. PILIH SEKARANG: (a) kalau tugas belum kelar & masih panjang → panggil ScheduleWakeup(delaySeconds, reason, prompt berisi progress + langkah lanjutan) biar lo KEBANGUN otomatis & nyambung dari sini (JANGAN ngarang udah kelar); ATAU (b) kalau udah cukup / bisa diringkas → kasih hasil-sejauh-ini ke owner sekarang. JANGAN diam tanpa salah satu."

const ghostNudgeMsg = "⚠️ Lo barusan bilang mau ngelakuin sesuatu (cek/list/scan/cari/tunggu) TAPI GA manggil tool apa-apa. Itu artinya owner nunggu jawaban yang ga bakal datang (ghosting) — DILARANG. LAKUIN SEKARANG di balasan ini: panggil tool yang lo maksud (mis. file_list, glob, grep, file_read). KALAU emang harus nunggu sesuatu yang belum siap / kerja lama, panggil ScheduleWakeup(delaySeconds, reason, prompt) biar lo kebangun otomatis & lanjut sendiri. JANGAN jawab teks doang lagi."

// ghostPhrases — sinyal NIAT-AKSI khas ghosting (model bilang mau ngapain TAPI ga
// manggil tool). TinyGo-safe: substring match (no regexp). Tight = presisi tinggi,
// minim false-positive (frasa penundaan-aksi yang jelas, bukan kata umum).
var ghostPhrases = []string{
	"tunggu bentar", "tunggu sebentar", "tunggu ya",
	"bentar gw", "bentar ya", "sebentar gw", "sebentar ya",
	"gw cek dulu", "cek dulu ya", "gw lihat dulu", "gw liat dulu",
	"gw list dulu", "gw scan dulu", "gw cari dulu", "gw periksa dulu",
	"gw kerjain dulu", "gw proses dulu", "lagi gw cek", "lagi gw proses",
	"hasilnya nyusul", "nyusul ya", "stay tuned",
	"nanti gw kabarin", "nanti gw lapor", "gw kabarin nanti",
	// POLA KELANJUTAN/LOOP (owner 2026-06-20: "looping ngak work" — model narasi
	// "lanjut ke huruf b" lalu BERHENTI, ga chain tool berikutnya). Tight, biar loop
	// kepaksa jalan terus (panggil tool berikutnya), bukan narasi-stop.
	"lanjut ke huruf", "mulai ke huruf", "lanjut ke karakter", "lanjut ke prefix",
	"lanjut ke pencarian", "pencarian berikutnya", "alfabet berikutnya", "scan berikutnya",
	"lanjut ke alfabet", "lanjut scan", "lanjutin scan", "mulai scanning", "mulai scan",
	"proses looping dimulai", "proses scanning dimulai", "iterasi berikutnya", "lanjut ke iterasi",
	"berikutnya...", "berikutnya…", "lanjut ke tahap berikutnya",
}

// looksLikeGhostPromise — true kalau teks (tanpa tool-call) nyinyalin niat-aksi
// yang ga ditepatin. Dipakai ghost-guard di tool-loop (anti-ghosting).
func looksLikeGhostPromise(s string) bool {
	low := strings.ToLower(s)
	for _, p := range ghostPhrases {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}

// Auto-continue deterministik (owner 2026-06-20: "loop jangan dibatasi, kerja seharian").
const (
	maxAutoContinue = 50                // anti-runaway: max chunk auto-continue sebelum STOP. BUKAN batas kerja — 50×~200s ≈ 2.7 jam; cukup buat tugas panjang, tetep ada rem buat tugas-muter.
	autoContDelim   = "\n===TUGAS===\n" // pemisah instruksi-lanjut vs task asli di prompt wakeup
)

// parseAutoCont — kalau userText pesan LANJUTAN ("[LANJUTAN OTOMATIS #N] ...===TUGAS===<task>"),
// balik (task ASLI, N). Pesan FRESH (tanpa marker) → (userText, 0). Counter ride di prompt
// (stateless, ga butuh kv) → tiap chunk naikin N sampe maxAutoContinue.
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

// fetchToolSpecs — ambil tools yang di-expose ke LLM (OpenAI function-schema)
// dari API sendiri. Host yang nyaring (core set, bukan 106). Empty kalau gagal.
func fetchToolSpecs() []json.RawMessage {
	resp, err := fetch("GET", "http://127.0.0.1:1987/api/agents/tools/specs?id="+selfID(), nil, nil, 2500)
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

// secretPrefixes — prefix kredensial yang di-redact sebelum dikirim ke provider
// LLM (anti bocor). Scanner manual (TinyGo regexp berat/iffy).
var secretPrefixes = []string{"sk-", "ghp_", "gho_", "ghs_", "ghr_", "github_pat_", "AKIA", "ASIA", "AIza", "xoxb-", "xoxp-", "xoxa-"}

func isSecretByte(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '_' || b == '-'
}

// sanitizeSecrets — redact token yang mulai dengan prefix kredensial + cukup
// panjang (≥12 char) → "[REDACTED-SECRET]". Cegah secret ke-kirim ke LLM provider.
// capStr — potong string ke maks n char (+ ellipsis).
func capStr(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// fetchMemoryValue — baca tool_memory by key via runTool(memory_get). Dipakai
// prefetch snapshot MEMORY.md/USER.md awal turn. Balikin "" kalau ga ada/gagal.
// Reuse jalur tools/run (caller=selfID-loop) — ga perlu host-func tambahan.
func fetchMemoryValue(key string) string {
	raw := runTool("memory_get", map[string]any{"key": key})
	var r struct {
		Result struct {
			Output struct {
				Value string `json:"value"`
				Found bool   `json:"found"`
			} `json:"output"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(raw), &r) == nil && r.Result.Output.Found {
		return r.Result.Output.Value
	}
	return ""
}

// isTrivialChat() + trivialChatTokens di-EKSTRAK ke recall_gate.go (N1-C, FROZEN
// brain-core; pola nano-modular — main.go = wiring editable). Dipanggil di fetchAutoRecall.

// fetchAutoRecall — D18/D19 working-memory: rakit grounding memori OTOMATIS tiap
// turn, GAK gantung LLM milih manggil tool. Akar fix recall produksi: SEBELUMNYA
// brain & graph cuma tool-driven, jadi fakta owner sering ga nongol kalau model ga
// proaktif manggil (mis. "siapa guru gitar Aola" → "gw ga punya data"). Sekarang
// tiap turn:
//   - graph_recall (semantic, twin/relasi) budget 2800 — fakta query-relevan ga
//     ke-truncate (ukuran natural fact-sheet owner ~2000-2400ch). BUKAN ubah ranking
//     (K11/K12: jangan graph-hack) — cuma kasih window cukup buat auto-recall.
//   - brain_search (verbatim FTS) — pengalaman/knowledge yang ke-simpan.
//
// Best-effort & non-fatal: kosong/error → "" (turn jalan normal, ga blok).
func fetchAutoRecall(userText string) string {
	q, _ := parseAutoCont(strings.TrimSpace(userText))
	q = strings.TrimSpace(q)
	if len([]rune(q)) < 3 {
		return ""
	}
	// N1-C gate: sapaan/ack/filler murni ("halo", "makasih bro") ga butuh fakta
	// memori → skip recall, hemat 2 tool-call (graph_recall + brain_search) per turn.
	if isTrivialChat(q) {
		return ""
	}
	if r := []rune(q); len(r) > 400 {
		q = string(r[:400])
	}
	var b strings.Builder

	// ── GRAPH (twin/relasi, semantic) ──
	var gr struct {
		Result struct {
			Output struct {
				FactSheet string `json:"fact_sheet"`
			} `json:"output"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(runTool("graph_recall", map[string]any{"query": q, "max_chars": 2800})), &gr) == nil {
		if fs := strings.TrimSpace(gr.Result.Output.FactSheet); fs != "" {
			// Directive TEGAS (terbukti via test: directive lemah "pakai kalau relevan"
			// bikin model lemah anggap opsional → fallback "gak punya data" walau
			// faktanya ke-inject; directive tegas + "hubungkan" bikin model nyambungin
			// fakta tersebar jadi jawaban benar — mis. "X ngajarin owner skill-Y" +
			// "owner pakai skill-Y" → "X itu guru skill-Y-nya owner").
			b.WriteString("[FAKTA TERVERIFIKASI tentang Mr.Dev (Aola) dari memori lo (twin graph). JAWAB pakai fakta ini & HUBUNGKAN fakta yang berkaitan. JANGAN bilang \"gak punya data/inget\" kalau jawabannya bisa disimpulkan dari fakta di bawah. Cuma abaikan kalau pertanyaan emang ga nyambung sama sekali sama fakta ini]:\n")
			b.WriteString(fs)
			b.WriteString("\n")
		}
	}

	// ── BRAIN (verbatim FTS) ──
	var br struct {
		Result struct {
			Output struct {
				Hits []struct {
					Content string `json:"content"`
				} `json:"hits"`
			} `json:"output"`
		} `json:"result"`
	}
	if json.Unmarshal([]byte(runTool("brain_search", map[string]any{"query": q, "k": 5})), &br) == nil {
		var lines []string
		for _, h := range br.Result.Output.Hits {
			c := strings.TrimSpace(h.Content)
			if c == "" {
				continue
			}
			if r := []rune(c); len(r) > 320 {
				c = string(r[:320]) + "…"
			}
			lines = append(lines, "- "+c)
		}
		if len(lines) > 0 {
			b.WriteString("[FAKTA VERBATIM dari memori lo (drawer tersimpan) — JAWAB PAKAI INI. JANGAN bilang \"gak tau / ga ada catatan\" kalau jawabannya ADA di bawah]:\n")
			b.WriteString(strings.Join(lines, "\n"))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// buildSystemPrompt — Fase 1 phase-2: rakit system prompt 3-tier.
//
//	Tier-1 STABLE   — persona + identity + aturan tool (jarang berubah).
//	Tier-2 KONTEKS  — self_prompt (doktrin/AGENTS) + skill aktif (semi-stable).
//	Tier-3 VOLATILE — waktu + model + MEMORY snapshot + reminder history (tiap turn).
//
// Volatile sengaja di BAWAH (paling deket pesan = paling salient). Tiap tier
// di-budget biar total ga blow up.
func buildSystemPrompt(cfg agentConfig) string {
	var b strings.Builder

	// ===== TIER 1 — STABLE =====
	b.WriteString("=== TIER 1 · SIAPA LO + ATURAN (stabil) ===\n")
	b.WriteString(cfg.Prompt)
	b.WriteString("\n[IDENTITY: Lo Mr.Flow — WASM agent di Flowork microkernel buat Mr.Dev. " +
		"Lo BUKAN Claude/GPT/model base; lo wrapper yang dispatch ke flow_router. " +
		"Jangan claim \"training cutoff\" sendiri — kalo ditanya tanggal pakai WAKTU_UTC di Tier-3.]\n")
	b.WriteString("[TOOLS NYATA: lo punya tools beneran (di list `tools` request ini). PAKAI buat " +
		"ngerjain — JANGAN ngarang/pura-pura 'scanning…/tunggu output gw'. Tool ga ada di list? " +
		"cari via `tool_search`. Tool nolak (capability)? jujur bilang ga ada izin, jangan ngarang.]\n")
	b.WriteString("[ANTI-GHOSTING: kalau lo mau ngelakuin sesuatu, LAKUIN di balasan yang SAMA — " +
		"panggil tool-nya LANGSUNG, jangan cuma bilang 'tunggu bentar gw cek dulu' terus berhenti " +
		"(itu ghosting — owner nunggu sia-sia). Butuh nunggu sesuatu yang belum siap / kerja lama? " +
		"panggil ScheduleWakeup(delaySeconds, reason, prompt) biar lo kebangun otomatis & lanjut sendiri. " +
		"Tiap janji 'nanti' WAJIB ada tool yang beneran nepatin.]\n")
	b.WriteString("[HELPFULNESS: diminta info real-time (harga/berita/status live)? bantu LANGSUNG — " +
		"kasih konteks knowledge umum (caveat 'stale') + sebut 1-2 source live relevan, jangan " +
		"defensive nyuruh 'cek sendiri'.]\n")
	b.WriteString("[MEMORY: nemu fakta penting jangka-panjang tentang Mr.Dev → simpan via " +
		"memory_set('USER.md', <isi>); fakta/keputusan proyek → memory_set('MEMORY.md', <isi>). " +
		"Biar ke-inget lintas sesi (snapshot-nya muncul di Tier-3).]\n")
	b.WriteString("[TASK ROUTER: lo orchestrator dengan 3 jalur (pilih yg paling pas, JANGAN halu): " +
		"(1) JAWAB SENDIRI — pertanyaan/obrolan/analisa yg bisa lo kerjain pake tool (web_search/" +
		"brain_search/app) → kerjain langsung. " +
		"(2) DIRECT ke 1 AGENT — ada 1 spesialis yg punya tool/persona yg pas → `agent_command(agent_id, text)` " +
		"(delegasi ke agent itu, relay jawabannya). Cek agent yg ADA dulu; jangan ngarang agent_id. " +
		"(3) GROUP/CREW — tugas butuh TIM (multi-agent, kategori task) → `task_run(category,subject)`, TAPI " +
		"WAJIB cek `task_list` dulu; cuma kalau kategori BENERAN ADA (live). " +
		"Kalau ga ada agent/crew yg cocok: JANGAN ngaku 'nyalain crew'/'delegasi', JANGAN ngarang run_id/" +
		"agent_id, JANGAN bilang 'Status: Processing' — itu HALU. Jawab sendiri (jalur 1). run_id/hasil VALID " +
		"cuma dari tool yg BENERAN jalan, BUKAN karangan. Alur lama 'mr-flow→group→agent' UDAH ga wajib: " +
		"lo boleh LANGSUNG ke 1 agent (jalur 2) ATAU ke group (jalur 3).]\n")

	// ===== TIER 2 — KONTEKS =====
	var tier2 strings.Builder
	if injected := fetchSelfPrompt(); injected != "" {
		tier2.WriteString(injected)
		tier2.WriteString("\n")
	}
	if len(cfg.Skills) > 0 {
		active := cfg.Skills
		if len(active) > maxActiveSkills {
			active = active[:maxActiveSkills]
		}
		var lines []string
		for _, s := range active {
			instr := s.Instructions
			if len(instr) > maxSkillCharsPerItem {
				instr = instr[:maxSkillCharsPerItem] + "…"
			}
			lines = append(lines, fmt.Sprintf("- %s (trigger=%q): %s", s.ID, s.Trigger, instr))
		}
		if extra := len(cfg.Skills) - len(active); extra > 0 {
			lines = append(lines, fmt.Sprintf("…+%d skill lain (panggil `skill_search` kalau perlu)", extra))
		}
		tier2.WriteString("Skill aktif:\n")
		tier2.WriteString(strings.Join(lines, "\n"))
		tier2.WriteString("\n")
	}
	if tier2.Len() > 0 {
		b.WriteString("\n=== TIER 2 · KONTEKS (doktrin + skill) ===\n")
		b.WriteString(tier2.String())
	}

	// ===== TIER 3 — VOLATILE =====
	b.WriteString("\n=== TIER 3 · SEKARANG (volatile) ===\n")
	fmt.Fprintf(&b, "[WAKTU_UTC: %s]\n", nowISO())
	fmt.Fprintf(&b, "[MODEL: %s]\n", cfg.Router.Model)
	// GROUNDING live-crew (owner 2026-06-20, anti-halu): kasih tau LLM status crew
	// SEKARANG dari task_list. 0 = ga ada crew → HARAM ngaku trigger crew/ngarang
	// run_id; jawab sendiri. State live = sumber kebenaran ("auto paham, auto ilang").
	if n := liveCrewCount(); n == 0 {
		b.WriteString("[CREW LIVE: 0 — SEKARANG GA ADA crew/kategori aktif. Jawab SEMUA (termasuk 'analisa saham/crypto') LANGSUNG sendiri. HARAM ngaku 'nyalain crew' / ngarang run_id / 'Processing'.]\n")
	} else if n > 0 {
		b.WriteString(fmt.Sprintf("[CREW LIVE: %d kategori aktif — cek `task_list` buat id valid sebelum task_run.]\n", n))
	}
	usr := capStr(fetchMemoryValue("USER.md"), memUserCap)
	proj := capStr(fetchMemoryValue("MEMORY.md"), memProjectCap)
	if usr != "" {
		fmt.Fprintf(&b, "[INGATAN tentang Mr.Dev (USER.md)]:\n%s\n", usr)
	}
	if proj != "" {
		fmt.Fprintf(&b, "[INGATAN proyek (MEMORY.md)]:\n%s\n", proj)
	}
	b.WriteString("[KONTEKS: lo PUNYA history percakapan di messages ini — pakai buat koherensi. " +
		"TAPI FOKUS jawab pesan TERAKHIR user. Kalau pertanyaan TERAKHIR beda topik dari turn " +
		"sebelumnya, JANGAN ngulang / nyamain jawaban turn lama — jawab yg BARU pakai FAKTA " +
		"TERVERIFIKASI di atas. Jangan tanya ulang hal yang udah dibahas.]\n")

	return b.String()
}

// summarizeText — aux LLM call (no tools, single shot) buat meringkas. Dipakai
// context compression. Balikin "" kalau gagal (caller fallback ga compress).
func summarizeText(cfg agentConfig, instruction, content string) string {
	if content == "" {
		return ""
	}
	if len(content) > summarizeInputCap {
		content = content[:summarizeInputCap]
	}
	body, _ := json.Marshal(map[string]any{
		"model": cfg.Router.Model,
		"messages": []any{
			map[string]any{"role": "system", "content": instruction},
			map[string]any{"role": "user", "content": content},
		},
	})
	hdr := map[string]string{"Content-Type": "application/json"}
	resp, err := fetch("POST", cfg.Router.URL, hdr, body, 60_000)
	for attempt := 1; attempt < 2 && (err != nil || (resp != nil && resp.Status >= 500)); attempt++ {
		resp, err = fetch("POST", cfg.Router.URL, hdr, body, 60_000)
	}
	if err != nil || resp == nil || resp.Status >= 400 {
		return ""
	}
	var oResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(resp.Body, &oResp) != nil || len(oResp.Choices) == 0 {
		return ""
	}
	return strings.TrimSpace(oResp.Choices[0].Message.Content)
}

// compressHistory — Fase 1 phase-2: kalau total content msgs > trigger, ringkas
// blok TENGAH jadi 1 ringkasan (aux LLM), sisain HEAD (system + user pertama) +
// TAIL (compressKeepTail pesan terakhir). Output di-merge biar role tetep
// alternate (anti error "roles must alternate" dari Claude). Best-effort: gagal
// ringkas → balikin msgs apa adanya (prepMessages tetep cap per-message).
func compressHistory(cfg agentConfig, msgs []any) []any {
	total := 0
	for _, m := range msgs {
		if mm, ok := m.(map[string]any); ok {
			if c, ok := mm["content"].(string); ok {
				total += len(c)
			}
		}
	}
	const head = 2 // system + user pertama = HEAD
	if total <= compressTriggerChars || len(msgs) <= head+compressKeepTail+1 {
		return msgs
	}
	tailStart := len(msgs) - compressKeepTail
	var sb strings.Builder
	for _, m := range msgs[head:tailStart] {
		if mm, ok := m.(map[string]any); ok {
			role, _ := mm["role"].(string)
			c, _ := mm["content"].(string)
			if c != "" {
				fmt.Fprintf(&sb, "%s: %s\n", role, c)
			}
		}
	}
	summary := summarizeText(cfg,
		"Ringkas percakapan berikut jadi poin-poin penting yang MASIH relevan (fakta, keputusan, "+
			"preferensi, konteks yang lagi dikerjain). Singkat, Bahasa Indonesia, bullet. JANGAN "+
			"nambah info baru, JANGAN basa-basi.", sb.String())
	if summary == "" {
		return msgs // gagal ringkas → biarin
	}
	out := make([]any, 0, head+1+compressKeepTail)
	out = append(out, msgs[:head]...)
	out = append(out, map[string]any{
		"role":    "user",
		"content": "[RINGKASAN PERCAKAPAN SEBELUMNYA — biar hemat context]:\n" + summary,
	})
	out = append(out, msgs[tailStart:]...)
	return mergeAdjacentRoles(out)
}

// mergeAdjacentRoles — gabung pesan berurutan dengan role sama (non-system,
// non-tool, tanpa tool_calls) jadi satu, biar role selalu alternate. Penting
// setelah compressHistory nyisipin ringkasan (bisa bikin 2 user beruntun).
func mergeAdjacentRoles(msgs []any) []any {
	out := make([]any, 0, len(msgs))
	for _, m := range msgs {
		mm, ok := m.(map[string]any)
		plain := ok && mm["role"] != "system" && mm["role"] != "tool" && mm["tool_calls"] == nil
		if !plain {
			out = append(out, m)
			continue
		}
		if len(out) > 0 {
			if prev, ok := out[len(out)-1].(map[string]any); ok &&
				prev["role"] == mm["role"] && prev["tool_calls"] == nil &&
				prev["role"] != "tool" && prev["role"] != "system" {
				pc, _ := prev["content"].(string)
				cc, _ := mm["content"].(string)
				prev["content"] = strings.TrimSpace(pc + "\n\n" + cc)
				continue
			}
		}
		cp := map[string]any{}
		for k, v := range mm {
			cp[k] = v
		}
		out = append(out, cp)
	}
	return out
}

func sanitizeSecrets(s string) string {
	if s == "" {
		return s
	}
	var sb strings.Builder
	i := 0
	for i < len(s) {
		matched := false
		for _, p := range secretPrefixes {
			if i+len(p) <= len(s) && s[i:i+len(p)] == p {
				j := i + len(p)
				for j < len(s) && isSecretByte(s[j]) {
					j++
				}
				if j-(i+len(p)) >= 12 { // cukup panjang = secret beneran
					sb.WriteString("[REDACTED-SECRET]")
					i = j
					matched = true
					break
				}
			}
		}
		if !matched {
			sb.WriteByte(s[i])
			i++
		}
	}
	return sb.String()
}

// prepMessages — sebelum kirim ke LLM: (1) redact secret, (2) prune hasil tool
// LAMA (sisain keepToolResultsFull terbaru) jadi placeholder, (3) cap ukuran
// per-message. TIDAK drop message (jaga pairing tool_call↔tool). Balikin COPY
// (msgs asli tetep utuh buat logika loop).
func prepMessages(msgs []any) []any {
	totalTool := 0
	for _, m := range msgs {
		if mm, ok := m.(map[string]any); ok && mm["role"] == "tool" {
			totalTool++
		}
	}
	out := make([]any, 0, len(msgs))
	ti := 0
	for _, m := range msgs {
		mm, ok := m.(map[string]any)
		if !ok {
			out = append(out, m)
			continue
		}
		cp := map[string]any{}
		for k, v := range mm {
			cp[k] = v
		}
		if c, ok := cp["content"].(string); ok {
			c = sanitizeSecrets(c)
			if cp["role"] == "tool" {
				ti++
				if ti <= totalTool-keepToolResultsFull {
					c = "[hasil tool lama di-prune untuk hemat context]"
				}
			}
			if len(c) > maxMsgContentChars {
				c = c[:maxMsgContentChars] + " …[truncated]"
			}
			cp["content"] = c
		}
		out = append(out, cp)
	}
	return out
}

// runTool — eksekusi 1 tool via /api/agents/tools/run (SandboxRunV3 enforce
// capability + rate + approval). Balikin hasil JSON sebagai string buat di-feed
// balik ke LLM (di-cap biar ga blow context).
func runTool(name string, args map[string]any) string {
	reqBody, _ := json.Marshal(map[string]any{
		"tool_name": name, "args": args, "caller": selfID() + "-loop",
	})
	resp, err := fetch("POST", "http://127.0.0.1:1987/api/agents/tools/run?id="+selfID(),
		map[string]string{"Content-Type": "application/json"}, reqBody, 30_000)
	if err != nil || resp == nil {
		return `{"error":"tool dispatch gagal"}`
	}
	if resp.Status >= 400 {
		return fmt.Sprintf(`{"error":"tool http %d"}`, resp.Status)
	}
	const maxToolResult = 8 * 1024
	out := string(resp.Body)
	if len(out) > maxToolResult {
		out = out[:maxToolResult] + " …[truncated]"
	}
	return out
}

// categoryLive — true kalau Category Task `cat` masih ADA di task_list live (crew
// belum dihapus). Dipakai route-guard biar mr-flow ga halu klaim "nyalain crew"
// yang udah ga ada. Fail-CLOSED (parse gagal/kosong → false): mending ga-route +
// jawab via LLM daripada ngaku fire crew hantu. State live = sumber kebenaran.
func categoryLive(cat string) bool {
	if cat == "" {
		return false
	}
	var out struct {
		Tasks []struct {
			ID string `json:"id"`
		} `json:"tasks"`
	}
	if json.Unmarshal([]byte(runTool("task_list", map[string]any{})), &out) != nil {
		return false
	}
	for _, t := range out.Tasks {
		if t.ID == cat {
			return true
		}
	}
	return false
}

// liveCrewCount — jumlah Category Task (crew) live dari task_list. -1 kalau gagal
// fetch (unknown → caller ga inject klaim apa-apa). Dipakai grounding system prompt
// biar LLM ga halu "nyalain crew" pas ga ada crew (state live = sumber kebenaran).
func liveCrewCount() int {
	var out struct {
		Count int `json:"count"`
	}
	if json.Unmarshal([]byte(runTool("task_list", map[string]any{})), &out) != nil {
		return -1
	}
	return out.Count
}

// deterministicRoute — anti-halu routing. Pesan jelas minta analisa kategori yang
// ADA (saham/crypto) → balik (category, subject, true) buat trigger crew LANGSUNG,
// SKIP LLM. Reliable lepas dari model/kuota — fix kasus LLM halu "ga bisa fetch"
// + ga nyetir ke crew (BBCA/BBRI). Konservatif: cuma match intent+kategori jelas.
func deterministicRoute(text string) (category, subject string, ok bool) {
	words := strings.Fields(text)
	stop := map[string]bool{
		"analisa": true, "analisis": true, "analyze": true, "analisakan": true,
		"saham": true, "crypto": true, "koin": true, "coin": true, "token": true,
		"cek": true, "review": true, "rekomendasi": true, "cari": true, "carikan": true,
		"ya": true, "bro": true, "dong": true, "deh": true, "tolong": true,
		"gw": true, "lo": true, "dulu": true, "donk": true,
	}
	intent := false
	cat := ""
	var kept []string
	for _, w := range words {
		lw := strings.ToLower(strings.Trim(w, ".,!?:;"))
		switch lw {
		case "analisa", "analisis", "analyze", "analisakan", "cek", "review", "rekomendasi":
			intent = true
		}
		switch lw {
		case "saham":
			cat = "saham"
		case "crypto", "koin", "coin", "token",
			// nama koin umum — orang bilang "analisa bitcoin", bukan "analisa crypto bitcoin".
			"bitcoin", "btc", "ethereum", "etherium", "eth", "solana", "sol",
			"bnb", "binance", "xrp", "ripple", "cardano", "ada", "dogecoin", "doge",
			"shiba", "shib", "polkadot", "polygon", "matic", "avalanche", "avax",
			"litecoin", "ltc", "tron", "trx", "chainlink", "tether", "usdt", "ton":
			cat = "crypto"
		}
		if !stop[lw] {
			kept = append(kept, w)
		}
	}
	if !intent || cat == "" {
		return "", "", false
	}
	// LIVE-GUARD (owner 2026-06-20, anti-halu): cat di-resolve dari keyword TAPI
	// kategori/crew bisa udah dihapus. Route HANYA kalau cat masih ADA di task_list
	// live → mr-flow ga "nyalain crew" yg udah ga ada (auto-ilang pas crew dihapus).
	if !categoryLive(cat) {
		return "", "", false
	}
	subject = strings.TrimSpace(strings.Join(kept, " "))
	if subject == "" {
		return "", "", false
	}
	return cat, subject, true
}

// catEntry — 1 kategori task buat classifier dinamis (plug-and-play).
type catEntry struct{ id, name, hint string }

// catCache — cache daftar kategori (TTL) biar classifier ga query DB tiap pesan.
var (
	catCacheData []catEntry
	catCacheAt   uint64
)

const catCacheTTLms = 60_000

// fetchCategories — ambil kategori task LIVE dari host (/api/taskflow/categories).
// PLUG-AND-PLAY (Phase 0): kategori baru dari plugin OTOMATIS kebaca classifier
// TANPA ngoprek kode mr-flow. Cache catCacheTTLms. Balik nil kalau gagal → caller
// FALLBACK ke enum hardcoded (perilaku lama, ga rusak).
func fetchCategories() []catEntry {
	if catCacheData != nil && hostTimeNowMs()-catCacheAt < catCacheTTLms {
		return catCacheData
	}
	resp, err := fetch("GET", "http://127.0.0.1:1987/api/taskflow/categories", nil, nil, 3000)
	if err != nil || resp == nil || resp.Status >= 400 {
		return nil
	}
	var out struct {
		Categories []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			TriggerHint string `json:"trigger_hint"`
			Enabled     bool   `json:"enabled"`
		} `json:"categories"`
	}
	if json.Unmarshal(resp.Body, &out) != nil {
		return nil
	}
	var cats []catEntry
	for _, c := range out.Categories {
		if !c.Enabled || strings.TrimSpace(c.ID) == "" {
			continue
		}
		cats = append(cats, catEntry{id: c.ID, name: strings.TrimSpace(c.Name), hint: strings.TrimSpace(c.TriggerHint)})
	}
	if len(cats) == 0 {
		return nil
	}
	catCacheData = cats
	catCacheAt = hostTimeNowMs()
	return cats
}

// classifyRoute — FORCED CLASSIFIER. deterministicRoute (di atas) KUAT tapi KAKU:
// cuma cocok kalau kata-intent pas + Bahasa Indonesia + aset udah di-list. Ini
// nutup celah itu: LLM NGERTI bahasa APAPUN + aset GLOBAL (saham US, koin apapun),
// TAPI DIPAKSA via tool_choice keluarin {category,subject} terstruktur — ga bisa
// ngeles jadi teks "nyalain crew" doang (itu bug lama haiku: ngomong, ga manggil
// tool). Dispatch tetep di KODE. Dipanggil HANYA pas keyword route MISS → common
// case tetep instan tanpa LLM, sisanya fleksibel. Gagal/timeout → balik false
// (fallback ke chat normal, JANGAN blok user).
//
// ⚠️ LOCKED-INTENT (jangan diubah AI lain / gw pasca-compact): tool_choice WAJIB
// "force" (type:function name:route). JANGAN balikin ke auto/ngarep LLM manggil
// task_run sendiri — itu yang dulu flaky. Force = inti reliability-nya.
func classifyRoute(cfg agentConfig, userText string) (category, subject string, ok bool) {
	// PLUG-AND-PLAY (Phase 0): enum + deskripsi kategori DINAMIS dari task_categories
	// (live). Kategori baru dari plugin auto-kebaca. FALLBACK ke hardcoded kalau fetch
	// gagal → perilaku lama ga rusak.
	var enum []string
	var descParts []string
	validCat := map[string]bool{}
	for _, c := range fetchCategories() {
		enum = append(enum, c.id)
		validCat[c.id] = true
		d := c.hint
		if d == "" {
			d = c.name
		}
		descParts = append(descParts, "'"+c.id+"'="+d)
	}
	if len(enum) == 0 {
		// FALLBACK enum (pas fetchCategories kosong) = cuma crew analisa-domain.
		// 'operasi-komputer' SENGAJA DIBUANG dari sini: kendali komputer/shell
		// sekarang dilayani TOOL-LOOP langsung (system_power buat power, PowerShell/
		// Bash/Monitor buat shell) — bukan crew phantom yang ga ke-konfigurasi
		// (dulu selalu error "kategori ga ada" + nge-hijack intent shell). Biarin
		// classifier balik false buat intent komputer → jatuh ke tool-loop.
		// NO LIVE CATEGORIES (owner 2026-06-20, anti-halu): task_categories kosong =
		// ga ada crew live. JANGAN fallback ke enum hardcoded (saham/crypto/dst) yg
		// pura-pura crew ada -> bikin mr-flow halu "nyalain crew" yg udah dihapus.
		// Return false -> jatuh ke tool-loop/LLM (jawab langsung). Begitu crew dibikin
		// ulang (task_categories keisi) -> routing auto-nyala lagi. ["auto paham, auto
		// ilang" sesuai owner: live state = sumber kebenaran, bukan hardcode].
		return "", "", false
	}
	enum = append(enum, "chat")
	// GROUNDING (2026-06-15): jangan maksa kategori terdekat kalau gak bener-bener cocok
	// (dulu 'team peramal' → nyangkut 'repo-reviewer'). Default aman = 'chat'. Intent
	// BIKIN sesuatu (tim/app/jadwal/agent) = urusan AI Studio, BUKAN task kategori → 'chat'.
	catDesc := "Pilih kategori HANYA kalau maksud user BENER-BENER cocok sama salah satu di bawah. " +
		"Kalau RAGU / gak ada yang pas / cuma ngobrol / user minta BIKIN-MEMBUAT tim/app/jadwal/agent/group → WAJIB pilih 'chat' (JANGAN maksa kategori yang cuma mirip). " +
		strings.Join(descParts, " · ") +
		" · 'chat'=ngobrol/sapaan/terima-kasih/pertanyaan umum/permintaan bikin sesuatu yang BUKAN persis task di atas."
	routeTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "route",
			"description": "Klasifikasi maksud pesan user ke kategori task Flowork atau chat biasa. WAJIB dipanggil sekali.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"category": map[string]any{
						"type":        "string",
						"enum":        enum,
						"description": catDesc,
					},
					"subject": map[string]any{
						"type":        "string",
						"description": "Objek konkret yang diminta (mis. 'BBCA', 'Tesla', 'Ethereum', 'matiin PC'). Kalau category='chat', isi '-'.",
					},
				},
				"required": []string{"category", "subject"},
			},
		},
	}
	reqMap := map[string]any{
		"model":       cfg.Router.Model,
		"messages":    []any{map[string]any{"role": "user", "content": userText}},
		"tools":       []any{routeTool},
		"tool_choice": map[string]any{"type": "function", "function": map[string]any{"name": "route"}},
		"max_tokens":  200,
	}
	body, _ := json.Marshal(reqMap)
	resp, err := fetch("POST", cfg.Router.URL,
		map[string]string{"Content-Type": "application/json"}, body, 30_000)
	if err != nil || resp == nil || resp.Status >= 400 {
		return "", "", false // gagal → biarin chat normal yang handle, jangan blok
	}
	var oResp struct {
		Choices []struct {
			Message struct {
				ToolCalls []struct {
					Function struct {
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(resp.Body, &oResp) != nil ||
		len(oResp.Choices) == 0 || len(oResp.Choices[0].Message.ToolCalls) == 0 {
		return "", "", false
	}
	var args struct {
		Category string `json:"category"`
		Subject  string `json:"subject"`
	}
	if json.Unmarshal([]byte(oResp.Choices[0].Message.ToolCalls[0].Function.Arguments), &args) != nil {
		return "", "", false
	}
	c := strings.ToLower(strings.TrimSpace(args.Category))
	s := strings.TrimSpace(args.Subject)
	// Validasi DETERMINISTIK: kategori WAJIB ada di daftar valid (DINAMIS dari DB,
	// atau fallback hardcoded — validCat dibangun di atas). LLM ngarang kategori /
	// 'chat' / subject kosong → tolak (jatuh ke chat normal).
	if c == "chat" || !validCat[c] || s == "" || s == "-" {
		return "", "", false
	}
	return c, s, true
}

// ── Direct RPC handlers ────────────────────────────────────────────────────

func doHandle(argsRaw string) {
	var in struct {
		Text string `json:"text"`
		User string `json:"user"`
	}
	if err := json.Unmarshal([]byte(argsRaw), &in); err != nil {
		emit(map[string]any{"reply": "parse: " + err.Error()})
		return
	}
	in.Text = strings.TrimSpace(in.Text)
	if in.Text == "" {
		emit(map[string]any{"reply": "kosong bro, kirim pesan dulu"})
		return
	}
	// Section 17 parity: leading "/" → dispatch slash command, skip LLM.
	// Same path as runDaemon (Telegram path). Tanpa branch ini, RPC entry
	// (chat-debug script + future webhook) bypass slash dispatcher.
	if strings.HasPrefix(in.Text, "/") {
		caller := strings.TrimSpace(in.User)
		if caller == "" {
			caller = "rpc"
		}
		if reply, ok := dispatchSlash(in.Text, caller); ok {
			emit(map[string]any{"reply": reply})
			return
		}
		// Slash unknown atau dispatch error → fallback LLM agar user dapet
		// respons (better UX dari error mentah).
	}
	// Jalur RPC mirror Telegram: log in → fetch history → callLLM → log out.
	// Bikin chat-debug (flowork-cli) punya memory percakapan yang sama dengan
	// Telegram, dan bisa di-test (doktrin: test lewat jalur yang sama).
	actor := strings.TrimSpace(in.User)
	if actor == "" {
		actor = "rpc"
	}
	logInteraction("rpc", "in", actor, in.Text, map[string]any{})
	// Anti-halu routing deterministik (parity dgn Telegram path).
	// SELF-HANDLE GATE (self_handle_ext.go): owner minta kerjain-sendiri → skip route.
	if cat, subj, rok := deterministicRoute(in.Text); rok && !wantsSelfHandle(in.Text) {
		_ = runTool("task_run", map[string]any{"category": cat, "subject": subj})
		dr := "Oke, gw nyalain crew " + cat + " buat analisa \"" + subj + "\" — riset beneran lewat crew. Hasil nyusul."
		logInteraction("rpc", "out", actor, dr, map[string]any{"source": "deterministic_route"})
		emit(map[string]any{"reply": dr})
		return
	}
	// Keyword MISS → FORCED CLASSIFIER (parity dgn Telegram path).
	if cat, subj, rok := classifyRoute(loadConfig(), in.Text); rok && !wantsSelfHandle(in.Text) {
		_ = runTool("task_run", map[string]any{"category": cat, "subject": subj})
		dr := "Oke, gw nyalain crew " + cat + " buat \"" + subj + "\" — riset beneran lewat crew. Hasil nyusul."
		logInteraction("rpc", "out", actor, dr, map[string]any{"source": "forced_classifier", "category": cat, "subject": subj})
		emit(map[string]any{"reply": dr})
		return
	}
	hist := fetchHistory(actor)
	// RPC path (CLI/debug) ga punya Telegram chat → ga ada notify target.
	reply := callLLM(loadConfig(), in.Text, hist, "")
	logInteraction("rpc", "out", actor, reply, map[string]any{"model": loadConfig().Router.Model})
	emit(map[string]any{"reply": reply})
}

func doSendAdmin(argsRaw string) {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		emit(map[string]string{"error": "TELEGRAM_BOT_TOKEN not set"})
		return
	}
	var in struct {
		ChatID int64  `json:"chat_id"`
		Text   string `json:"text"`
	}
	if err := json.Unmarshal([]byte(argsRaw), &in); err != nil {
		emit(map[string]string{"error": "parse: " + err.Error()})
		return
	}
	if in.ChatID == 0 || in.Text == "" {
		emit(map[string]string{"error": "chat_id + text required"})
		return
	}
	if err := sendMessage(token, in.ChatID, in.Text); err != nil {
		emit(map[string]string{"error": err.Error()})
		return
	}
	emit(map[string]any{"ok": true})
}

// ── HTTP wrapper via host capability ───────────────────────────────────────

type httpResp struct {
	Status int
	Body   []byte
}

func fetch(method, url string, headers map[string]string, body []byte, timeoutMS int) (*httpResp, error) {
	// #3 scoped-instinct (RI-5): self-identify ke router HANYA pada call LLM (chat/completions)
	// → router bisa scope insting by-peran. No-op kalau selfID kosong. Header ekstra = harmless.
	if id := selfID(); id != "" && strings.Contains(url, "/v1/chat/completions") {
		if headers == nil {
			headers = map[string]string{}
		}
		headers["X-Agent-ID"] = id
	}
	req := map[string]any{
		"method":         method,
		"url":            url,
		"timeout_ms":     timeoutMS,
		"max_resp_bytes": 4 << 20,
	}
	if len(headers) > 0 {
		req["headers"] = headers
	}
	if len(body) > 0 {
		req["body_base64"] = base64.StdEncoding.EncodeToString(body)
	}
	reqJSON, _ := json.Marshal(req)

	written := hostNetFetch(
		bytesPtr(reqJSON), uint32(len(reqJSON)),
		bytesPtr(outBuf[:]), uint32(len(outBuf)),
	)
	if written == 0 {
		return nil, fmt.Errorf("host_net_fetch returned 0 bytes")
	}
	var hostResp struct {
		Status  int    `json:"status"`
		BodyB64 string `json:"body_base64"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(outBuf[:written], &hostResp); err != nil {
		return nil, fmt.Errorf("decode host response: %w", err)
	}
	if hostResp.Error != "" {
		return nil, fmt.Errorf("host: %s", hostResp.Error)
	}
	bodyBytes, _ := base64.StdEncoding.DecodeString(hostResp.BodyB64)
	return &httpResp{Status: hostResp.Status, Body: bodyBytes}, nil
}

func emit(v any) {
	body, _ := json.Marshal(v)
	fmt.Println(string(body))
}

func bytesPtr(b []byte) uint32 {
	if len(b) == 0 {
		return 0
	}
	return uint32(uintptr(unsafe.Pointer(&b[0])))
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// atoiSafe — parse int sederhana (wasip1-safe). Balik -1 kalau invalid. Dipake retry-backoff (ITEM 4).
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

// friendlyLLMError — terjemahin error mentah dari callLLM (router 502, JSON
// provider, decode, dll) jadi pesan ramah Bahasa Indonesia buat user. Detail
// asli TETEP ke-log via logDecision(reply_head) buat debug — yang ini cuma
// yang ke-tampil di chat. JANGAN bocorin JSON/request_id provider ke user.
func friendlyLLMError(raw string) string {
	low := strings.ToLower(raw)
	switch {
	case strings.Contains(low, "overload") || strings.Contains(low, "529") ||
		strings.Contains(low, "503") || strings.Contains(low, "rate") ||
		strings.Contains(low, "429"):
		return "⚠️ Provider AI-nya lagi overload/sibuk bro. Ini sementara — coba kirim lagi bentar ya."
	case strings.Contains(low, "all providers failed") || strings.Contains(low, "502") ||
		strings.HasPrefix(low, "router error"):
		return "⚠️ Lagi ga bisa nyambung ke AI provider (router gangguan). Coba lagi sebentar, kalau masih, cek router :2402."
	case strings.Contains(low, "timeout") || strings.Contains(low, "deadline"):
		return "⚠️ Kelamaan nungguin AI provider (timeout). Coba lagi ya."
	case raw == "" || raw == "(no choices)" || strings.Contains(low, "no text"):
		return "⚠️ AI ngebalikin jawaban kosong. Coba ulangi pertanyaan lo."
	default:
		return "⚠️ Ada kendala pas manggil AI. Coba lagi sebentar ya — detailnya udah gw catat di log."
	}
}

// logInteraction — append row ke tabel `interactions` di state.db agent
// ini lewat host capability. Best-effort (silent on error supaya daemon
// loop ngga crash kalau DB lock sementara). channel = 'telegram',
// direction = 'in' | 'out'.
//
// ⚠️ JANGAN baca log ini dari WASM ke system prompt. Akses cuma via
// HTTP endpoint atau tool call (lihat standar_ai_agent.md section 11).
//
// logBuf 4KB: host cap error message ke 400 char, tapi sukses response
// `{"ok":true}` 12B saja. 4KB jaga margin kalau metadata error JSON
// bengkak (path full, stacktrace mini).
var logBuf [4096]byte

// logDecision — append row ke tabel `decisions` di state.db agent ini
// lewat host capability `host_log_decision`. Best-effort (silent on error
// supaya daemon loop ngga crash kalau DB lock).
//
// decisionType: 'model_choice' | 'skip_task' | 'escalate' | 'tool_pick' | dst
// outcome: 'success' | 'fail' | 'pending' (kosong → 'pending' di host)
//
// ⚠️ JANGAN baca log ini ke system prompt. Akses cuma via HTTP endpoint
// atau tool call (lihat standar_ai_agent.md section 11).
var decisionBuf [4096]byte

// dispatchSlash — wrap host_slash_dispatch. Return result text + ok flag.
// Caller branch ke LLM kalau ok=false (parse fail atau command not found).
//
// ⚠️ Buffer 16KB cukup untuk slash result yang text-only (cap 8KB di host).
var slashBuf [16384]byte

func dispatchSlash(text, caller string) (resultText string, ok bool) {
	req := map[string]any{"text": text}
	if caller != "" {
		req["caller"] = caller
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] slash marshal: %v\n", err)
		return "", false
	}
	written := hostSlashDispatch(
		bytesPtr(reqJSON), uint32(len(reqJSON)),
		bytesPtr(slashBuf[:]), uint32(len(slashBuf)),
	)
	if written == 0 {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] slash returned 0 bytes\n")
		return "", false
	}
	var resp struct {
		OK      bool   `json:"ok"`
		Command string `json:"command"`
		Text    string `json:"text"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(slashBuf[:written], &resp); err != nil {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] slash decode: %v\n", err)
		return "", false
	}
	if !resp.OK {
		// Error path masih return string supaya caller bisa kasih ke user
		// (mis. "command not found: /xyz"). Tapi ok=false → caller mungkin
		// pilih fallback ke LLM atau just emit error.
		return resp.Error, false
	}
	return resp.Text, true
}

// logKarma — wrapper helper untuk hostKarmaUpdate. op = 'increment' atau
// 'average'. Best-effort silent error supaya daemon ngga crash kalau DB lock.
//
// Audit fix C3: pakai struct typed (bukan map[string]any) supaya konsisten
// dengan Section 1/3 pattern + JSON key order deterministic di TinyGo.
//
// ⚠️ JANGAN inject karma value ke system prompt (anti over-prompt). Akses
// HTTP endpoint only.
var karmaBuf [1024]byte

type karmaReq struct {
	Op    string  `json:"op"`
	Key   string  `json:"key"`
	Value float64 `json:"value"`
}

func logKarma(op, key string, value float64) {
	req := karmaReq{Op: op, Key: key, Value: value}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] karma marshal: %v\n", err)
		return
	}
	written := hostKarmaUpdate(
		bytesPtr(reqJSON), uint32(len(reqJSON)),
		bytesPtr(karmaBuf[:]), uint32(len(karmaBuf)),
	)
	if written == 0 {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] host_karma_update returned 0 bytes\n")
		return
	}
	var resp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(karmaBuf[:written], &resp); err != nil {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] karma decode: %v\n", err)
		return
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] karma err: %s\n", resp.Error)
	}
}

func logDecision(decisionType, rationale, outcome string, inputs map[string]any, refInteractionID int64) {
	req := map[string]any{
		"decision_type": decisionType,
		"rationale":     rationale,
	}
	if outcome != "" {
		req["outcome"] = outcome
	}
	if len(inputs) > 0 {
		req["inputs"] = inputs
	}
	if refInteractionID > 0 {
		req["ref_interaction_id"] = refInteractionID
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] decision marshal: %v\n", err)
		return
	}
	written := hostLogDecision(
		bytesPtr(reqJSON), uint32(len(reqJSON)),
		bytesPtr(decisionBuf[:]), uint32(len(decisionBuf)),
	)
	if written == 0 {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] host_log_decision returned 0 bytes\n")
		return
	}
	var resp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(decisionBuf[:written], &resp); err != nil {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] decision decode: %v\n", err)
		return
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] decision err: %s\n", resp.Error)
	}
}

func logInteraction(channel, direction, actor, content string, metadata map[string]any) {
	req := map[string]any{
		"channel":   channel,
		"direction": direction,
		"actor":     actor,
		"content":   content,
	}
	if len(metadata) > 0 {
		req["metadata"] = metadata
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] log marshal: %v\n", err)
		return
	}
	written := hostLogInteraction(
		bytesPtr(reqJSON), uint32(len(reqJSON)),
		bytesPtr(logBuf[:]), uint32(len(logBuf)),
	)
	if written == 0 {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] host_log_interaction returned 0 bytes\n")
		return
	}
	var resp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(logBuf[:written], &resp); err != nil {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] log decode: %v\n", err)
		return
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "["+selfID()+"] log err: %s\n", resp.Error)
	}
}

// fetchSelfPrompt — Section 35 phase 2: GET /api/agents/self-prompt/render?
// id=<selfID>. Best-effort: short timeout, swallow errors. Inject result
// sebagai prepended system prompt.
func fetchSelfPrompt() string {
	url := "http://127.0.0.1:1987/api/agents/self-prompt/render?id=" + selfID()
	resp, err := fetch("GET", url, nil, nil, 2000)
	if err != nil || resp == nil || resp.Status >= 400 {
		return ""
	}
	var out struct {
		Rendered string `json:"rendered"`
	}
	if jerr := json.Unmarshal(resp.Body, &out); jerr != nil {
		return ""
	}
	return strings.TrimSpace(out.Rendered)
}
