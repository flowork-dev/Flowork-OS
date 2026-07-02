// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// inject_budget_ext.go — sibling NON-frozen (Rule 7): BUDGET AGREGAT SYSTEM-INJECT (roadmap F-A3).
//
// AKAR: tiap injector (knowledge/skill/insting/antibodi) punya cap SENDIRI, tapi ga ada
// yang ngejumlahin TOTAL-nya → worst-case semua nyuntik bareng = prompt bengkak ("muntah").
// Fix: lewat seam `applyInjectShaper` (POLA B di dispatcher.go), pasca SEMUA injeksi,
// jumlahin karakter suntikan yang dikenal; lebih dari budget → BUANG PESAN UTUH per-prioritas:
//   kelas 1 (buang duluan): knowledge/skill brain — bisa di-retrieve ulang turn depan
//   kelas 2: insting (WHEN→THEN)
//   kelas 3 (paling akhir): antibodi (kesalahan terbukti)
//   TIDAK PERNAH disentuh: doktrin SACRED ("## Project doctrine") + persona/system caller.
// Buang utuh (bukan potong tengah) biar ga ada instruksi kepotong setengah (anti-halu).
//
// SWITCH (GUI = kebenaran): FLOWORK_INJECT_BUDGET (char; 0 = OFF = perilaku lama).
// Saran 6000–12000. File ini DIHAPUS → seam balik no-op → inti tetep jalan (delete-test).

package router

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/flowork-os/flowork_Router/internal/store"
)

// Header suntikan yang DIKENAL (harus sinkron sama builder frozen: brainenrich.go,
// instinctenrich.go, mistakeenrich.go). Ganti header di builder = update di sini.
const (
	hdrBrainPersona = "You are operating with a shared knowledge brain"
	hdrKnowledge    = "## Relevant knowledge"
	hdrSkills       = "## Applicable skills"
	hdrInstinct     = "## Insting —"
	hdrAntibody     = "## Antibodi —"
)

// classifyInjected — kelas prioritas-buang suntikan router. 0 = BUKAN suntikan yang
// boleh disentuh (doktrin/persona caller/pesan biasa).
func classifyInjected(content string) int {
	c := strings.TrimSpace(content)
	switch {
	case strings.HasPrefix(c, hdrKnowledge), strings.HasPrefix(c, hdrBrainPersona), strings.HasPrefix(c, hdrSkills):
		return 1
	case strings.HasPrefix(c, hdrInstinct):
		return 2
	case strings.HasPrefix(c, hdrAntibody):
		return 3
	}
	return 0
}

// injectBudgetChars — budget total (char) dari switch GUI; <=0 = off. Dibaca
// per-request → live via fwswitch watcher.
func injectBudgetChars() int {
	v := strings.TrimSpace(os.Getenv("FLOWORK_INJECT_BUDGET"))
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func init() {
	prev := applyInjectShaper
	applyInjectShaper = func(ctx context.Context, req OpenAIRequest, settings *store.Settings) OpenAIRequest {
		req = prev(ctx, req, settings)
		budget := injectBudgetChars()
		if budget <= 0 {
			return req
		}
		total := 0
		for _, m := range req.Messages {
			if m.Role == "system" && classifyInjected(m.Content) > 0 {
				total += len(m.Content)
			}
		}
		if total <= budget {
			return req
		}
		// Tandai yang dibuang per-kelas (1 → 2 → 3), dari BELAKANG (yang paling
		// baru disuntik duluan), sampe total masuk budget.
		drop := map[int]bool{}
		for class := 1; class <= 3 && total > budget; class++ {
			for i := len(req.Messages) - 1; i >= 0 && total > budget; i-- {
				m := req.Messages[i]
				if drop[i] || m.Role != "system" || classifyInjected(m.Content) != class {
					continue
				}
				drop[i] = true
				total -= len(m.Content)
				log.Printf("flow_router inject-budget: buang suntikan kelas %d (%d char) — total lewat budget %d", class, len(m.Content), budget)
			}
		}
		if len(drop) == 0 {
			return req
		}
		out := make([]OpenAIMessage, 0, len(req.Messages)-len(drop))
		for i, m := range req.Messages {
			if !drop[i] {
				out = append(out, m)
			}
		}
		req.Messages = out
		return req
	}
}
