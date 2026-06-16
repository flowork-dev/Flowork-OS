// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-05-30 · re-edit 2026-06-17 (owner-approved): count LIVE drawers only
//   (deleted_at IS NULL) — report corpus retrievable beneran, bukan 5M (83% tombstoned). Re-LOCK.
// Reason: Audit pass — Brain drawer/embedding/skills storage.

package brain

import (
	"context"
	"os"
)

// WingCount — drawer count for one wing.
type WingCount struct {
	Wing  string `json:"wing"`
	Count int64  `json:"count"`
}

// Stats — a snapshot of the brain DB for the dashboard.
type Stats struct {
	Available bool        `json:"available"`
	Path      string      `json:"path"`
	SizeBytes int64       `json:"sizeBytes"`
	Drawers   int64       `json:"drawers"`
	Wings     []WingCount `json:"wings"`
	Skills    int         `json:"skills"` // embedded skill library size
}

// GetStats reports availability + lightweight content stats. Counts are
// best-effort: a query error leaves the field zero rather than failing.
func GetStats(ctx context.Context) Stats {
	st := Stats{Path: DBPath(), Skills: len(Skills())}
	if !Available() {
		return st
	}
	st.Available = true
	if info, err := os.Stat(st.Path); err == nil {
		st.SizeBytes = info.Size()
	}
	db, err := Open()
	if err != nil {
		return st
	}
	// Count LIVE drawers only (deleted_at IS NULL): per 2026-06-17 retrieval (FTS +
	// semantic) SKIP tombstoned, so report what the brain can REALLY retrieve — the
	// live corpus — not the 5M total that's 83% soft-deleted.
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM drawers WHERE deleted_at IS NULL`).Scan(&st.Drawers)
	rows, err := db.QueryContext(ctx, `SELECT wing, COUNT(*) c FROM drawers
		WHERE deleted_at IS NULL GROUP BY wing ORDER BY c DESC LIMIT 12`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var w WingCount
			if err := rows.Scan(&w.Wing, &w.Count); err == nil {
				st.Wings = append(st.Wings, w)
			}
		}
	}
	return st
}
