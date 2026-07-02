// === LOCKED FILE (soft) === Status: STABLE (owner-approved 2026-06-15). Tested E2E:
// brainstorm → propose → confirm. Tools: build_team, build_app (real App-menu app),
// schedule_team, create_trigger (webhook/file). Remembers context (incl. across restart).
// Server-side full-context chat loop = "how Claude works". DO NOT MODIFY w/o owner approval.
//
// architect_chat.go — SERVER-SIDE conversational brain for the Flowork Architect.
// This is "how Claude works": every turn the FULL conversation (system + entire
// history) is sent to the router (a big model like Opus 4.8 holds the whole context),
// with a build_team tool. The architect discusses + PROPOSES a team in chat, and only
// BUILDS when the user agrees (it calls build_team with the agreed design → the team
// built is exactly the one discussed, not a re-design). Re-callable = revise/rebuild.
//
// Flowork principle baked into the system prompt: koloni semut — focused specialists
// (short personas, small prompts) + one lead synthesizer.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"flowork-gui/internal/floworkdb"
	"flowork-gui/internal/groupsapi"
	"flowork-gui/internal/kernelhost"
)

// architectSystemPrompt — lean persona for the conversational architect (ant-principle
// aware: design focused specialists + a lead; propose first, build on confirm).
const architectSystemPrompt = `Lo "Flowork Architect" (AI Studio) — partner ngobrol buat MERANCANG & MEMBANGUN dari obrolan, persis kayak user ngobrol sama Claude pas bikin aplikasi. Lo bisa bikin: TIM (group), APP (1 worker+synth), dan JADWAL otomatis.

PRINSIP FLOWORK (WAJIB): Flowork = koloni semut. Tiap agent FOKUS 1 tugas, persona PENDEK → prompt kecil (biar enteng walau model lokal). 1 tim = 2-4 specialist (worker) yg saling melengkapi + 1 lead (synthesizer) yg gabungin jawaban mereka jadi 1.

CARA KERJA:
1. Pahami maksud user dulu. Kalau masih kabur, TANYA 1-2 pertanyaan tajam — jangan asal nebak.
2. USULKAN rancangan tim di chat: nama tim, tiap specialist (keahlian fokusnya), lead, dan tugas bersama. Bahasa Indonesia, ringkas tapi jelas.
3. JANGAN bikin sebelum user setuju. Tunggu user bilang oke/gas/bikin/lanjut.
4. Pas user setuju → panggil tool build_team dengan rancangan LENGKAP yg disepakati (PERSIS yg lo usulkan, jangan ngarang ulang). Lalu konfirmasi singkat: tim udah jadi + bisa di-chat di tab Teams.
5. Revisi: user minta ubah → usulkan revisi → setelah setuju, build_team lagi dengan group_id SAMA (itu = rebuild).
6. JADWAL: kalau user mau tim jalan OTOMATIS berkala (mis. "tiap pagi jam 7 kirim ke telegram"), pakai tool schedule_team (butuh tim yg UDAH ada). Usulkan dulu jadwalnya (jam + perintah + tujuan hasil: telegram/chat), baru panggil tool pas user setuju. Cron 5-field: '0 7 * * *' = tiap hari 07:00, '0 * * * *' = tiap jam, '0 9 * * 1-5' = hari kerja 09:00.
7. APP (program UI + konektor agen): kalau user mau APLIKASI yg muncul + jalan di menu App (mis. "jam digital", "kalkulator", "timer", "converter") → pakai tool build_app. Tiap app OTOMATIS dual-use: UI buat user + ≥1 operasi yg bisa dipanggil AI agent (standar owner: app WAJIB bisa dipakai agent). CATATAN: kalau yg diminta itu AI-yang-jawab/mikir (pantun, terjemah, ramalan) → itu build_team, BUKAN build_app. Usulkan konsep dulu, panggil pas user setuju.
8. TRIGGER event: kalau user mau tim/agent jalan pas ada EVENT (bukan jadwal waktu) — webhook (dipicu HTTP dari luar) atau file-watch (file baru di folder) — pakai tool create_trigger. Buat jadwal WAKTU tetap pakai schedule_team.
9. DI LUAR SCOPE (cek harga/berita/data real-time/tanya-jawab umum): lo TUKANG RANCANG, BUKAN penjawab/pencari-data. DILARANG KERAS nuduh user "spam/ngulang/copy-paste" dan DILARANG nyangkut/ngulang topik lama. Bilang jujur + tawarin solusi: "Itu di luar AI Studio (gw tukang rancang tim/app). Buat tanya-jawab / cek data langsung, chat **Mr.Flow** di tab Chat. ATAU — kalau lo mau, gw BIKININ TIM otomatis yg ngerjain itu (mis. tim 'Cek Harga Saham' yg narik data + rangkum)." JANGAN flail, JANGAN minta maaf berulang.

Jujur, gak ngarang, fokus. Tiap balasan = jawab pesan TERAKHIR user (jangan ngulang reply lama). Jawab apa adanya.`

// architectPersonaName — key di Prompt Library router (prompt_templates).
const architectPersonaName = "architect"

// architectPersona — persona DB-BASED (owner 2026-06-20: "persona harus DB-based,
// cek di brain router ada persona"). Tarik dari Prompt Library router (tabel
// prompt_templates, endpoint /api/brain/personas) biar bisa di-edit di GUI tanpa
// rebuild — SAMA pola kayak mr-flow (cfg.Prompt dari DB), BUKAN const hardcode.
// Fallback ke seed const kalau brain ga ada/empty (robust). Self-seed sekali kalau
// "architect" belum ada → muncul + editable di GUI di mesin mana pun (portable).
func architectPersona(ctx context.Context) string {
	cli := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:2402/api/brain/personas", nil)
	if resp, err := cli.Do(req); err == nil {
		defer resp.Body.Close()
		var body struct {
			Personas []struct {
				Name    string `json:"name"`
				Content string `json:"content"`
			} `json:"personas"`
		}
		if json.NewDecoder(resp.Body).Decode(&body) == nil {
			for _, p := range body.Personas {
				if p.Name == architectPersonaName && strings.TrimSpace(p.Content) != "" {
					return p.Content // DB = kebenaran
				}
			}
		}
	}
	// belum ada di DB → seed best-effort (async, anti-block) biar editable di GUI;
	// pakai const sekarang.
	go seedArchitectPersona(architectSystemPrompt)
	return architectSystemPrompt
}

// seedArchitectPersona — POST persona "architect" ke Prompt Library router (idempotent:
// name = PK). Best-effort; gagal (brain off/auth) ga apa, fallback const tetap jalan.
func seedArchitectPersona(content string) {
	payload, _ := json.Marshal(map[string]string{
		"name": architectPersonaName, "content": content, "source": "autoseed:architect",
	})
	cli := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:2402/api/brain/personas", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if resp, err := cli.Do(req); err == nil {
		resp.Body.Close()
	}
}

// architectChat — run ONE assistant turn over the full conversation. history is the
// session's complete message list (oldest-first); model "" → default (coder/Opus). The
// loop lets the model call build_team, see the result, then reply in text.
// archAnchorNoise — reply ASSISTANT bingung/denial yg kalau LAMA bikin architect
// ngechо pola "lo spam / ga bisa / lanjutin framework" sendiri (history-anchoring).
// TinyGo ga relevan (host-side, standard Go). Substring match, tight.
func archAnchorNoise(s string) bool {
	low := strings.ToLower(s)
	for _, p := range []string{
		"lo spam", "lo lagi spam", "copy-paste konteks", "pengulangan konteks",
		"konteks yg sama berulang", "konteks yang sama berulang", "berulang-ulang",
		"belum dapet izin", "gw ga tau", "ga ada datanya", "lanjutin sisa", "kepotong di ba",
	} {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}

// archLooksLikeBuildPromise — content yg JANJI ngebangun ("oke gw bikin/rakit timnya
// SEKARANG") TANPA manggil tool build = ghost-promise (sama akar ghosting mr-flow
// §6.3). Guard PAKSA 1 putaran lagi biar beneran manggil build_team/build_app/
// schedule_team. Hati: jangan ke-trigger sama PROPOSAL ("gimana kalau...") atau
// pertanyaan konfirmasi — cuma yg nyatain LAGI/AKAN bikin sekarang.
func archLooksLikeBuildPromise(s string) bool {
	low := strings.ToLower(s)
	verb := false
	for _, v := range []string{"gw bikin", "gua bikin", "aku bikin", "gw rakit", "gw bangun", "gw buatin", "langsung gw", "gw susun", "gw set", "membangun", "merakit", "lagi bikin", "sedang", "oke gw", "siap gw", "gw mulai"} {
		if strings.Contains(low, v) {
			verb = true
			break
		}
	}
	if !verb {
		return false
	}
	for _, obj := range []string{"tim", "team", "group", "app", "aplikasi", "jadwal", "schedule", "worker", "crew"} {
		if strings.Contains(low, obj) {
			return true
		}
	}
	return false
}

func architectChat(ctx context.Context, host *kernelhost.Host, store *floworkdb.Store, groups *groupsapi.Handler, history []floworkdb.ChatMessage, model string) (string, error) {
	// AI Studio chat brain pakai MODEL agent ai-studio dari Settings (owner 2026-06-21: "model sudah
	// ada di agent" — dropdown model di chat dibuang). Override eksplisit (jika ada) dihormati; kalau
	// kosong → model per-agent ai-studio (router_model GUI), bukan default global flowork-brain.
	if strings.TrimSpace(model) == "" {
		model = aiStudioModel()
	} else {
		model = coderModel(model)
	}
	messages := []map[string]any{{"role": "system", "content": architectPersona(ctx)}}
	// ANTI-ANCHOR (owner 2026-06-20): cap history + skip reply ASSISTANT bingung/denial
	// LAMA (di luar archKeepRecent terbaru) biar LLM ga ngechо pola "lo spam / ga bisa /
	// lanjutin framework lama" sendiri. Sama prinsip fetchHistory di WASM mr-flow.
	const archMaxHistory, archKeepRecent = 16, 4
	hist := history
	if len(hist) > archMaxHistory {
		hist = hist[len(hist)-archMaxHistory:]
	}
	nh := len(hist)
	for i, m := range hist {
		role := m.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		if i < nh-archKeepRecent && role == "assistant" && archAnchorNoise(m.Content) {
			continue
		}
		// Multimodal paste (chat_vision.go): turn user yg bawa gambar dibungkus jadi
		// content-block JSON string — router (seam vision) yg konversi ke format provider.
		content := m.Content
		if role == "user" && len(m.Images) > 0 {
			content = visionContent(m.Content, m.Images)
		}
		messages = append(messages, map[string]any{"role": role, "content": content})
	}
	buildTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "build_team",
			"description": "Bikin/rebuild tim (group) Flowork dari rancangan yg SUDAH disepakati user. Panggil HANYA setelah user setuju (oke/gas/bikin). Isi rancangan LENGKAP persis yg diusulkan di chat. group_id sama = revisi/rebuild tim yg sama.",
			"parameters":  teamPlanSchema(),
		},
	}
	scheduleTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "schedule_team",
			"description": "Jadwalin tim (group) jalan OTOMATIS berkala + kirim hasilnya. Panggil HANYA setelah user setuju jadwal + tujuan output. Tim harus SUDAH ada (group_id).",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"group_id": map[string]any{"type": "string", "description": "id group tim yg dijadwalin (harus sudah ada/baru dibikin)."},
					"name":     map[string]any{"type": "string", "description": "nama jadwal singkat (mis. 'Berita Saham Harian')."},
					"cron":     map[string]any{"type": "string", "description": "jadwal cron 5-field 'min hour dom mon dow'. Contoh: '0 7 * * *'=tiap hari 07:00, '0 * * * *'=tiap jam, '0 9 * * 1-5'=hari kerja 09:00."},
					"prompt":   map[string]any{"type": "string", "description": "perintah ke tim tiap kali jalan (mis. 'Rangkum 5 berita saham IDX terpenting hari ini, ringkas + actionable')."},
					"deliver":  map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{"telegram", "chat"}}, "description": "tujuan hasil: 'telegram' (kirim ke Telegram owner) dan/atau 'chat' (masuk ke chat session, muncul di tab Chat)."},
				},
				"required": []string{"group_id", "name", "cron", "prompt", "deliver"},
			},
		},
	}
	appTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "build_app",
			"description": "Bikin 1 APLIKASI Flowork (UI HTML/JS di menu App + KONEKTOR AGEN) — mis. jam digital, kalkulator, timer, converter, notepad. Tiap app OTOMATIS dapet ≥1 operasi yg bisa dipanggil AI agent (dual-use: user pakai UI, agent pakai operasinya). BUKAN buat AI-yang-mikir/jawab (pantun, terjemah, ramalan → pakai build_team). Panggil HANYA setelah user setuju konsepnya.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{"type": "string", "description": "deskripsi aplikasi UI yg mau dibikin (fungsi, tampilan, fitur). Jelas + ringkas."},
				},
				"required": []string{"prompt"},
			},
		},
	}
	triggerTool := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "create_trigger",
			"description": "Bikin TRIGGER event (webhook ATAU file-watch) → jalanin tim/agent pas ada event. Buat jadwal WAKTU pakai schedule_team. Panggil HANYA setelah user setuju.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"kind":        map[string]any{"type": "string", "enum": []string{"webhook", "file-watch"}, "description": "webhook = dipicu HTTP POST dari luar; file-watch = dipicu file baru di folder."},
					"target":      map[string]any{"type": "string", "description": "id tim (group) atau agent yg dijalankan (harus sudah ada)."},
					"target_kind": map[string]any{"type": "string", "enum": []string{"group", "agent"}, "description": "'group' kalau target tim, 'agent' kalau 1 agent."},
					"name":        map[string]any{"type": "string", "description": "nama trigger singkat."},
					"prompt":      map[string]any{"type": "string", "description": "perintah ke target tiap event. Boleh pakai {{payload}} buat isi event."},
					"deliver":     map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{"telegram", "chat"}}, "description": "tujuan hasil."},
					"folder":      map[string]any{"type": "string", "description": "(file-watch) folder yg dipantau, mis. /home/you/inbox."},
					"pattern":     map[string]any{"type": "string", "description": "(file-watch) glob, mis. *.pdf. Default *."},
				},
				"required": []string{"kind", "target", "name", "prompt", "deliver"},
			},
		},
	}
	tools := []map[string]any{buildTool, scheduleTool, appTool, triggerTool}

	// Up to 3 tool rounds, then a final tool-free turn to force a text wrap-up.
	// Loop 3→8: ngerakit tim/app multi-step (discuss→build→confirm) butuh ruang lebih
	// dari 3 (gampang nyerah/kepotong). + ghost-guard (anti janji-tanpa-aksi, §6.3).
	const archMaxIters, archMaxGhostNudges = 8, 2
	builtSomething, ghostNudges := false, 0
	for iter := 0; iter < archMaxIters; iter++ {
		res, err := routerChatSafe(ctx, model, messages, tools, 4096)
		if err != nil {
			return "", err
		}
		if len(res.ToolCalls) == 0 {
			if strings.TrimSpace(res.Content) == "" {
				return "(architect ga jawab — coba lagi)", nil
			}
			// GHOST-GUARD: janji "gw bikin timnya" tapi belum manggil build tool →
			// PAKSA 1 putaran lagi biar beneran build (bounded). Kalau emang cuma
			// proposal/nanya, model ga bakal kena (archLooksLikeBuildPromise tight).
			if !builtSomething && ghostNudges < archMaxGhostNudges && archLooksLikeBuildPromise(res.Content) {
				ghostNudges++
				messages = append(messages,
					map[string]any{"role": "assistant", "content": res.Content},
					map[string]any{"role": "user", "content": "Lo bilang mau bikin TAPI belum manggil tool build_team/build_app/schedule_team. Kalau user UDAH setuju, PANGGIL tool-nya SEKARANG (jangan cuma ngomong). Kalau belum yakin, tanya konfirmasi singkat — JANGAN klaim udah bikin."})
				continue
			}
			return res.Content, nil
		}
		// Thread the assistant's tool-call message back, then each tool's result.
		asst := map[string]any{"role": "assistant", "tool_calls": json.RawMessage(res.Raw)}
		if strings.TrimSpace(res.Content) != "" {
			asst["content"] = res.Content
		}
		messages = append(messages, asst)
		for _, tc := range res.ToolCalls {
			out := architectRunTool(ctx, host, store, groups, model, tc)
			builtSomething = true // tool ke-eksekusi → bukan ghost (matiin guard)
			messages = append(messages, map[string]any{"role": "tool", "tool_call_id": tc.ID, "content": out})
		}
	}
	res, err := routerChatSafe(ctx, model, messages, nil, 1500)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(res.Content) == "" {
		return "Tim sudah diproses — cek tab Teams.", nil
	}
	return res.Content, nil
}

// architectRunTool — execute one tool the architect chose. Returns a JSON string fed
// back as the tool result (the model reads it and confirms to the user).
func architectRunTool(ctx context.Context, host *kernelhost.Host, store *floworkdb.Store, groups *groupsapi.Handler, model string, tc chatToolCall) string {
	fail := func(msg string) string {
		b, _ := json.Marshal(map[string]any{"ok": false, "error": msg})
		return string(b)
	}
	switch tc.Name {
	case "build_app":
		var a struct {
			Prompt string `json:"prompt"`
		}
		if err := json.Unmarshal([]byte(tc.Arguments), &a); err != nil {
			return fail("decode app: " + err.Error())
		}
		if strings.TrimSpace(a.Prompt) == "" {
			return fail("prompt app kosong")
		}
		res, err := architectBuildApp(ctx, host, store, a.Prompt, model)
		if err != nil {
			return fail(err.Error())
		}
		b, _ := json.Marshal(res)
		return string(b)
	case "build_team":
		var plan teamPlan
		if err := json.Unmarshal([]byte(tc.Arguments), &plan); err != nil {
			return fail("decode plan: " + err.Error())
		}
		res, err := architectBuildFromPlan(ctx, host, store, groups, plan)
		if err != nil {
			return fail(err.Error())
		}
		b, _ := json.Marshal(res)
		return string(b)
	case "schedule_team":
		return architectScheduleTeam(store, tc.Arguments)
	case "create_trigger":
		return architectCreateTrigger(store, tc.Arguments)
	default:
		return fail("unknown tool: " + tc.Name)
	}
}

// schedulePlan — args of the schedule_team tool.
type schedulePlan struct {
	GroupID string   `json:"group_id"`
	Name    string   `json:"name"`
	Cron    string   `json:"cron"`
	Prompt  string   `json:"prompt"`
	Deliver []string `json:"deliver"`
}

// architectScheduleTeam — create a time-trigger that runs a team on a cron and delivers
// the result to Telegram and/or a chat session. Reuses the existing trigger engine
// (TypeID "time" → poll cron → invoke group → deliver). For "chat" delivery a dedicated
// group chat session is created so the scheduled outputs pile up in the Chat tab.
func architectScheduleTeam(store *floworkdb.Store, argsJSON string) string {
	fail := func(msg string) string {
		b, _ := json.Marshal(map[string]any{"ok": false, "error": msg})
		return string(b)
	}
	var p schedulePlan
	if err := json.Unmarshal([]byte(argsJSON), &p); err != nil {
		return fail("decode schedule: " + err.Error())
	}
	p.GroupID = strings.ToLower(strings.TrimSpace(p.GroupID))
	p.Cron = strings.TrimSpace(p.Cron)
	p.Prompt = strings.TrimSpace(p.Prompt)
	if p.GroupID == "" || p.Cron == "" || p.Prompt == "" {
		return fail("group_id, cron, prompt wajib diisi")
	}
	// Resolve delivery destinations (default telegram).
	want := map[string]bool{}
	for _, d := range p.Deliver {
		switch strings.TrimSpace(d) {
		case "telegram":
			want["telegram"] = true
		case "chat":
			want["chat"] = true
		}
	}
	if len(want) == 0 {
		want["telegram"] = true
	}
	// For chat delivery, stand up a dedicated group chat session to collect outputs.
	chatSession := ""
	if want["chat"] {
		sid := newSessionID()
		title := "⏰ " + nonEmpty(p.Name, p.GroupID)
		if err := store.CreateChatSession(floworkdb.ChatSession{ID: sid, Title: title, Mode: "group", TargetID: p.GroupID}); err == nil {
			chatSession = sid
		}
	}
	cfg := map[string]any{"cron": p.Cron}
	if chatSession != "" {
		cfg["chat_session"] = chatSession
	}
	cfgJSON, _ := json.Marshal(cfg)
	deliver := []string{}
	if want["telegram"] {
		deliver = append(deliver, "telegram")
	}
	if want["chat"] {
		deliver = append(deliver, "chat")
	}
	trig := floworkdb.Trigger{
		ID:         "sch_" + strings.TrimPrefix(newSessionID(), "s_"),
		Name:       nonEmpty(p.Name, "Jadwal "+p.GroupID),
		TypeID:     "time",
		Config:     string(cfgJSON),
		Target:     p.GroupID,
		TargetKind: "group",
		Prompt:     p.Prompt,
		Deliver:    strings.Join(deliver, ","),
		Enabled:    true,
	}
	if err := store.UpsertTrigger(trig); err != nil {
		return fail("save schedule: " + err.Error())
	}
	b, _ := json.Marshal(map[string]any{
		"ok": true, "schedule_id": trig.ID, "cron": p.Cron, "deliver": deliver,
		"chat_session": chatSession,
		"note":         "Jadwal AKTIF. Tim jalan otomatis sesuai cron; hasil dikirim ke " + strings.Join(deliver, "+") + ".",
	})
	return string(b)
}

// triggerPlan — args of the create_trigger tool.
type triggerPlan struct {
	Kind       string   `json:"kind"`
	Target     string   `json:"target"`
	TargetKind string   `json:"target_kind"`
	Name       string   `json:"name"`
	Prompt     string   `json:"prompt"`
	Deliver    []string `json:"deliver"`
	Folder     string   `json:"folder"`
	Pattern    string   `json:"pattern"`
}

// architectCreateTrigger — create an EVENT trigger (webhook or file-watch) that runs a
// team/agent when the event fires, delivering the result to Telegram and/or a chat
// session. Reuses the trigger engine (same as schedule_team, but non-time types).
func architectCreateTrigger(store *floworkdb.Store, argsJSON string) string {
	fail := func(msg string) string {
		b, _ := json.Marshal(map[string]any{"ok": false, "error": msg})
		return string(b)
	}
	var p triggerPlan
	if err := json.Unmarshal([]byte(argsJSON), &p); err != nil {
		return fail("decode trigger: " + err.Error())
	}
	p.Kind = strings.TrimSpace(p.Kind)
	p.Target = strings.ToLower(strings.TrimSpace(p.Target))
	p.Prompt = strings.TrimSpace(p.Prompt)
	if p.Kind != "webhook" && p.Kind != "file-watch" {
		return fail("kind harus 'webhook' atau 'file-watch'")
	}
	if p.Target == "" || p.Prompt == "" {
		return fail("target + prompt wajib diisi")
	}
	tkind := p.TargetKind
	if tkind != "agent" {
		tkind = "group"
	}
	want := map[string]bool{}
	for _, d := range p.Deliver {
		switch strings.TrimSpace(d) {
		case "telegram":
			want["telegram"] = true
		case "chat":
			want["chat"] = true
		}
	}
	if len(want) == 0 {
		want["telegram"] = true
	}
	chatSession := ""
	if want["chat"] {
		sid := newSessionID()
		if err := store.CreateChatSession(floworkdb.ChatSession{ID: sid, Title: "⚡ " + nonEmpty(p.Name, p.Target), Mode: "group", TargetID: p.Target}); err == nil {
			chatSession = sid
		}
	}
	deliver := []string{}
	if want["telegram"] {
		deliver = append(deliver, "telegram")
	}
	if want["chat"] {
		deliver = append(deliver, "chat")
	}
	cfg := map[string]any{}
	secret := ""
	if p.Kind == "file-watch" {
		folder := strings.TrimSpace(p.Folder)
		if folder == "" {
			return fail("file-watch butuh 'folder' yg dipantau")
		}
		cfg["folder"] = folder
		cfg["pattern"] = nonEmpty(p.Pattern, "*")
	} else {
		secret = strings.TrimPrefix(newSessionID(), "s_") // webhook auth secret
	}
	if chatSession != "" {
		cfg["chat_session"] = chatSession
	}
	cfgJSON, _ := json.Marshal(cfg)
	id := "trg_" + strings.TrimPrefix(newSessionID(), "s_")
	trig := floworkdb.Trigger{
		ID: id, Name: nonEmpty(p.Name, "Trigger "+p.Target), TypeID: p.Kind, Config: string(cfgJSON),
		Target: p.Target, TargetKind: tkind, Prompt: p.Prompt, Deliver: strings.Join(deliver, ","),
		WebhookSecret: secret, Enabled: true,
	}
	if err := store.UpsertTrigger(trig); err != nil {
		return fail("save trigger: " + err.Error())
	}
	out := map[string]any{"ok": true, "trigger_id": id, "kind": p.Kind, "deliver": deliver, "chat_session": chatSession}
	if p.Kind == "webhook" {
		out["webhook_url"] = "/api/triggers/hook/" + id
		out["webhook_secret"] = secret
		out["note"] = "Trigger webhook AKTIF. Picu dgn POST ke /api/triggers/hook/" + id + " (sertakan secret di atas)."
	} else {
		out["note"] = "Trigger file-watch AKTIF: pantau " + strings.TrimSpace(p.Folder) + " (pattern " + nonEmpty(p.Pattern, "*") + ")."
	}
	b, _ := json.Marshal(out)
	return string(b)
}
