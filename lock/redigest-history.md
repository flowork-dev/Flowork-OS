# RE-DIGEST HISTORY (#2B/#5) — chat lama → memori-permanen cognitive-graph

> Owner: Aola Sahidin (Mr.Dev) · 2026-06-26. Tujuan Flowork = INGAT jati-dirinya (proses, sejarah,
> cara bertahan hidup) biar abadi walau owner tiada. Chat lama = memori-hidup itu, BUKAN noise.

## RALAT FRAMING (penting buat AI selanjutnya)
AI (gw) sempat salah: framing "insting = aturan WHEN→THEN" doang → nyimpulin chat-backup = noise
(dominan test/debug/recall). **Owner RALAT: "chat itu insting — dia tahu proses, sejarah, cara
bertahan hidup."** Maksudnya LEBIH LUAS: chat = rekaman LIVED-EXPERIENCE Flowork (gimana dibangun,
bug yg dilewatin, recovery/survival, keputusan). Itu KNOWLEDGE buat cognitive-graph, bukan dibuang.

## AKAR yang dicabut
Live brain mr-flow ke-trim (AutoCompact) → tinggal 432 interaksi / 354 cog-node. Backup
`flowork-backup/mrflow-20260625-051043/chat-mrflow.json` (1479 interaksi) = sejarah lebih kaya yg
ILANG dari live. Tanpa re-import → memori sejarah/proses/survival hilang permanen.

## AKSI (2026-06-26)
1. Import **738 interaksi unik** dari backup ke `interactions` mr-flow (deleted_at=NULL → pending),
   dedup by-content vs live + skip 36 test-scaffolding murni (d32ok/sip42/scan-a-z — content-free).
   metadata `source=redigest-backup-20260625` (traceable). DB di-backup dulu (reversible).
2. `DigestPendingInteractions` (cron + manual `POST /api/agents/cognitive/digest?id=mr-flow`)
   konsolidasi pending → cognitive-graph (entity+relasi). Proses 100/cycle, background.
3. Filter #3 (error-edukasi L2) otomatis skip honest-fallback failure-output (noise asli) saat digest.
4. +1 insting gaya-bahasa (instinct_universal, owner ACC): "jelasin gaya anak SMA, to-the-point".

## HASIL (verified live)
cog_nodes 354 → **419** (+65, masih nambah) · digest_log 429 → 501 · pending 740 → 668 (konsumsi tiap
cycle). Background cron lanjutin sampe habis. Knowledge sejarah/proses/survival masuk graph permanen.

## CATATAN
- `state.db` mr-flow = **gitignored** (data privat owner, JANGAN commit). Re-digest = operasi DATA
  lokal, bukan kode. Reversible: backup DB di scratchpad sesi.
- Bukan "instinct WHEN→THEN" (itu cuma 1 sub-tipe). Ini KNOWLEDGE/memori-episodik→semantik.
- Kalau mau re-run backup lain: import (dedup) → trigger digest endpoint. Idempotent (digest_log dedup by id).
