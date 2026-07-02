# LLM call timeout + env-forward guest (2026-07-02)

## Masalah (akar, bukan gejala)
LLM lokal + tugas berat → `router error: host: fetch: Post ":2402/v1/chat/completions":
context deadline exceeded` berulang. Akar (3 lapis):
1. mr-flow hardcode timeout call LLM **90_000ms** (agentkit udah 240s — outlier) → model
   lokal yang mikir >90s/completion PASTI timeout.
2. Timeout ikut **di-retry** (re-POST) padahal engine lokal masih ngunyah request lama →
   antrian dobel di 1 GPU → makin macet.
3. Switch GUI `FLOWORK_*` yang call-site-nya DI DALAM guest WASM ga pernah nyampe guest
   (wazero cuma inject `i.env` hasil `buildAgentEnv`, bukan environ host; slot
   `kernelhost.EnvForwardKeys` cuma 1 & di-hard-assign frozen main.go).

## Arsitektur final
- **`agent/agents/mr-flow/main.go` (FROZEN, re-hash):** seam Pola B
  - `llmFetchTimeoutMs(loopStartMs)` — default **240_000ms**, override switch GUI
    `FLOWORK_LLM_TIMEOUT_MS` (min 15000), ADAPTIF: dipotong ke sisa jendela turn
    (290s − elapsed − 20s margin, floor 15s) biar ga nembus turn-timeout.
    Rantai atas: router→upstream 300s (`FLOWORK_ROUTER_HTTP_TIMEOUT`) · host netFetch
    cap 300s (>5menit di-reset 60s — JANGAN set switch >290000) · turn 290-300s.
  - `llmRetryable(err, resp)` — timeout client ("deadline exceeded"/"Client.Timeout")
    TIDAK retry; net error lain + 5xx/429/408 tetep retry (backoff+jitter,
    `FLOWORK_ROUTER_RETRY`).
  - Override lanjutan: sibling `_ext.go` baru boleh nge-wrap dua var seam ini (chain).
- **`agent/envfwd_seam.go` (FROZEN, baru):** papan colokan Pola A `RegisterEnvForward(f)`
  → `mergedEnvForwardKeys` (nilai `kernelhost.EnvForwardKeys`, di-assign main.go).
  Default cuma `connections.GlobalSecretEnvKeys` (dicolok di init() seam = perilaku lama;
  delete-test aman).
- **`agent/envfwd_switches_ext.go` (non-frozen, DELETABLE):** colok daftar key guest-side:
  `FLOWORK_LLM_TIMEOUT_MS · ROUTER_RETRY · PARALLEL_TOOLS · PROMPT_CACHE ·
  TOOL_RESULT_MAX · TG_FORMAT · TG_CHUNK · TG_MEDIA · SELF_HANDLE_PHRASES`.
  ⚠️ env guest kebentuk pas agent LOAD → ganti switch guest-side = restart stack.
- **`agent/main.go` (FROZEN, re-hash, 1 baris):** assign `EnvForwardKeys =
  mergedEnvForwardKeys` (dulu hard-assign connections).
- **`agent/internal/fwswitch/registry.go` (non-frozen):** +switch
  `FLOWORK_LLM_TIMEOUT_MS` (int, default `240000`, Router / Resilience).

## File & status
| File | Status |
|---|---|
| `agents/mr-flow/main.go` | FROZEN (re-hash 2026-07-02) + wasm rebuilt & staged |
| `main.go` (agent) | FROZEN (re-hash 2026-07-02) |
| `envfwd_seam.go` | FROZEN (entri baru) |
| `envfwd_switches_ext.go` | non-frozen, deletable |
| `internal/fwswitch/registry.go` | non-frozen |

QC 2026-07-02: build ✓ vet ✓ test ./... ✓ TestKernelFreeze ✓ (720 file) gembok ✓
delete-test (cabut ext → build OK) ✓ smoke bahasa-manusia via /api/chat ✓.
