package brain

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

type Node struct {
	ID         string
	Label      string
	Type       string
	Properties string
}

type Edge struct {
	FromID   string
	ToID     string
	Relation string
}

// RunDreamCycle runs the Cognitive Digestion and Knowledge Graph generation pipeline.
func RunDreamCycle(ctx context.Context) (int, error) {
	// PHASE 0 — data-loss neutralize (roadmap_opus8.md). The legacy pipeline below uses a
	// 3-topic MOCK extractor and previously soft-deleted EVERY queried memory unconditionally
	// (even when 0 nodes were extracted) = data loss. Disabled by default until rebuilt as the
	// real LLM extractor + safe digest. Dev opt-in only: FLOWORK_LEGACY_DREAM=1.
	if os.Getenv("FLOWORK_LEGACY_DREAM") != "1" {
		return 0, fmt.Errorf("dream cycle disabled: under rebuild (data-loss fix, roadmap Phase 0)")
	}
	db, err := OpenRW()
	if err != nil {
		return 0, fmt.Errorf("open rw: %w", err)
	}

	// 1. Query active memories
	query := `SELECT id, title, content FROM memories 
	          WHERE category IN ('chat', 'interaction', 'project_notes') 
	            AND deleted_at IS NULL`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("query memories: %w", err)
	}
	defer rows.Close()

	type memory struct {
		id      int
		title   string
		content string
	}
	var memories []memory
	for rows.Next() {
		var m memory
		if err := rows.Scan(&m.id, &m.title, &m.content); err != nil {
			return 0, fmt.Errorf("scan memory: %w", err)
		}
		memories = append(memories, m)
	}

	// Start a transaction for digestion
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	insertedNodes := 0
	insertedEdges := 0

	for _, mem := range memories {
		lowerContent := strings.ToLower(mem.content)

		var nodes []Node
		var edges []Edge

		// Semantic rules mirroring Python Mock Extractor
		if strings.Contains(lowerContent, "fable") || strings.Contains(lowerContent, "instinct") {
			nodes = append(nodes, Node{ID: "instinct", Label: "Flowork Instinct", Type: "concept", Properties: `{"category": "ai_memory"}`})
			nodes = append(nodes, Node{ID: "flowork", Label: "Flowork OS", Type: "concept", Properties: `{"category": "operating_system"}`})
			edges = append(edges, Edge{FromID: "flowork", ToID: "instinct", Relation: "has_capability"})
		}

		if strings.Contains(lowerContent, "aola") || strings.Contains(lowerContent, "mr.dev") {
			nodes = append(nodes, Node{ID: "mr_dev", Label: "Aola Sahidin (Mr.Dev)", Type: "who", Properties: `{"role": "architect"}`})
			nodes = append(nodes, Node{ID: "flowork", Label: "Flowork OS", Type: "concept", Properties: `{"category": "operating_system"}`})
			edges = append(edges, Edge{FromID: "mr_dev", ToID: "flowork", Relation: "is_creator_of"})
		}

		if strings.Contains(lowerContent, "neonstrike") || strings.Contains(lowerContent, "fps") {
			nodes = append(nodes, Node{ID: "neonstrike", Label: "NeonStrike Game", Type: "concept", Properties: `{"genre": "FPS"}`})
			nodes = append(nodes, Node{ID: "webgl", Label: "WebGL2 Raytracer", Type: "how", Properties: `{"technology": "graphics"}`})
			nodes = append(nodes, Node{ID: "audio_dsp", Label: "Pure DSP Audio", Type: "how", Properties: `{"technology": "sound"}`})
			edges = append(edges, Edge{FromID: "neonstrike", ToID: "webgl", Relation: "requires_method"})
			edges = append(edges, Edge{FromID: "neonstrike", ToID: "audio_dsp", Relation: "requires_method"})
			edges = append(edges, Edge{FromID: "mr_dev", ToID: "neonstrike", Relation: "is_creator_of"})
			edges = append(edges, Edge{FromID: "flowork", ToID: "neonstrike", Relation: "has_project"})
		}

		sourceRef := fmt.Sprintf("memory_%d", mem.id)

		// Insert nodes
		for _, node := range nodes {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO cognitive_nodes (id, label, type, properties, source)
				VALUES (?, ?, ?, ?, ?)
				ON CONFLICT(id) DO UPDATE SET
					label=excluded.label,
					type=excluded.type,
					properties=excluded.properties,
					source=excluded.source,
					last_accessed=CURRENT_TIMESTAMP`,
				node.ID, node.Label, node.Type, node.Properties, sourceRef)
			if err != nil {
				return 0, fmt.Errorf("insert node %s: %w", node.ID, err)
			}
			insertedNodes++
		}

		// Insert edges
		for _, edge := range edges {
			// Ensure referenced nodes exist
			_, _ = tx.ExecContext(ctx, "INSERT OR IGNORE INTO cognitive_nodes (id, label, type) VALUES (?, ?, 'concept')", edge.FromID, edge.FromID)
			_, _ = tx.ExecContext(ctx, "INSERT OR IGNORE INTO cognitive_nodes (id, label, type) VALUES (?, ?, 'concept')", edge.ToID, edge.ToID)

			_, err = tx.ExecContext(ctx, `
				INSERT INTO cognitive_edges (from_id, to_id, relation_type, strength)
				VALUES (?, ?, ?, 1.0)
				ON CONFLICT(from_id, to_id, relation_type) DO UPDATE SET
					strength=1.0`,
				edge.FromID, edge.ToID, edge.Relation)
			if err != nil {
				return 0, fmt.Errorf("insert edge %s->%s: %w", edge.FromID, edge.ToID, err)
			}
			insertedEdges++
		}

		// PHASE 0 — DO NOT delete digested memories. The original code soft-deleted every
		// queried memory unconditionally (even when 0 nodes extracted) = data loss. Digestion
		// must only MARK digested (via a digest-log), never delete the raw memory. Removed on
		// purpose so the bug cannot return when this pipeline is rebuilt.
	}

	// 2. Self Healing
	// Decay edge strengths
	_, err = tx.ExecContext(ctx, "UPDATE cognitive_edges SET strength = strength * 0.95")
	if err != nil {
		return 0, fmt.Errorf("decay edges: %w", err)
	}

	// Delete dead/weak relations
	_, err = tx.ExecContext(ctx, "DELETE FROM cognitive_edges WHERE strength < 0.1")
	if err != nil {
		return 0, fmt.Errorf("delete weak edges: %w", err)
	}

	// Remove orphaned nodes
	_, err = tx.ExecContext(ctx, `
		DELETE FROM cognitive_nodes 
		WHERE id NOT IN (SELECT from_id FROM cognitive_edges)
		  AND id NOT IN (SELECT to_id FROM cognitive_edges)`)
	if err != nil {
		return 0, fmt.Errorf("remove orphans: %w", err)
	}

	// 2.5 Ingest all system core entities (Agents, Constitution, Skills, Personas) into the Knowledge Graph
	if err := syncCoreEntitiesToGraph(ctx, tx); err != nil {
		return 0, fmt.Errorf("sync core entities to graph: %w", err)
	}

	// 3. Sync Graph to RAG drawers
	if err := syncGraphToRAGTx(ctx, tx); err != nil {
		return 0, fmt.Errorf("sync graph: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return len(memories), nil
}

func SyncGraphToRAG(ctx context.Context) error {
	db, err := OpenRW()
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := syncCoreEntitiesToGraph(ctx, tx); err != nil {
		return err
	}

	if err := syncGraphToRAGTx(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func syncGraphToRAGTx(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM drawers WHERE room = 'cognitive_graph'")
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM memory_fts WHERE room = 'cognitive_graph'")
	if err != nil {
		return err
	}

	// Sync Nodes
	nRows, err := tx.QueryContext(ctx, "SELECT id, label, type, properties FROM cognitive_nodes")
	if err != nil {
		return err
	}
	type dbNode struct {
		id    string
		label string
		t     string
		props string
	}
	var dbNodes []dbNode
	for nRows.Next() {
		var n dbNode
		if err := nRows.Scan(&n.id, &n.label, &n.t, &n.props); err != nil {
			nRows.Close()
			return err
		}
		dbNodes = append(dbNodes, n)
	}
	nRows.Close()

	for _, n := range dbNodes {
		content := fmt.Sprintf("Concept Node: [%s] (type: %s, id: %s). Properties: %s", n.label, n.t, n.id, n.props)
		sum := sha256.Sum256([]byte(content))
		h := hex.EncodeToString(sum[:])
		drawerID := h[:16]

		_, err = tx.ExecContext(ctx, `
			INSERT INTO drawers (id, content, wing, room, source_type, chunk_index, importance, normalize_version, content_hash, mem_type)
			VALUES (?, ?, 'training_data', 'cognitive_graph', 'graph', 0, 5.0, 1, ?, 'project')`,
			drawerID, content, h)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO memory_fts (drawer_id, content, wing, room, source_file)
			VALUES (?, ?, 'training_data', 'cognitive_graph', 'knowledge_graph')`,
			drawerID, content)
		if err != nil {
			return err
		}
	}

	// Sync Edges
	eRows, err := tx.QueryContext(ctx, `
		SELECT n1.label, n1.type, e.relation_type, n2.label, n2.type, e.strength 
		FROM cognitive_edges e
		JOIN cognitive_nodes n1 ON e.from_id = n1.id
		JOIN cognitive_nodes n2 ON e.to_id = n2.id`)
	if err != nil {
		return err
	}
	type dbEdge struct {
		fromLabel string
		fromType  string
		relType   string
		toLabel   string
		toType    string
		strength  float64
	}
	var dbEdges []dbEdge
	for eRows.Next() {
		var e dbEdge
		if err := eRows.Scan(&e.fromLabel, &e.fromType, &e.relType, &e.toLabel, &e.toType, &e.strength); err != nil {
			eRows.Close()
			return err
		}
		dbEdges = append(dbEdges, e)
	}
	eRows.Close()

	for _, e := range dbEdges {
		content := fmt.Sprintf("Relationship: [%s] (%s) --%s--> [%s] (%s). Link strength: %.2f", e.fromLabel, e.fromType, e.relType, e.toLabel, e.toType, e.strength)
		sum := sha256.Sum256([]byte(content))
		h := hex.EncodeToString(sum[:])
		drawerID := h[:16]

		_, err = tx.ExecContext(ctx, `
			INSERT INTO drawers (id, content, wing, room, source_type, chunk_index, importance, normalize_version, content_hash, mem_type)
			VALUES (?, ?, 'training_data', 'cognitive_graph', 'graph', 0, 5.0, 1, ?, 'project')`,
			drawerID, content, h)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO memory_fts (drawer_id, content, wing, room, source_file)
			VALUES (?, ?, 'training_data', 'cognitive_graph', 'knowledge_graph')`,
			drawerID, content)
		if err != nil {
			return err
		}
	}

	return nil
}

func syncCoreEntitiesToGraph(ctx context.Context, tx *sql.Tx) error {
	// 1. Clean up old system-generated entries to prevent duplication
	_, err := tx.ExecContext(ctx, "DELETE FROM cognitive_edges WHERE from_id LIKE 'agent_%' OR to_id LIKE 'agent_%' OR from_id LIKE 'constitution_rule_%' OR to_id LIKE 'constitution_rule_%' OR from_id LIKE 'skill_%' OR to_id LIKE 'skill_%' OR from_id LIKE 'persona_%' OR to_id LIKE 'persona_%'")
	if err != nil {
		return fmt.Errorf("clear system edges: %w", err)
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM cognitive_nodes WHERE source IN ('system_agent', 'system_constitution', 'system_skill', 'system_persona')")
	if err != nil {
		return fmt.Errorf("clear system nodes: %w", err)
	}

	// Make sure the main flowork node exists
	_, _ = tx.ExecContext(ctx, "INSERT OR IGNORE INTO cognitive_nodes (id, label, type, properties, source) VALUES ('flowork', 'FLowork OS', 'concept', '{}', 'system')")

	// 2. Sync Agents
	rows, err := tx.QueryContext(ctx, "SELECT name, display_name, role, model FROM agents")
	if err == nil {
		type dbAgent struct {
			Name        string
			DisplayName string
			Role        string
			Model       string
		}
		var agents []dbAgent
		for rows.Next() {
			var a dbAgent
			if err := rows.Scan(&a.Name, &a.DisplayName, &a.Role, &a.Model); err == nil {
				agents = append(agents, a)
			}
		}
		rows.Close()

		for _, a := range agents {
			nodeID := "agent_" + a.Name
			props := fmt.Sprintf(`{"role": %q, "model": %q}`, a.Role, a.Model)
			_, err = tx.ExecContext(ctx, `
				INSERT INTO cognitive_nodes (id, label, type, properties, source)
				VALUES (?, ?, 'who', ?, 'system_agent')
				ON CONFLICT(id) DO UPDATE SET
					label=excluded.label,
					properties=excluded.properties`,
				nodeID, "Agent: "+a.DisplayName, props)
			if err != nil {
				return fmt.Errorf("insert agent node: %w", err)
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO cognitive_edges (from_id, to_id, relation_type, strength)
				VALUES (?, 'flowork', 'member_of', 1.0)
				ON CONFLICT(from_id, to_id, relation_type) DO UPDATE SET strength=1.0`,
				nodeID)
			if err != nil {
				return fmt.Errorf("insert agent edge: %w", err)
			}
		}
	}

	// 3. Sync Constitution
	rows, err = tx.QueryContext(ctx, "SELECT id, section, content FROM constitution WHERE deleted_at IS NULL")
	if err == nil {
		type dbRule struct {
			ID      int
			Section string
			Content string
		}
		var rules []dbRule
		for rows.Next() {
			var r dbRule
			if err := rows.Scan(&r.ID, &r.Section, &r.Content); err == nil {
				rules = append(rules, r)
			}
		}
		rows.Close()

		for _, r := range rules {
			nodeID := fmt.Sprintf("constitution_rule_%d", r.ID)
			preview := r.Content
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			props := fmt.Sprintf(`{"section": %q, "content": %q}`, r.Section, preview)
			_, err = tx.ExecContext(ctx, `
				INSERT INTO cognitive_nodes (id, label, type, properties, source)
				VALUES (?, ?, 'concept', ?, 'system_constitution')
				ON CONFLICT(id) DO UPDATE SET
					label=excluded.label,
					properties=excluded.properties`,
				nodeID, "Rule: "+r.Section, props)
			if err != nil {
				return fmt.Errorf("insert constitution node: %w", err)
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO cognitive_edges (from_id, to_id, relation_type, strength)
				VALUES (?, 'flowork', 'governs', 1.0)
				ON CONFLICT(from_id, to_id, relation_type) DO UPDATE SET strength=1.0`,
				nodeID)
			if err != nil {
				return fmt.Errorf("insert constitution edge: %w", err)
			}
		}
	}

	// 4. Sync Instincts / Skills
	rows, err = tx.QueryContext(ctx, "SELECT skill_name, agent_id, content FROM skills WHERE deleted_at IS NULL AND active = 1")
	if err == nil {
		type dbSkill struct {
			Name    string
			AgentID int
			Content string
		}
		var skills []dbSkill
		for rows.Next() {
			var s dbSkill
			if err := rows.Scan(&s.Name, &s.AgentID, &s.Content); err == nil {
				skills = append(skills, s)
			}
		}
		rows.Close()

		for _, s := range skills {
			nodeID := "skill_" + s.Name
			preview := s.Content
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			props := fmt.Sprintf(`{"content_preview": %q}`, preview)
			_, err = tx.ExecContext(ctx, `
				INSERT INTO cognitive_nodes (id, label, type, properties, source)
				VALUES (?, ?, 'skill', ?, 'system_skill')
				ON CONFLICT(id) DO UPDATE SET
					label=excluded.label,
					properties=excluded.properties`,
				nodeID, "Skill: "+s.Name, props)
			if err != nil {
				return fmt.Errorf("insert skill node: %w", err)
			}

			var agentName string
			_ = tx.QueryRowContext(ctx, "SELECT name FROM agents WHERE id = ?", s.AgentID).Scan(&agentName)
			targetNode := "flowork"
			if agentName != "" {
				targetNode = "agent_" + agentName
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO cognitive_edges (from_id, to_id, relation_type, strength)
				VALUES (?, ?, 'belongs_to', 1.0)
				ON CONFLICT(from_id, to_id, relation_type) DO UPDATE SET strength=1.0`,
				nodeID, targetNode)
			if err != nil {
				return fmt.Errorf("insert skill edge: %w", err)
			}
		}
	}

	// 5. Sync Personas (prompt_templates)
	rows, err = tx.QueryContext(ctx, "SELECT name, content FROM prompt_templates")
	if err == nil {
		type dbPersona struct {
			Name    string
			Content string
		}
		var personas []dbPersona
		for rows.Next() {
			var p dbPersona
			if err := rows.Scan(&p.Name, &p.Content); err == nil {
				personas = append(personas, p)
			}
		}
		rows.Close()

		for _, p := range personas {
			nodeID := "persona_" + p.Name
			preview := p.Content
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			props := fmt.Sprintf(`{"prompt_preview": %q}`, preview)
			_, err = tx.ExecContext(ctx, `
				INSERT INTO cognitive_nodes (id, label, type, properties, source)
				VALUES (?, ?, 'who', ?, 'system_persona')
				ON CONFLICT(id) DO UPDATE SET
					label=excluded.label,
					properties=excluded.properties`,
				nodeID, "Persona: "+p.Name, props)
			if err != nil {
				return fmt.Errorf("insert persona node: %w", err)
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO cognitive_edges (from_id, to_id, relation_type, strength)
				VALUES (?, 'flowork', 'persona_of', 1.0)
				ON CONFLICT(from_id, to_id, relation_type) DO UPDATE SET strength=1.0`,
				nodeID)
			if err != nil {
				return fmt.Errorf("insert persona edge: %w", err)
			}
		}
	}

	return nil
}
