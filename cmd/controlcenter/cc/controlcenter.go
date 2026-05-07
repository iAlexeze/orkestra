package controlcenter

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ccversion "github.com/orkspace/orkestra-cc/version"
)

//go:embed assets/templates/*.html assets/static/* assets/static/css/* assets/static/js/*
var Assets embed.FS

const TemplateDir = "assets/templates"

// ─────────────────────────────────────────────────────────────────────────────
// Config
// ─────────────────────────────────────────────────────────────────────────────

type Config struct {
	RefreshInterval      time.Duration
	LogLevel             string
	Version              string
	EnableRuntimeManager bool
}

// ─────────────────────────────────────────────────────────────────────────────
// Instance — one connected Orkestra runtime
// ─────────────────────────────────────────────────────────────────────────────
type Instance struct {
	URL       string
	Client    *Client
	Katalog   *KatalogResponse
	Healthy   bool
	LastError string
	LastCheck time.Time
	Status    string // "online", "starting" or "degraded"
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
	// Load previously stored URLs
	stored, _ := LoadRuntimeStorage()
	allURLs := make(map[string]bool)
	for _, u := range urls {
		allURLs[normalizeURL(u)] = true
	}
	if stored != nil {
		for _, u := range stored.URLs {
			allURLs[normalizeURL(u)] = true
		}
	}
	finalURLs := make([]string, 0, len(allURLs))
	for u := range allURLs {
		finalURLs = append(finalURLs, u)
	}

	instances := make(map[string]*Instance)
	for _, url := range finalURLs {
		instances[url] = &Instance{
			URL:    url,
			Client: NewClient(url, config.RefreshInterval, config.LogLevel),
		}
	}
	cc := &ControlCenter{
		urls:      finalURLs,
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
	// Snapshot instance URLs without holding the lock during network I/O.
	// Holding a write lock across HTTP calls blocks all concurrent reads
	// (CRD detail pages, snapshot API) for the full fetch duration, causing
	// intermittent "pending / 0 reconciles" displays even when the runtime is healthy.
	cc.mu.RLock()
	urls := make([]string, 0, len(cc.instances))
	for u := range cc.instances {
		urls = append(urls, u)
	}
	cc.mu.RUnlock()

	anyOK := false

	for _, u := range urls {
		cc.mu.RLock()
		inst, ok := cc.instances[u]
		cc.mu.RUnlock()
		if !ok {
			continue // instance was removed between snapshot and now
		}

		now := time.Now()
		kat, err := inst.Client.FetchKatalog()

		// Update only this instance under a brief write lock
		cc.mu.Lock()
		if inst, ok = cc.instances[u]; ok {
			inst.LastCheck = now
			if err != nil {
				inst.Katalog = nil
				inst.Status = "offline"
				inst.Healthy = false
				inst.LastError = err.Error()
				log.Printf("WARN: fetch katalog from %s: %v", u, err)
			} else {
				inst.Katalog = kat
				if kat.OrkReady {
					inst.Status = "online"
					inst.Healthy = true
					inst.LastError = ""
				} else {
					inst.Status = "starting"
					inst.Healthy = false
					inst.LastError = ""
				}
				log.Printf("INFO: fetched katalog %q from %s (%d CRDs)",
					kat.Name, u, len(kat.CRDs))
			}
		}
		cc.mu.Unlock()

		if err == nil {
			anyOK = true
		}
	}

	cc.ready.Store(anyOK)
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
			fmt.Fprintf(w, "data: update\n\n")
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
//	/docs                                               → docs landing
//	/docs/{katalog}/{crd}                               → CRD docs page
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

	case path == "/api/snapshot":
		cc.handleAPISnapshot(w, r)

	case path == "/metrics":
		cc.handleMetricsPage(w, r)

	case path == "/debug/file":
		cc.handleDebugFile(w, r)

	case path == "/docs":
		cc.handleDocsLanding(w, r)

	case strings.HasPrefix(path, "/docs/"):
		// /docs/{katalog}/{crd}
		parts := strings.SplitN(strings.TrimPrefix(path, "/docs/"), "/", 2)
		if len(parts) == 2 {
			cc.handleCRDDocs(w, r, parts[0], parts[1])
		} else {
			cc.handleDocsLanding(w, r)
		}

	case strings.HasPrefix(path, "/katalog/"):
		// Strip leading slash and split
		parts := strings.Split(strings.Trim(path, "/"), "/")
		cc.routeKatalog(w, r, parts)
	case path == "/api/instances" && r.Method == http.MethodGet:
		cc.handleListInstances(w, r)
	case path == "/api/instances" && r.Method == http.MethodPost:
		cc.handleAddInstance(w, r)
	case strings.HasPrefix(path, "/api/instances/") && r.Method == http.MethodPut:
		cc.handleUpdateInstance(w, r)
	case strings.HasPrefix(path, "/api/instances/") && r.Method == http.MethodDelete:
		cc.handleDeleteInstance(w, r)
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

	// /katalog/{name}/crd/{crd}/raw  or  /katalog/{name}/crd/{crd}/enriched
	case len(parts) == 5 && parts[2] == "crd" && (parts[4] == "raw" || parts[4] == "enriched"):
		cc.handleProxyCRDSpec(w, r, katalogName, parts[3], parts[4])

	// /katalog/{name}/crd/{crd}
	case len(parts) >= 4 && parts[2] == "crd":
		cc.handleCRDDetail(w, r, katalogName, parts[3])

	// /katalog/{name}/raw  or  /katalog/{name}/enriched
	case len(parts) == 3 && (parts[2] == "raw" || parts[2] == "enriched"):
		cc.handleProxyKatalogSpec(w, r, katalogName, parts[2])

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
// Raw / Enriched proxy handlers
// ─────────────────────────────────────────────────────────────────────────────

// handleProxyCRDSpec proxies GET /katalog/{crd}/raw or /katalog/{crd}/enriched
// from the Orkestra runtime and returns the JSON body to the browser.
func (cc *ControlCenter) handleProxyCRDSpec(w http.ResponseWriter, r *http.Request, katalogName, crdName, kind string) {
	inst, ok := cc.instanceByKatalogName(katalogName)
	if !ok {
		http.Error(w, "katalog not found", http.StatusNotFound)
		return
	}
	target := inst.URL + "/katalog/" + crdName + "/" + kind
	cc.proxyJSON(w, target)
}

// handleProxyKatalogSpec proxies GET /katalog/raw or /katalog/enriched
// from the Orkestra runtime and returns the JSON body to the browser.
func (cc *ControlCenter) handleProxyKatalogSpec(w http.ResponseWriter, r *http.Request, katalogName, kind string) {
	inst, ok := cc.instanceByKatalogName(katalogName)
	if !ok {
		http.Error(w, "katalog not found", http.StatusNotFound)
		return
	}
	target := inst.URL + "/katalog/" + kind
	cc.proxyJSON(w, target)
}

// proxyJSON fetches target and pipes the JSON response body to w.
func (cc *ControlCenter) proxyJSON(w http.ResponseWriter, target string) {
	resp, err := http.Get(target) //nolint:gosec
	if err != nil {
		http.Error(w, "upstream error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body) //nolint:errcheck
}

// ─────────────────────────────────────────────────────────────────────────────
// Page handlers
// ─────────────────────────────────────────────────────────────────────────────

// ─────────────────────────────────────────────────────────────────────────────
// Live snapshot API — used by JS for partial DOM updates (no full-page reload)
// ─────────────────────────────────────────────────────────────────────────────

type StatsSnap struct {
	TotalKatalogs   int `json:"totalKatalogs"`
	HealthyKatalogs int `json:"healthyKatalogs"`
	TotalCRDs       int `json:"totalCRDs"`
	TotalWorkers    int `json:"totalWorkers"`
	TotalResources  int `json:"totalResources"`
}

type KatalogSnap struct {
	Name           string `json:"name"`
	Healthy        bool   `json:"healthy"`
	TotalCRDs      int    `json:"totalCRDs"`
	HealthyCRDs    int    `json:"healthyCRDs"`
	TotalWorkers   int    `json:"totalWorkers"`
	TotalResources int    `json:"totalResources"`
}

type SnapshotData struct {
	Stats    StatsSnap     `json:"stats"`
	Katalogs []KatalogSnap `json:"katalogs"`
	Updated  string        `json:"updated"`
}

func (cc *ControlCenter) handleAPISnapshot(w http.ResponseWriter, _ *http.Request) {
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

	stats := StatsSnap{}
	var katalogs []KatalogSnap

	for _, inst := range insts {
		kat := inst.Katalog
		healthyCRDs := 0
		for _, crd := range kat.CRDs {
			if crd.Healthy {
				healthyCRDs++
			}
		}
		stats.TotalKatalogs++
		stats.TotalCRDs += len(kat.CRDs)
		stats.TotalWorkers += sumWorkers(kat.CRDs)
		stats.TotalResources += sumResources(kat.CRDs)
		if kat.Healthy {
			stats.HealthyKatalogs++
		}
		katalogs = append(katalogs, KatalogSnap{
			Name:           kat.Name,
			Healthy:        kat.Healthy,
			TotalCRDs:      len(kat.CRDs),
			HealthyCRDs:    healthyCRDs,
			TotalWorkers:   sumWorkers(kat.CRDs),
			TotalResources: sumResources(kat.CRDs),
		})
	}

	if katalogs == nil {
		katalogs = []KatalogSnap{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(SnapshotData{
		Stats:    stats,
		Katalogs: katalogs,
		Updated:  time.Now().UTC().Format(time.RFC3339),
	})
}

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
		Katalogs:             summaries,
		TotalKatalogs:        len(summaries),
		HealthyKatalogs:      healthyKatalogs,
		TotalCRDs:            totalCRDs,
		TotalWorkers:         totalWorkers,
		TotalResources:       totalResources,
		AnyHealthy:           len(summaries) > 0,
		OrkestraURLs:         strings.Join(cc.urls, ", "),
		CCVersion:            ccversion.Short(),
		EnableRuntimeManager: cc.config.EnableRuntimeManager,
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

	// Sort CRDs by name for consistent display
	sortedCRDs := make([]CRDSummary, len(kat.CRDs))
	copy(sortedCRDs, kat.CRDs)
	sort.Slice(sortedCRDs, func(i, j int) bool {
		return sortedCRDs[i].Name < sortedCRDs[j].Name
	})

	cc.renderTemplate(w, "katalog.html", KatalogData{
		CRDs:               sortedCRDs,
		OrkReady:           kat.OrkReady,
		DeletionProtection: kat.DeletionProtection,
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
		RuntimeVersion:     kat.RuntimeVersion,
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
		cc.renderError(w, r, fmt.Sprintf("Could not load CR list for %s: \n%v", crdName, err))
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
		if namespace != "" {
			cc.renderError(w, r, fmt.Sprintf("Could not load CR '%s/%s': \n%v", namespace, name, err))
		} else {
			cc.renderError(w, r, fmt.Sprintf("Could not load CR '%s': \n%v", name, err))
		}
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
		ChildGroups: normalizeChildGroups(detail.Children),
		BackURL:     fmt.Sprintf("/controlcenter/katalog/%s/crd/%s", katalogName, crdName),
	})
}

func (cc *ControlCenter) handleListInstances(w http.ResponseWriter, r *http.Request) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	type InstanceDTO struct {
		URL       string    `json:"url"`
		Healthy   bool      `json:"healthy"`
		LastError string    `json:"lastError,omitempty"`
		LastCheck time.Time `json:"lastCheck"`
	}

	out := make([]InstanceDTO, 0, len(cc.instances))
	for _, inst := range cc.instances {
		out = append(out, InstanceDTO{
			URL:       inst.URL,
			Healthy:   inst.Healthy,
			LastError: inst.LastError,
			LastCheck: inst.LastCheck,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"instances": out,
	})
}

func (cc *ControlCenter) handleAddInstance(w http.ResponseWriter, r *http.Request) {
	var req struct{ URL string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if err := cc.AddInstance(req.URL); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "added"})
}

func (cc *ControlCenter) handleUpdateInstance(w http.ResponseWriter, r *http.Request) {
	// Path: /api/instances/encodedOldURL
	encoded := strings.TrimPrefix(r.URL.Path, "/api/instances/")
	oldURL, err := url.PathUnescape(encoded)
	if err != nil {
		http.Error(w, `{"error":"invalid URL"}`, http.StatusBadRequest)
		return
	}
	var req struct{ URL string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	newURL := normalizeURL(req.URL)
	if err := cc.UpdateInstance(oldURL, newURL); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (cc *ControlCenter) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	encoded := strings.TrimPrefix(r.URL.Path, "/api/instances/")
	urlStr, err := url.PathUnescape(encoded)
	if err != nil {
		http.Error(w, `{"error":"invalid URL"}`, http.StatusBadRequest)
		return
	}
	if err := cc.DeleteInstance(urlStr); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// normalizeURL adds scheme and removes trailing slash.
func normalizeURL(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "http://" + u
	}
	return strings.TrimSuffix(u, "/")
}

// ─────────────────────────────────────────────────────────────────────────────
// Asset and utility handlers
// ─────────────────────────────────────────────────────────────────────────────

func (cc *ControlCenter) serveAsset(w http.ResponseWriter, r *http.Request, filePath string) {
	data, err := Assets.ReadFile("assets/" + filePath)
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
	data, err := Assets.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(w, "ERROR: %v\n", err)
		return
	}
	fmt.Fprintf(w, "OK: %d bytes\n", len(data))

	entries, _ := Assets.ReadDir("assets/static")
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
