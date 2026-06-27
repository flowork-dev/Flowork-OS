// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah fitur TANPA buka frozen: file sibling baru + registry (RegisterMeshFilter/
// RegisterExtraRoute/RegisterGraphProjection) + SWITCH fwswitch. Pola: lock/frozen-core.md

package router

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	toolCallTagRe = regexp.MustCompile(`(?s)<tool_call>\s*(\{.*?\})\s*</tool_call>`)
	leniNameRe    = regexp.MustCompile(`(?:"|')?name(?:"|')?\s*:\s*(?:"|')?([a-zA-Z0-9_.\-]+)(?:"|')?`)
	leniArgsRe    = regexp.MustCompile(`(?s)(?:"|')?(?:parameters|arguments)(?:"|')?\s*:\s*(\{.*\})`)
)

func toolcallRecoverEnabled() bool {
	return strings.TrimSpace(strings.ToLower(os.Getenv("FLOWORK_TOOLCALL_RECOVER"))) != "0"
}

// RecoveredToolCall — 1 tool-call yg dipulihin dari teks muntah (name + args JSON-string).
type RecoveredToolCall struct {
	Name string
	Args string
}

// extraToolcallExtractors — SWITCH (POLA A, Rule #7). Extractor format-muntah TAMBAHAN
// (```json fence, JSON telanjang, syntax func(...)) didaftar lewat sibling init() TANPA
// buka freeze. Tiap extractor balik (calls, cleanedContent); kalau calls>0 → dipakai.
// Default KOSONG = perilaku persis lama (cuma <tool_call> tag). Self-sufficient: hapus
// sibling → registry kosong → aman.
var extraToolcallExtractors []func(content string) (calls []RecoveredToolCall, cleaned string)

// RegisterToolcallExtractor — daftarin extractor format-muntah baru (nil di-skip). Tiap
// extractor WAJIB konservatif: cuma balik calls kalau yakin itu tool-call (anti false-positive
// yg ngerusak balasan teks normal).
func RegisterToolcallExtractor(fn func(content string) (calls []RecoveredToolCall, cleaned string)) {
	if fn != nil {
		extraToolcallExtractors = append(extraToolcallExtractors, fn)
	}
}

func recoverTextToolCalls(resp *OpenAIResponse) {
	if resp == nil || !toolcallRecoverEnabled() {
		return
	}
	type fnObj struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}
	type tcall struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function fnObj  `json:"function"`
	}
	mk := func(j int, name, args string) tcall {
		return tcall{ID: "call_recover_" + strconv.Itoa(j), Type: "function", Function: fnObj{Name: name, Arguments: args}}
	}
	for i := range resp.Choices {
		msg := &resp.Choices[i].Message
		if hasNativeToolCalls(msg.ToolCalls) {
			continue
		}
		var calls []tcall
		via := ""
		// 1) <tool_call> tag (PRIMER — paling banyak dipake model lokal). Perilaku lama.
		if strings.Contains(msg.Content, "<tool_call>") {
			for _, m := range toolCallTagRe.FindAllStringSubmatch(msg.Content, -1) {
				name, args := parseToolCallInner(m[1])
				if name != "" {
					calls = append(calls, mk(len(calls), name, args))
				}
			}
			if len(calls) > 0 {
				msg.Content = stripToolCallTags(msg.Content)
				via = "<tool_call>"
			}
		}
		// 2) Extractor TAMBAHAN (sibling) — cuma kalau tag ga ngasih hasil. Format lain
		//    (```json fence, JSON telanjang, func-syntax). Konservatif per-extractor.
		if len(calls) == 0 {
			for _, ex := range extraToolcallExtractors {
				rcs, cleaned := ex(msg.Content)
				if len(rcs) == 0 {
					continue
				}
				for _, rc := range rcs {
					if strings.TrimSpace(rc.Name) != "" {
						calls = append(calls, mk(len(calls), rc.Name, rc.Args))
					}
				}
				if len(calls) > 0 {
					msg.Content = strings.TrimSpace(cleaned)
					via = "extractor"
					break
				}
			}
		}
		if len(calls) > 0 {
			if b, err := json.Marshal(calls); err == nil {
				msg.ToolCalls = b
				resp.Choices[i].FinishReason = "tool_calls"
				log.Printf("flow_router toolcall-recover: %d %s teks → native tool_calls (anti-bocor)", len(calls), via)
			}
		}
	}
}

func parseToolCallInner(inner string) (name, args string) {
	var raw struct {
		Name       string          `json:"name"`
		Arguments  json.RawMessage `json:"arguments"`
		Parameters json.RawMessage `json:"parameters"`
	}
	if json.Unmarshal([]byte(inner), &raw) == nil && strings.TrimSpace(raw.Name) != "" {
		a := raw.Arguments
		if len(a) == 0 {
			a = raw.Parameters
		}
		if len(a) == 0 {
			a = json.RawMessage("{}")
		}
		return raw.Name, string(a)
	}

	nm := leniNameRe.FindStringSubmatch(inner)
	if nm == nil {
		return "", ""
	}
	args = "{}"
	if am := leniArgsRe.FindStringSubmatch(inner); am != nil && json.Valid([]byte(am[1])) {
		args = am[1]
	}
	return nm[1], args
}

func hasNativeToolCalls(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s != "" && s != "null" && s != "[]"
}

func stripToolCallTags(s string) string {
	s = toolCallTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "<tool_call>", "")
	s = strings.ReplaceAll(s, "</tool_call>", "")
	return strings.TrimSpace(s)
}
