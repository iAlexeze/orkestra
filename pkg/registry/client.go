// pkg/registry/client.go
//
// ORAS-based OCI client for pushing and pulling Orkestra artifacts.
//
// Authentication uses ~/.docker/config.json via oras.land/oras-go/v2.
// No separate login step — docker login ghcr.io is sufficient.
//
// Push: validates the directory, reads each file, pushes as OCI layers.
// Pull: fetches the manifest, extracts layers to the cache directory.
// Info: fetches the manifest only, reads annotations.
// List: fetches the index pattern from the registry root.
package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

// Push validates the directory, auto-detects the pattern kind, and pushes all
// files to the registry. Returns the manifest digest on success.
// e2eMeta and simulateMeta are optional; when non-nil their fields are embedded
// as OCI annotations on the published artifact.
// typedMeta is optional; when non-nil it is written as typed-operator annotations.
func (c *Client) Push(ctx context.Context, ref *Ref, dir string, e2eMeta *PatternE2E, simulateMeta *PatternSimulate, typedMeta *PatternTyped, progress func(file string, size int64)) (string, error) {
	patternKind, spec, files, err := ValidatePatternDirectory(dir)
	if err != nil {
		return "", fmt.Errorf("validation failed: %w", err)
	}

	meta, err := LoadPatternMeta(dir, spec)
	if err != nil {
		return "", fmt.Errorf("reading metadata: %w", err)
	}

	if e2eMeta != nil {
		meta.E2E = e2eMeta
	}
	if simulateMeta != nil {
		meta.Simulate = simulateMeta
	}
	if typedMeta != nil {
		meta.Typed = typedMeta
	}

	store := memory.New()

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
		desc := content.NewDescriptorFromBytes(mediaTypeForPatternFile(f, patternKind), data)
		desc.Annotations = map[string]string{
			"org.opencontainers.image.title": f,
		}
		if err := store.Push(ctx, desc, bytes.NewReader(data)); err != nil {
			return "", fmt.Errorf("staging %s: %w", f, err)
		}
		descs = append(descs, desc)
	}

	annotations := artifactMetaToAnnotations(meta, ref)
	manifestDesc, err := oras.Pack(ctx, store, spec.MediaType, descs, oras.PackOptions{
		PackImageManifest:   true,
		ManifestAnnotations: annotations,
	})
	if err != nil {
		return "", fmt.Errorf("packing manifest: %w", err)
	}

	if err := store.Tag(ctx, manifestDesc, ref.Tag); err != nil {
		return "", fmt.Errorf("tagging manifest: %w", err)
	}

	repo, err := c.remoteRepo(ref)
	if err != nil {
		return "", err
	}

	if _, err := oras.Copy(ctx, store, ref.Tag, repo, ref.Tag, oras.DefaultCopyOptions); err != nil {
		return "", fmt.Errorf("pushing: %w", err)
	}

	entry := PatternEntry{
		Name:          meta.Name,
		LatestVersion: meta.Version,
		Description:   meta.Description,
		Tags:          meta.Tags,
		Author:        meta.Author,
		Kind:          string(patternKind),
	}
	if meta.E2E != nil {
		entry.E2EStatus = meta.E2E.Status
	}
	if meta.Simulate != nil {
		entry.SimulateStatus = meta.Simulate.Status
	}
	if meta.Deprecated != nil {
		entry.Deprecated = true
	}
	if err := c.updateIndex(ctx, ref, entry); err != nil {
		fmt.Fprintf(os.Stderr, "warning: index update failed: %v\n", err)
	}

	return manifestDesc.Digest.String(), nil
}

// Pull fetches an pattern from the registry into the local cache.
// Returns the cache directory path.
func (c *Client) Pull(ctx context.Context, ref *Ref, refresh bool) (string, error) {
	cacheDir, err := ref.CachePath()
	if err != nil {
		return "", err
	}

	if !refresh && ref.IsCached() {
		return cacheDir, nil
	}

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("creating cache dir: %w", err)
	}

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
		os.RemoveAll(cacheDir)
		return "", fmt.Errorf("pulling: %w", err)
	}

	return cacheDir, nil
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

// List fetches the pattern index from the given registry URL.
// Returns an empty index (not an error) when no artifacts have been pushed yet.
func (c *Client) List(ctx context.Context, registryURL string) (*PatternIndex, error) {
	if registryURL == "" {
		registryURL = DefaultPatternRegistry
	}
	clean := strings.TrimSuffix(strings.TrimPrefix(registryURL, "oci://"), "/")
	idxRef, err := parseRef(clean + "/index:latest")
	if err != nil {
		return nil, fmt.Errorf("building index ref: %w", err)
	}
	index, err := c.fetchIndex(ctx, idxRef)
	if err != nil {
		return &PatternIndex{UpdatedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	return index, nil
}

// ── Index management ──────────────────────────────────────────────────────────

func indexRefFrom(ref *Ref) (*Ref, error) {
	lastSlash := strings.LastIndex(ref.Repository, "/")
	if lastSlash < 0 {
		return nil, fmt.Errorf("cannot derive index path from ref %q", ref.Full)
	}
	namespace := ref.Repository[:lastSlash]
	return parseRef(ref.Registry + "/" + namespace + "/index:latest")
}

func (c *Client) fetchIndex(ctx context.Context, idxRef *Ref) (*PatternIndex, error) {
	repo, err := c.remoteRepo(idxRef)
	if err != nil {
		return nil, err
	}

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

func (c *Client) updateIndex(ctx context.Context, ref *Ref, entry PatternEntry) error {
	idxRef, err := indexRefFrom(ref)
	if err != nil {
		return err
	}

	index, err := c.fetchIndex(ctx, idxRef)
	if err != nil {
		index = &PatternIndex{}
	}

	found := false
	for i, e := range index.Entries {
		if e.Name == entry.Name {
			index.Entries[i] = entry
			found = true
			break
		}
	}
	if !found {
		index.Entries = append(index.Entries, entry)
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

// artifactMetaToAnnotations converts PatternMeta to OCI manifest annotations.
func artifactMetaToAnnotations(meta *PatternMeta, ref *Ref) map[string]string {
	ann := map[string]string{
		"org.opencontainers.image.created":     time.Now().UTC().Format(time.RFC3339),
		"org.opencontainers.image.title":       meta.Name,
		"org.opencontainers.image.version":     meta.Version,
		"org.opencontainers.image.description": meta.Description,
		"io.orkestra.pattern.kind":             string(meta.Kind),
		"io.orkestra.pattern.name":             meta.Name,
		"io.orkestra.pattern.version":          meta.Version,
		"io.orkestra.pattern.author":           meta.Author,
		"io.orkestra.pattern.license":          meta.License,
		"io.orkestra.pattern.tags":             strings.Join(meta.Tags, ","),
	}
	if meta.Author != "" {
		ann["org.opencontainers.image.authors"] = meta.Author
	}
	if meta.E2E != nil {
		ann["io.orkestra.e2e.status"] = meta.E2E.Status
		if meta.E2E.Duration != "" {
			ann["io.orkestra.e2e.duration"] = meta.E2E.Duration
		}
		if meta.E2E.TestedAt != "" {
			ann["io.orkestra.e2e.tested_at"] = meta.E2E.TestedAt
		}
		if meta.E2E.Runner != "" {
			ann["io.orkestra.e2e.runner"] = meta.E2E.Runner
		}
		if meta.E2E.Assertions > 0 {
			ann["io.orkestra.e2e.assertions"] = strconv.Itoa(meta.E2E.Assertions)
		}
	}
	if meta.Simulate != nil {
		ann["io.orkestra.simulate.status"] = meta.Simulate.Status
		if meta.Simulate.Duration != "" {
			ann["io.orkestra.simulate.duration"] = meta.Simulate.Duration
		}
		if meta.Simulate.TestedAt != "" {
			ann["io.orkestra.simulate.tested_at"] = meta.Simulate.TestedAt
		}
		if meta.Simulate.Assertions > 0 {
			ann["io.orkestra.simulate.assertions"] = strconv.Itoa(meta.Simulate.Assertions)
		}
	}
	if meta.Typed != nil {
		if meta.Typed.HasHooks {
			ann["io.orkestra.katalog.has_hooks"] = "true"
		}
		if meta.Typed.HasConstructor {
			ann["io.orkestra.katalog.has_constructor"] = "true"
		}
		if meta.Typed.HasHooks || meta.Typed.HasConstructor {
			ann["io.orkestra.katalog.typed"] = "true"
		}
	}
	if meta.Deprecated != nil {
		ann["io.orkestra.katalog.deprecated"] = "true"
		if meta.Deprecated.MigratedTo != "" {
			ann["io.orkestra.katalog.deprecated.migrated_to"] = meta.Deprecated.MigratedTo
		}
		if meta.Deprecated.Message != "" {
			ann["io.orkestra.katalog.deprecated.message"] = meta.Deprecated.Message
		}
	}
	return ann
}

// annotationsToMeta reconstructs PatternMeta from OCI manifest annotations.
func annotationsToMeta(ann map[string]string) *PatternMeta {
	tags := []string{}
	name := ann["io.orkestra.pattern.name"]
	if name == "" {
		name = ann["io.orkestra.pattern.name"]
	}
	version := ann["io.orkestra.pattern.version"]
	if version == "" {
		version = ann["io.orkestra.pattern.version"]
	}
	author := ann["io.orkestra.pattern.author"]
	if author == "" {
		author = ann["io.orkestra.pattern.author"]
	}
	license := ann["io.orkestra.pattern.license"]
	if license == "" {
		license = ann["io.orkestra.pattern.license"]
	}
	kindStr := ann["io.orkestra.pattern.kind"]
	if t := ann["io.orkestra.pattern.tags"]; t != "" {
		tags = strings.Split(t, ",")
	} else if t := ann["io.orkestra.pattern.tags"]; t != "" {
		tags = strings.Split(t, ",")
	}
	meta := &PatternMeta{
		Kind:        PatternKind(kindStr),
		Name:        name,
		Version:     version,
		Description: ann["org.opencontainers.image.description"],
		Author:      author,
		License:     license,
		Tags:        tags,
	}
	if status := ann["io.orkestra.e2e.status"]; status != "" {
		n, _ := strconv.Atoi(ann["io.orkestra.e2e.assertions"])
		meta.E2E = &PatternE2E{
			Status:     status,
			Duration:   ann["io.orkestra.e2e.duration"],
			TestedAt:   ann["io.orkestra.e2e.tested_at"],
			Runner:     ann["io.orkestra.e2e.runner"],
			Assertions: n,
		}
	}
	if status := ann["io.orkestra.simulate.status"]; status != "" {
		n, _ := strconv.Atoi(ann["io.orkestra.simulate.assertions"])
		meta.Simulate = &PatternSimulate{
			Status:     status,
			Duration:   ann["io.orkestra.simulate.duration"],
			TestedAt:   ann["io.orkestra.simulate.tested_at"],
			Assertions: n,
		}
	}
	if ann["io.orkestra.katalog.typed"] == "true" {
		meta.Typed = &PatternTyped{
			HasHooks:       ann["io.orkestra.katalog.has_hooks"] == "true",
			HasConstructor: ann["io.orkestra.katalog.has_constructor"] == "true",
		}
	}
	if ann["io.orkestra.katalog.deprecated"] == "true" {
		meta.Deprecated = &PatternDeprecated{
			MigratedTo: ann["io.orkestra.katalog.deprecated.migrated_to"],
			Message:    ann["io.orkestra.katalog.deprecated.message"],
		}
	}
	return meta
}
