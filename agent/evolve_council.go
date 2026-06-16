// === LOCKED FILE (soft) === Status: STABLE — owner-approved 2026-06-16 (autonomous sprint A1 dewan).
// LOCKED ≠ FREEZE (boleh diedit dgn izin owner).
//
// evolve_council.go — A1 DEWAN ADVERSARIAL: logika debat (di-inject ke agentmgr.EvolveCouncilHandler).
// Pakai model KUAT (coderModel = Opus). 5 panggilan: Pembela → Penantang (semi-veto) → Hakim panel-3
// (3 framing kriteria berbeda = panel beragam) → sintesa konservatif.
package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/agentmgr"
)

const evolvePillarDesc = "ekonomi (cari duit/biayai-diri), keamanan (LANTAI KERAS — gak boleh dikorbanin), " +
	"warga (manfaat + mudahin agent/user lain), kecerdasan (cerdas + evolusi), mandiri (hidup tanpa owner)"

// evolveCouncilJudge — dewan adversarial atas 1 proposal. Konservatif by-design.
func evolveCouncilJudge() agentmgr.CouncilJudge {
	return func(ctx context.Context, p agentdb.EvolveProposal) (agentmgr.CouncilVerdict, error) {
		model := coderModel("")
		v := agentmgr.CouncilVerdict{Model: model}
		prop := fmt.Sprintf("PROPOSAL EVOLUSI:\n- kind: %s\n- target: %s\n- pilar (auto-tag): %s\n- risk: %s\n- alasan: %s",
			p.Kind, p.TargetFile, nonEmpty(p.Pillar, "(belum)"), p.Risk, p.Rationale)

		// 1) PEMBELA (advocate, fresh-eyes — bukan pengusul, biar objektif)
		res, e := routerChat(ctx, model, []map[string]any{
			{"role": "system", "content": "Kamu PEMBELA di dewan evolusi Flowork. Argumenkan PRO proposal ini: " +
				"petakan ke 5 PILAR (" + evolvePillarDesc + "), manfaat KONKRET, kenapa AMAN + ADDITIVE + reversibel. " +
				"Persuasif TAPI JUJUR — jangan ngarang manfaat. Max 120 kata."},
			{"role": "user", "content": prop},
		}, nil, 500)
		if e != nil {
			return v, fmt.Errorf("pembela: %w", e)
		}
		v.Pembela = strings.TrimSpace(res.Content)

		// 2) PENANTANG (skeptic, semi-veto)
		res, e = routerChat(ctx, model, []map[string]any{
			{"role": "system", "content": "Kamu PENANTANG di dewan evolusi Flowork. Tugasmu CARI CACAT: risiko, mutasi-LETAL, " +
				"cara proposal ini BISA NGERUSAK / MBOBOL / nge-DESTABILISASI Flowork, kerusakan LINTAS-PILAR (majuin 1 pilar " +
				"tapi NGORBANIN lain — terutama KEAMANAN = lantai keras). Default SKEPTIS; ragu = flag. Kalau ada risiko FATAL " +
				"yang gak kebantah, MULAI balasan dengan baris 'VETO: <alasan singkat>'. Max 120 kata."},
			{"role": "user", "content": prop + "\n\n--- ARGUMEN PEMBELA ---\n" + v.Pembela},
		}, nil, 500)
		if e != nil {
			return v, fmt.Errorf("penantang: %w", e)
		}
		v.Penantang = strings.TrimSpace(res.Content)
		v.PenantangVeto = strings.Contains(strings.ToUpper(v.Penantang), "VETO:")

		// 3) HAKIM panel-3 — tiap hakim 1 kriteria (panel beragam, anti rubber-stamp)
		frames := []string{
			"Fokusmu: NET-POSITIF lintas-pilar — proposal majuin >=1 pilar TANPA ngerusak yang lain (keamanan = lantai keras, gak boleh turun).",
			"Fokusmu: kecocokan JIWA/DOKTRIN Flowork + apakah proposal GROUNDED/nyata (anti-halu, bukan ngelantur).",
			"Fokusmu: MANFAAT buat agent/user lain + REVERSIBILITAS (additive, gampang di-rollback kalau salah).",
		}
		approve, reject := 0, 0
		for i, fr := range frames {
			res, e = routerChat(ctx, model, []map[string]any{
				{"role": "system", "content": "Kamu HAKIM-GERBANG #" + strconv.Itoa(i+1) + " di dewan evolusi Flowork. " + fr +
					" Timbang argumen Pembela vs Penantang. RAGU = stage/reject (KONSERVATIF). Kalau keamanan dikorbanin = reject. " +
					"Balas PERSIS format ini, tanpa prosa lain:\nDECISION: <approve|stage|reject>\nSCORE: <0-10>\nREASON: <1 kalimat>"},
				{"role": "user", "content": prop + "\n\n--- PEMBELA ---\n" + v.Pembela + "\n\n--- PENANTANG ---\n" + v.Penantang},
			}, nil, 220)
			if e != nil {
				return v, fmt.Errorf("hakim#%d: %w", i+1, e)
			}
			dec, score, reason := parseJudgeVote(res.Content)
			v.Judges = append(v.Judges, agentmgr.CouncilJudgeVote{Decision: dec, Score: score, Reason: reason})
			switch dec {
			case "approve":
				approve++
			case "reject":
				reject++
			}
		}

		// 4) SINTESA (konservatif): veto fatal tanpa override bulat → reject; mayoritas reject → reject;
		//    mayoritas approve TANPA veto → approve; selain itu (ragu/split) → STAGE buat review owner.
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
			v.Reasoning = "Putusan gak bulat / ada keraguan → STAGE buat review owner (konservatif: ragu = jangan auto)."
		}
		return v, nil
	}
}

// parseJudgeVote — ambil DECISION/SCORE/REASON dari balasan hakim (toleran).
func parseJudgeVote(s string) (decision string, score int, reason string) {
	decision = "stage" // default konservatif kalau gagal parse
	for _, ln := range strings.Split(s, "\n") {
		t := strings.TrimSpace(ln)
		up := strings.ToUpper(t)
		switch {
		case strings.HasPrefix(up, "DECISION:"):
			d := strings.ToLower(strings.TrimSpace(t[len("DECISION:"):]))
			for _, k := range []string{"approve", "reject", "stage"} {
				if strings.Contains(d, k) {
					decision = k
				}
			}
		case strings.HasPrefix(up, "SCORE:"):
			fields := strings.FieldsFunc(t[len("SCORE:"):], func(r rune) bool { return r < '0' || r > '9' })
			if len(fields) > 0 {
				if n, err := strconv.Atoi(fields[0]); err == nil {
					score = n
				}
			}
		case strings.HasPrefix(up, "REASON:"):
			reason = strings.TrimSpace(t[len("REASON:"):])
		}
	}
	return decision, score, reason
}
