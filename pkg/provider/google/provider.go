// pkg/provider/google/provider.go
//
// Google Cloud provider for Orkestra.
//
// Handles the "google" block in Katalog declarations.
// Uses the official Google Cloud Go SDKs.
//
// Supported resource kinds:
//
//	gcs      — Google Cloud Storage bucket (create, set labels, delete)
//	pubsub   — Pub/Sub topic (create, set labels, delete) and subscription (create, delete)
//	cloudsql — Cloud SQL instance (create, update tier, delete — PostgreSQL / MySQL)
//
// Installation:
//
//	go get cloud.google.com/go/storage
//	go get cloud.google.com/go/pubsub
//	go get google.golang.org/api/sqladmin/v1
//	go get golang.org/x/oauth2/google
//
// Registration:
//
//	p, err := googleprovider.NewFromAuth(ctx, auth)
//	registry.Register(p)
//
// Auth keys (providers[].auth block):
//
//	project         — GCP project ID (required)
//	credentialsFile — path to service account JSON (default: ADC / GOOGLE_APPLICATION_CREDENTIALS)
//	credentialsJSON — inline service account JSON (alternative to credentialsFile)
//
// Application Default Credentials (ADC) are used when no explicit credentials are provided.
// This means the provider works out of the box on GKE with Workload Identity.
//
// Katalog:
//
//	providers:
//	  - name: google
//	    required: true
//	    auth:
//	      project: "$GCP_PROJECT"
//
//	operatorBox:
//	  providers:
//	    google:
//	      - gcs:
//	          name: "{{ .metadata.name }}-{{ .metadata.namespace }}"
//	          location: US
//	          storageClass: STANDARD
//	          uniformAccess: "true"
//
//	      - pubsub:
//	          topic: "{{ .metadata.name }}-events"
//	          subscription: "{{ .metadata.name }}-events-sub"
//	          ackDeadline: "20"
//
//	      - cloudsql:
//	          name: "{{ .metadata.name }}-db"
//	          databaseVersion: POSTGRES_15
//	          tier: db-f1-micro
//	          region: us-central1
//	          when:
//	            - field: spec.needsDatabase
//	              equals: "true"
package googleprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	orktypes "github.com/orkspace/orkestra/pkg/types"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1"
)

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

// Provider implements orktypes.Provider for the "google" block.
type Provider struct {
	project string
	opts    []option.ClientOption // credential options passed to each GCP client
}

// New creates a Google Cloud provider for the given project using Application Default Credentials.
func New(project string) *Provider {
	return &Provider{project: project}
}

// NewFromAuth creates a Google Cloud provider from a Katalog auth map.
// Keys: project, credentialsFile, credentialsJSON.
func NewFromAuth(_ context.Context, auth map[string]string) (*Provider, error) {
	project := auth["project"]
	if project == "" {
		return nil, fmt.Errorf("google: auth.project is required")
	}

	var opts []option.ClientOption
	switch {
	case auth["credentialsJSON"] != "":
		opts = append(opts, option.WithCredentialsJSON([]byte(auth["credentialsJSON"])))
	case auth["credentialsFile"] != "":
		opts = append(opts, option.WithCredentialsFile(auth["credentialsFile"]))
		// else: use ADC
	}

	return &Provider{project: project, opts: opts}, nil
}

func (p *Provider) Name() string { return "google" }

// ─────────────────────────────────────────────────────────────────────────────
// Reconcile
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) Reconcile(ctx context.Context, req orktypes.ReconcileRequest) error {
	for _, decl := range req.Declarations {
		var err error
		switch decl.Kind {
		case "gcs":
			err = p.reconcileGCS(ctx, req, decl)
		case "pubsub":
			err = p.reconcilePubSub(ctx, req, decl)
		case "cloudsql":
			err = p.reconcileCloudSQL(ctx, req, decl)
		default:
			req.Logger.Warn().
				Str("kind", decl.Kind).
				Msg("google: unknown resource kind — skipped")
			continue
		}
		if err != nil {
			return fmt.Errorf("google.%s: %w", decl.Kind, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) Delete(ctx context.Context, req orktypes.DeleteRequest) error {
	for i := len(req.Declarations) - 1; i >= 0; i-- {
		decl := req.Declarations[i]
		var err error
		switch decl.Kind {
		case "gcs":
			err = p.deleteGCS(ctx, req, decl)
		case "pubsub":
			err = p.deletePubSub(ctx, req, decl)
		case "cloudsql":
			err = p.deleteCloudSQL(ctx, req, decl)
		}
		if err != nil {
			return fmt.Errorf("google.%s delete: %w", decl.Kind, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// GCS
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileGCS(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	name, err := decl.Require("name")
	if err != nil {
		return err
	}

	client, err := storage.NewClient(ctx, p.opts...)
	if err != nil {
		return fmt.Errorf("creating GCS client: %w", err)
	}
	defer client.Close()

	bucket := client.Bucket(name)
	_, err = bucket.Attrs(ctx)
	if err != nil {
		if !errors.Is(err, storage.ErrBucketNotExist) {
			return fmt.Errorf("checking bucket %q: %w", name, err)
		}

		// Create bucket
		attrs := &storage.BucketAttrs{
			Location:     decl.Field("location", "US"),
			StorageClass: decl.Field("storageClass", "STANDARD"),
			Labels: map[string]string{
				"orkestra-owner":     req.OwnerName,
				"orkestra-namespace": req.OwnerNamespace,
			},
		}
		if decl.Field("uniformAccess", "false") == "true" {
			attrs.UniformBucketLevelAccess = storage.UniformBucketLevelAccess{Enabled: true}
		}
		if err := bucket.Create(ctx, p.project, attrs); err != nil {
			return fmt.Errorf("creating bucket %q: %w", name, err)
		}
		req.Logger.Info().Str("bucket", name).Msg("google: GCS bucket created")
		return nil
	}

	req.Logger.Debug().Str("bucket", name).Msg("google: GCS bucket already exists — no-op")
	return nil
}

func (p *Provider) deleteGCS(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	name := decl.Field("name", "")
	if name == "" {
		return nil
	}

	client, err := storage.NewClient(ctx, p.opts...)
	if err != nil {
		return fmt.Errorf("creating GCS client: %w", err)
	}
	defer client.Close()

	if err := client.Bucket(name).Delete(ctx); err != nil {
		if errors.Is(err, storage.ErrBucketNotExist) {
			return nil // already gone
		}
		return fmt.Errorf("deleting bucket %q: %w", name, err)
	}

	req.Logger.Info().Str("bucket", name).Msg("google: GCS bucket deleted")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Pub/Sub
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcilePubSub(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	topicID, err := decl.Require("topic")
	if err != nil {
		return err
	}

	client, err := pubsub.NewClient(ctx, p.project, p.opts...)
	if err != nil {
		return fmt.Errorf("creating Pub/Sub client: %w", err)
	}
	defer client.Close()

	topic := client.Topic(topicID)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("checking topic %q: %w", topicID, err)
	}
	if !exists {
		if _, err := client.CreateTopic(ctx, topicID); err != nil {
			return fmt.Errorf("creating topic %q: %w", topicID, err)
		}
		req.Logger.Info().Str("topic", topicID).Msg("google: Pub/Sub topic created")
	} else {
		req.Logger.Debug().Str("topic", topicID).Msg("google: Pub/Sub topic already exists — no-op")
	}

	// Optional subscription
	subID := decl.Field("subscription", "")
	if subID == "" {
		return nil
	}

	sub := client.Subscription(subID)
	subExists, err := sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("checking subscription %q: %w", subID, err)
	}
	if subExists {
		req.Logger.Debug().Str("subscription", subID).Msg("google: Pub/Sub subscription already exists — no-op")
		return nil
	}

	ackDeadline := 10
	if s := decl.Field("ackDeadline", ""); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			ackDeadline = n
		}
	}

	if _, err := client.CreateSubscription(ctx, subID, pubsub.SubscriptionConfig{
		Topic:       topic,
		AckDeadline: time.Duration(ackDeadline) * time.Second,
	}); err != nil {
		return fmt.Errorf("creating subscription %q: %w", subID, err)
	}

	req.Logger.Info().Str("subscription", subID).Msg("google: Pub/Sub subscription created")
	return nil
}

func (p *Provider) deletePubSub(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	topicID := decl.Field("topic", "")
	if topicID == "" {
		return nil
	}

	client, err := pubsub.NewClient(ctx, p.project, p.opts...)
	if err != nil {
		return fmt.Errorf("creating Pub/Sub client: %w", err)
	}
	defer client.Close()

	// Delete subscription first if declared
	if subID := decl.Field("subscription", ""); subID != "" {
		sub := client.Subscription(subID)
		if exists, _ := sub.Exists(ctx); exists {
			if err := sub.Delete(ctx); err != nil {
				return fmt.Errorf("deleting subscription %q: %w", subID, err)
			}
			req.Logger.Info().Str("subscription", subID).Msg("google: Pub/Sub subscription deleted")
		}
	}

	topic := client.Topic(topicID)
	if exists, _ := topic.Exists(ctx); exists {
		if err := topic.Delete(ctx); err != nil {
			return fmt.Errorf("deleting topic %q: %w", topicID, err)
		}
		req.Logger.Info().Str("topic", topicID).Msg("google: Pub/Sub topic deleted")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Cloud SQL
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileCloudSQL(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	name, err := decl.Require("name")
	if err != nil {
		return err
	}

	svc, err := sqladmin.NewService(ctx, p.opts...)
	if err != nil {
		return fmt.Errorf("creating Cloud SQL client: %w", err)
	}

	dbVersion := decl.Field("databaseVersion", "POSTGRES_15")
	tier := decl.Field("tier", "db-f1-micro")
	region := decl.Field("region", "us-central1")

	_, err = svc.Instances.Get(p.project, name).Context(ctx).Do()
	if err != nil {
		if !isGoogleNotFound(err) {
			return fmt.Errorf("checking Cloud SQL instance %q: %w", name, err)
		}

		// Create
		if _, err := svc.Instances.Insert(p.project, &sqladmin.DatabaseInstance{
			Name:            name,
			DatabaseVersion: dbVersion,
			Region:          region,
			Settings: &sqladmin.Settings{
				Tier: tier,
				UserLabels: map[string]string{
					"orkestra-owner":     req.OwnerName,
					"orkestra-namespace": req.OwnerNamespace,
				},
			},
		}).Context(ctx).Do(); err != nil {
			return fmt.Errorf("creating Cloud SQL instance %q: %w", name, err)
		}

		req.Logger.Info().
			Str("instance", name).
			Str("version", dbVersion).
			Str("tier", tier).
			Msg("google: Cloud SQL instance creation initiated — will be available in a few minutes")
		return nil
	}

	req.Logger.Debug().Str("instance", name).Msg("google: Cloud SQL instance already exists — no-op")
	return nil
}

func (p *Provider) deleteCloudSQL(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	name := decl.Field("name", "")
	if name == "" {
		return nil
	}

	svc, err := sqladmin.NewService(ctx, p.opts...)
	if err != nil {
		return fmt.Errorf("creating Cloud SQL client: %w", err)
	}

	if _, err := svc.Instances.Delete(p.project, name).Context(ctx).Do(); err != nil {
		if isGoogleNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting Cloud SQL instance %q: %w", name, err)
	}

	req.Logger.Info().Str("instance", name).Msg("google: Cloud SQL instance deletion initiated")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func isGoogleNotFound(err error) bool {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusNotFound
	}
	return false
}
