package controlcenter

import (
	"fmt"
	"log"
)

// AddInstance adds and persists a new runtime.
func (cc *ControlCenter) AddInstance(rawURL string) error {
	url := normalizeURL(rawURL)
	if url == "" {
		return fmt.Errorf("invalid URL")
	}
	cc.mu.Lock()
	defer cc.mu.Unlock()
	for _, existing := range cc.urls {
		if existing == url {
			return fmt.Errorf("instance already exists")
		}
	}
	// Add to memory
	client := NewClient(url, cc.config.RefreshInterval, cc.config.LogLevel)
	cc.instances[url] = &Instance{URL: url, Client: client}
	cc.urls = append(cc.urls, url)

	// Persist
	store := &RuntimeStorage{URLs: cc.urls}
	if err := SaveRuntimeStorage(store); err != nil {
		log.Printf("WARN: failed to save runtime list: %v", err)
	}
	// Fetch its Katalog in background
	go func() {
		if kat, err := client.FetchKatalog(); err == nil {
			cc.mu.Lock()
			cc.instances[url].Katalog = kat
			cc.mu.Unlock()
		}
	}()
	return nil
}

// DeleteInstance removes a runtime by URL.
func (cc *ControlCenter) DeleteInstance(rawURL string) error {
	url := normalizeURL(rawURL)
	if url == "" {
		return fmt.Errorf("invalid URL")
	}
	cc.mu.Lock()
	defer cc.mu.Unlock()
	newURLs := make([]string, 0, len(cc.urls))
	found := false
	for _, u := range cc.urls {
		if u == url {
			found = true
			continue
		}
		newURLs = append(newURLs, u)
	}
	if !found {
		return fmt.Errorf("instance not found")
	}
	delete(cc.instances, url)
	cc.urls = newURLs
	store := &RuntimeStorage{URLs: cc.urls}
	if err := SaveRuntimeStorage(store); err != nil {
		log.Printf("WARN: failed to save runtime list: %v", err)
	}
	return nil
}

// UpdateInstance replaces an old URL with a new one.
func (cc *ControlCenter) UpdateInstance(oldURL, newURL string) error {
	if err := cc.DeleteInstance(oldURL); err != nil {
		return err
	}
	return cc.AddInstance(newURL)
}

// ListInstances returns all managed URLs.
func (cc *ControlCenter) ListInstances() []string {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	out := make([]string, len(cc.urls))
	copy(out, cc.urls)
	return out
}
