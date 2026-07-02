// secscrub_ext.go — SIBLING ext (deletable): colok scrubber rahasia rekursif
// (internal/secscrub) ke seam sink DB kernelhost (SanitizeLogged*). File ini
// dihapus → sink balik apa-adanya (default aman), core tetep build.
// 📄 Dok: FLowork_os/lock/secscrub.md
package main

import (
	"flowork-gui/internal/agentdb"
	"flowork-gui/internal/kernelhost"
	"flowork-gui/internal/secscrub"
)

func init() {
	// AKAR: chokepoint agentdb — SEMUA jalur sink (kernelhost, agentmgr,
	// builtins, slashcmd, runtime host) ke-cover sekali di sini.
	agentdb.SanitizeText = secscrub.RedactString
	agentdb.SanitizeMeta = secscrub.RedactMap
	// Lapisan kedua (defense in depth) di choke-point invoke kernelhost.
	kernelhost.SanitizeLogged = secscrub.RedactString
	kernelhost.SanitizeLoggedMap = secscrub.RedactMap
}
