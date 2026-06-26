// Flowork OS — Dev: Aola Sahidin — github.com/flowork-os/Flowork-OS · floworkos.com
// Tab GUI: Tunnel (registry plug-and-play) → dok lock/gui/Tunnel.md  ⚠️ FROZEN — jangan edit.
// Nambah provider tunnel: file sibling tunnel_<x>_ext.go + RegisterTunnelProvider. Cara:
// CARAFREEZE.MD (POLA A) + lock/plug-and-play.md. Pola freeze: lock/frozen-core.md

package main

import "sort"

type TunnelProvider struct {
	Name    string
	Detect  func() bool
	Status  func() map[string]any
	Enable  func(target string) (map[string]any, error)
	Disable func() (map[string]any, error)
}

var tunnelProviders = map[string]TunnelProvider{}

func RegisterTunnelProvider(p TunnelProvider) {
	if p.Name == "" || p.Enable == nil {
		return
	}
	tunnelProviders[p.Name] = p
}

func tunnelProviderNames() []string {
	names := make([]string, 0, len(tunnelProviders))
	for n := range tunnelProviders {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func getTunnelProvider(name string) (TunnelProvider, bool) {
	p, ok := tunnelProviders[name]
	return p, ok
}
