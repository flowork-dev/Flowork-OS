// fwswitch_boot.go — GROWTH-POINT (NON-frozen). Boot plug-and-play settings (lihat
// internal/fwswitch). init() jalan SEBELUM main() → switch fitur GUI (file lintas-proses)
// ke-apply ke os.Setenv sebelum server router nyala. main.go ga disentuh (frozen-safe).
package main

import "github.com/flowork-os/flowork_Router/internal/fwswitch"

func init() { fwswitch.Boot() }
