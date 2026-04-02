package dashboard

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
)

type SimpleSystemMetrics struct {
	Goroutines int     `json:"goroutines"`
	Threads    int     `json:"threads"`
	Gomaxprocs int     `json:"gomaxprocs"`
	HeapAlloc  uint64  `json:"heapAlloc"`
	HeapSys    uint64  `json:"heapSys"`
	GCPauseAvg float64 `json:"gcPauseAvg"`
}

func (d *Dashboard) handleMetricsAPI(w http.ResponseWriter, r *http.Request) {
	// Fetch metrics from the orkestra metrics endpoint
	resp, err := http.Get(d.client.baseURL + "/metrics")
	if err != nil {
		http.Error(w, "Failed to fetch metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	metricsText, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse metrics
	systemMetrics := parseSimpleSystemMetrics(string(metricsText))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(systemMetrics)
}

func parseSimpleSystemMetrics(text string) SimpleSystemMetrics {
	metrics := SimpleSystemMetrics{}

	// Parse goroutines
	reGoroutines := regexp.MustCompile(`go_goroutines (\d+)`)
	if matches := reGoroutines.FindStringSubmatch(text); len(matches) > 1 {
		metrics.Goroutines, _ = strconv.Atoi(matches[1])
	}

	// Parse threads
	reThreads := regexp.MustCompile(`go_threads (\d+)`)
	if matches := reThreads.FindStringSubmatch(text); len(matches) > 1 {
		metrics.Threads, _ = strconv.Atoi(matches[1])
	}

	// Parse GOMAXPROCS
	reGomaxprocs := regexp.MustCompile(`go_sched_gomaxprocs_threads (\d+)`)
	if matches := reGomaxprocs.FindStringSubmatch(text); len(matches) > 1 {
		metrics.Gomaxprocs, _ = strconv.Atoi(matches[1])
	}

	// Parse heap alloc
	reHeapAlloc := regexp.MustCompile(`go_memstats_heap_alloc_bytes (\d+)`)
	if matches := reHeapAlloc.FindStringSubmatch(text); len(matches) > 1 {
		metrics.HeapAlloc, _ = strconv.ParseUint(matches[1], 10, 64)
	}

	// Parse heap sys
	reHeapSys := regexp.MustCompile(`go_memstats_heap_sys_bytes (\d+)`)
	if matches := reHeapSys.FindStringSubmatch(text); len(matches) > 1 {
		metrics.HeapSys, _ = strconv.ParseUint(matches[1], 10, 64)
	}

	// Parse GC pause average
	reGCPauseCount := regexp.MustCompile(`go_gc_duration_seconds_count (\d+)`)
	reGCPauseSum := regexp.MustCompile(`go_gc_duration_seconds_sum ([\d\.]+)`)

	var gcCount int64
	var gcSum float64

	if matches := reGCPauseCount.FindStringSubmatch(text); len(matches) > 1 {
		gcCount, _ = strconv.ParseInt(matches[1], 10, 64)
	}
	if matches := reGCPauseSum.FindStringSubmatch(text); len(matches) > 1 {
		gcSum, _ = strconv.ParseFloat(matches[1], 64)
	}

	if gcCount > 0 {
		metrics.GCPauseAvg = gcSum / float64(gcCount)
	}

	return metrics
}
