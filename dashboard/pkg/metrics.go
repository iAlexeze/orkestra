package dashboard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Metrics API structures
type CRDMetrics struct {
    Name              string                 `json:"name"`
    QueueDepth        int                    `json:"queueDepth"`
    WorkersActive     int                    `json:"workersActive"`
    ResourceCount     int                    `json:"resourceCount"`
    ReconcileTotal    ReconcileTotalMetrics  `json:"reconcileTotal"`
    ReconcileDuration ReconcileDurationMetrics `json:"reconcileDuration"`
    Conversion        ConversionMetrics      `json:"conversion"`
}

type ReconcileTotalMetrics struct {
    Total   int64 `json:"total"`
    Success int64 `json:"success"`
    Error   int64 `json:"error"`
}

type ReconcileDurationMetrics struct {
    Count   int64   `json:"count"`
    Sum     float64 `json:"sum"`
    Avg     float64 `json:"avg"`
    P50     float64 `json:"p50"`
    P95     float64 `json:"p95"`
    P99     float64 `json:"p99"`
}

type ConversionMetrics struct {
    TotalRequests   int64                   `json:"totalRequests"`
    Success         int64                   `json:"success"`
    Failures        int64                   `json:"failures"`
    AvgLatencyMs    float64                 `json:"avgLatencyMs"`
    P95LatencyMs    float64                 `json:"p95LatencyMs"`
    ByDirection     map[string]*ConversionDirection `json:"byDirection"`
}

type ConversionDirection struct {
    Count    int64   `json:"count"`
    AvgMs    float64 `json:"avgMs"`
    P95Ms    float64 `json:"p95Ms"`
}

type SystemMetrics struct {
    Timestamp     int64            `json:"timestamp"`
    CRDs          []CRDMetrics     `json:"crds"`
    GoMetrics     GoMetrics        `json:"go"`
    ProcessMetrics ProcessMetrics  `json:"process"`
}

type GoMetrics struct {
    Goroutines    int     `json:"goroutines"`
    HeapAlloc     uint64  `json:"heapAlloc"`
    HeapSys       uint64  `json:"heapSys"`
    GCPercent     int     `json:"gcPercent"`
    GCTotalPause  float64 `json:"gcTotalPause"`
}

type ProcessMetrics struct {
    CPU          float64 `json:"cpuSeconds"`
    Memory       uint64  `json:"memoryBytes"`
    OpenFDs      int     `json:"openFDs"`
    StartTime    int64   `json:"startTime"`
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
    systemMetrics := parseMetrics(string(metricsText))
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(systemMetrics)
}

func parseMetrics(metricsText string) SystemMetrics {
    metrics := SystemMetrics{
        Timestamp: time.Now().Unix(),
        CRDs:      []CRDMetrics{},
    }
    
    // Parse controller metrics
    crdMetrics := make(map[string]*CRDMetrics)
    
    lines := strings.Split(metricsText, "\n")
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        // Parse metric lines
        parseMetricLine(line, crdMetrics)
    }
    
    // Convert map to slice
    for _, cm := range crdMetrics {
        metrics.CRDs = append(metrics.CRDs, *cm)
    }
    
    // Parse Go and Process metrics
    parseGoMetrics(metricsText, &metrics)
    parseProcessMetrics(metricsText, &metrics)
    
    return metrics
}

func parseMetricLine(line string, crdMetrics map[string]*CRDMetrics) {
    // Format: metric_name{labels} value
    parts := strings.SplitN(line, " ", 2)
    if len(parts) != 2 {
        return
    }
    
    metricPart := parts[0]
    valuePart := parts[1]
    
    // Extract metric name and labels
    openBrace := strings.Index(metricPart, "{")
    if openBrace == -1 {
        return
    }
    
    metricName := metricPart[:openBrace]
    labelsPart := metricPart[openBrace+1 : len(metricPart)-1]
    
    // Parse labels
    labels := parseLabels(labelsPart)
    crdName := labels["crd"]
    if crdName == "" {
        return
    }
    
    value, _ := strconv.ParseFloat(valuePart, 64)
    
    // Initialize CRD metrics if not exists
    if _, ok := crdMetrics[crdName]; !ok {
        crdMetrics[crdName] = &CRDMetrics{
            Name: crdName,
            ReconcileDuration: ReconcileDurationMetrics{},
            Conversion: ConversionMetrics{
                ByDirection: make(map[string]*ConversionDirection),
            },
        }
    }
    
    cm := crdMetrics[crdName]
    
    switch metricName {
    case "controller_queue_depth":
        cm.QueueDepth = int(value)
    case "controller_workers_active":
        cm.WorkersActive = int(value)
    case "controller_resource_count":
        cm.ResourceCount = int(value)
    case "controller_reconcile_total":
        result := labels["result"]
        total := int64(value)
        cm.ReconcileTotal.Total += total
        if result == "success" {
            cm.ReconcileTotal.Success += total
        } else if result == "error" {
            cm.ReconcileTotal.Error += total
        }
    case "controller_reconcile_duration_seconds_count":
        cm.ReconcileDuration.Count = int64(value)
    case "controller_reconcile_duration_seconds_sum":
        cm.ReconcileDuration.Sum = value
        if cm.ReconcileDuration.Count > 0 {
            cm.ReconcileDuration.Avg = cm.ReconcileDuration.Sum / float64(cm.ReconcileDuration.Count)
        }
    case "orkestra_conversion_requests_total":
        fromVer := labels["from_version"]
        toVer := labels["to_version"]
        kind := labels["kind"]
        result := labels["result"]
        
        direction := fmt.Sprintf("%s->%s (%s)", fromVer, toVer, kind)
        if _, ok := cm.Conversion.ByDirection[direction]; !ok {
            cm.Conversion.ByDirection[direction] = &ConversionDirection{}
        }
        
        count := int64(value)
        if result == "success" {
            cm.Conversion.Success += count
        } else {
            cm.Conversion.Failures += count
        }
        cm.Conversion.TotalRequests += count
        cm.Conversion.ByDirection[direction].Count += count
    case "orkestra_conversion_duration_seconds_sum":
        fromVer := labels["from_version"]
        toVer := labels["to_version"]
        kind := labels["kind"]
        direction := fmt.Sprintf("%s->%s (%s)", fromVer, toVer, kind)
        
        if _, ok := cm.Conversion.ByDirection[direction]; !ok {
            cm.Conversion.ByDirection[direction] = &ConversionDirection{}
        }
        
        sum := value
        count := int64(0)
        // We'll need to find the count separately
        cm.Conversion.ByDirection[direction].AvgMs = sum / float64(max(1, count)) * 1000
    }
}

func parseLabels(labelStr string) map[string]string {
    labels := make(map[string]string)
    
    // Split by commas but respect quotes
    parts := strings.Split(labelStr, ",")
    for _, part := range parts {
        kv := strings.SplitN(part, "=", 2)
        if len(kv) == 2 {
            key := strings.TrimSpace(kv[0])
            value := strings.Trim(kv[1], "\"")
            labels[key] = value
        }
    }
    
    return labels
}

func parseGoMetrics(text string, metrics *SystemMetrics) {
    reGoroutines := regexp.MustCompile(`go_goroutines (\d+)`)
    reHeapAlloc := regexp.MustCompile(`go_memstats_heap_alloc_bytes (\d+)`)
    reHeapSys := regexp.MustCompile(`go_memstats_heap_sys_bytes (\d+)`)
    reGCPercent := regexp.MustCompile(`go_gc_gogc_percent (\d+)`)
    reGCPause := regexp.MustCompile(`go_gc_duration_seconds_sum ([\d\.]+)`)
    
    if matches := reGoroutines.FindStringSubmatch(text); len(matches) > 1 {
        metrics.GoMetrics.Goroutines, _ = strconv.Atoi(matches[1])
    }
    if matches := reHeapAlloc.FindStringSubmatch(text); len(matches) > 1 {
        metrics.GoMetrics.HeapAlloc, _ = strconv.ParseUint(matches[1], 10, 64)
    }
    if matches := reHeapSys.FindStringSubmatch(text); len(matches) > 1 {
        metrics.GoMetrics.HeapSys, _ = strconv.ParseUint(matches[1], 10, 64)
    }
    if matches := reGCPercent.FindStringSubmatch(text); len(matches) > 1 {
        metrics.GoMetrics.GCPercent, _ = strconv.Atoi(matches[1])
    }
    if matches := reGCPause.FindStringSubmatch(text); len(matches) > 1 {
        metrics.GoMetrics.GCTotalPause, _ = strconv.ParseFloat(matches[1], 64)
    }
}

func parseProcessMetrics(text string, metrics *SystemMetrics) {
    reCPU := regexp.MustCompile(`process_cpu_seconds_total ([\d\.]+)`)
    reMemory := regexp.MustCompile(`process_resident_memory_bytes (\d+)`)
    reOpenFDs := regexp.MustCompile(`process_open_fds (\d+)`)
    reStartTime := regexp.MustCompile(`process_start_time_seconds (\d+)`)
    
    if matches := reCPU.FindStringSubmatch(text); len(matches) > 1 {
        metrics.ProcessMetrics.CPU, _ = strconv.ParseFloat(matches[1], 64)
    }
    if matches := reMemory.FindStringSubmatch(text); len(matches) > 1 {
        metrics.ProcessMetrics.Memory, _ = strconv.ParseUint(matches[1], 10, 64)
    }
    if matches := reOpenFDs.FindStringSubmatch(text); len(matches) > 1 {
        metrics.ProcessMetrics.OpenFDs, _ = strconv.Atoi(matches[1])
    }
    if matches := reStartTime.FindStringSubmatch(text); len(matches) > 1 {
        metrics.ProcessMetrics.StartTime, _ = strconv.ParseInt(matches[1], 10, 64)
    }
}

func max(a, b int64) int64 {
    if a > b {
        return a
    }
    return b
}