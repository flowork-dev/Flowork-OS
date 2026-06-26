# Pricing

> Owner: Aola Sahidin (Mr.Dev) · github.com/flowork-os/Flowork-OS · floworkos.com
> Dok tab GUI Flowork Router (:2402). Standar freeze: lock/frozen-core.md.

## Fungsi
Rate card per model (USD per 1 juta token) buat estimasi cost dan milih kombinasi cost-optimal. Ada dua jalur: rate card sederhana per (provider, model) di tabel `pricing`, dan pricing rules berbasis tier di tabel `pricing_rules` (input/output per 1M USD, bisa di-enable/disable). Tab juga nyediain lookup harga single model.

## Endpoint (router/routes.go)
- `GET/POST/DELETE /api/pricing` → `pricingHandler` (registerProviderRoutes)
- `GET  /api/pricing/lookup` → `pricingLookupHandler` (registerProviderRoutes)
- `GET/POST /api/pricing/rules` → `PricingRulesHandler` (registerAuthRoutes, di `handlers_llm_policy.go`)
- Terkait (di file policy yang sama): `/api/pricing/calc` → `PricingCalcHandler`, `/api/pricing/log_call` → `PricingLogCallHandler`

## Logic / Alur
- **pricingHandler**:
  - GET: opsional filter `provider`, `ListPricing` → `{data, count}`.
  - POST: decode `store.Pricing`, wajib `provider`+`model`, `UpsertPricing`.
  - DELETE: query `provider`+`model` wajib, `DeletePricing`.
- **pricingLookupHandler (GET)**: wajib query `provider`+`model`, `GetPricing`; balik 404 `{error:"no rate card"}` kalau belum ada.
- **PricingRulesHandler**:
  - GET: query langsung `SELECT ... FROM pricing_rules ORDER BY id`, balikin `{items, count}` (field `enabled` jadi bool).
  - POST: decode body (`rule_name`, `provider`, `model`, `tier`, `input_per_1m_usd`, `output_per_1m_usd`, `notes`), wajib `provider`+`model`, default `tier="default"`, `INSERT ... ON CONFLICT(provider,model,tier) DO UPDATE` (upsert). Balik `{ok, id}`.

## File yang dilewati
- Handler rate card: `router/handlers_pricing.go`
- Handler pricing rules: `router/handlers_llm_policy.go` (`PricingRulesHandler`)
- Store: `router/internal/store/pricing.go` (`Pricing` struct, `ListPricing`, `GetPricing`, `LookupPricingByModel`, `UpsertPricing`, `DeletePricing`, `SeedDefaultPricing`)
- Migrasi tabel `pricing_rules`: `router/internal/store/llm_pricing_policy_migrations.go`
- Frontend: `router/web/static/index.html` (`data-tab="pricing"`)

## Teknologi
Go `net/http`, SQLite. Tabel `pricing` lewat helper store; tabel `pricing_rules` di-query SQL langsung dari handler (upsert pakai `ON CONFLICT(provider,model,tier)`). Struct `Pricing` punya field input/output/cache-read/cache-write USD per 1M + currency + source.

## Status freeze
FROZEN — `handlers_pricing.go`, `handlers_llm_policy.go`, dan `internal/store/pricing.go` punya header `⚠️ FROZEN`. Penambahan fitur lewat SEAM non-frozen + SWITCH (`internal/fwswitch/registry.go`). GUI `web/static/index.html` TIDAK frozen.
