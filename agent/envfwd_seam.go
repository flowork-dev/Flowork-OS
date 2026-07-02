// envfwd_seam.go — papan colokan (Pola A) daftar env key yang diforward ke guest WASM.
// 📄 Dok: FLowork_os/lock/llm-timeout.md
//
// AKAR (2026-07-02): kernelhost.EnvForwardKeys cuma 1 slot & main.go (frozen)
// hard-assign 1 sumber (connections) → env switch GUI FLOWORK_* yang dibaca DI
// DALAM guest (mr-flow/agentkit: FLOWORK_ROUTER_RETRY, FLOWORK_PARALLEL_TOOLS,
// FLOWORK_LLM_TIMEOUT_MS, dst) GA PERNAH nyampe guest — wazero cuma inject
// i.env hasil buildAgentEnv, BUKAN environ host → switch GUI diem-diem ga
// ngefek di guest (ngelanggar "GUI = kebenaran"). Papan ini ngegabung banyak
// sumber; sumber baru dicolok via file sibling _ext.go (deletable) →
// RegisterEnvForward. Semua ext dihapus → papan kosong KECUALI connections
// (dicolok di init() sini = perilaku lama utuh, delete-test aman).
//
// Catatan: buildAgentEnv jalan saat agent LOAD/RELOAD — ganti switch guest-side
// di GUI butuh restart stack / reload agent biar kebaca guest.
package main

import "flowork-gui/internal/connections"

var envFwdFuncs []func(agentID string) []string

// RegisterEnvForward — colok 1 sumber daftar env key buat diforward ke guest.
func RegisterEnvForward(f func(agentID string) []string) {
	if f != nil {
		envFwdFuncs = append(envFwdFuncs, f)
	}
}

// mergedEnvForwardKeys — nilai kernelhost.EnvForwardKeys (di-assign main.go).
// Gabungan semua sumber; buildAgentEnv tetep cuma forward key yang nilainya
// non-kosong di env host.
func mergedEnvForwardKeys(agentID string) []string {
	var out []string
	for _, f := range envFwdFuncs {
		out = append(out, f(agentID)...)
	}
	return out
}

func init() { RegisterEnvForward(connections.GlobalSecretEnvKeys) }
