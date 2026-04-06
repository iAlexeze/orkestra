package controlcenter

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//go:embed assets/templates/*.html assets/static/* assets/static/css/* assets/static/js/*
var assets embed.FS

const assetsDir = "assets/templates"

// ─────────────────────────────────────────────────────────────────────────────
// Config
// ─────────────────────────────────────────────────────────────────────────────

type Config struct {
	RefreshInterval time.Duration
	LogLevel        string
	Version         string
}

// ─────────────────────────────────────────────────────────────────────────────
// Instance — one connected Orkestra runtime
// ─────────────────────────────────────────────────────────────────────────────

type Instance struct {
	URL     string
	Client  *Client
	Katalog *KatalogResponse
}

// ─────────────────────────────────────────────────────────────────────────────
// ControlCenter
// ─────────────────────────────────────────────────────────────────────────────

type ControlCenter struct {
	urls        []string
	instances   map[string]*Instance // keyed by URL
	mu          sync.RWMutex
	config      Config
	ready       atomic.Bool
	subscribers sync.Map // map[chan struct{}]struct{}
}

// New creates and starts a ControlCenter. Background fetches begin immediately.
func New(urls []string, config Config) *ControlCenter {
	instances := make(map[string]*Instance, len(urls))
	for _, url := range urls {
		instances[url] = &Instance{
			URL:    url,
			Client: NewClient(url, config.RefreshInterval, config.LogLevel),
		}
	}
	cc := &ControlCenter{
		urls:      urls,
		instances: instances,
		config:    config,
	}
	go cc.backgroundFetchLoop()
	return cc
}

func (cc *ControlCenter) IsReady() bool { return cc.ready.Load() }

// ─────────────────────────────────────────────────────────────────────────────
// Background fetch
// ─────────────────────────────────────────────────────────────────────────────

func (cc *ControlCenter) backgroundFetchLoop() {
	cc.fetchAllKatalogs()
	t := time.NewTicker(cc.config.RefreshInterval)
	defer t.Stop()
	for range t.C {
		cc.fetchAllKatalogs()
	}
}

func (cc *ControlCenter) fetchAllKatalogs() {
	cc.mu.Lock()
	anyOK := false
	for _, inst := range cc.instances {
		kat, err := inst.Client.FetchKatalog()
		if err != nil {
			log.Printf("WARN: fetch from %s: %v", inst.URL, err)
			continue
		}
		inst.Katalog = kat
		anyOK = true
		log.Printf("INFO: fetched katalog %q from %s (%d CRDs)", kat.Name, inst.URL, len(kat.CRDs))
	}
	cc.ready.Store(anyOK)
	cc.mu.Unlock()
	cc.notifySubscribers()
}

// notifySubscribers sends a signal to all connected SSE clients.
func (cc *ControlCenter) notifySubscribers() {
	cc.subscribers.Range(func(key, _ interface{}) bool {
		ch := key.(chan struct{})
		select {
		case ch <- struct{}{}:
		default:
		}
		return true
	})
}

// ServeSSE handles Server-Sent Events for live page reloads.
func (cc *ControlCenter) ServeSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan struct{}, 1)
	cc.subscribers.Store(ch, struct{}{})
	defer cc.subscribers.Delete(ch)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial connection confirmation
	fmt.Fprintf(w, "data: connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-ch:
			fmt.Fprintf(w, "data: reload\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Instance lookups
// ─────────────────────────────────────────────────────────────────────────────

// instanceByKatalogName returns the instance whose loaded katalog has the given name.
func (cc *ControlCenter) instanceByKatalogName(name string) (*Instance, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	for _, inst := range cc.instances {
		if inst.Katalog != nil && inst.Katalog.Name == name {
			return inst, true
		}
	}
	return nil, false
}

// clientFor returns the Client for an instance URL.
// Falls back to the first available client if the URL is not found.
func (cc *ControlCenter) clientFor(instanceURL string) *Client {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	if inst, ok := cc.instances[instanceURL]; ok {
		return inst.Client
	}
	// URL not found — return first available client as fallback
	for _, inst := range cc.instances {
		return inst.Client
	}
	return NewClient(instanceURL, cc.config.RefreshInterval, cc.config.LogLevel)
}

// ─────────────────────────────────────────────────────────────────────────────
// Rendering helpers
// ─────────────────────────────────────────────────────────────────────────────

// renderTemplate parses and executes a named template from the embedded FS.
// Writes directly to w. On error, renders a 500 page.
func (cc *ControlCenter) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	tmpl, err := template.New(name).Funcs(templateFuncs).ParseFS(assets, assetsDir+"/"+name)
	if err != nil {
		log.Printf("ERROR: parse %s: %v", name, err)
		cc.renderError(w, nil, fmt.Sprintf("Template error: %v", err))
		return
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		log.Printf("ERROR: execute %s: %v", name, err)
		cc.renderError(w, nil, fmt.Sprintf("Render error: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w)
}

// renderError renders an inline error page. r is optional (may be nil).
func (cc *ControlCenter) renderError(w http.ResponseWriter, _ *http.Request, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Error – Orkestra Control Center</title>
<link rel="stylesheet" href="/controlcenter/assets/static/css/style.css">
<link rel="icon" type="image/png" href="/controlcenter/assets/static/logo.png">
<link rel="stylesheet" href="/controlcenter/assets/static/css/style.css">
<script>(function(){var t=localStorage.getItem('cc-theme')||'dark';document.documentElement.setAttribute('data-theme',t);})();</script>
</head>
<body style="display:flex;align-items:center;justify-content:center;min-height:100vh;background:var(--bg-base)">
  <div style="text-align:center;padding:40px;max-width:480px">
    <div style="font-size:36px;margin-bottom:16px;opacity:0.6">⚠</div>
    <h1 style="font-size:20px;font-weight:700;color:var(--text-primary);margin-bottom:8px">Something went wrong</h1>
    <p style="font-size:12px;font-family:monospace;color:var(--text-muted);margin-bottom:24px;word-break:break-all">%s</p>
    <a href="/controlcenter" style="display:inline-flex;align-items:center;gap:6px;padding:8px 16px;background:var(--accent);color:#fff;border-radius:6px;text-decoration:none;font-size:13px">
      ← Control Center
    </a>
  </div>
</body></html>`, message)
}

// handleNotFound renders a 404 page.
func (cc *ControlCenter) handleNotFound(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>404 – Orkestra Control Center</title>
<link rel="icon" type="image/png" href="/controlcenter/assets/static/logo.png">
<link rel="stylesheet" href="/controlcenter/assets/static/css/style.css">
<script>(function(){var t=localStorage.getItem('cc-theme')||'dark';document.documentElement.setAttribute('data-theme',t);})();</script>
</head>
<body style="display:flex;align-items:center;justify-content:center;min-height:100vh;background:var(--bg-base)">
  <div style="text-align:center;padding:40px;max-width:400px">
    <div style="font-size:60px;font-weight:700;color:var(--text-muted);margin-bottom:8px">404</div>
    <h1 style="font-size:18px;font-weight:600;color:var(--text-primary);margin-bottom:8px">Page not found</h1>
    <p style="font-size:13px;color:var(--text-muted);margin-bottom:24px">The page you're looking for doesn't exist.</p>
    <a href="/controlcenter" style="display:inline-flex;align-items:center;gap:6px;padding:8px 16px;background:var(--accent);color:#fff;border-radius:6px;text-decoration:none;font-size:13px">
      ← Back to Control Center
    </a>
  </div>
</body></html>`)
}

// ─────────────────────────────────────────────────────────────────────────────
// Router
// ─────────────────────────────────────────────────────────────────────────────

// ServeHTTP dispatches all /controlcenter/** requests.
//
// Route table (after stripping /controlcenter prefix):
//
//	/                                                   → index
//	/assets/**                                          → static files
//	/metrics                                            → metrics page
//	/debug/file                                         → debug
//	/katalog/{katalog}                                  → katalog panel
//	/katalog/{katalog}/crd/{crd}                        → CRD detail
//	/katalog/{katalog}/crd/{crd}/cr                     → CR list
//	/katalog/{katalog}/crd/{crd}/cr/{name}              → CR detail (cluster-scoped)
//	/katalog/{katalog}/crd/{crd}/cr/{ns}/{name}         → CR detail (namespaced)
func (cc *ControlCenter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/controlcenter")
	if path == "" || path == "/" {
		cc.handleIndex(w, r)
		return
	}

	log.Printf("DEBUG: %s %s", r.Method, path)

	switch {
	case strings.HasPrefix(path, "/assets/"):
		cc.serveAsset(w, r, strings.TrimPrefix(path, "/assets/"))

	case path == "/sse":
		cc.ServeSSE(w, r)

	case path == "/metrics":
		cc.handleMetricsPage(w, r)

	case path == "/debug/file":
		cc.handleDebugFile(w, r)

	case strings.HasPrefix(path, "/katalog/"):
		// Strip leading slash and split
		parts := strings.Split(strings.Trim(path, "/"), "/")
		cc.routeKatalog(w, r, parts)

	default:
		cc.handleNotFound(w, r)
	}
}

// routeKatalog dispatches /katalog/** paths.
// parts has no leading slash: ["katalog", "{name}", ...]
func (cc *ControlCenter) routeKatalog(w http.ResponseWriter, r *http.Request, parts []string) {
	// parts[0] == "katalog"
	if len(parts) < 2 {
		cc.handleNotFound(w, r)
		return
	}

	katalogName := parts[1]

	switch {
	// /katalog/{name}/crd/{crd}/cr[/...]
	case len(parts) >= 5 && parts[2] == "crd" && parts[4] == "cr":
		crdName := parts[3]
		inst, ok := cc.instanceByKatalogName(katalogName)
		if !ok {
			cc.handleNotFound(w, r)
			return
		}
		cc.routeCR(w, r, inst.URL, inst.Katalog.Name, crdName, parts[4:])

	// /katalog/{name}/crd/{crd}
	case len(parts) >= 4 && parts[2] == "crd":
		cc.handleCRDDetail(w, r, katalogName, parts[3])

	// /katalog/{name}
	default:
		cc.handleKatalogPanel(w, r, katalogName)
	}
}

// routeCR dispatches CR sub-paths.
// crParts: ["cr"], ["cr", "{name}"], ["cr", "{ns}", "{name}"]
func (cc *ControlCenter) routeCR(w http.ResponseWriter, r *http.Request, instanceURL, katalogName, crdName string, crParts []string) {
	// crParts[0] == "cr"
	switch len(crParts) {
	case 1:
		// /cr → list
		cc.handleCRList(w, r, instanceURL, katalogName, crdName)
	case 2:
		// /cr/{name} → cluster-scoped detail
		cc.handleCRDetail(w, r, instanceURL, katalogName, crdName, "", crParts[1])
	case 3:
		// /cr/{ns}/{name} → namespaced detail
		cc.handleCRDetail(w, r, instanceURL, katalogName, crdName, crParts[1], crParts[2])
	default:
		cc.handleNotFound(w, r)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Page handlers
// ─────────────────────────────────────────────────────────────────────────────

func (cc *ControlCenter) handleIndex(w http.ResponseWriter, _ *http.Request) {
	cc.mu.RLock()
	insts := make([]*Instance, 0, len(cc.instances))
	for _, inst := range cc.instances {
		if inst.Katalog != nil {
			insts = append(insts, inst)
		}
	}
	cc.mu.RUnlock()

	sort.Slice(insts, func(i, j int) bool {
		return insts[i].Katalog.Name < insts[j].Katalog.Name
	})

	var summaries []KatalogSummary
	totalCRDs, totalWorkers, totalResources, healthyKatalogs := 0, 0, 0, 0

	for _, inst := range insts {
		kat := inst.Katalog
		healthyCRDs := 0
		for _, crd := range kat.CRDs {
			if crd.Healthy {
				healthyCRDs++
			}
		}
		summaries = append(summaries, KatalogSummary{
			Name:           kat.Name,
			Description:    kat.Description,
			Version:        kat.Version,
			Healthy:        kat.Healthy,
			TotalCRDs:      len(kat.CRDs),
			HealthyCRDs:    healthyCRDs,
			TotalWorkers:   sumWorkers(kat.CRDs),
			TotalResources: sumResources(kat.CRDs),
		})
		totalCRDs += len(kat.CRDs)
		totalWorkers += sumWorkers(kat.CRDs)
		totalResources += sumResources(kat.CRDs)
		if kat.Healthy {
			healthyKatalogs++
		}
	}

	cc.renderTemplate(w, "index.html", IndexData{
		Katalogs:        summaries,
		TotalKatalogs:   len(summaries),
		HealthyKatalogs: healthyKatalogs,
		TotalCRDs:       totalCRDs,
		TotalWorkers:    totalWorkers,
		TotalResources:  totalResources,
		AnyHealthy:      len(summaries) > 0,
		OrkestraURLs:    strings.Join(cc.urls, ", "),
	})
}

func (cc *ControlCenter) handleKatalogPanel(w http.ResponseWriter, r *http.Request, katalogName string) {
	inst, ok := cc.instanceByKatalogName(katalogName)
	if !ok {
		// Try a fresh fetch once before giving up
		cc.fetchAllKatalogs()
		inst, ok = cc.instanceByKatalogName(katalogName)
		if !ok {
			cc.handleNotFound(w, r)
			return
		}
	}

	kat := inst.Katalog
	cc.renderTemplate(w, "katalog.html", KatalogData{
		CRDs:               kat.CRDs,
		OrkReady:           kat.OrkReady,
		TotalCRDs:          len(kat.CRDs),
		TotalWorkers:       sumWorkers(kat.CRDs),
		TotalResources:     sumResources(kat.CRDs),
		HealthyCount:       countHealthyCRDs(kat.CRDs),
		KatalogName:        kat.Name,
		KatalogDescription: kat.Description,
		KatalogHealthy:     kat.Healthy,
		KatalogVersion:     kat.Version,
		KatalogAuthor:      kat.Author,
		KatalogLicense:     kat.License,
		DegradedReason:     kat.DegradedReason,
		StatusCounts:       kat.StatusCounts,
	})
}

func (cc *ControlCenter) handleCRDDetail(w http.ResponseWriter, r *http.Request, katalogName, crdName string) {
	inst, ok := cc.instanceByKatalogName(katalogName)
	if !ok {
		cc.renderError(w, r, fmt.Sprintf("Katalog %q not found", katalogName))
		return
	}

	crd, err := inst.Client.FetchCRDDetail(crdName)
	if err != nil {
		log.Printf("WARN: fetch CRD %s: %v", crdName, err)
		// Render a degraded view rather than a hard error
		crd = &CRDDetail{
			Name:        crdName,
			State:       "offline",
			Healthy:     false,
			Description: "Unable to connect to Orkestra runtime",
			GVK:         "unknown",
			LastError:   err.Error(),
		}
	}

	cc.renderTemplate(w, "crd.html", map[string]interface{}{
		"CRD":         crd,
		"KatalogName": katalogName,
	})
}

// handleCRList renders the CR instance list for one CRD.
// Route: /katalog/{katalog}/crd/{crd}/cr
func (cc *ControlCenter) handleCRList(w http.ResponseWriter, r *http.Request, instanceURL, katalogName, crdName string) {
	client := cc.clientFor(instanceURL)

	list, err := client.FetchCRList(instanceURL, crdName)
	if err != nil {
		cc.renderError(w, r, fmt.Sprintf("Could not load CR list for %s: %v", crdName, err))
		return
	}

	cc.renderTemplate(w, "cr_list.html", CRListView{
		KatalogName: katalogName,
		Instance:    instanceURL,
		CRDName:     crdName,
		GVK:         list.GVK,
		Total:       list.Total,
		Items:       list.Items,
		BackURL:     fmt.Sprintf("/controlcenter/katalog/%s/crd/%s", katalogName, crdName),
	})
}

// handleCRDetail renders one CR instance with its children and events.
// Route: /katalog/{katalog}/crd/{crd}/cr/{ns}/{name}  (namespace may be empty)
func (cc *ControlCenter) handleCRDetail(w http.ResponseWriter, r *http.Request, instanceURL, katalogName, crdName, namespace, name string) {
	client := cc.clientFor(instanceURL)

	detail, err := client.FetchCRDetail(instanceURL, crdName, namespace, name)
	if err != nil {
		cc.renderError(w, r, fmt.Sprintf("Could not load CR %s/%s: %v", namespace, name, err))
		return
	}

	// Events are best-effort — never block or error on failure
	var events []CREvent
	var eventTotal int
	if evResp, err := client.FetchCREvents(instanceURL, crdName, namespace, name); err == nil {
		events = evResp.Events
		eventTotal = evResp.Total
	}

	// Extract phase from status for template convenience
	phase := ""
	if detail.Status != nil {
		if p, ok := detail.Status["phase"].(string); ok {
			phase = p
		}
	}

	cc.renderTemplate(w, "cr_detail.html", CRDetailView{
		KatalogName: katalogName,
		Instance:    instanceURL,
		CRDName:     crdName,
		CR:          *detail,
		Events:      events,
		EventTotal:  eventTotal,
		Phase:       phase,
		BackURL:     fmt.Sprintf("/controlcenter/katalog/%s/crd/%s", katalogName, crdName),
	})
}

// ─────────────────────────────────────────────────────────────────────────────
// Asset and utility handlers
// ─────────────────────────────────────────────────────────────────────────────

func (cc *ControlCenter) serveAsset(w http.ResponseWriter, r *http.Request, filePath string) {
	data, err := assets.ReadFile("assets/" + filePath)
	if err != nil {
		cc.handleNotFound(w, r)
		return
	}
	switch {
	case strings.HasSuffix(filePath, ".css"):
		w.Header().Set("Content-Type", "text/css")
	case strings.HasSuffix(filePath, ".js"):
		w.Header().Set("Content-Type", "application/javascript")
	case strings.HasSuffix(filePath, ".png"):
		w.Header().Set("Content-Type", "image/png")
	case strings.HasSuffix(filePath, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Write(data)
}

func (cc *ControlCenter) handleMetricsPage(w http.ResponseWriter, _ *http.Request) {
	cc.renderTemplate(w, "metrics.html", nil)
}

func (cc *ControlCenter) handleDebugFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		filePath = "assets/static/logo.png"
	}
	fmt.Fprintf(w, "Reading: %s\n\n", filePath)
	data, err := assets.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(w, "ERROR: %v\n", err)
		return
	}
	fmt.Fprintf(w, "OK: %d bytes\n", len(data))

	entries, _ := assets.ReadDir("assets/static")
	fmt.Fprintf(w, "\nassets/static/:\n")
	for _, e := range entries {
		fmt.Fprintf(w, "  %s\n", e.Name())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Aggregate helpers
// ─────────────────────────────────────────────────────────────────────────────

func sumWorkers(crds []CRDSummary) int {
	n := 0
	for _, c := range crds {
		n += c.Workers
	}
	return n
}

func sumResources(crds []CRDSummary) int {
	n := 0
	for _, c := range crds {
		n += c.ResourceCount
	}
	return n
}

func countHealthyCRDs(crds []CRDSummary) int {
	n := 0
	for _, c := range crds {
		if c.Healthy {
			n++
		}
	}
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
