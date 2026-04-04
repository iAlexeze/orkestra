package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	controlcenter "github.com/orkspace/orkestra-cc/pkg"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	var (
		orkestraURLs = flag.String("u", "http://localhost:8080", "Comma-separated URLs of Orkestra runtime instances")
		port         = flag.String("p", "8090", "Port to serve the control center on")
		refresh      = flag.Duration("refresh", 10*time.Second, "Refresh interval for fetching Katalogs")
		logLevel     = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
		showVersion  = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("orkestra-control-center version %s (commit: %s, built: %s)\n", version, commit, buildDate)
		os.Exit(0)
	}

	// Parse and deduplicate URLs
	urls := parseAndDedupeURLs(*orkestraURLs)
	if len(urls) == 0 {
		log.Fatal("ERROR: at least one Orkestra URL is required. Use -u flag.")
	}

	// Create control center
	cc := controlcenter.New(urls, controlcenter.Config{
		RefreshInterval: *refresh,
		LogLevel:        *logLevel,
		Version:         version,
	})

	// Setup routes
	mux := setupRoutes(cc, version, commit, buildDate)

	// Setup HTTP server
	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("🚀 Orkestra Control Center starting on :%s", *port)
		log.Printf("📡 Watching instances: %v", urls)
		log.Printf("🔄 Refresh interval: %v", *refresh)
		log.Printf("🌐 Control Center URL: http://localhost:%s/controlcenter", *port)
		log.Printf("🏥 Health endpoint: http://localhost:%s/controlcenter/health", *port)
		log.Printf("✅ Ready endpoint: http://localhost:%s/controlcenter/ready", *port)
		log.Printf("📌 Version endpoint: http://localhost:%s/controlcenter/version", *port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("🛑 Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("⚠️ Shutdown error: %v", err)
	}

	log.Println("✅ Control Center stopped")
}

// parseAndDedupeURLs parses a comma-separated URL string and removes duplicates
func parseAndDedupeURLs(input string) []string {
	parts := strings.Split(input, ",")
	seen := make(map[string]bool)
	urls := make([]string, 0, len(parts))

	for _, p := range parts {
		url := strings.TrimSpace(p)
		if url == "" {
			continue
		}

		// Add scheme if missing
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "http://" + url
		}

		// Remove trailing slash
		url = strings.TrimSuffix(url, "/")

		// Deduplicate
		if !seen[url] {
			seen[url] = true
			urls = append(urls, url)
		}
	}

	return urls
}
