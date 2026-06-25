// fwswitch_boot.go — ⚠️ FROZEN 2026-06-26. Boot plug-and-play settings (lihat
// internal/fwswitch). init() jalan SEBELUM main() → switch fitur GUI (file lintas-proses)
// ke-apply ke os.Setenv sebelum server router nyala. main.go ga disentuh (frozen-safe).
package main

import "github.com/flowork-os/flowork_Router/internal/fwswitch"

func init() { fwswitch.Boot() }
