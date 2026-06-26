// sysstatus_ext.go — SYSTEM-AWARENESS (NON-frozen seam). Sisipin kondisi PC + WAKTU sekarang ke
// SETIAP chat → agent SADAR: spek/OS/CPU/GPU/temp/RAM/disk/load + tanggal-jam (biar tau data lama/
// baru, + kalau panas bisa nyaranin jeda). Switch GUI FLOWORK_SYS_STATUS (default ON).
//
// MULTI-OS (Rule 6): kode COMPILE di semua OS (cuma os.ReadFile + exec, no syscall OS-specific).
//   linux/android → /proc + /sys (CPU/RAM/load/temp lengkap). windows → wmic. darwin → sysctl.
//   cores: runtime.NumCPU (semua OS). GPU: nvidia-smi (OS apapun yg ada GPU NVIDIA). disk: df (unix).
package main

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flowork-os/flowork_Router/internal/router"
)

func sysStatusOn() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("FLOWORK_SYS_STATUS"))) {
	case "0", "false", "off", "no":
		return false
	}
	return true
}

func isProcOS() bool { return runtime.GOOS == "linux" || runtime.GOOS == "android" }

// ---- static (cache sekali) ----
var (
	sysStaticOnce sync.Once
	sysOSRel      string
	sysCPUModel   string
	sysRAMTotalGB float64
)

func loadSysStatic() {
	sysStaticOnce.Do(func() {
		sysOSRel = runtime.GOOS
		switch {
		case isProcOS():
			if b, e := os.ReadFile("/proc/sys/kernel/osrelease"); e == nil {
				sysOSRel = runtime.GOOS + " " + strings.TrimSpace(string(b))
			}
			if f, e := os.Open("/proc/cpuinfo"); e == nil {
				sc := bufio.NewScanner(f)
				for sc.Scan() {
					l := sc.Text()
					if strings.HasPrefix(l, "model name") {
						if i := strings.Index(l, ":"); i >= 0 {
							sysCPUModel = strings.TrimSpace(l[i+1:])
							break
						}
					}
				}
				f.Close()
			}
			if mt := readMeminfoKB("MemTotal"); mt > 0 {
				sysRAMTotalGB = float64(mt) / 1024 / 1024
			}
		case runtime.GOOS == "darwin":
			sysCPUModel = strings.TrimSpace(execOut("sysctl", "-n", "machdep.cpu.brand_string"))
			if v, _ := strconv.ParseFloat(strings.TrimSpace(execOut("sysctl", "-n", "hw.memsize")), 64); v > 0 {
				sysRAMTotalGB = v / 1024 / 1024 / 1024
			}
		case runtime.GOOS == "windows":
			sysCPUModel = winWmicVal("cpu", "name")
			if v, _ := strconv.ParseFloat(winWmicVal("ComputerSystem", "TotalPhysicalMemory"), 64); v > 0 {
				sysRAMTotalGB = v / 1024 / 1024 / 1024
			}
		}
	})
}

func execOut(name string, args ...string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	out, e := exec.Command(p, args...).Output()
	if e != nil {
		return ""
	}
	return string(out)
}

// winWmicVal — ambil value dari `wmic <obj> get <prop>` (baris ke-2).
func winWmicVal(obj, prop string) string {
	out := execOut("wmic", obj, "get", prop)
	lines := strings.Split(out, "\n")
	for i, l := range lines {
		if i == 0 {
			continue // header
		}
		if s := strings.TrimSpace(l); s != "" {
			return s
		}
	}
	return ""
}

func readMeminfoKB(key string) int64 {
	f, e := os.Open("/proc/meminfo")
	if e != nil {
		return 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()
		if strings.HasPrefix(l, key+":") {
			if fs := strings.Fields(l); len(fs) >= 2 {
				v, _ := strconv.ParseInt(fs[1], 10, 64)
				return v
			}
		}
	}
	return 0
}

func cpuTempC() float64 {
	if !isProcOS() {
		return 0
	}
	b, e := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if e != nil {
		return 0
	}
	v, _ := strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
	return v / 1000.0
}

func loadAvg() string {
	if isProcOS() {
		if b, e := os.ReadFile("/proc/loadavg"); e == nil {
			if fs := strings.Fields(string(b)); len(fs) >= 1 {
				return fs[0]
			}
		}
		return ""
	}
	if runtime.GOOS == "darwin" {
		// vm.loadavg → "{ 1.20 1.05 0.98 }"
		s := strings.Trim(strings.TrimSpace(execOut("sysctl", "-n", "vm.loadavg")), "{} ")
		if fs := strings.Fields(s); len(fs) >= 1 {
			return fs[0]
		}
	}
	return ""
}

func ramUsedGB() float64 {
	if isProcOS() {
		return sysRAMTotalGB - float64(readMeminfoKB("MemAvailable"))/1024/1024
	}
	if runtime.GOOS == "windows" {
		if free, _ := strconv.ParseFloat(winWmicVal("OS", "FreePhysicalMemory"), 64); free > 0 {
			return sysRAMTotalGB - free/1024/1024 // FreePhysicalMemory = KB
		}
	}
	return 0 // darwin: skip used (vm_stat ribet) → tampil total aja
}

// diskFree — "<free>/<total> GB" root, via df (unix). windows skip.
func diskFree() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	out := execOut("df", "-k", "/")
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return ""
	}
	fs := strings.Fields(lines[1])
	if len(fs) < 4 {
		return ""
	}
	total, _ := strconv.ParseFloat(fs[1], 64)
	avail, _ := strconv.ParseFloat(fs[3], 64)
	if total <= 0 {
		return ""
	}
	return strconv.FormatFloat(avail/1024/1024, 'f', 0, 64) + "/" + strconv.FormatFloat(total/1024/1024, 'f', 0, 64) + " GB free"
}

// GPU via nvidia-smi (cache 30s).
var (
	gpuMu     sync.Mutex
	gpuCache  string
	gpuExpiry time.Time
)

func gpuStatus() string {
	gpuMu.Lock()
	defer gpuMu.Unlock()
	if time.Now().Before(gpuExpiry) {
		return gpuCache
	}
	gpuExpiry = time.Now().Add(30 * time.Second)
	gpuCache = ""
	out := execOut("nvidia-smi", "--query-gpu=name,temperature.gpu,utilization.gpu", "--format=csv,noheader,nounits")
	if out == "" {
		return ""
	}
	parts := strings.Split(strings.TrimSpace(strings.Split(out, "\n")[0]), ",")
	if len(parts) >= 3 {
		gpuCache = strings.TrimSpace(parts[0]) + " " + strings.TrimSpace(parts[1]) + "°C util " + strings.TrimSpace(parts[2]) + "%"
	}
	return gpuCache
}

// systemStatusText — blok [STATUS_PC] live (per-chat).
func systemStatusText() string {
	loadSysStatic()
	now := time.Now()
	var b strings.Builder
	b.WriteString("[STATUS_PC] waktu: " + now.Format("2006-01-02 15:04 MST") + " (UTC " + now.UTC().Format("2006-01-02 15:04") + ")")
	b.WriteString(" | OS: " + sysOSRel)
	cpu := sysCPUModel
	if cpu == "" {
		cpu = "CPU"
	}
	b.WriteString(" | CPU: " + cpu + " ×" + strconv.Itoa(runtime.NumCPU()))
	if l := loadAvg(); l != "" {
		b.WriteString(" load " + l)
	}
	if sysRAMTotalGB > 0 {
		if used := ramUsedGB(); used > 0 {
			b.WriteString(" | RAM: " + strconv.FormatFloat(used, 'f', 1, 64) + "/" + strconv.FormatFloat(sysRAMTotalGB, 'f', 0, 64) + " GB")
		} else {
			b.WriteString(" | RAM: " + strconv.FormatFloat(sysRAMTotalGB, 'f', 0, 64) + " GB")
		}
	}
	if d := diskFree(); d != "" {
		b.WriteString(" | disk: " + d)
	}
	if g := gpuStatus(); g != "" {
		b.WriteString(" | GPU: " + g)
	}
	if t := cpuTempC(); t > 0 {
		b.WriteString(" | CPU " + strconv.FormatFloat(t, 'f', 0, 64) + "°C")
	}
	b.WriteString("\nCatatan: pengetahuan/data lo relatif ke WAKTU di atas — bandingin sama timestamp data (mis. occurred_at di interaction_recall) buat nilai lama/baru, jangan asumsi cutoff. Kalau GPU/CPU temp tinggi (>80°C) atau load berat, hindari kerjaan berat barengan / sarankan jeda biar PC ga overheat.")
	return b.String()
}

// InjectSystemStatus — prepend system message [STATUS_PC] ke req (switch ON & belum ada).
func InjectSystemStatus(req *router.OpenAIRequest) {
	if req == nil || !sysStatusOn() {
		return
	}
	for _, m := range req.Messages {
		if m.Role == "system" && strings.Contains(m.Content, "[STATUS_PC]") {
			return
		}
	}
	req.Messages = append([]router.OpenAIMessage{{Role: "system", Content: systemStatusText()}}, req.Messages...)
}
