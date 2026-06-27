// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// SEAM (NON-frozen, sibling, DELETABLE). Extractor format-muntah TAMBAHAN buat
// recoverTextToolCalls (toolcall_recover_ext.go, frozen) — didaftar lewat RegisterToolcallExtractor
// (POLA A). Tujuan: model LOKAL (fallback pas provider 429) sering ngeluarin tool-call
// BUKAN sebagai <tool_call> tag tapi: ```json fence / JSON telanjang / build_app({...}).
// Tanpa ini, build dari model lokal BOCOR jadi teks. Hapus file ini → balik ke perilaku
// lama (cuma <tool_call> tag), build tetap jalan (self-sufficient).
//
// PRINSIP ANTI-FALSE-POSITIVE: tiap extractor KETAT — cuma nyala kalau yakin itu tool-call
// (JSON wajib ber-shape {name, arguments|parameters}; func-syntax wajib WHOLE-content).
// Biar balasan teks/kode normal GA salah dijadiin tool-call.
package router

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	// ```json {…}``` (label opsional: json/tool/tool_call). Non-greedy biar multi-fence aman.
	muntahFenceRe = regexp.MustCompile("(?s)```(?:json|tool|tool_call|tool_use)?\\s*(\\{.*?\\})\\s*```")
	// SELURUH content = `nama({...})` (mis. build_app({...})). Whole-match → anti code-in-prosa.
	muntahFuncRe = regexp.MustCompile(`(?s)^\s*([a-zA-Z_][a-zA-Z0-9_]{2,48})\s*\(\s*(\{.*\})\s*\)\s*$`)
)

// strictToolJSON — terima HANYA kalau JSON ber-shape {"name":<str non-kosong>, "arguments"|
// "parameters":<object>}. Ini kunci anti-false-positive: data JSON biasa (tanpa name+args) DITOLAK.
func strictToolJSON(raw string) (name, args string, ok bool) {
	var m map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &m) != nil {
		return "", "", false
	}
	if json.Unmarshal(m["name"], &name) != nil || strings.TrimSpace(name) == "" {
		return "", "", false
	}
	a, has := m["arguments"]
	if !has {
		a, has = m["parameters"]
	}
	if !has {
		return "", "", false
	}
	as := strings.TrimSpace(string(a))
	if !strings.HasPrefix(as, "{") {
		// arguments kadang STRING berisi json (gaya OpenAI). Terima kalau valid object.
		var inner string
		if json.Unmarshal(a, &inner) == nil {
			inner = strings.TrimSpace(inner)
			if strings.HasPrefix(inner, "{") && json.Valid([]byte(inner)) {
				return name, inner, true
			}
		}
		return "", "", false
	}
	if !json.Valid([]byte(as)) {
		return "", "", false
	}
	return name, as, true
}

func init() {
	// A) ```json fence ber-shape tool. Strip HANYA fence tool-nya (fence data lain dibiarin).
	RegisterToolcallExtractor(func(content string) ([]RecoveredToolCall, string) {
		var calls []RecoveredToolCall
		cleaned := content
		for _, m := range muntahFenceRe.FindAllStringSubmatch(content, -1) {
			if name, args, ok := strictToolJSON(m[1]); ok {
				calls = append(calls, RecoveredToolCall{Name: name, Args: args})
				cleaned = strings.Replace(cleaned, m[0], "", 1)
			}
		}
		if len(calls) == 0 {
			return nil, content
		}
		return calls, cleaned
	})

	// B) JSON TELANJANG: SELURUH content (trim) = 1 object tool-shaped.
	RegisterToolcallExtractor(func(content string) ([]RecoveredToolCall, string) {
		t := strings.TrimSpace(content)
		if !strings.HasPrefix(t, "{") || !strings.HasSuffix(t, "}") {
			return nil, content
		}
		if name, args, ok := strictToolJSON(t); ok {
			return []RecoveredToolCall{{Name: name, Args: args}}, ""
		}
		return nil, content
	})

	// C) FUNC-SYNTAX: SELURUH content = `nama({...})` (whole-match, anti code-in-prosa).
	RegisterToolcallExtractor(func(content string) ([]RecoveredToolCall, string) {
		m := muntahFuncRe.FindStringSubmatch(content)
		if m == nil {
			return nil, content
		}
		args := strings.TrimSpace(m[2])
		if !json.Valid([]byte(args)) {
			return nil, content
		}
		return []RecoveredToolCall{{Name: m[1], Args: args}}, ""
	})
}
