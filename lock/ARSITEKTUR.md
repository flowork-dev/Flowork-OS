# ARSITEKTUR FLOWORK — Abadi, Evolusi-able, Anti-Tamper

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Master doc arsitektur. Detail per-subsistem: lock/{frozen-core,fwswitch,integrity,mesh}.md + lock/gui/*.

## Filosofi
Flowork didesain ABADI: bisa berevolusi (nambah fitur) bahkan setelah owner mati, TANPA hasil
evolusi merobohkan yang sudah jalan. Caranya: pisahkan tegas **INTI yang dibekukan** dari
**permukaan yang tumbuh**. AI masa depan menambah lewat permukaan; inti tak tersentuh.

## 4 LAPISAN

```
┌─ DATA (DB SQLite) ─────────── paling cair: provider, model, combo, skill, mcp, persona,
│                                knowledge. Tambah/copot/share-mesh BEBAS. NOL kode.
├─ SWITCH (fwswitch) ────────── perilaku via GUI: internal/fwswitch/registry.go (NON-frozen).
│                                ~80 FLOWORK_* key, lintas-proses, live 3 dtk tanpa restart.
├─ SEAM (registry + sibling) ── nambah KODE tanpa buka frozen: file *_ext.go / sibling baru +
│                                init(){ Register*(...) }. Mekanisme registry-nya FROZEN.
└─ FROZEN CORE (engine .go) ─── beku: chattr +i + hash di KERNEL_FREEZE.md (TestKernelFreeze).
                                 Immutable: tak bisa edit MAUPUN hapus tanpa unfreeze (sudo).
```

Aturan emas: **kebenaran di GUI, jangan hardcode**. Tiap fitur eksternal WAJIB ada di DATA/SWITCH/SEAM,
JANGAN dikunci di FROZEN CORE — biar tetap bisa ditambah & di-share mesh.

## SELF-SUFFICIENCY (terbukti)
Frozen core berdiri sendiri: **hapus SEMUA file non-frozen non-test non-GUI → `go build ./...`
tetap sukses** (diuji empiris di git worktree, 2026-06-27). Artinya permukaan (seam/switch/data)
murni ADITIF — copot = fitur mati mulus, core tak patah. Router: 409/510 .go frozen; sisa non-frozen
= seam/sibling/test/GUI. Detail: lock/frozen-core.md.

## SEAM yang tersedia (cara nambah TANPA buka frozen)
| Mau nambah | Caranya | Registry (frozen) |
|---|---|---|
| Endpoint HTTP | file `handlers_<x>_ext.go` + `RegisterExtraRoute` | routes_ext.go |
| Provider LLM (protokol ADA) | baris DB (GUI): format/baseURL/key/model | store/providers.go |
| Protokol/dialect BARU | file `internal/translator/{request,response}/<x>.go` + `translator.Register` | translator/registry.go |
| Provider media (embed/img/tts/stt) | file `internal/providers/<kat>/<x>.go` + `Register` | providers/<kat>/<kat>.go |
| Executor tool (cursor/codex/…) | file `internal/executors/<x>.go` + `Register` | executors/executor.go |
| Lapis filter mesh | file `internal/mesh/filter_<x>.go` + `RegisterMeshFilter` | mesh/filter_ext.go |
| Proyeksi graph | file + `RegisterGraphProjection` | brain/graph_extras_ext.go |
| Skill / persona / model / combo / MCP | baris DB (GUI/API) | store/* |
| Perilaku on/off/tuning | entri di fwswitch/registry.go → GUI Switch | fwswitch |

## INTEGRITY GATE (anti-mesh-jahat) — yang dikunci HANYA engine inti
Tiap node hitung **root-hash** dari semua file FROZEN router (lock/integrity.md). Kalau ada file
frozen berubah → node **tampered** → gate L0 (seam RegisterMeshFilter) **tolak semua pembelajaran
mesh masuk**. 

PENTING: yang masuk root-hash & dikunci = **ENGINE INTI (.go frozen)** saja. Fitur eksternal
(model, provider, skill, combo, knowledge = DATA/DB + sibling non-frozen) **TIDAK** masuk root-hash
→ menambah/men-share-nya via mesh **TIDAK** bikin node dianggap tampered. Jadi: inti kebal mesh-jahat,
permukaan tetap bebas tumbuh & berbagi. Switch: `FLOWORK_INTEGRITY_GATE`.

## BATAS CORE vs EKSTERNAL (prinsip pembeda)
- **CORE (boleh frozen):** engine routing/translator/dispatch, brain RAG (vecindex/semantic),
  mesh pipeline+signing+integrity, creds/auth, freeze-enforcement. Stabil, security-sensitif,
  jarang berubah, BUKAN sesuatu yang user/komunitas tambah.
- **EKSTERNAL (HARAM frozen-hardcode):** daftar model, koneksi provider, target proxy/tunnel,
  CLI tools, profil cloaking, MCP server, skill, combo, persona. Ini yang user tambah & share
  mesh → WAJIB DATA/SEAM. Kalau ke-hardcode di frozen = cacat (lihat lock/plug-and-play.md).

## VERIFIKASI (standar, jangan klaim tanpa ini)
`go build ./...` + `go vet` + `go test ./...` + `TestKernelFreeze` + Rule-9 (mr-flow `/api/chat`
bahasa-manusia) + QC GUI live + delete-test (hapus non-frozen → build OK). Lalu push 2 repo.
