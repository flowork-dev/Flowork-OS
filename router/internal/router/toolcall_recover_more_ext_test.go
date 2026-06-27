package router

import (
	"strings"
	"testing"
)

func recovered(content string) *OpenAIResponse {
	r := &OpenAIResponse{Choices: []OpenAIChoice{{Message: OpenAIMessage{Content: content}}}}
	recoverTextToolCalls(r)
	return r
}

func mustRecover(t *testing.T, content, wantName string) {
	t.Helper()
	r := recovered(content)
	if !hasNativeToolCalls(r.Choices[0].Message.ToolCalls) {
		t.Fatalf("EXPECT recover, GOT none. content=%q tc=%s", r.Choices[0].Message.Content, r.Choices[0].Message.ToolCalls)
	}
	if !strings.Contains(string(r.Choices[0].Message.ToolCalls), wantName) {
		t.Fatalf("name %q hilang: %s", wantName, r.Choices[0].Message.ToolCalls)
	}
}

func mustNotRecover(t *testing.T, content string) {
	t.Helper()
	r := recovered(content)
	if hasNativeToolCalls(r.Choices[0].Message.ToolCalls) {
		t.Fatalf("FALSE-POSITIVE: teks normal kejadiin tool-call. content=%q tc=%s", content, r.Choices[0].Message.ToolCalls)
	}
	if r.Choices[0].Message.Content != content {
		t.Fatalf("content teks normal keubah: %q → %q", content, r.Choices[0].Message.Content)
	}
}

// ── POSITIF: format muntah model lokal HARUS ke-recover ──────────────────────
func TestRecoverMore_FencedJSON(t *testing.T) {
	mustRecover(t, "Oke:\n```json\n{\"name\":\"build_app\",\"arguments\":{\"prompt\":\"pantun\"}}\n```", "build_app")
}
func TestRecoverMore_FencedParameters(t *testing.T) {
	mustRecover(t, "```\n{\"name\":\"bikin_kotak\",\"parameters\":{\"warna\":\"merah\"}}\n```", "bikin_kotak")
}
func TestRecoverMore_BareJSON(t *testing.T) {
	mustRecover(t, `{"name":"build_app","arguments":{"prompt":"x"}}`, "build_app")
}
func TestRecoverMore_FuncSyntaxWhole(t *testing.T) {
	// kasus NYATA architect: balasan = build_app({...}) doang.
	mustRecover(t, "build_app({\n  \"prompt\": \"Buat app Pantun Kocak\"\n})", "build_app")
}
func TestRecoverMore_ArgsAsString(t *testing.T) {
	// arguments = STRING berisi json (gaya OpenAI native).
	mustRecover(t, `{"name":"foo","arguments":"{\"a\":1}"}`, "foo")
}

// ── NEGATIF: teks/kode/data normal JANGAN kejadiin tool-call ──────────────────
func TestRecoverMore_PlainText(t *testing.T) {
	mustNotRecover(t, "halo bro, app-nya udah jadi ya, tinggal klik tombolnya")
}
func TestRecoverMore_CodeInProse(t *testing.T) {
	// func-call di TENGAH prosa → bukan whole-match → JANGAN kena.
	mustNotRecover(t, "contoh pemakaian: panggil updateUser({\"id\":5}) terus refresh halaman")
}
func TestRecoverMore_DataJSONNotTool(t *testing.T) {
	// JSON telanjang TAPI bukan shape tool (ga ada name+arguments) → JANGAN kena.
	mustNotRecover(t, `{"status":"ok","count":3}`)
}
func TestRecoverMore_FencedDataNotTool(t *testing.T) {
	// fence json data biasa → JANGAN kena.
	mustNotRecover(t, "hasilnya:\n```json\n{\"warna\":\"merah\",\"ukuran\":5}\n```")
}
func TestRecoverMore_JSONWithNameButNoArgs(t *testing.T) {
	// punya "name" tapi tanpa arguments/parameters → bukan tool-call → JANGAN kena.
	mustNotRecover(t, `{"name":"Budi","umur":30}`)
}
