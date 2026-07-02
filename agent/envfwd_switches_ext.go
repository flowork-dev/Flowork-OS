// envfwd_switches_ext.go — SIBLING ext (deletable, non-frozen): colok daftar
// switch GUI/env yang DIBACA DI DALAM guest WASM ke papan envfwd_seam.go →
// nilai GUI (fwswitch udah os.Setenv di host) beneran nyampe os.Getenv guest.
// Nambah key guest-side baru = tambah 1 baris di sini. 📄 Dok: lock/llm-timeout.md
package main

func init() {
	RegisterEnvForward(func(string) []string {
		return []string{
			// switch GUI (fwswitch registry) yang call-site-nya di guest:
			"FLOWORK_LLM_TIMEOUT_MS", // mr-flow: timeout call LLM→router
			"FLOWORK_ROUTER_RETRY",   // mr-flow + agentkit: attempt retry transient
			"FLOWORK_PARALLEL_TOOLS", // mr-flow: parallel_tool_calls
			"FLOWORK_PROMPT_CACHE",   // mr-flow: cache_control Claude
			"FLOWORK_TOOL_RESULT_MAX", // mr-flow: cap hasil tool
			// env-only (bukan registry) yang juga dibaca guest:
			"FLOWORK_TG_FORMAT",
			"FLOWORK_TG_CHUNK",
			"FLOWORK_TG_MEDIA",
			"FLOWORK_SELF_HANDLE_PHRASES",
		}
	})
}
