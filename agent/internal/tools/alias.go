// === LOCKED FILE ===
// Status: STABLE — DO NOT MODIFY without owner approval.
// Owner: Aola Sahidin (Mr.Dev)
// Repo: https://github.com/flowork-os/Flowork-OS
// Locked at: 2026-06-14
// Reason: Alias layer (surface vocabulary → canonical) VERIFIED via mr-flow:
//   Read→file_read + Write→file_write round-trip OK; advertising (DisplayName)
//   confirmed for all 12 renamed tools. Edit nama tool baru → tambah entry di
//   surfaceToCanonical + canonToSurface, JANGAN ubah logic resolve.
//
// alias.go — alias layer for tool names.
//
// PURPOSE:
//   The local model is distilled on agent-style traces and an external driver
//   model (via router) both speak a common, ergonomic tool vocabulary
//   (Read/Write/Edit/Bash/...). The kernel keeps its own canonical internal
//   names (file_read/file_write/edit/bash/...). This file bridges the two:
//
//     - canonicalToolName: incoming name (model/driver vocabulary, any case) →
//       canonical registry name. Used by registry.go Lookup (alias layer).
//     - DisplayName: canonical name → the name advertised to the LLM in the
//       tool schema. Used by agentmgr/tool_specs.go.
//
//   DESIGN NOTE: identity lives in the SOUL (persona/constitution), NOT in tool
//   names. Tool names are plumbing. Renaming the surface vocabulary changes
//   nothing about who the agent is. Internal implementations keep their names,
//   so every capability/tier/permission gate (which classifies by the resolved
//   tool's Name()) keeps working unchanged.
//
//   FORGIVING: case-insensitive on aliases, trims whitespace. Names not in the
//   alias table pass through untouched — including mcp__* tools (kept verbatim).

package tools

import "strings"

// surfaceToCanonical — surface vocabulary name (lowercased) → canonical registry
// name. Only general-purpose tools that have a canonical Flowork twin are mapped.
// Flowork-native tools (brain_add, scanner, ...) have no surface alias by design.
var surfaceToCanonical = map[string]string{
	"read":            "file_read",
	"write":           "file_write",
	"edit":            "edit",
	"bash":            "bash",
	"grep":            "grep",
	"glob":            "glob",
	"websearch":       "web_search",
	"webfetch":        "webfetch",
	"toolsearch":      "tool_search",
	"skill":           "skill",
	"agent":           "agent_command",
	"askuserquestion": "askuser",
}

// canonToSurface — reverse map: canonical name → surface name advertised to the
// LLM. Only the renamed tools differ; everything else keeps its native name.
var canonToSurface = map[string]string{
	"file_read":     "Read",
	"file_write":    "Write",
	"edit":          "Edit",
	"bash":          "Bash",
	"grep":          "Grep",
	"glob":          "Glob",
	"web_search":    "WebSearch",
	"webfetch":      "WebFetch",
	"tool_search":   "ToolSearch",
	"skill":         "Skill",
	"agent_command": "Agent",
	"askuser":       "AskUserQuestion",
}

// canonicalToolName resolves a possibly-surface (or mis-cased) tool name to the
// canonical registry name. If no alias matches, returns the trimmed input so the
// caller's exact registry lookup still runs. Pure (no lock) — safe to call from
// inside Lookup's read lock.
func canonicalToolName(name string) string {
	n := strings.TrimSpace(name)
	if c, ok := surfaceToCanonical[strings.ToLower(n)]; ok {
		return c
	}
	return n
}

// DisplayName returns the name advertised to the LLM for a canonical tool name —
// surface vocabulary for the renamed tools, native otherwise.
func DisplayName(canonical string) string {
	if d, ok := canonToSurface[canonical]; ok {
		return d
	}
	return canonical
}
