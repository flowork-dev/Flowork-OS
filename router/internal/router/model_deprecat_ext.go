// model_deprecat_ext.go — F-G: auto-remap model DEPRECATED + warning (anti-404).
// 📄 Dok: FLowork_os/lock/model-deprecat.md
//
// NON-FROZEN sibling: wrap seam `applyInjectShaper` (jalan PERSIS sebelum
// resolveModel di dispatcher + dispatcher_stream) → req.Model yang nunjuk id
// model yang UDAH pensiun di-remap ke pengganti hidup + WARNING di log, BUKAN
// diteruskan mentah (yang bikin upstream balikin 404 → crash chain). File beku
// (dispatcher/modelresolve) NOL disentuh; chain composable (pola inject_budget_ext).
//
// KONSERVATIF: cuma id yang JELAS pensiun yang dipetakan; id Flowork yang hidup
// (claude-haiku-4-5, claude-opus-4-8, dst) TIDAK disentuh. Switch
// FLOWORK_MODEL_REMAP=0 buat matiin (escape-hatch). Nambah entri = 1 baris di map.
package router

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/flowork-os/flowork_Router/internal/store"
)

// deprecatedModelMap — id pensiun → pengganti hidup (prefix-insensitive; dicek
// setelah normalizeClaudeModel mencabut cc//anthropic//claude/). Sumber: model
// Anthropic/OpenAI yang di-retire. Kalau pengganti pun kelak pensiun, update sini.
var deprecatedModelMap = map[string]string{
	// Anthropic retired → generasi hidup terdekat (peran setara).
	"claude-1":                   "claude-haiku-4-5",
	"claude-instant-1":           "claude-haiku-4-5",
	"claude-instant-1.2":         "claude-haiku-4-5",
	"claude-2":                   "claude-opus-4-8",
	"claude-2.0":                 "claude-opus-4-8",
	"claude-2.1":                 "claude-opus-4-8",
	"claude-3-haiku-20240307":    "claude-haiku-4-5",
	"claude-3-sonnet-20240229":   "claude-sonnet-5",
	"claude-3-opus-20240229":     "claude-opus-4-8",
	"claude-3-opus-latest":       "claude-opus-4-8",
	"claude-3.5-sonnet":          "claude-sonnet-5",
	"claude-3-5-sonnet-20240620": "claude-sonnet-5",
	"claude-3-5-sonnet-20241022": "claude-sonnet-5",
	"claude-3-5-haiku-20241022":  "claude-haiku-4-5",
	// OpenAI retired → pengganti umum (kalau provider OpenAI dipakai).
	"gpt-4":         "gpt-4o",
	"gpt-4-32k":     "gpt-4o",
	"gpt-3.5-turbo": "gpt-4o-mini",
	"text-davinci-003": "gpt-4o-mini",
}

func modelRemapEnabled() bool {
	v := strings.TrimSpace(os.Getenv("FLOWORK_MODEL_REMAP"))
	return v == "" || v == "1" || strings.EqualFold(v, "true") // default ON (anti-404)
}

// remapDeprecatedModel — return (baru, true) kalau m pensiun; else (m, false).
func remapDeprecatedModel(m string) (string, bool) {
	key := strings.TrimSpace(m)
	for _, p := range []string{"cc/", "anthropic/", "claude/", "openai/"} {
		key = strings.TrimPrefix(key, p)
	}
	if repl, ok := deprecatedModelMap[key]; ok && repl != key {
		return repl, true
	}
	return m, false
}

func init() {
	prev := applyInjectShaper
	applyInjectShaper = func(ctx context.Context, req OpenAIRequest, settings *store.Settings) OpenAIRequest {
		req = prev(ctx, req, settings)
		if !modelRemapEnabled() {
			return req
		}
		if repl, remapped := remapDeprecatedModel(req.Model); remapped {
			log.Printf("flow_router ⚠️ model DEPRECATED %q → auto-remap %q (set FLOWORK_MODEL_REMAP=0 buat matiin; update config/alias-mu)", req.Model, repl)
			modelRemapWarn(req.Model, repl) // surface ke GUI (seam, default no-op)
			req.Model = repl
		}
		return req
	}
}

// modelRemapWarn — SEAM surface warning ke GUI/owner (default no-op). Diisi non-frozen
// (mis. badge GUI / notif). Dipisah biar remap tetep jalan walau surfacing belum ada.
var modelRemapWarn = func(oldModel, newModel string) {}
