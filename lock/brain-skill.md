# BRAIN — SKILL SUBSYSTEM (pecahan `lock/brain.md`)

> Bagian arsitektur memori Flowork. Overview + index: **`lock/brain.md`**. Topik file ini: 2 sistem skill (router :2402) + registry komunitas.
> ⚠️ KE-TRACK repo → NOL data personal owner.

---

## 14. SKILL SUBSYSTEM (router :2402) — 2 sistem + registry komunitas

Ada **2 konsep "skill" terpisah** di router (jangan ketuker):

**(A) Prompt-template skills** (`store.Skill`, disimpan di config DB `kv` prefix `skill:<uuid>`):
- Template prompt reusable: `{name(slug), description, systemPrompt, userTemplate (pakai {{var}}), defaultModel, temperature, maxTokens}`. Variabel **auto-extract** dari `{{...}}` (`extractVariables`).
- CRUD: `GET/POST /api/skills` + `PUT/DELETE /api/skills/<id>` — handler THIN di `handlers_resources.go` (non-frozen), logic di `internal/store/skills.go` (**FROZEN**: ListSkills/GetSkillByName/UpsertSkill/DeleteSkill/RenderSkillTemplate).
- Invoke: `GET /v1/skills/` (list) · `POST /v1/skills/<name>` `{variables,model?,temperature?,max_tokens?,stream?}` → render template (`{{var}}`→nilai) → susun pesan (system+user) → `DispatchChatCompletion` → balikin completion (`handlers_skills_invoke.go`, **FROZEN**). Model kosong → `defaultModel` skill (boleh `flowork-brain` lokal).
- GUI: tab **Skills** (create/list/run). NON-frozen (viz/form evolve).

**(B) SKILL.md behavioral skills** (markdown + frontmatter `---`, di `DynamicSkillsDir` = `~/.flow_router/skills`):
- Skill BAWAAN = embedded (`//go:embed`); skill AUTHORED = file `.md` di dir. Di-inject ke request lewat Brain config "Inject skills" (topK) — `internal/brain/skills.go` (**FROZEN**: DynamicSkillsDir/loadDynamicSkills/Skills/SelectSkills).

**REGISTRY KOMUNITAS** (`internal/skillregistry/registry.go` + `handlers_skillregistry.go`, **FROZEN**) — share SKILL.md via GitHub `flowork-os/flowork-skills` (override env `FLOWORK_SKILL_REGISTRY`):
- **3 GERBANG kepercayaan:** (1) **publish** butuh karma-gate (`skillpack.CanPublish`: endorsed-owner ATAU proven-lokal uses≥min & positif≥min) + **sign** provenance (`mesh.SignData`). (2) **pull** = verify **signature** (`mesh.VerifyData`) + verify **content** (`skillpack.VerifyContent`: tolak dangerous/injection) + frontmatter `---` wajib + anti path-traversal, SEBELUM import (registry = untrusted). Karma+pack di `handlers_skillpack.go` + `internal/skillpack/{skillpack,karma}.go`.
- Endpoint: `GET status/browse` (publik, **tanpa token**) · `POST pull?name=` · `POST publish?skill=` (loopback-only + `FLOWORK_GITHUB_TOKEN`). Browse/pull baca **fresh** via GitHub contents API (`Accept: raw`, anti-cache CDN). Publish = PUT `skills/<n>/<n>.fwskill` + merge `registry/index.json`.
- **Bukti e2e (2026-06-22):** endorse→publish(sign+push GitHub)→browse(count:1, fresh)→pull(verify-sig+verify-content+import) **round-trip LULUS**. Repo `flowork-os/flowork-skills` (public) seed skill `ringkas-terstruktur`.
