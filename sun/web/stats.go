package web

import (
	"fmt"
	"math"
	"net/http"
	"runtime"
	"time"

	"github.com/labstack/echo"
)

func (s *Server) stats(c echo.Context) error {
	updateSystemStatus()
	return c.JSON(http.StatusOK, sysStatus)
}

// https://github.com/gogits/gogs/blob/master/pkg/tool/file.go
var (
	startTime = time.Now()
	sysStatus struct {
		Uptime       string `json:"uptime"`
		NumGoroutine int    `json:"num_goroutine"`

		// General statistics.
		MemAllocated string `json:"mem_allocated"` // bytes allocated and still in use
		MemTotal     string `json:"mem_total"`     // bytes allocated (even if freed)
		MemSys       string `json:"mem_sys"`       // bytes obtained from system (sum of XxxSys below)
		Lookups      uint64 `json:"lookups"`       // number of pointer lookups
		MemMallocs   uint64 `json:"mem_mallocs"`   // number of mallocs
		MemFrees     uint64 `json:"mem_frees"`     // number of frees

		// Main allocation heap statistics.
		HeapAlloc    string `json:"heap_alloc"`    // bytes allocated and still in use
		HeapSys      string `json:"heap_sys"`      // bytes obtained from system
		HeapIdle     string `json:"heap_idle"`     // bytes in idle spans
		HeapInuse    string `json:"heap_inuse"`    // bytes in non-idle span
		HeapReleased string `json:"heap_released"` // bytes released to the OS
		HeapObjects  uint64 `json:"heap_objects"`  // total number of allocated objects

		// Low-level fixed-size structure allocator statistics.
		//	Inuse is bytes used now.
		//	Sys is bytes obtained from system.
		StackInuse  string `json:"stack_inuse"` // bootstrap stacks
		StackSys    string `json:"stack_sys"`
		MSpanInuse  string `json:"mspan_inuse"` // mspan structures
		MSpanSys    string `json:"mspan_sys"`
		MCacheInuse string `json:"mcache_inuse"` // mcache structures
		MCacheSys   string `json:"mcache_sys"`
		BuckHashSys string `json:"buck_hash_sys"` // profiling bucket hash table
		GCSys       string `json:"gc_sys"`        // GC metadata
		OtherSys    string `json:"other_sys"`     // other system allocations

		// Garbage collector statistics.
		NextGC       string `json:"next_gc"` // next run in HeapAlloc time (bytes)
		LastGC       string `json:"last_gc"` // last run in absolute time (ns)
		PauseTotalNs string `json:"pause_total_ns"`
		PauseNs      string `json:"pause_ns"` // circular buffer of recent GC pause times, most recent at [(NumGC+255)%256]
		NumGC        uint32 `json:"num_gc"`
	}
	sizes = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
)

func updateSystemStatus() {
	sysStatus.Uptime = fmt.Sprintf("%v", time.Now().Sub(startTime))

	m := new(runtime.MemStats)
	runtime.ReadMemStats(m)
	sysStatus.NumGoroutine = runtime.NumGoroutine()

	sysStatus.MemAllocated = humanateBytes(uint64(m.Alloc))
	sysStatus.MemTotal = humanateBytes(uint64(m.TotalAlloc))
	sysStatus.MemSys = humanateBytes(uint64(m.Sys))
	sysStatus.Lookups = m.Lookups
	sysStatus.MemMallocs = m.Mallocs
	sysStatus.MemFrees = m.Frees

	sysStatus.HeapAlloc = humanateBytes(uint64(m.HeapAlloc))
	sysStatus.HeapSys = humanateBytes(uint64(m.HeapSys))
	sysStatus.HeapIdle = humanateBytes(uint64(m.HeapIdle))
	sysStatus.HeapInuse = humanateBytes(uint64(m.HeapInuse))
	sysStatus.HeapReleased = humanateBytes(uint64(m.HeapReleased))
	sysStatus.HeapObjects = m.HeapObjects

	sysStatus.StackInuse = humanateBytes(uint64(m.StackInuse))
	sysStatus.StackSys = humanateBytes(uint64(m.StackSys))
	sysStatus.MSpanInuse = humanateBytes(uint64(m.MSpanInuse))
	sysStatus.MSpanSys = humanateBytes(uint64(m.MSpanSys))
	sysStatus.MCacheInuse = humanateBytes(uint64(m.MCacheInuse))
	sysStatus.MCacheSys = humanateBytes(uint64(m.MCacheSys))
	sysStatus.BuckHashSys = humanateBytes(uint64(m.BuckHashSys))
	sysStatus.GCSys = humanateBytes(uint64(m.GCSys))
	sysStatus.OtherSys = humanateBytes(uint64(m.OtherSys))

	sysStatus.NextGC = humanateBytes(uint64(m.NextGC))
	sysStatus.LastGC = fmt.Sprintf("%.1fs", float64(time.Now().UnixNano()-int64(m.LastGC))/1000/1000/1000)
	sysStatus.PauseTotalNs = fmt.Sprintf("%.1fs", float64(m.PauseTotalNs)/1000/1000/1000)
	sysStatus.PauseNs = fmt.Sprintf("%.3fs", float64(m.PauseNs[(m.NumGC+255)%256])/1000/1000/1000)
	sysStatus.NumGC = m.NumGC
}

func humanateBytes(n uint64) string {
	if n < 10 {
		return fmt.Sprintf("%d B", n)
	}

	e := math.Floor(logn(float64(n), 1024))
	suffix := sizes[int(e)]
	val := float64(n) / math.Pow(1024, math.Floor(e))
	f := "%.0f"
	if val < 10 {
		f = "%.1f"
	}

	return fmt.Sprintf(f+" %s", val, suffix)
}

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}
