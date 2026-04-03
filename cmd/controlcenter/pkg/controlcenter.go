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

//go:embed assets/templates/*.html assets/static/*
var assets embed.FS
var assetsDir = "assets/templates"

// Config holds configuration for the control center
type Config struct {
	RefreshInterval time.Duration
	LogLevel        string
	Version         string
}

// Instance represents a single Orkestra runtime instance
type Instance struct {
	URL     string
	Client  *Client
	Katalog *KatalogResponse
}

// ControlCenter aggregates data from multiple Orkestra instances
type ControlCenter struct {
	urls      []string
	instances map[string]*Instance
	mu        sync.RWMutex
	config    Config
	ready     atomic.Bool
}

// New creates a new ControlCenter instance
func New(urls []string, config Config) *ControlCenter {
	instances := make(map[string]*Instance)
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
	cc.ready.Store(false)
	go cc.backgroundFetchLoop()
	return cc
}

func (cc *ControlCenter) backgroundFetchLoop() {
	cc.fetchAllKatalogs()
	ticker := time.NewTicker(cc.config.RefreshInterval)
	defer ticker.Stop()
	for range ticker.C {
		cc.fetchAllKatalogs()
	}
}

func (cc *ControlCenter) fetchAllKatalogs() {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	anySuccess := false
	for _, instance := range cc.instances {
		katalog, err := instance.Client.FetchKatalog()
		if err != nil {
			log.Printf("Failed to fetch from %s: %v", instance.URL, err)
			continue
		}
		instance.Katalog = katalog
		anySuccess = true
		log.Printf("Fetched Katalog '%s' from %s (%d CRDs)", katalog.Name, instance.URL, len(katalog.CRDs))
	}
	cc.ready.Store(anySuccess)
}

// ServeHTTP implements http.Handler
func (cc *ControlCenter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/controlcenter")
	if path == "" {
		path = "/"
	}
	log.Printf("DEBUG: Request: %s %s -> path: %s", r.Method, r.URL.Path, path)

	switch {
	case path == "/":
		cc.handleIndex(w, r)
	case strings.HasPrefix(path, "/katalog/"):
		relativePath := strings.TrimPrefix(path, "/")
		cc.handleKatalog(w, r, relativePath)
	case path == "/metrics":
		cc.handleMetricsPage(w, r)
	case path == "/debug/file":
		cc.handleDebugFile(w, r)
	case strings.HasPrefix(path, "/assets/"):
		filePath := strings.TrimPrefix(path, "/assets/")
		data, err := assets.ReadFile("assets/" + filePath)
		if err != nil {
			log.Printf("DEBUG: File not found: assets/%s", filePath)
			http.NotFound(w, r)
			return
		}

		contentType := "application/octet-stream"
		if strings.HasSuffix(filePath, ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(filePath, ".css") {
			contentType = "text/css"
		} else if strings.HasSuffix(filePath, ".js") {
			contentType = "application/javascript"
		}

		w.Header().Set("Content-Type", contentType)
		w.Write(data)
	default:
		log.Printf("DEBUG: 404 Not Found - path: %s", path)
		http.NotFound(w, r)
	}
}

func (cc *ControlCenter) getInstanceByName(katalogName string) (*Instance, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	for _, instance := range cc.instances {
		if instance.Katalog != nil && instance.Katalog.Name == katalogName {
			return instance, true
		}
	}
	return nil, false
}

func (cc *ControlCenter) handleIndex(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: handleIndex called")

	cc.mu.RLock()
	instances := make([]*Instance, 0, len(cc.instances))
	for _, instance := range cc.instances {
		if instance.Katalog != nil {
			instances = append(instances, instance)
		}
	}
	cc.mu.RUnlock()

	log.Printf("DEBUG: Found %d instances with Katalogs", len(instances))

	sort.Slice(instances, func(i, j int) bool {
		return instances[i].Katalog.Name < instances[j].Katalog.Name
	})

	var summaries []KatalogSummary
	totalCRDs, totalWorkers, totalResources, healthyKatalogs := 0, 0, 0, 0

	for _, instance := range instances {
		katalog := instance.Katalog
		healthyCRDs := 0
		for _, crd := range katalog.CRDs {
			if crd.Healthy {
				healthyCRDs++
			}
		}
		summaries = append(summaries, KatalogSummary{
			Name:           katalog.Name,
			Description:    katalog.Description,
			Version:        katalog.Version,
			Healthy:        katalog.Healthy,
			TotalCRDs:      len(katalog.CRDs),
			HealthyCRDs:    healthyCRDs,
			TotalWorkers:   sumWorkers(katalog.CRDs),
			TotalResources: sumResources(katalog.CRDs),
		})
		totalCRDs += len(katalog.CRDs)
		totalWorkers += sumWorkers(katalog.CRDs)
		totalResources += sumResources(katalog.CRDs)
		if katalog.Healthy {
			healthyKatalogs++
		}
	}

	data := IndexData{
		Katalogs:        summaries,
		TotalKatalogs:   len(summaries),
		HealthyKatalogs: healthyKatalogs,
		TotalCRDs:       totalCRDs,
		TotalWorkers:    totalWorkers,
		TotalResources:  totalResources,
		AnyHealthy:      len(summaries) > 0,
		OrkestraURLs:    strings.Join(cc.urls, ", "),
	}

	tmpl, err := template.New("index.html").Funcs(templateFuncs).ParseFS(assets, assetsDir+"/index.html")
	if err != nil {
		log.Printf("ERROR: Template parse error: %v", err)
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		log.Printf("ERROR: Template execute error: %v", err)
		http.Error(w, fmt.Sprintf("Execute error: %v", err), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
	log.Printf("DEBUG: handleIndex completed successfully")
}

func (cc *ControlCenter) handleKatalog(w http.ResponseWriter, r *http.Request, relativePath string) {
	// relativePath is like "katalog/hello-website" or "katalog/hello-website/crd/website"
	parts := strings.Split(relativePath, "/")
	log.Printf("DEBUG: handleKatalog - relativePath: %s, parts: %v", relativePath, parts)

	if len(parts) < 2 {
		log.Printf("DEBUG: Invalid path, need at least 2 parts")
		http.NotFound(w, r)
		return
	}

	katalogName := parts[1] // parts[0] is "katalog", parts[1] is the name
	log.Printf("DEBUG: Looking for Katalog: %s", katalogName)

	// Handle CRD detail view
	if len(parts) >= 4 && parts[2] == "crd" {
		crdName := parts[3]
		log.Printf("DEBUG: CRD detail view - crd: %s", crdName)
		cc.handleCRDDetail(w, r, katalogName, crdName)
		return
	}

	// Find the instance
	instance, found := cc.getInstanceByName(katalogName)
	if !found || instance.Katalog == nil {
		log.Printf("DEBUG: Katalog %s not found, attempting fetch...", katalogName)
		cc.fetchAllKatalogs()
		instance, found = cc.getInstanceByName(katalogName)
		if !found || instance.Katalog == nil {
			log.Printf("DEBUG: Katalog %s still not found after fetch", katalogName)

			cc.mu.RLock()
			available := make([]string, 0)
			for _, inst := range cc.instances {
				if inst.Katalog != nil {
					available = append(available, inst.Katalog.Name)
				}
			}
			cc.mu.RUnlock()
			log.Printf("DEBUG: Available Katalogs: %v", available)

			http.Error(w, fmt.Sprintf("Katalog '%s' not found. Available: %v", katalogName, available), http.StatusNotFound)
			return
		}
	}

	katalog := instance.Katalog
	log.Printf("DEBUG: Rendering Katalog: %s with %d CRDs", katalog.Name, len(katalog.CRDs))

	data := KatalogData{
		CRDs:               katalog.CRDs,
		OrkReady:           katalog.OrkReady,
		TotalCRDs:          len(katalog.CRDs),
		TotalWorkers:       sumWorkers(katalog.CRDs),
		TotalResources:     sumResources(katalog.CRDs),
		HealthyCount:       countHealthyCRDs(katalog.CRDs),
		KatalogName:        katalog.Name,
		KatalogDescription: katalog.Description,
		KatalogHealthy:     katalog.Healthy,
		KatalogVersion:     katalog.Version,
		KatalogAuthor:      katalog.Author,
		KatalogLicense:     katalog.License,
		DegradedReason:     katalog.DegradedReason,
		StatusCounts:       katalog.StatusCounts,
	}

	tmpl, err := template.New("katalog.html").Funcs(templateFuncs).ParseFS(assets, assetsDir+"/katalog.html")
	if err != nil {
		log.Printf("ERROR: Template parse error: %v", err)
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		log.Printf("ERROR: Template execute error: %v", err)
		http.Error(w, fmt.Sprintf("Execute error: %v", err), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
	log.Printf("DEBUG: handleKatalog completed successfully")
}

func (cc *ControlCenter) handleCRDDetail(w http.ResponseWriter, r *http.Request, katalogName, crdName string) {
	log.Printf("DEBUG: handleCRDDetail - katalog: %s, crd: %s", katalogName, crdName)

	instance, found := cc.getInstanceByName(katalogName)
	if !found {
		log.Printf("DEBUG: Katalog %s not found for CRD detail", katalogName)
		http.Error(w, fmt.Sprintf("Katalog '%s' not found", katalogName), http.StatusNotFound)
		return
	}

	crd, err := instance.Client.FetchCRDDetail(crdName)
	if err != nil {
		log.Printf("Failed to fetch CRD %s: %v", crdName, err)
		crd = &CRDDetail{
			Name:        crdName,
			State:       "offline",
			Healthy:     false,
			Description: "Unable to connect to Orkestra runtime",
			GVK:         "unknown",
			LastError:   err.Error(),
		}
	}

	tmpl, err := template.New("crd.html").Funcs(templateFuncs).ParseFS(assets, assetsDir+"/crd.html")
	if err != nil {
		log.Printf("ERROR: Template parse error: %v", err)
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
		return
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, map[string]interface{}{
		"CRD":         crd,
		"KatalogName": katalogName,
	}); err != nil {
		log.Printf("ERROR: Template execute error: %v", err)
		http.Error(w, fmt.Sprintf("Execute error: %v", err), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}

func (cc *ControlCenter) handleMetricsPage(w http.ResponseWriter, r *http.Request) {
	log.Printf("DEBUG: handleMetricsPage called")
	tmpl, err := template.ParseFS(assets, assetsDir+"/metrics.html")
	if err != nil {
		log.Printf("ERROR: Template parse error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse template: %v", err), http.StatusInternalServerError)
		return
	}
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, nil); err != nil {
		log.Printf("ERROR: Template execute error: %v", err)
		http.Error(w, fmt.Sprintf("Failed to execute template: %v", err), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}

func (cc *ControlCenter) IsReady() bool {
	return cc.ready.Load()
}

// Helper functions
func sumWorkers(crds []CRDSummary) int {
	sum := 0
	for _, crd := range crds {
		sum += crd.Workers
	}
	return sum
}

func sumResources(crds []CRDSummary) int {
	sum := 0
	for _, crd := range crds {
		sum += crd.ResourceCount
	}
	return sum
}

func countHealthyCRDs(crds []CRDSummary) int {
	count := 0
	for _, crd := range crds {
		if crd.Healthy {
			count++
		}
	}
	return count
}

// Debug
func (cc *ControlCenter) handleDebugFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	// Get the file path from query param
	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		filePath = "assets/static/logo.png"
	}

	fmt.Fprintf(w, "Trying to read: %s\n\n", filePath)

	// Try to read the file
	data, err := assets.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(w, "ERROR: %v\n\n", err)
	} else {
		fmt.Fprintf(w, "SUCCESS: Read %d bytes\n", len(data))
		fmt.Fprintf(w, "First 20 bytes: %x\n", data[:min(20, len(data))])
	}

	// List all files in assets/static/
	fmt.Fprintf(w, "\nFiles in assets/static/:\n")
	staticFiles, err := assets.ReadDir("assets/static")
	if err != nil {
		fmt.Fprintf(w, "Error reading assets/static: %v\n", err)
		return
	}

	for _, f := range staticFiles {
		if f.IsDir() {
			fmt.Fprintf(w, "  📁 %s/\n", f.Name())
			subFiles, _ := assets.ReadDir("assets/static/" + f.Name())
			for _, sf := range subFiles {
				fmt.Fprintf(w, "    📄 %s\n", sf.Name())
			}
		} else {
			fmt.Fprintf(w, "  📄 %s\n", f.Name())
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
