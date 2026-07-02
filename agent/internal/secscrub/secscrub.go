// secscrub — F-G: scrubber rahasia REKURSIF buat sink log/trace/DB.
// 📄 Dok: FLowork_os/lock/secscrub.md
//
// Beda sama piistrip (router, PII linear di prompt): ini nyasar KREDENSIAL
// (API key/token/password) di OBJEK terstruktur (map/slice) sebelum
// dipersist ke DB agent (interactions/decisions/mistakes) — biar token yang
// ke-echo di chat/error/log ga awet di disk. Nilai di-potong jadi
// "<prefix>…<4 char terakhir>" biar masih bisa dikorelasi tanpa bocor.
//
// Package BARU non-frozen; dicolok ke sink via seam kernelhost.SanitizeLogged*
// (lihat agent/secscrub_ext.go). Konservatif: yang ga match pola = utuh.
package secscrub

import (
	"regexp"
	"strings"
)

// tokenRe — pola kredensial yang bentuknya khas (aman di-redact agresif).
var tokenRe = []*regexp.Regexp{
	regexp.MustCompile(`sk-ant-[A-Za-z0-9_\-]{8,}`),                       // Anthropic
	regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`),                             // OpenAI-style
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`),                      // GitHub
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`),                    // GitHub fine-grained
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                                // AWS access key id
	regexp.MustCompile(`xox[baprs]-[A-Za-z0-9\-]{10,}`),                   // Slack
	regexp.MustCompile(`AIza[0-9A-Za-z_\-]{30,}`),                         // Google API
	regexp.MustCompile(`eyJ[A-Za-z0-9_\-]{15,}\.[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{5,}`), // JWT
	regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._~+/\-]{16,}`),            // Authorization: Bearer …
}

// assignRe — `password=…` / `"api_key": "…"` gaya key=value di dalam string.
var assignRe = regexp.MustCompile(
	`(?i)(password|passwd|secret|api[_\-]?key|token|credential|private[_\-]?key)(["']?\s*[:=]\s*["']?)([^\s"',;}{]{8,})`)

// secretKeyRe — nama KEY map yang isinya pasti rahasia (redact walau valuenya
// ga match pola — konservatif ke arah aman buat field yang namanya ngaku).
var secretKeyRe = regexp.MustCompile(
	`(?i)^(password|passwd|secret|api[_\-]?key|token|access[_\-]?token|refresh[_\-]?token|authorization|auth|cookie|credential|private[_\-]?key)$`)

const maxDepth = 12

// clip — "<max 6 char awal>…<4 char akhir>" (roadmap: "potong sk-ant-…XXXX").
func clip(s string) string {
	head := s
	if len(head) > 6 {
		head = head[:6]
	}
	tail := ""
	if len(s) >= 4 {
		tail = s[len(s)-4:]
	}
	return head + "…" + tail
}

// RedactString — scrub semua pola kredensial di 1 string.
func RedactString(s string) string {
	if s == "" {
		return s
	}
	for _, re := range tokenRe {
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			return "[REDACTED:" + clip(m) + "]"
		})
	}
	s = assignRe.ReplaceAllString(s, `$1$2[REDACTED]`)
	return s
}

// RedactMap — versi map (buat metadata/inputs sink). Non-destruktif: return
// COPY; input caller ga diubah (anti side-effect ke pemakai lain).
func RedactMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out, _ := redactAny(m, 0).(map[string]any)
	if out == nil {
		return m
	}
	return out
}

// Redact — jalan rekursif ke semua struktur umum (map/slice/string).
func Redact(v any) any { return redactAny(v, 0) }

func redactAny(v any, depth int) any {
	if depth > maxDepth {
		return v
	}
	switch t := v.(type) {
	case string:
		return RedactString(t)
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if s, ok := val.(string); ok && secretKeyRe.MatchString(strings.TrimSpace(k)) && s != "" {
				out[k] = "[REDACTED:" + clip(s) + "]"
				continue
			}
			out[k] = redactAny(val, depth+1)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(t))
		for k, s := range t {
			if secretKeyRe.MatchString(strings.TrimSpace(k)) && s != "" {
				out[k] = "[REDACTED:" + clip(s) + "]"
				continue
			}
			out[k] = RedactString(s)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = redactAny(val, depth+1)
		}
		return out
	case []string:
		out := make([]string, len(t))
		for i, s := range t {
			out[i] = RedactString(s)
		}
		return out
	default:
		return v
	}
}
