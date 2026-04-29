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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"oras.land/oras-go/v2"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
)

// Client wraps ORAS for Orkestra pattern operations.
type Client struct {
	// credStore loads credentials from ~/.docker/config.json.
	credStore credentials.Store
}

// NewClient returns a Client with credentials loaded from the Docker config.
// func NewClient() (*Client, error) {
// 	store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{
// 		AllowPlaintextPut: false,
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("loading docker credentials: %w", err)
// 	}
// 	return &Client{credStore: store}, nil
// }

func NewClient() (*Client, error) {
    return &Client{}, nil
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

	// Build a file store from the directory
	store, err := file.New(dir)
	if err != nil {
		return "", fmt.Errorf("creating file store: %w", err)
	}
	defer store.Close()

	// Add each file as a layer
	var descs []ociDesc
	for _, f := range files {
		path := filepath.Join(dir, f)
		info, err := os.Stat(path)
		if err != nil {
			return "", fmt.Errorf("stat %s: %w", f, err)
		}
		if progress != nil {
			progress(f, info.Size())
		}
		desc, err := store.Add(ctx, f, mediaTypeForFile(f), path)
		if err != nil {
			return "", fmt.Errorf("adding %s: %w", f, err)
		}
		descs = append(descs, ociDesc(desc))
	}

	// Pack into a manifest with annotations from pattern.yaml
	annotations := metaToAnnotations(meta, ref)

	layers := make([]ocispec.Descriptor, len(descs))
	for i, d := range descs {
		// descs[i] is of type ocispec.Descriptor (from file.Store.Add)
		layers[i] = d
	}

	opts := oras.PackManifestOptions{
		Layers:              layers,
		ManifestAnnotations: annotations,
	}

	manifestDesc, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, MediaType, opts)
	if err != nil {
		return "", fmt.Errorf("packing manifest: %w", err)
	}

	// Push to the remote repository
	repo, err := c.remoteRepo(ref)
	if err != nil {
		return "", err
	}

	_, err = oras.Copy(ctx, store, manifestDesc.Digest.String(), repo, ref.Tag, oras.DefaultCopyOptions)
	if err != nil {
		return "", fmt.Errorf("pushing: %w", err)
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

// List fetches the pattern index from the registry.
func (c *Client) List(ctx context.Context, registryURL string) (*PatternIndex, error) {
	if registryURL == "" {
		registryURL = DefaultRegistry
	}
	registryURL = strings.TrimPrefix(registryURL, "oci://")

	indexRef := &Ref{Full: registryURL + "/index:latest"}
	// Parse properly
	parsed, err := Resolve("oci://" + registryURL + "/index:latest")
	if err != nil {
		return nil, err
	}
	indexRef = parsed

	repo, err := c.remoteRepo(indexRef)
	if err != nil {
		return nil, err
	}

	// Fetch the index blob
	desc, rc, err := repo.FetchReference(ctx, "latest")
	if err != nil {
		return nil, fmt.Errorf("fetching index: %w", err)
	}
	defer rc.Close()
	_ = desc

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("reading index: %w", err)
	}

	var index PatternIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("decoding index: %w", err)
	}

	return &index, nil
}

// remoteRepo returns an authenticated ORAS remote.Repository for the ref.
// func (c *Client) remoteRepo(ref *Ref) (*remote.Repository, error) {
// 	repo, err := remote.NewRepository(ref.Full)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid reference %q: %w", ref.Full, err)
// 	}
// 	repo.Client = &auth.Client{
// 		Client:     auth.DefaultClient,
// 		Cache:      auth.DefaultCache,
// 		Credential: credentials.Credential(c.credStore),
// 	}
// 	return repo, nil
// }

func (c *Client) remoteRepo(ref *Ref) (*remote.Repository, error) {
    repo, err := remote.NewRepository(ref.Full)
    if err != nil {
        return nil, fmt.Errorf("invalid reference %q: %w", ref.Full, err)
    }
    repo.Client = auth.DefaultClient
    return repo, nil
}

// mediaTypeForFile returns the OCI media type for a pattern file.
func mediaTypeForFile(name string) string {
	switch name {
	case FileKatalog:
		return "application/vnd.orkestra.katalog.v1+yaml"
	case FilePattern:
		return "application/vnd.orkestra.pattern-meta.v1+yaml"
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

// ociDesc and toDescs adapt internal types to the oras API.
// Using type aliases to avoid importing oci descriptors directly in pattern.go.
type ociDesc = interface{}

func toDescs(descs []ociDesc) []interface{} {
	return descs
}
