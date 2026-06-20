// evolve_council_group.go — DEWAN ADVERSARIAL via 5 AGENT (otak pindah ke grup
// self-evolution). Owner 2026-06-20 "pindahin otaknya". Tiap peran (pembela/penantang/
// 3 hakim) = 1 AGENT persona-DB yang di-invoke (bukan routerChat hardcoded). ALUR +
// SINTESA KONSERVATIF SAMA PERSIS evolve_council.go.
//
// ⚠️ INI CUMA OTAK (deliberasi). GATE KEAMANAN (EvolveGateDeps: mode=auto + karma matang
// + ModelStrong cloud-kuat + fail-CLOSED) di-WRAP di LUAR judge ini (EvolveScheduleAutoApply
// / EvolveCouncilHandler) — TIDAK disentuh. Judge cuma ngasih verdict; gate yang mutusin
// boleh-apply. Error invoke agent → judge return error → harness perlakuin konservatif
// (hold/ga auto-apply). Sintesa voting TETAP di harness (deterministik), BUKAN didelegasi.
package main

import (
	"context"
	"fmt"
	"strings"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/agentmgr"
	"flowork-gui/internal/kernelhost"
)

// evolveCouncilJudgeViaGroup — judge yang invoke 5 agent council. Drop-in pengganti
// evolveCouncilJudge() (inline). Gate di-pasang TERPISAH di pemanggil — ga berubah.
func evolveCouncilJudgeViaGroup(host *kernelhost.Host) agentmgr.CouncilJudge {
	return func(ctx context.Context, p agentdb.EvolveProposal) (agentmgr.CouncilVerdict, error) {
		v := agentmgr.CouncilVerdict{Model: coderModel("") + " (grup self-evolution)"}
		prop := fmt.Sprintf("PROPOSAL EVOLUSI:\n- kind: %s\n- target: %s\n- pilar (auto-tag): %s\n- risk: %s\n- alasan: %s",
			p.Kind, p.TargetFile, nonEmpty(p.Pillar, "(belum)"), p.Risk, p.Rationale)

		ask := func(agentID, text string) (string, error) {
			raw, err := host.InvokeAgentMessage(ctx, agentID, text, "evolve-council")
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(extractReply(raw)), nil
		}

		// 1) PEMBELA (advocate)
		pembela, e := ask("evo-pembela", prop)
		if e != nil {
			return v, fmt.Errorf("pembela: %w", e)
		}
		v.Pembela = pembela

		// 2) PENANTANG (skeptic, semi-veto)
		penantang, e := ask("evo-penantang", prop+"\n\n--- ARGUMEN PEMBELA ---\n"+pembela)
		if e != nil {
			return v, fmt.Errorf("penantang: %w", e)
		}
		v.Penantang = penantang
		v.PenantangVeto = strings.Contains(strings.ToUpper(penantang), "VETO:")

		// 3) HAKIM panel-3 (framing beda, di persona masing-masing agent)
		approve, reject := 0, 0
		for _, hakim := range []string{"evo-hakim-1", "evo-hakim-2", "evo-hakim-3"} {
			vote, e := ask(hakim, prop+"\n\n--- PEMBELA ---\n"+pembela+"\n\n--- PENANTANG ---\n"+penantang)
			if e != nil {
				return v, fmt.Errorf("%s: %w", hakim, e)
			}
			dec, score, reason := parseJudgeVote(vote)
			v.Judges = append(v.Judges, agentmgr.CouncilJudgeVote{Decision: dec, Score: score, Reason: reason})
			switch dec {
			case "approve":
				approve++
			case "reject":
				reject++
			}
		}

		// 4) SINTESA konservatif — SAMA PERSIS evolve_council.go (harness, deterministik).
		switch {
		case v.PenantangVeto && approve < 3:
			v.Decision = "reject"
			v.Reasoning = "Penantang angkat VETO (risiko fatal) & hakim gak BULAT override → reject."
		case reject >= 2:
			v.Decision = "reject"
			v.Reasoning = "Mayoritas hakim (>=2/3) REJECT."
		case approve >= 2 && !v.PenantangVeto:
			v.Decision = "approve"
			v.Reasoning = "Mayoritas hakim (>=2/3) APPROVE, gak ada veto fatal."
		default:
			v.Decision = "stage"
			v.Reasoning = "Putusan gak bulat / ada keraguan → STAGE buat review owner (konservatif)."
		}
		return v, nil
	}
}
