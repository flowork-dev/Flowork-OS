# SKILL CENTRAL — propagasi skill dari Router ke agent

> Goal owner: "skill jadi CENTRAL di router" — edit skill sekali di router, agent ikut update
> (gak re-import manual). Owner: Mr.Dev.

## MASALAH (akar)
"Browse Router Catalog" COPY body skill ke `skills[]` agent saat import → kalau skill di router
DIUBAH, copy di agent BASI (drift). Gak ada propagasi.

## FIX (jalur AMAN — GUI non-frozen, gak bedah kernel/runtime sakral)
Tombol **"🔄 Sync from Router"** di section Skill (agent setting GUI, `agent/web/tabs/agents.js`):
loop `skills[]`, tiap skill yang ID-nya ADA di Router Catalog → re-pull body terbaru via endpoint
proxy yang SUDAH ada (`GET /api/agents/router-skills/get?id=<agent>&name=<skill>`) → update
`instructions` kalau beda. Skill lokal (gak ada di router) di-SKIP. Owner pencet Save buat persist.
Efek: edit skill di router → klik Sync → agent kepakai versi baru = CENTRAL (propagasi).

## KENAPA BUKAN reference-model penuh (sekarang)
Reference murni (agent simpan ref, resolve live tiap run) butuh edit file PALING SAKRAL & FROZEN:
`kernelhost.go` (build FLOWORK_AGENT_CONFIG) atau `mr-flow/main.go` (runtime), plus seluruh skill
subsystem router (handlers_brain_skills/store/skills — semua frozen). Blast-radius tinggi → ditunda
biar gak ngerusak inti. Versi tombol ini = 80% value, ~0% risiko (cuma GUI + endpoint yg ada).

## SISA (opus_roadmap.md Bagian 2)
- Auto-sync periodik (ticker) — sekarang manual (tombol). 
- Reference-model penuh + version/hash + GUI link-vs-copy (S0-S3) = butuh buka skill-core frozen
  (sesi owner-attended fokus).

## FILE
- `agent/web/tabs/agents.js` — tombol #cf-skills-sync-router + handler (non-frozen GUI).
- Proxy dipakai (frozen, gak diubah): `agent/internal/agentmgr/router_skills.go` GET.

## UPDATE 2026-06-26 — AUTO-SYNC (reference-model praktis SELESAI)
`feature_autosync.go` (NON-frozen, seam RegisterFeature): loop `skillResyncLoop` tiap interval
re-pull skill ter-link dari Router → update agent OTOMATIS (gak cuma tombol manual). Edit skill di
router → NYEBAR ke agent. Switch GUI: `FLOWORK_SKILL_AUTOSYNC` (default ON) + `_MIN` (default 30).
Reference-model PURE (zero-copy resolve live di kernelhost frozen) tetap parked (sakral); auto-sync
ini nutup goal praktisnya.
