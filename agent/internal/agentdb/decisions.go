// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Cara kerja sistem: lihat os/.  ⚠️ FROZEN — jangan edit file ini.
// Nambah/ubah fitur TANPA buka frozen: pakai SEAM non-frozen + SWITCH
// (internal/fwswitch/registry.go). Pola lengkap: lock/frozen-core.md

package agentdb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type Decision struct {
	ID               int64          `json:"id"`
	DecisionType     string         `json:"decision_type"`
	Rationale        string         `json:"rationale"`
	Inputs           map[string]any `json:"inputs"`
	Outcome          string         `json:"outcome"`
	RefInteractionID int64          `json:"ref_interaction_id"`
	OccurredAt       string         `json:"occurred_at"`
}

func (s *Store) LogDecision(decisionType, rationale, outcome string, inputs map[string]any, refInteractionID int64) (int64, error) {
	rationale = SanitizeText(rationale) // seam scrub rahasia (sanitize_seam.go)
	inputs = SanitizeMeta(inputs)
	s.mu.Lock()
	defer s.mu.Unlock()

	if decisionType == "" || rationale == "" {
		return 0, fmt.Errorf("decision_type + rationale required")
	}

	const maxRationaleBytes = 4 * 1024
	if len(rationale) > maxRationaleBytes {
		rationale = rationale[:maxRationaleBytes] + "…[truncated]"
	}

	var inputsJSON string
	if len(inputs) > 0 {
		b, err := json.Marshal(inputs)
		if err == nil {
			inputsJSON = string(b)
		}
	}
	if inputsJSON == "" {
		inputsJSON = "{}"
	}

	if outcome == "" {
		outcome = "pending"
	}

	ts := time.Now().UTC().Format(time.RFC3339)

	var refArg any
	if refInteractionID > 0 {
		refArg = refInteractionID
	}

	res, err := s.db.Exec(
		`INSERT INTO decisions(decision_type, rationale, inputs, outcome, ref_interaction_id, occurred_at)
		 VALUES(?, ?, ?, ?, ?, ?)`,
		decisionType, rationale, inputsJSON, outcome, refArg, ts,
	)
	if err != nil {
		return 0, fmt.Errorf("insert decision: %w", err)
	}
	return res.LastInsertId()
}

func (s *Store) ListDecisions(decisionType string, limit int) ([]Decision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 || limit > 500 {
		limit = 50
	}

	query := `SELECT id, decision_type, rationale, inputs, outcome,
	                 COALESCE(ref_interaction_id, 0), occurred_at
	          FROM decisions WHERE deleted_at IS NULL`
	args := []any{}
	if decisionType != "" {
		query += ` AND decision_type = ?`
		args = append(args, decisionType)
	}
	query += ` ORDER BY occurred_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query decisions: %w", err)
	}
	defer rows.Close()

	var out []Decision
	for rows.Next() {
		var d Decision
		var inputsRaw string
		if err := rows.Scan(&d.ID, &d.DecisionType, &d.Rationale, &inputsRaw,
			&d.Outcome, &d.RefInteractionID, &d.OccurredAt); err != nil {
			return nil, err
		}
		if inputsRaw != "" && inputsRaw != "{}" {
			_ = json.Unmarshal([]byte(inputsRaw), &d.Inputs)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) PruneDecisions(olderThan time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	res, err := s.db.Exec(
		`UPDATE decisions SET deleted_at = CURRENT_TIMESTAMP
		 WHERE deleted_at IS NULL AND occurred_at < ?`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("prune decisions: %w", err)
	}
	return res.RowsAffected()
}

func (s *Store) CountDecisions() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var n int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM decisions WHERE deleted_at IS NULL`).Scan(&n); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}
