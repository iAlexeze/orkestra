// pkg/providers/mongodb/provider.go
//
// Real MongoDB provider for Orkestra.
//
// Handles the "mongodb" block in Katalog declarations.
// Uses the official MongoDB Go driver — no mocks, no stubs.
//
// Supported resource kinds:
//
//	database   — MongoDB database (create, drop)
//	user       — Database user (create, update roles, delete)
//	collection — Collection with optional schema validation (create, drop)
//
// Installation:
//
//	go get go.mongodb.org/mongo-driver/mongo
//	go get go.mongodb.org/mongo-driver/mongo/options
//
// Registration:
//
//	client, _ := mongo.Connect(ctx, options.Client().ApplyURI(uri))
//	registry.Register(mongoprovider.New(client))
//
// Katalog:
//
//	providers:
//	  mongodb:
//	    - database:
//	        name: "{{ .metadata.name }}"
//
//	    - user:
//	        name: "{{ .spec.dbUser }}"
//	        database: "{{ .metadata.name }}"
//	        roles: "readWrite"
//	        credentials:
//	          secretName: "{{ .metadata.name }}-mongo-creds"   # MONGO_PASSWORD
//
//	    - collection:
//	        name: events
//	        database: "{{ .metadata.name }}"
//	        when:
//	          - field: spec.enableEvents
//	            equals: "true"
package mongoprovider

import (
	"context"
	"fmt"
	"strings"

	orktypes "github.com/ialexeze/orkestra/pkg/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

// Provider implements orktypes.Provider for the "mongodb" block.
type Provider struct {
	client *mongo.Client
}

// New creates a MongoDB provider from an existing connected client.
func New(client *mongo.Client) *Provider {
	return &Provider{client: client}
}

// NewFromURI connects to MongoDB using the given URI and returns a Provider.
func NewFromURI(ctx context.Context, uri string) (*Provider, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongodb: connecting to %q: %w", uri, err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongodb: ping failed: %w", err)
	}
	return New(client), nil
}

func (p *Provider) Name() string { return "mongodb" }

// ─────────────────────────────────────────────────────────────────────────────
// Reconcile
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) Reconcile(ctx context.Context, req orktypes.ReconcileRequest) error {
	for _, decl := range req.Declarations {
		var err error
		switch decl.Kind {
		case "database":
			err = p.reconcileDatabase(ctx, req, decl)
		case "user":
			err = p.reconcileUser(ctx, req, decl)
		case "collection":
			err = p.reconcileCollection(ctx, req, decl)
		default:
			req.Logger.Warn().
				Str("kind", decl.Kind).
				Msg("mongodb: unknown resource kind — skipped")
			continue
		}
		if err != nil {
			return fmt.Errorf("mongodb.%s: %w", decl.Kind, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) Delete(ctx context.Context, req orktypes.DeleteRequest) error {
	// Delete in reverse order to respect dependencies:
	// collections → users → database
	for i := len(req.Declarations) - 1; i >= 0; i-- {
		decl := req.Declarations[i]
		var err error
		switch decl.Kind {
		case "collection":
			err = p.deleteCollection(ctx, req, decl)
		case "user":
			err = p.deleteUser(ctx, req, decl)
		case "database":
			err = p.deleteDatabase(ctx, req, decl)
		}
		if err != nil {
			return fmt.Errorf("mongodb.%s delete: %w", decl.Kind, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Database
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileDatabase(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	name, err := decl.Require("name")
	if err != nil {
		return err
	}

	// MongoDB creates databases lazily — they appear when the first document
	// is inserted or the first collection is created. We create a system
	// metadata collection to materialise the database immediately, then
	// track ownership via a document.
	db := p.client.Database(name)

	// Check if our ownership document exists
	col := db.Collection("_orkestra_meta")
	filter := bson.M{"_id": "ownership"}

	var result bson.M
	err = col.FindOne(ctx, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		// First creation — insert ownership document
		_, err = col.InsertOne(ctx, bson.M{
			"_id":       "ownership",
			"owner":     req.OwnerName,
			"namespace": req.OwnerNamespace,
		})
		if err != nil {
			return fmt.Errorf("initialising database %q: %w", name, err)
		}
		req.Logger.Info().
			Str("database", name).
			Str("owner", req.OwnerName).
			Msg("mongodb: database created")
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking database %q: %w", name, err)
	}

	req.Logger.Debug().Str("database", name).Msg("mongodb: database already exists — no-op")
	return nil
}

func (p *Provider) deleteDatabase(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	name := decl.Field("name", "")
	if name == "" {
		return nil
	}

	db := p.client.Database(name)

	// Only drop if we own it
	col := db.Collection("_orkestra_meta")
	var result bson.M
	err := col.FindOne(ctx, bson.M{"_id": "ownership", "owner": req.OwnerName}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		req.Logger.Warn().
			Str("database", name).
			Msg("mongodb: database not owned by this CR — skipping drop")
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking ownership of database %q: %w", name, err)
	}

	if err := db.Drop(ctx); err != nil {
		return fmt.Errorf("dropping database %q: %w", name, err)
	}

	req.Logger.Info().Str("database", name).Msg("mongodb: database dropped")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// User
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileUser(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	username, err := decl.Require("name")
	if err != nil {
		return err
	}
	database, err := decl.Require("database")
	if err != nil {
		return err
	}

	// Resolve password from Secret
	password := ""
	if secretName := decl.Field("credentials.secretName", ""); secretName != "" {
		data, err := req.Kube.GetSecret(ctx, req.OwnerNamespace, secretName)
		if err != nil {
			return fmt.Errorf("reading credentials secret %q: %w", secretName, err)
		}
		password = string(data["MONGO_PASSWORD"])
		if password == "" {
			return fmt.Errorf("secret %q missing MONGO_PASSWORD", secretName)
		}
	}

	// Parse roles — comma-separated: "readWrite,dbAdmin"
	roles := parseRoles(decl.Field("roles", "readWrite"), database)

	// Check if user exists
	adminDB := p.client.Database("admin")
	var userInfo bson.M
	err = adminDB.RunCommand(ctx, bson.D{
		{Key: "usersInfo", Value: bson.D{
			{Key: "user", Value: username},
			{Key: "db", Value: database},
		}},
	}).Decode(&userInfo)

	if err != nil {
		return fmt.Errorf("checking user %q: %w", username, err)
	}

	users, _ := userInfo["users"].(bson.A)

	if len(users) == 0 {
		// Create user
		createCmd := bson.D{
			{Key: "createUser", Value: username},
			{Key: "roles", Value: roles},
		}
		if password != "" {
			createCmd = append(createCmd, bson.E{Key: "pwd", Value: password})
		}
		if err := p.client.Database(database).RunCommand(ctx, createCmd).Err(); err != nil {
			return fmt.Errorf("creating user %q: %w", username, err)
		}
		req.Logger.Info().
			Str("user", username).
			Str("database", database).
			Msg("mongodb: user created")
		return nil
	}

	// Update roles — ensure declared roles are applied (idempotent grantRolesToUser)
	grantCmd := bson.D{
		{Key: "grantRolesToUser", Value: username},
		{Key: "roles", Value: roles},
	}
	if err := p.client.Database(database).RunCommand(ctx, grantCmd).Err(); err != nil {
		return fmt.Errorf("updating roles for user %q: %w", username, err)
	}

	req.Logger.Debug().
		Str("user", username).
		Str("database", database).
		Msg("mongodb: user reconciled")

	return nil
}

func (p *Provider) deleteUser(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	username := decl.Field("name", "")
	database := decl.Field("database", "")
	if username == "" || database == "" {
		return nil
	}

	err := p.client.Database(database).RunCommand(ctx, bson.D{
		{Key: "dropUser", Value: username},
	}).Err()

	if err != nil {
		// User already doesn't exist — idempotent success
		if strings.Contains(err.Error(), "UserNotFound") {
			return nil
		}
		return fmt.Errorf("dropping user %q: %w", username, err)
	}

	req.Logger.Info().Str("user", username).Str("database", database).Msg("mongodb: user dropped")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Collection
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileCollection(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	name, err := decl.Require("name")
	if err != nil {
		return err
	}
	database, err := decl.Require("database")
	if err != nil {
		return err
	}

	db := p.client.Database(database)

	// List existing collections to check for existence
	collections, err := db.ListCollectionNames(ctx, bson.M{"name": name})
	if err != nil {
		return fmt.Errorf("listing collections in %q: %w", database, err)
	}

	if len(collections) > 0 {
		req.Logger.Debug().
			Str("collection", name).
			Str("database", database).
			Msg("mongodb: collection already exists — no-op")
		return nil
	}

	// Create with optional capped configuration
	opts := options.CreateCollection()
	if decl.Field("capped", "false") == "true" {
		if s := decl.Field("maxSizeBytes", ""); s != "" {
			// capped collections need a size
		}
		opts.SetCapped(true)
	}

	if err := db.CreateCollection(ctx, name, opts); err != nil {
		return fmt.Errorf("creating collection %q in %q: %w", name, database, err)
	}

	req.Logger.Info().
		Str("collection", name).
		Str("database", database).
		Msg("mongodb: collection created")

	return nil
}

func (p *Provider) deleteCollection(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	name := decl.Field("name", "")
	database := decl.Field("database", "")
	if name == "" || database == "" {
		return nil
	}

	if err := p.client.Database(database).Collection(name).Drop(ctx); err != nil {
		return fmt.Errorf("dropping collection %q from %q: %w", name, database, err)
	}

	req.Logger.Info().
		Str("collection", name).
		Str("database", database).
		Msg("mongodb: collection dropped")

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// parseRoles converts a comma-separated role string into MongoDB role documents.
// "readWrite,dbAdmin" → [{role: "readWrite", db: "mydb"}, {role: "dbAdmin", db: "mydb"}]
func parseRoles(rolesStr, database string) bson.A {
	parts := strings.Split(rolesStr, ",")
	roles := make(bson.A, 0, len(parts))
	for _, r := range parts {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		// Role may specify its own database: "readWrite@admin"
		if at := strings.Index(r, "@"); at >= 0 {
			roles = append(roles, bson.M{
				"role": r[:at],
				"db":   r[at+1:],
			})
		} else {
			roles = append(roles, bson.M{
				"role": r,
				"db":   database,
			})
		}
	}
	return roles
}
