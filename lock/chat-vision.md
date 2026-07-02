# 👁️ CHAT VISION — multimodal paste (Ctrl+V screenshot → LLM vision)

Status: LIVE (verified E2E 2026-07-02: gemini via Antigravity ✓ + claude via Anthropic ✓,
history multi-turn ✓, GUI render ✓). Feature files NON-FROZEN (deletable).

## Cara kerja (alur)
1. **GUI** `agent/web/js/chatui.js` — paste gambar di textarea Chat tab → chip preview
   (`.cu-attach`) → send `POST /api/chat/send {session_id, text, images:[dataURL]}`.
   Thumbnail dirender di bubble user + history. Maks 4 gambar/pesan.
2. **Agent** `agent/chat_sessions.go` (soft-lock) — terima `images`, validasi
   (`validChatImages`, data:image/*;base64, ≤8MB/gambar, ≤20MB total, body cap 16MB),
   persist via `AddChatMessageImages` → kolom `chat_message.images` (JSON array,
   ADDITIVE `ALTER TABLE ADD COLUMN`, `agent/internal/floworkdb/chatdb.go`).
3. **Architect brain** `agent/architect_chat.go` — turn user ber-gambar dibungkus
   `visionContent()` (`agent/chat_vision.go`) jadi **content-block JSON string**
   (konvensi sama dgn `preprocess_content.go`):
   `[{"type":"text","text":...},{"type":"image_url","image_url":{"url":"data:image/png;base64,..."}}]`
4. **Router** decode string itu jadi format provider lewat 2 SEAM (Pola B):
   - `internal/executors/antigravity.go` (FROZEN) → `AntigravityPartsHook` — diisi
     sibling `antigravity_vision_ext.go` → parts Gemini `{"inlineData":{mimeType,data}}`.
   - `internal/router/tools.go` (FROZEN) → `anthropicUserContentHook` — diisi sibling
     `vision_anthropic_ext.go` → block Anthropic `{"type":"image","source":{base64,...}}`.
   - Parser bersama: `internal/visionblocks/` (Parse ketat: semua entri harus dikenal,
     wajib ada ≥1 gambar data-URL; selain itu → bukan block → teks apa adanya).

## Guard konteks (agent/llm_context_safe.go)
- `msgContentLen` → `chatContentEstLen`: block vision dihitung teks + ~6400 char/gambar
  (≈1600 token), BUKAN panjang base64 — tanpa ini compactor motong base64 (gambar korup).
- `compactMessages` SKIP truncate content block-vision (`isVisionBlockContent`).
- `ctxBudgetTokens`: + case gemini → 180000.

## Switch
- Kill-switch router: `FLOWORK_VISION=0` → hook ga dipasang (balik text-only).
- Hapus sibling ext (`antigravity_vision_ext.go`, `vision_anthropic_ext.go`,
  `internal/visionblocks/`, `agent/chat_vision.go` + pemakainya) → seam nil → aman
  (delete-test router PASS 2026-07-02).

## Batas yang disengaja (bukan bug)
- Mode GROUP text-only: gambar ditandai teks `[📷 user melampirkan gambar]`
  (`buildGroupTranscript`) — vision penuh = jalur architect.
- Provider openai-compat/local (llama) ga di-decode → block string keliatan sbg teks
  JSON. Fallback-of-fallback, model lokal ga vision. Kalau nanti ada provider OpenAI
  vision: tambah seam serupa di marshal openai (pola sama, tiru 2 seam di atas).
- mr-flow WASM chat (/api/chat) bukan jalur ini (beda pipeline).

## File (status)
| File | Status |
|---|---|
| `router/internal/executors/antigravity.go` | FROZEN (hash updated 2026-07-02, seam nambah) |
| `router/internal/router/tools.go` | FROZEN (hash updated 2026-07-02, seam nambah) |
| `router/internal/executors/antigravity_vision_ext.go` | non-frozen, deletable |
| `router/internal/router/vision_anthropic_ext.go` | non-frozen, deletable |
| `router/internal/visionblocks/*` | non-frozen, deletable (+unit test) |
| `agent/chat_vision.go` | non-frozen, deletable |
| `agent/chat_sessions.go` / `architect_chat.go` / `llm_context_safe.go` / `internal/floworkdb/chatdb.go` | soft-lock (edit seizin roadmap owner 2026-07-02) |
| `agent/web/js/chatui.js` | soft-lock (embedded — ubah = rebuild binari) |
