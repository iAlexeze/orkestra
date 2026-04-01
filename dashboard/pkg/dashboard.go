package dashboard

import (
    "embed"
    "html/template"
    "net/http"
    "strings"
)

//go:embed templates/*.html static/*
var assets embed.FS

type Dashboard struct {
    client *Client
}

func New(orkestraURL string) *Dashboard {
    return &Dashboard{
        client: NewClient(orkestraURL),
    }
}

func (d *Dashboard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    path := strings.TrimPrefix(r.URL.Path, "/dashboard")
    if path == "" || path == "/" {
        d.handleIndex(w, r)
        return
    }

    // CRD endpoints
    if strings.HasPrefix(path, "/crd/") {
        name := strings.TrimPrefix(path, "/crd/")
        d.handleCRD(w, r, name)
        return
    }

    // API endpoints for metrics data
    if strings.HasPrefix(path, "/api/metrics") {
        d.handleMetricsAPI(w, r)
        return
    }

    // Status page
    if path == "/status" {
        d.handleStatus(w, r)
        return
    }
    if path == "/metrics" {
        d.handleMetricsPage(w, r)
        return
    }
    // serve static files
    http.FileServer(http.FS(assets)).ServeHTTP(w, r)
}

type IndexData struct {
    CRDs          []CRDSummary
    TotalCRDs     int
    TotalWorkers  int
    TotalResources int
    HealthyCount  int
}

// Define template functions once and reuse them
var templateFuncs = template.FuncMap{
    "mul": func(a, b int) int {
        return a * b
    },
    "div": func(a, b int) int {
        if b == 0 {
            return 0
        }
        return a / b
    },
    "title": strings.Title,
}

func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
    katalog, err := d.client.FetchKatalog()
    if err != nil {
        http.Error(w, "Failed to fetch katalog: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Calculate summary statistics
    totalCRDs := len(katalog.CRDs)
    totalWorkers := 0
    totalResources := 0
    healthyCount := 0

    for _, crd := range katalog.CRDs {
        totalWorkers += crd.Workers
        totalResources += crd.ResourceCount
        if crd.Healthy {
            healthyCount++
        }
    }

    // Prepare data for template
    data := IndexData{
        CRDs:          katalog.CRDs,
        TotalCRDs:     totalCRDs,
        TotalWorkers:  totalWorkers,
        TotalResources: totalResources,
        HealthyCount:  healthyCount,
    }

    // Parse template with function map
    tmpl := template.New("index-v1.html").Funcs(templateFuncs)
    tmpl, err = tmpl.ParseFS(assets, "templates/index-v1.html")
    if err != nil {
        http.Error(w, "Failed to parse template: "+err.Error(), http.StatusInternalServerError)
        return
    }

    err = tmpl.Execute(w, data)
    if err != nil {
        http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
        return
    }
}

func (d *Dashboard) handleCRD(w http.ResponseWriter, r *http.Request, name string) {
    crd, err := d.client.FetchCRDDetail(name)
    if err != nil {
        http.Error(w, "Failed to fetch CRD: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Parse the template with the same function map
    tmpl := template.New("crd-v1.html").Funcs(templateFuncs)
    tmpl, err = tmpl.ParseFS(assets, "templates/crd-v1.html")
    if err != nil {
        http.Error(w, "Failed to parse template: "+err.Error(), http.StatusInternalServerError)
        return
    }

    err = tmpl.Execute(w, map[string]interface{}{
        "CRD": crd,
    })
    if err != nil {
        http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
        return
    }
}

func (d *Dashboard) handleStatus(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFS(assets, "templates/status.html"))
    tmpl.Execute(w, nil)
}

func (d *Dashboard) handleMetricsPage(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFS(assets, "templates/metrics.html"))
    tmpl.Execute(w, nil)
}