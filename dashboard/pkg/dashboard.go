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
    if strings.HasPrefix(path, "/crd/") {
        name := strings.TrimPrefix(path, "/crd/")
        d.handleCRD(w, r, name)
        return
    }
    if path == "/status" {
        d.handleStatus(w, r)
        return
    }
    if path == "/metrics" {
        d.handleMetrics(w, r)
        return
    }
    // serve static files
    http.FileServer(http.FS(assets)).ServeHTTP(w, r)
}

func (d *Dashboard) handleIndex(w http.ResponseWriter, r *http.Request) {
    katalog, err := d.client.FetchKatalog()
    if err != nil {
        http.Error(w, "Failed to fetch katalog: "+err.Error(), http.StatusInternalServerError)
        return
    }

    tmpl := template.Must(template.ParseFS(assets, "templates/index.html"))
    tmpl.Execute(w, map[string]interface{}{
        "CRDs": katalog.CRDs,
    })
}

func (d *Dashboard) handleCRD(w http.ResponseWriter, r *http.Request, name string) {
    crd, err := d.client.FetchCRDDetail(name)
    if err != nil {
        http.Error(w, "Failed to fetch CRD: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Parse the template fresh for each request (or cache it)
    tmpl, err := template.ParseFS(assets, "templates/crd.html")
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

func (d *Dashboard) handleMetrics(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFS(assets, "templates/metrics.html"))
    tmpl.Execute(w, nil)
}