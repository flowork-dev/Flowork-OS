package tools

import (
	"context"
	"errors"
	"testing"
)

// fakeCapTool — tool dummy buat tes cap-gate (declare capability tertentu).
type fakeCapTool struct {
	cap string
	ran *bool
}

func (f fakeCapTool) Name() string      { return "fake_cap_tool" }
func (f fakeCapTool) Capability() string { return f.cap }
func (f fakeCapTool) Schema() Schema     { return Schema{Description: "tes cap-gate"} }
func (f fakeCapTool) Run(_ context.Context, _ map[string]any) (Result, error) {
	if f.ran != nil {
		*f.ran = true
	}
	return Result{Output: "ok"}, nil
}

// TestSandboxCapGate — KUNCI keamanan fondasional. Ini yg bikin arah "buang
// subscription-gating" AMAN: exposure (apa yg agent LIHAT) dipisah dari permission
// (apa yg agent boleh RUN). Tool ber-cap yg agent-nya GAK punya = DITOLAK
// (ErrSandboxCapDenied) + Run TIDAK PERNAH jalan. Validasi LIVE 2026-06-25:
// fb-repofinder/fbspecial (non-privileged) → git(exec:git)/system_power(exec:power)
// DENIED di kondisi group-OFF maupun group-ON. Test ini ngunci jaminan itu di unit-level
// supaya gak ke-regresi diam-diam (mis. ada yg longgarin Gate 1 sandbox.go).
func TestSandboxCapGate(t *testing.T) {
	deny := CapsChecker(func(string) bool { return false })
	allow := CapsChecker(func(string) bool { return true })
	noStore := SandboxOpts{SkipDisabledGate: true, SkipRateLimit: true}

	t.Run("deny_cap_kurang_ditolak_dan_Run_tak_jalan", func(t *testing.T) {
		ran := false
		ctx := WithCapsChecker(context.Background(), deny)
		_, err := SandboxRun(ctx, fakeCapTool{cap: "exec:power", ran: &ran}, nil, noStore)
		if !errors.Is(err, ErrSandboxCapDenied) {
			t.Fatalf("harus ErrSandboxCapDenied, dapet: %v", err)
		}
		if ran {
			t.Fatal("BOCOR: Run jalan padahal cap ditolak")
		}
	})

	t.Run("allow_cap_dipunya_jalan", func(t *testing.T) {
		ran := false
		ctx := WithCapsChecker(context.Background(), allow)
		_, err := SandboxRun(ctx, fakeCapTool{cap: "exec:power", ran: &ran}, nil, noStore)
		if err != nil {
			t.Fatalf("cap dipunya tapi error: %v", err)
		}
		if !ran {
			t.Fatal("Run tak jalan padahal cap diizinin")
		}
	})

	t.Run("tool_tanpa_cap_tidak_di_gate", func(t *testing.T) {
		// DOKUMENTASI PENTING: tool ber-Capability()=="" TIDAK di-gate (lolos walau
		// checker nolak semua). Konsekuensi: tiap tool BAHAYA WAJIB declare cap —
		// kalau lupa, "buang subscription" bikin dia bisa dipanggil siapa pun.
		ran := false
		ctx := WithCapsChecker(context.Background(), deny)
		_, err := SandboxRun(ctx, fakeCapTool{cap: "", ran: &ran}, nil, noStore)
		if err != nil {
			t.Fatalf("tool tanpa cap harusnya lolos, error: %v", err)
		}
		if !ran {
			t.Fatal("tool tanpa cap tidak jalan")
		}
	})

	t.Run("tanpa_caps_checker_default_allow_backward_compat", func(t *testing.T) {
		// Phase-1 backward-compat: ctx TANPA CapsChecker → default-allow (mis. endpoint
		// admin). Di runtime nyata, ToolRunHandler SELALU inject CapsCheckerForAgent →
		// gate aktif (kebukti live). Test ini cuma ngedokumentasiin perilaku default.
		ran := false
		_, err := SandboxRun(context.Background(), fakeCapTool{cap: "exec:power", ran: &ran}, nil, noStore)
		if err != nil {
			t.Fatalf("tanpa checker harusnya default-allow, error: %v", err)
		}
		if !ran {
			t.Fatal("default-allow tapi Run tak jalan")
		}
	})
}
