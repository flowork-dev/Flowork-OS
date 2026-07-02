// allow_health_ext.go — SIBLING ext (deletable): colok /api/health ke papan
// allowlist loopback (allow_seam.go). Dokumen rilis + tutorial portable ngerujuk
// endpoint ini (F-F); loopback-only GET, tanpa sesi. 📄 Dok: lock/approval-gate.md
package floworkauth

func init() { RegisterLoopbackPublic("/api/health", "GET") }
