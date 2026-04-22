// pkg/provider/postgres/provider.go
//
// PostgreSQL provider for Orkestra.
//
// Handles the "postgres" block in Katalog declarations.
// Uses pgx/v5 — the standard PostgreSQL driver for Go.
//
// Supported resource kinds:
//
//	database   — PostgreSQL database (create, drop)
//	role       — Database role/user (create, update password and privileges, drop)
//	extension  — PostgreSQL extension (CREATE EXTENSION IF NOT EXISTS, drop)
//
// Installation:
//
//	go get github.com/jackc/pgx/v5
//
// Registration:
//
//	p, err := pgsprovider.NewFromAuth(ctx, auth)
//	registry.Register(p)
//
// Auth keys (providers[].auth block):
//
//	host     — server hostname or IP (default: localhost)
//	port     — server port (default: 5432)
//	user     — superuser name (must be able to CREATE DATABASE/ROLE)
//	password — superuser password (or set PGPASSWORD env var)
//	sslmode  — disable | require | verify-ca | verify-full (default: require)
//
// Katalog:
//
//	providers:
//	  - name: postgres
//	    required: true
//	    auth:
//	      host: "$PG_HOST"
//	      user: "$PG_USER"
//	      password: "$PG_PASSWORD"
//
//	operatorBox:
//	  providers:
//	    postgres:
//	      - database:
//	          name: "{{ .metadata.name }}"
//	          owner: "{{ .spec.dbUser }}"
//
//	      - role:
//	          name: "{{ .spec.dbUser }}"
//	          password: "{{ .spec.dbPassword }}"
//	          privileges: "LOGIN NOSUPERUSER"
//	          credentials:
//	            secretName: "{{ .metadata.name }}-pg-creds"   # PG_PASSWORD key
//
//	      - extension:
//	          name: pgcrypto
//	          database: "{{ .metadata.name }}"
package pgsprovider

import (
	"context"
	"fmt"
	"strings"

	pgx "github.com/jackc/pgx/v5"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

// Provider implements orktypes.Provider for the "postgres" block.
type Provider struct {
	connStr string // base DSN used to open admin connections
}

// New creates a PostgreSQL provider from a DSN string.
// DSN format: "postgres://user:password@host:port/postgres?sslmode=require"
func New(dsn string) *Provider {
	return &Provider{connStr: dsn}
}

// NewFromAuth creates a PostgreSQL provider from a Katalog auth map.
// Keys: host, port, user, password, sslmode.
func NewFromAuth(_ context.Context, auth map[string]string) (*Provider, error) {
	host := auth["host"]
	if host == "" {
		host = "localhost"
	}
	port := auth["port"]
	if port == "" {
		port = "5432"
	}
	user := auth["user"]
	if user == "" {
		return nil, fmt.Errorf("postgres: auth.user is required")
	}
	password := auth["password"]
	sslmode := auth["sslmode"]
	if sslmode == "" {
		sslmode = "require"
	}
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=%s",
		user, password, host, port, sslmode)
	return New(dsn), nil
}

func (p *Provider) Name() string { return "postgres" }

// connect opens a short-lived admin connection.
// Callers must defer conn.Close() immediately.
func (p *Provider) connect(ctx context.Context) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, p.connStr)
	if err != nil {
		return nil, fmt.Errorf("postgres: connecting: %w", err)
	}
	return conn, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Reconcile
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) Reconcile(ctx context.Context, req orktypes.ReconcileRequest) error {
	for _, decl := range req.Declarations {
		var err error
		switch decl.Kind {
		case "database":
			err = p.reconcileDatabase(ctx, req, decl)
		case "role":
			err = p.reconcileRole(ctx, req, decl)
		case "extension":
			err = p.reconcileExtension(ctx, req, decl)
		default:
			req.Logger.Warn().
				Str("kind", decl.Kind).
				Msg("postgres: unknown resource kind — skipped")
			continue
		}
		if err != nil {
			return fmt.Errorf("postgres.%s: %w", decl.Kind, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) Delete(ctx context.Context, req orktypes.DeleteRequest) error {
	// Delete in reverse order: extensions → roles → database
	for i := len(req.Declarations) - 1; i >= 0; i-- {
		decl := req.Declarations[i]
		var err error
		switch decl.Kind {
		case "extension":
			err = p.deleteExtension(ctx, req, decl)
		case "role":
			err = p.deleteRole(ctx, req, decl)
		case "database":
			err = p.deleteDatabase(ctx, req, decl)
		}
		if err != nil {
			return fmt.Errorf("postgres.%s delete: %w", decl.Kind, err)
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
	conn, err := p.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	var exists bool
	err = conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", name,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("checking database %q: %w", name, err)
	}

	if exists {
		req.Logger.Debug().Str("database", name).Msg("postgres: database already exists — no-op")
		// Optionally update owner if specified
		if owner := decl.Field("owner", ""); owner != "" {
			_, err = conn.Exec(ctx,
				fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", pgQuoteIdent(name), pgQuoteIdent(owner)))
			if err != nil {
				return fmt.Errorf("updating owner of database %q: %w", name, err)
			}
		}
		return nil
	}

	createSQL := fmt.Sprintf("CREATE DATABASE %s", pgQuoteIdent(name))
	if owner := decl.Field("owner", ""); owner != "" {
		createSQL += fmt.Sprintf(" OWNER %s", pgQuoteIdent(owner))
	}
	if _, err = conn.Exec(ctx, createSQL); err != nil {
		return fmt.Errorf("creating database %q: %w", name, err)
	}
	req.Logger.Info().Str("database", name).Msg("postgres: database created")
	return nil
}

func (p *Provider) deleteDatabase(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	name := decl.Field("name", "")
	if name == "" {
		return nil
	}
	conn, err := p.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	// Terminate existing connections before dropping
	_, _ = conn.Exec(ctx,
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()",
		name)

	if _, err := conn.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", pgQuoteIdent(name))); err != nil {
		return fmt.Errorf("dropping database %q: %w", name, err)
	}
	req.Logger.Info().Str("database", name).Msg("postgres: database dropped")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Role
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileRole(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	name, err := decl.Require("name")
	if err != nil {
		return err
	}

	// Resolve password — from Secret if specified, else from declaration
	password := decl.Field("password", "")
	if secretName := decl.Field("credentials.secretName", ""); secretName != "" {
		data, err := req.Kube.GetSecret(ctx, req.OwnerNamespace, secretName)
		if err != nil {
			return fmt.Errorf("reading credentials secret %q: %w", secretName, err)
		}
		if p := string(data["PG_PASSWORD"]); p != "" {
			password = p
		}
	}

	conn, err := p.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	var exists bool
	if err := conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname = $1)", name,
	).Scan(&exists); err != nil {
		return fmt.Errorf("checking role %q: %w", name, err)
	}

	// Parse options (LOGIN, NOSUPERUSER, etc.)
	opts := decl.Field("privileges", "LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE")

	if !exists {
		sql := fmt.Sprintf("CREATE ROLE %s %s", pgQuoteIdent(name), opts)
		if password != "" {
			sql += fmt.Sprintf(" PASSWORD '%s'", strings.ReplaceAll(password, "'", "''"))
		}
		if _, err := conn.Exec(ctx, sql); err != nil {
			return fmt.Errorf("creating role %q: %w", name, err)
		}
		req.Logger.Info().Str("role", name).Msg("postgres: role created")
		return nil
	}

	// Update password if provided
	if password != "" {
		if _, err := conn.Exec(ctx,
			fmt.Sprintf("ALTER ROLE %s PASSWORD '%s'",
				pgQuoteIdent(name), strings.ReplaceAll(password, "'", "''"))); err != nil {
			return fmt.Errorf("updating password for role %q: %w", name, err)
		}
	}
	req.Logger.Debug().Str("role", name).Msg("postgres: role reconciled")
	return nil
}

func (p *Provider) deleteRole(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	name := decl.Field("name", "")
	if name == "" {
		return nil
	}
	conn, err := p.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, fmt.Sprintf("DROP ROLE IF EXISTS %s", pgQuoteIdent(name))); err != nil {
		return fmt.Errorf("dropping role %q: %w", name, err)
	}
	req.Logger.Info().Str("role", name).Msg("postgres: role dropped")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Extension
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileExtension(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	name, err := decl.Require("name")
	if err != nil {
		return err
	}
	database, err := decl.Require("database")
	if err != nil {
		return err
	}

	// Connect to the target database (not the admin "postgres" db)
	dsn := strings.Replace(p.connStr, "/postgres?", "/"+database+"?", 1)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connecting to database %q: %w", database, err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx,
		fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", pgQuoteIdent(name))); err != nil {
		return fmt.Errorf("creating extension %q in %q: %w", name, database, err)
	}
	req.Logger.Info().
		Str("extension", name).
		Str("database", database).
		Msg("postgres: extension created (or already exists)")
	return nil
}

func (p *Provider) deleteExtension(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	name := decl.Field("name", "")
	database := decl.Field("database", "")
	if name == "" || database == "" {
		return nil
	}

	dsn := strings.Replace(p.connStr, "/postgres?", "/"+database+"?", 1)
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connecting to database %q: %w", database, err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx,
		fmt.Sprintf("DROP EXTENSION IF EXISTS %s", pgQuoteIdent(name))); err != nil {
		return fmt.Errorf("dropping extension %q from %q: %w", name, database, err)
	}
	req.Logger.Info().Str("extension", name).Str("database", database).Msg("postgres: extension dropped")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// pgQuoteIdent wraps an identifier in double quotes for safe SQL injection prevention.
// Double quotes inside the identifier are doubled per SQL standard.
func pgQuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
