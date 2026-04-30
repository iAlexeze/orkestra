// pkg/registry/client.go
//
// ORAS-based OCI client for pushing and pulling Orkestra patterns.
//
// Authentication uses ~/.docker/config.json via oras.land/oras-go/v2.
// No separate login step — docker login ghcr.io is sufficient.
//
// Push: validates the directory, reads each file, pushes as OCI layers.
// Pull: fetches the manifest, extracts layers to the cache directory.
// Info: fetches the manifest only, reads annotations.
// List: fetches the index artifact from the registry root.
package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// Client wraps ORAS for Orkestra pattern operations.
type Client struct {
	// credStore loads credentials from ~/.docker/config.json.
	credStore credentials.Store
}

// NewClient returns a Client with credentials loaded from the Docker config.
func NewClient() (*Client, error) {
	store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{
		AllowPlaintextPut: false,
	})
	if err != nil {
		return nil, fmt.Errorf("loading docker credentials: %w", err)
	}
	return &Client{credStore: store}, nil
}

// Push validates the directory and pushes all pattern files to the registry.
// Returns the manifest digest on success.
func (c *Client) Push(ctx context.Context, ref *Ref, dir string, progress func(file string, size int64)) (string, error) {
	// Validate first — fail before any network call
	meta, files, err := ValidateDirectory(dir)
	if err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}
	_ = meta // used for annotations below

	// Use an in-memory store so nothing is written back to dir.
	store := memory.New()

	// Read each file and push it into the memory store as a blob.
	var descs []ocispec.Descriptor
	for _, f := range files {
		path := filepath.Join(dir, f)
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", f, err)
		}
		if progress != nil {
			progress(f, int64(len(data)))
		}
		desc := content.NewDescriptorFromBytes(mediaTypeForFile(f), data)
		desc.Annotations = map[string]string{
			"org.opencontainers.image.title": f,
		}
		if err := store.Push(ctx, desc, bytes.NewReader(data)); err != nil {
			return "", fmt.Errorf("staging %s: %w", f, err)
		}
		descs = append(descs, desc)
	}

	// Pack into a manifest with annotations from pattern.yaml
	annotations := metaToAnnotations(meta, ref)
	manifestDesc, err := oras.Pack(ctx, store, MediaType, descs, oras.PackOptions{
		PackImageManifest:   true,
		ManifestAnnotations: annotations,
	})
	if err != nil {
		return "", fmt.Errorf("packing manifest: %w", err)
	}

	// Tag the manifest in the local store so oras.Copy can resolve it by name.
	if err := store.Tag(ctx, manifestDesc, ref.Tag); err != nil {
		return "", fmt.Errorf("tagging manifest: %w", err)
	}

	// Push to the remote repository
	repo, err := c.remoteRepo(ref)
	if err != nil {
		return "", err
	}

	if _, err := oras.Copy(ctx, store, ref.Tag, repo, ref.Tag, oras.DefaultCopyOptions); err != nil {
		return "", fmt.Errorf("pushing: %w", err)
	}

	// Update the shared index so ork registry list reflects the new pattern.
	entry := PatternEntry{
		Name:          meta.Name,
		LatestVersion: meta.Version,
		Description:   meta.Description,
		Tags:          meta.Tags,
		Author:        meta.Author,
	}
	if err := c.updateIndex(ctx, ref, entry); err != nil {
		// Non-fatal: push succeeded; index update failure is best-effort.
		fmt.Fprintf(os.Stderr, "warning: index update failed: %v\n", err)
	}

	return manifestDesc.Digest.String(), nil
}

// Pull fetches a pattern from the registry into the local cache.
// Returns the cache directory path.
func (c *Client) Pull(ctx context.Context, ref *Ref, refresh bool) (string, error) {
	cacheDir, err := ref.CachePath()
	if err != nil {
		return "", err
	}

	// Serve from cache unless refresh is requested
	if !refresh && ref.IsCached() {
		return cacheDir, nil
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("creating cache dir: %w", err)
	}

	// Pull into the cache directory
	store, err := file.New(cacheDir)
	if err != nil {
		return "", fmt.Errorf("creating file store: %w", err)
	}
	defer store.Close()

	repo, err := c.remoteRepo(ref)
	if err != nil {
		return "", err
	}

	if _, err := oras.Copy(ctx, repo, ref.Tag, store, ref.Tag, oras.DefaultCopyOptions); err != nil {
		// Clean up partial cache on failure
		os.RemoveAll(cacheDir)
		return "", fmt.Errorf("pulling: %w", err)
	}

	return cacheDir, nil
}

// PatternInfo holds the metadata returned by Info.
type PatternInfo struct {
	Ref      *Ref
	Digest   string
	Size     int64
	PushedAt time.Time
	Meta     *PatternMeta
}

// Info fetches manifest metadata without downloading the pattern files.
func (c *Client) Info(ctx context.Context, ref *Ref) (*PatternInfo, error) {
	repo, err := c.remoteRepo(ref)
	if err != nil {
		return nil, err
	}

	desc, _, err := repo.FetchReference(ctx, ref.Tag)
	if err != nil {
		return nil, fmt.Errorf("fetching manifest: %w", err)
	}

	// Read annotations from the manifest
	rc, err := repo.Fetch(ctx, desc)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}
	defer rc.Close()

	var manifest struct {
		Annotations map[string]string `json:"annotations"`
	}
	if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decoding manifest: %w", err)
	}

	meta := annotationsToMeta(manifest.Annotations)

	info := &PatternInfo{
		Ref:    ref,
		Digest: desc.Digest.String(),
		Size:   desc.Size,
		Meta:   meta,
	}

	if ts, ok := manifest.Annotations["org.opencontainers.image.created"]; ok {
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			info.PushedAt = t
		}
	}

	return info, nil
}

// List fetches the pattern index from the registry index artifact.
// Returns an empty index (not an error) when no patterns have been pushed yet.
func (c *Client) List(ctx context.Context, registryURL string) (*PatternIndex, error) {
	if registryURL == "" {
		registryURL = DefaultRegistry
	}
	clean := strings.TrimSuffix(strings.TrimPrefix(registryURL, "oci://"), "/")
	idxRef, err := parseRef(clean + "/index:latest")
	if err != nil {
		return nil, fmt.Errorf("building index ref: %w", err)
	}
	index, err := c.fetchIndex(ctx, idxRef)
	if err != nil {
		// Index doesn't exist yet — return empty, not an error.
		return &PatternIndex{UpdatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	return index, nil
}

// ── Index management ──────────────────────────────────────────────────────────

const indexMediaType = "application/vnd.orkestra.index.v1+json"

// indexRef derives the index repository ref from a pattern ref.
// "ghcr.io/orkspace/orkestra-registry/patterns/website:v1"
// →  "ghcr.io/orkspace/orkestra-registry/patterns/index:latest"
func indexRefFrom(ref *Ref) (*Ref, error) {
	lastSlash := strings.LastIndex(ref.Repository, "/")
	if lastSlash < 0 {
		return nil, fmt.Errorf("cannot derive index path from ref %q", ref.Full)
	}
	namespace := ref.Repository[:lastSlash]
	return parseRef(ref.Registry + "/" + namespace + "/index:latest")
}

// fetchIndex pulls the index JSON from the registry.
func (c *Client) fetchIndex(ctx context.Context, idxRef *Ref) (*PatternIndex, error) {
	repo, err := c.remoteRepo(idxRef)
	if err != nil {
		return nil, err
	}

	// Pull to a temp directory.
	tmp, err := os.MkdirTemp("", "ork-index-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	store, err := file.New(tmp)
	if err != nil {
		return nil, err
	}
	defer store.Close()

	if _, err := oras.Copy(ctx, repo, idxRef.Tag, store, idxRef.Tag, oras.DefaultCopyOptions); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(tmp, "index.json"))
	if err != nil {
		return nil, fmt.Errorf("reading index blob: %w", err)
	}

	var index PatternIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("decoding index: %w", err)
	}
	return &index, nil
}

// pushIndex writes the index JSON to the registry as an OCI artifact.
func (c *Client) pushIndex(ctx context.Context, idxRef *Ref, index *PatternIndex) error {
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}

	store := memory.New()

	blob := content.NewDescriptorFromBytes("application/json", data)
	if err := store.Push(ctx, blob, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("staging index blob: %w", err)
	}

	// Also stage as a named file so file-based pulls extract it as index.json.
	// We encode the descriptor's filename in the title annotation.
	blob.Annotations = map[string]string{
		"org.opencontainers.image.title": "index.json",
	}

	manifestDesc, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, indexMediaType, oras.PackManifestOptions{
		Layers: []ocispec.Descriptor{blob},
	})
	if err != nil {
		return fmt.Errorf("packing index manifest: %w", err)
	}

	if err := store.Tag(ctx, manifestDesc, idxRef.Tag); err != nil {
		return fmt.Errorf("tagging index: %w", err)
	}

	repo, err := c.remoteRepo(idxRef)
	if err != nil {
		return err
	}

	if _, err := oras.Copy(ctx, store, idxRef.Tag, repo, idxRef.Tag, oras.DefaultCopyOptions); err != nil {
		return fmt.Errorf("pushing index: %w", err)
	}
	return nil
}

// updateIndex fetches the existing index (if any), upserts the entry, and pushes.
func (c *Client) updateIndex(ctx context.Context, ref *Ref, entry PatternEntry) error {
	idxRef, err := indexRefFrom(ref)
	if err != nil {
		return err
	}

	index, err := c.fetchIndex(ctx, idxRef)
	if err != nil {
		index = &PatternIndex{} // first push — start fresh
	}

	found := false
	for i, p := range index.Patterns {
		if p.Name == entry.Name {
			index.Patterns[i] = entry
			found = true
			break
		}
	}
	if !found {
		index.Patterns = append(index.Patterns, entry)
	}
	index.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return c.pushIndex(ctx, idxRef, index)
}

// remoteRepo returns an authenticated ORAS remote.Repository for the ref.
func (c *Client) remoteRepo(ref *Ref) (*remote.Repository, error) {
	repo, err := remote.NewRepository(ref.Full)
	if err != nil {
		return nil, fmt.Errorf("invalid reference %q: %w", ref.Full, err)
	}
	repo.Client = &auth.Client{
		Client:     retry.DefaultClient,
		Cache:      auth.DefaultCache,
		Credential: credentials.Credential(c.credStore),
	}
	return repo, nil
}

// mediaTypeForFile returns the OCI media type for a pattern file.
func mediaTypeForFile(name string) string {
	switch name {
	case FileKatalog:
		return "application/vnd.orkestra.katalog.v1+yaml"
	case FileCRD:
		return "application/vnd.kubernetes.crd.v1+yaml"
	case FileCR:
		return "application/vnd.kubernetes.cr.v1+yaml"
	case FileReadme:
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}

// metaToAnnotations converts PatternMeta to OCI manifest annotations.
func metaToAnnotations(meta *PatternMeta, ref *Ref) map[string]string {
	ann := map[string]string{
		"org.opencontainers.image.created":     time.Now().UTC().Format(time.RFC3339),
		"org.opencontainers.image.title":       meta.Name,
		"org.opencontainers.image.version":     meta.Version,
		"org.opencontainers.image.description": meta.Description,
		"io.orkestra.pattern.name":             meta.Name,
		"io.orkestra.pattern.version":          meta.Version,
		"io.orkestra.pattern.author":           meta.Author,
		"io.orkestra.pattern.license":          meta.License,
		"io.orkestra.pattern.tags":             strings.Join(meta.Tags, ","),
	}
	if meta.Author != "" {
		ann["org.opencontainers.image.authors"] = meta.Author
	}
	return ann
}

// annotationsToMeta reconstructs PatternMeta from OCI annotations.
func annotationsToMeta(ann map[string]string) *PatternMeta {
	tags := []string{}
	if t := ann["io.orkestra.pattern.tags"]; t != "" {
		tags = strings.Split(t, ",")
	}
	return &PatternMeta{
		Name:        ann["io.orkestra.pattern.name"],
		Version:     ann["io.orkestra.pattern.version"],
		Description: ann["org.opencontainers.image.description"],
		Author:      ann["io.orkestra.pattern.author"],
		License:     ann["io.orkestra.pattern.license"],
		Tags:        tags,
	}
}
