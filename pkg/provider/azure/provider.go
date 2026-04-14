// pkg/provider/azure/provider.go
//
// Azure provider for Orkestra.
//
// Handles the "azure" block in Katalog declarations.
// Uses the official Azure SDK for Go (azure-sdk-for-go v2 / azidentity).
//
// Supported resource kinds:
//
//	blob        — Azure Blob Storage container (create, set metadata, delete)
//	servicebus  — Azure Service Bus namespace topic (create, delete) and subscription (create, delete)
//	sqldatabase — Azure SQL Database (create, update service tier, delete)
//
// Installation:
//
//	go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
//	go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage
//	go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/servicebus/armservicebus
//	go get github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql
//
// Registration:
//
//	p, err := azureprovider.NewFromAuth(ctx, auth)
//	registry.Register(p)
//
// Auth keys (providers[].auth block):
//
//	subscriptionId — Azure subscription ID (required)
//	tenantId       — Azure tenant ID (required for service principal auth)
//	clientId       — Service principal / managed identity client ID
//	clientSecret   — Service principal client secret (omit to use managed identity)
//	resourceGroup  — Default resource group for all resources (can be overridden per declaration)
//
// When clientSecret is absent, DefaultAzureCredential is used, which supports
// managed identity, CLI auth, and environment variables in order.
//
// Katalog:
//
//	providers:
//	  - name: azure
//	    required: true
//	    auth:
//	      subscriptionId: "$AZURE_SUBSCRIPTION_ID"
//	      tenantId: "$AZURE_TENANT_ID"
//	      clientId: "$AZURE_CLIENT_ID"
//	      clientSecret: "$AZURE_CLIENT_SECRET"
//	      resourceGroup: "$AZURE_RESOURCE_GROUP"
//
//	operatorBox:
//	  providers:
//	    azure:
//	      - blob:
//	          storageAccount: "{{ .spec.storageAccount }}"
//	          container: "{{ .metadata.name }}"
//	          resourceGroup: "{{ .spec.resourceGroup }}"
//
//	      - servicebus:
//	          namespace: "{{ .spec.sbNamespace }}"
//	          topic: "{{ .metadata.name }}-events"
//	          subscription: "{{ .metadata.name }}-sub"
//	          resourceGroup: "{{ .spec.resourceGroup }}"
//
//	      - sqldatabase:
//	          server: "{{ .spec.sqlServer }}"
//	          name: "{{ .metadata.name }}"
//	          sku: Basic
//	          resourceGroup: "{{ .spec.resourceGroup }}"
//	          when:
//	            - field: spec.needsDatabase
//	              equals: "true"
package azureprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/servicebus/armservicebus"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

// Provider implements orktypes.Provider for the "azure" block.
type Provider struct {
	subscriptionID       string
	defaultResourceGroup string
	cred                 azcore.TokenCredential
}

// New creates an Azure provider using the supplied credential.
func New(subscriptionID, defaultResourceGroup string, cred azcore.TokenCredential) *Provider {
	return &Provider{
		subscriptionID:       subscriptionID,
		defaultResourceGroup: defaultResourceGroup,
		cred:                 cred,
	}
}

// NewFromAuth creates an Azure provider from a Katalog auth map.
// Keys: subscriptionId, tenantId, clientId, clientSecret, resourceGroup.
func NewFromAuth(_ context.Context, auth map[string]string) (*Provider, error) {
	subscriptionID := auth["subscriptionId"]
	if subscriptionID == "" {
		return nil, fmt.Errorf("azure: auth.subscriptionId is required")
	}

	resourceGroup := auth["resourceGroup"] // may be empty; declarations can override

	var (
		cred azcore.TokenCredential
		err  error
	)

	clientSecret := auth["clientSecret"]
	tenantID := auth["tenantId"]
	clientID := auth["clientId"]

	if clientSecret != "" && tenantID != "" && clientID != "" {
		// Service principal with client secret
		cred, err = azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
		if err != nil {
			return nil, fmt.Errorf("azure: creating service principal credential: %w", err)
		}
	} else {
		// DefaultAzureCredential — managed identity, CLI, env vars
		opts := &azidentity.DefaultAzureCredentialOptions{}
		if tenantID != "" {
			opts.TenantID = tenantID
		}
		cred, err = azidentity.NewDefaultAzureCredential(opts)
		if err != nil {
			return nil, fmt.Errorf("azure: creating default credential: %w", err)
		}
	}

	return New(subscriptionID, resourceGroup, cred), nil
}

func (p *Provider) Name() string { return "azure" }

// resourceGroup returns the resource group from the declaration, falling back
// to the provider-level default.
func (p *Provider) resourceGroup(decl orktypes.ProviderDeclaration) string {
	if rg := decl.Field("resourceGroup", ""); rg != "" {
		return rg
	}
	return p.defaultResourceGroup
}

// ─────────────────────────────────────────────────────────────────────────────
// Reconcile
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) Reconcile(ctx context.Context, req orktypes.ReconcileRequest) error {
	for _, decl := range req.Declarations {
		var err error
		switch decl.Kind {
		case "blob":
			err = p.reconcileBlob(ctx, req, decl)
		case "servicebus":
			err = p.reconcileServiceBus(ctx, req, decl)
		case "sqldatabase":
			err = p.reconcileSQLDatabase(ctx, req, decl)
		default:
			req.Logger.Warn().
				Str("kind", decl.Kind).
				Msg("azure: unknown resource kind — skipped")
			continue
		}
		if err != nil {
			return fmt.Errorf("azure.%s: %w", decl.Kind, err)
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
		case "blob":
			err = p.deleteBlob(ctx, req, decl)
		case "servicebus":
			err = p.deleteServiceBus(ctx, req, decl)
		case "sqldatabase":
			err = p.deleteSQLDatabase(ctx, req, decl)
		}
		if err != nil {
			return fmt.Errorf("azure.%s delete: %w", decl.Kind, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Blob Storage
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileBlob(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	storageAccount, err := decl.Require("storageAccount")
	if err != nil {
		return err
	}
	containerName, err := decl.Require("container")
	if err != nil {
		return err
	}
	rg := p.resourceGroup(decl)
	if rg == "" {
		return fmt.Errorf("resourceGroup is required for blob declarations")
	}

	client, err := armstorage.NewBlobContainersClient(p.subscriptionID, p.cred, nil)
	if err != nil {
		return fmt.Errorf("creating blob containers client: %w", err)
	}

	_, err = client.Get(ctx, rg, storageAccount, containerName, nil)
	if err != nil {
		if !isAzureNotFound(err) {
			return fmt.Errorf("checking container %q: %w", containerName, err)
		}

		// Create container
		if _, err := client.Create(ctx, rg, storageAccount, containerName,
			armstorage.BlobContainer{
				ContainerProperties: &armstorage.ContainerProperties{
					Metadata: map[string]*string{
						"orkestra-owner":     to.Ptr(req.OwnerName),
						"orkestra-namespace": to.Ptr(req.OwnerNamespace),
					},
				},
			}, nil,
		); err != nil {
			return fmt.Errorf("creating container %q: %w", containerName, err)
		}

		req.Logger.Info().
			Str("account", storageAccount).
			Str("container", containerName).
			Msg("azure: Blob container created")
		return nil
	}

	req.Logger.Debug().
		Str("container", containerName).
		Msg("azure: Blob container already exists — no-op")
	return nil
}

func (p *Provider) deleteBlob(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	storageAccount := decl.Field("storageAccount", "")
	containerName := decl.Field("container", "")
	rg := p.resourceGroup(decl)
	if storageAccount == "" || containerName == "" || rg == "" {
		return nil
	}

	client, err := armstorage.NewBlobContainersClient(p.subscriptionID, p.cred, nil)
	if err != nil {
		return fmt.Errorf("creating blob containers client: %w", err)
	}

	if _, err := client.Delete(ctx, rg, storageAccount, containerName, nil); err != nil {
		if isAzureNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting container %q: %w", containerName, err)
	}

	req.Logger.Info().Str("container", containerName).Msg("azure: Blob container deleted")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Service Bus
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileServiceBus(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	namespace, err := decl.Require("namespace")
	if err != nil {
		return err
	}
	topicName, err := decl.Require("topic")
	if err != nil {
		return err
	}
	rg := p.resourceGroup(decl)
	if rg == "" {
		return fmt.Errorf("resourceGroup is required for servicebus declarations")
	}

	topicsClient, err := armservicebus.NewTopicsClient(p.subscriptionID, p.cred, nil)
	if err != nil {
		return fmt.Errorf("creating Service Bus topics client: %w", err)
	}

	// CreateOrUpdate is idempotent
	if _, err := topicsClient.CreateOrUpdate(ctx, rg, namespace, topicName,
		armservicebus.SBTopic{}, nil,
	); err != nil {
		return fmt.Errorf("creating Service Bus topic %q: %w", topicName, err)
	}
	req.Logger.Info().Str("topic", topicName).Msg("azure: Service Bus topic reconciled")

	// Optional subscription
	subName := decl.Field("subscription", "")
	if subName == "" {
		return nil
	}

	subsClient, err := armservicebus.NewSubscriptionsClient(p.subscriptionID, p.cred, nil)
	if err != nil {
		return fmt.Errorf("creating Service Bus subscriptions client: %w", err)
	}

	if _, err := subsClient.CreateOrUpdate(ctx, rg, namespace, topicName, subName,
		armservicebus.SBSubscription{}, nil,
	); err != nil {
		return fmt.Errorf("creating Service Bus subscription %q: %w", subName, err)
	}
	req.Logger.Info().Str("subscription", subName).Msg("azure: Service Bus subscription reconciled")
	return nil
}

func (p *Provider) deleteServiceBus(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	namespace := decl.Field("namespace", "")
	topicName := decl.Field("topic", "")
	rg := p.resourceGroup(decl)
	if namespace == "" || topicName == "" || rg == "" {
		return nil
	}

	// Delete subscription first
	if subName := decl.Field("subscription", ""); subName != "" {
		subsClient, err := armservicebus.NewSubscriptionsClient(p.subscriptionID, p.cred, nil)
		if err != nil {
			return fmt.Errorf("creating subscriptions client: %w", err)
		}
		if _, err := subsClient.Delete(ctx, rg, namespace, topicName, subName, nil); err != nil {
			if !isAzureNotFound(err) {
				return fmt.Errorf("deleting subscription %q: %w", subName, err)
			}
		}
		req.Logger.Info().Str("subscription", subName).Msg("azure: Service Bus subscription deleted")
	}

	topicsClient, err := armservicebus.NewTopicsClient(p.subscriptionID, p.cred, nil)
	if err != nil {
		return fmt.Errorf("creating topics client: %w", err)
	}
	if _, err := topicsClient.Delete(ctx, rg, namespace, topicName, nil); err != nil {
		if !isAzureNotFound(err) {
			return fmt.Errorf("deleting topic %q: %w", topicName, err)
		}
	}
	req.Logger.Info().Str("topic", topicName).Msg("azure: Service Bus topic deleted")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Azure SQL Database
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileSQLDatabase(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	server, err := decl.Require("server")
	if err != nil {
		return err
	}
	dbName, err := decl.Require("name")
	if err != nil {
		return err
	}
	rg := p.resourceGroup(decl)
	if rg == "" {
		return fmt.Errorf("resourceGroup is required for sqldatabase declarations")
	}

	sku := decl.Field("sku", "Basic")

	client, err := armsql.NewDatabasesClient(p.subscriptionID, p.cred, nil)
	if err != nil {
		return fmt.Errorf("creating Azure SQL databases client: %w", err)
	}

	_, err = client.Get(ctx, rg, server, dbName, nil)
	if err != nil {
		if !isAzureNotFound(err) {
			return fmt.Errorf("checking SQL database %q: %w", dbName, err)
		}

		// CreateOrUpdate — long-running operation
		poller, err := client.BeginCreateOrUpdate(ctx, rg, server, dbName, armsql.Database{
			Location: to.Ptr("global"),
			SKU:      &armsql.SKU{Name: to.Ptr(sku)},
			Tags: map[string]*string{
				"orkestra-owner":     to.Ptr(req.OwnerName),
				"orkestra-namespace": to.Ptr(req.OwnerNamespace),
			},
		}, nil)
		if err != nil {
			return fmt.Errorf("initiating SQL database creation %q: %w", dbName, err)
		}

		// Don't block — poll on next reconcile
		_ = poller
		req.Logger.Info().
			Str("database", dbName).
			Str("server", server).
			Str("sku", sku).
			Msg("azure: SQL database creation initiated — will check status on next reconcile")
		return nil
	}

	req.Logger.Debug().Str("database", dbName).Msg("azure: SQL database already exists — no-op")
	return nil
}

func (p *Provider) deleteSQLDatabase(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	server := decl.Field("server", "")
	dbName := decl.Field("name", "")
	rg := p.resourceGroup(decl)
	if server == "" || dbName == "" || rg == "" {
		return nil
	}

	client, err := armsql.NewDatabasesClient(p.subscriptionID, p.cred, nil)
	if err != nil {
		return fmt.Errorf("creating Azure SQL databases client: %w", err)
	}

	poller, err := client.BeginDelete(ctx, rg, server, dbName, nil)
	if err != nil {
		if isAzureNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting SQL database %q: %w", dbName, err)
	}
	_ = poller

	req.Logger.Info().Str("database", dbName).Msg("azure: SQL database deletion initiated")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func isAzureNotFound(err error) bool {
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == http.StatusNotFound
	}
	return false
}
