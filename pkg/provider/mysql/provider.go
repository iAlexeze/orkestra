// pkg/provider/mysql/provider.go
//
// MySQL provider for Orkestra.
//
// Handles the "mysql" block in Katalog declarations.
// Uses database/sql with the go-sql-driver/mysql driver.
//
// Supported resource kinds:
//
//	database — MySQL database / schema (create, drop)
//	user     — MySQL user (create, update password and privileges, drop)
//
// Installation:
//
//	go get github.com/go-sql-driver/mysql
//
// Registration:
//
//	p, err := mysqlprovider.NewFromAuth(ctx, auth)
//	registry.Register(p)
//
// Auth keys (providers[].auth block):
//
//	host     — server hostname or IP (default: 127.0.0.1)
//	port     — server port (default: 3306)
//	user     — admin user (must have GRANT OPTION)
//	password — admin password
//	tls      — false | true | skip-verify | preferred (default: false)
//
// Katalog:
//
//	providers:
//	  - name: mysql
//	    required: true
//	    auth:
//	      host: "$MYSQL_HOST"
//	      user: "$MYSQL_USER"
//	      password: "$MYSQL_PASSWORD"
//
//	operatorBox:
//	  providers:
//	    mysql:
//	      - database:
//	          name: "{{ .metadata.name }}"
//	          charset: utf8mb4
//	          collation: utf8mb4_unicode_ci
//
//	      - user:
//	          name: "{{ .spec.dbUser }}"
//	          host: "%"
//	          database: "{{ .metadata.name }}"
//	          privileges: "SELECT, INSERT, UPDATE, DELETE"
//	          credentials:
//	            secretName: "{{ .metadata.name }}-mysql-creds"   # MYSQL_PASSWORD key
package mysqlprovider

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql" // register mysql driver
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// ─────────────────────────────────────────────────────────────────────────────
// Provider
// ─────────────────────────────────────────────────────────────────────────────

// Provider implements orktypes.Provider for the "mysql" block.
type Provider struct {
	dsn string // DSN used to open admin connections
}

// New creates a MySQL provider from a DSN string.
// DSN format: "user:password@tcp(host:port)/?tls=false&parseTime=true"
func New(dsn string) *Provider {
	return &Provider{dsn: dsn}
}

// NewFromAuth creates a MySQL provider from a Katalog auth map.
// Keys: host, port, user, password, tls.
func NewFromAuth(ctx context.Context, auth map[string]string) (*Provider, error) {
	host := auth["host"]
	if host == "" {
		host = "127.0.0.1"
	}
	port := auth["port"]
	if port == "" {
		port = "3306"
	}
	user := auth["user"]
	if user == "" {
		return nil, fmt.Errorf("mysql: auth.user is required")
	}
	password := auth["password"]
	tlsMode := auth["tls"]
	if tlsMode == "" {
		tlsMode = "false"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?tls=%s&parseTime=true",
		user, password, host, port, tlsMode)

	p := New(dsn)
	// Verify connectivity at construction time
	db, err := p.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("mysql: ping failed: %w", err)
	}
	return p, nil
}

func (p *Provider) Name() string { return "mysql" }

// open returns a new *sql.DB backed by the provider DSN.
// Callers must defer db.Close().
func (p *Provider) open() (*sql.DB, error) {
	db, err := sql.Open("mysql", p.dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql: opening connection: %w", err)
	}
	return db, nil
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
		case "user":
			err = p.reconcileUser(ctx, req, decl)
		default:
			req.Logger.Warn().
				Str("kind", decl.Kind).
				Msg("mysql: unknown resource kind — skipped")
			continue
		}
		if err != nil {
			return fmt.Errorf("mysql.%s: %w", decl.Kind, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Delete
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) Delete(ctx context.Context, req orktypes.DeleteRequest) error {
	// Reverse: users → database
	for i := len(req.Declarations) - 1; i >= 0; i-- {
		decl := req.Declarations[i]
		var err error
		switch decl.Kind {
		case "user":
			err = p.deleteUser(ctx, req, decl)
		case "database":
			err = p.deleteDatabase(ctx, req, decl)
		}
		if err != nil {
			return fmt.Errorf("mysql.%s delete: %w", decl.Kind, err)
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

	charset := decl.Field("charset", "utf8mb4")
	collation := decl.Field("collation", "utf8mb4_unicode_ci")

	db, err := p.open()
	if err != nil {
		return err
	}
	defer db.Close()

	// CREATE DATABASE IF NOT EXISTS is idempotent
	stmt := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s COLLATE %s",
		mysqlEscapeIdent(name), charset, collation,
	)
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("creating database %q: %w", name, err)
	}

	req.Logger.Info().Str("database", name).Msg("mysql: database created (or already exists)")
	return nil
}

func (p *Provider) deleteDatabase(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	name := decl.Field("name", "")
	if name == "" {
		return nil
	}

	db, err := p.open()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", mysqlEscapeIdent(name))
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("dropping database %q: %w", name, err)
	}

	req.Logger.Info().Str("database", name).Msg("mysql: database dropped")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// User
// ─────────────────────────────────────────────────────────────────────────────

func (p *Provider) reconcileUser(ctx context.Context, req orktypes.ReconcileRequest, decl orktypes.ProviderDeclaration) error {
	name, err := decl.Require("name")
	if err != nil {
		return err
	}
	database, err := decl.Require("database")
	if err != nil {
		return err
	}

	// Host from which the user connects — % means any host
	host := decl.Field("host", "%")

	// Resolve password
	password := decl.Field("password", "")
	if secretName := decl.Field("credentials.secretName", ""); secretName != "" {
		data, err := req.Kube.GetSecret(ctx, req.OwnerNamespace, secretName)
		if err != nil {
			return fmt.Errorf("reading credentials secret %q: %w", secretName, err)
		}
		if pw := string(data["MYSQL_PASSWORD"]); pw != "" {
			password = pw
		}
	}

	privileges := decl.Field("privileges", "SELECT, INSERT, UPDATE, DELETE")

	db, err := p.open()
	if err != nil {
		return err
	}
	defer db.Close()

	// Check if user exists
	var count int
	row := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM mysql.user WHERE User = ? AND Host = ?",
		name, host,
	)
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("checking user %q@%q: %w", name, host, err)
	}

	if count == 0 {
		// CREATE USER
		createStmt := fmt.Sprintf("CREATE USER '%s'@'%s'",
			mysqlEscapeStr(name), mysqlEscapeStr(host))
		if password != "" {
			createStmt += fmt.Sprintf(" IDENTIFIED BY '%s'", mysqlEscapeStr(password))
		}
		if _, err := db.ExecContext(ctx, createStmt); err != nil {
			return fmt.Errorf("creating user %q@%q: %w", name, host, err)
		}
		req.Logger.Info().Str("user", name).Str("host", host).Msg("mysql: user created")
	} else if password != "" {
		// Update password
		alterStmt := fmt.Sprintf("ALTER USER '%s'@'%s' IDENTIFIED BY '%s'",
			mysqlEscapeStr(name), mysqlEscapeStr(host), mysqlEscapeStr(password))
		if _, err := db.ExecContext(ctx, alterStmt); err != nil {
			return fmt.Errorf("updating password for user %q@%q: %w", name, host, err)
		}
	}

	// GRANT — idempotent
	grantStmt := fmt.Sprintf("GRANT %s ON `%s`.* TO '%s'@'%s'",
		privileges, mysqlEscapeIdent(database),
		mysqlEscapeStr(name), mysqlEscapeStr(host))
	if _, err := db.ExecContext(ctx, grantStmt); err != nil {
		return fmt.Errorf("granting privileges to user %q@%q on %q: %w", name, host, database, err)
	}

	if _, err := db.ExecContext(ctx, "FLUSH PRIVILEGES"); err != nil {
		req.Logger.Warn().Err(err).Msg("mysql: FLUSH PRIVILEGES failed — continuing")
	}

	req.Logger.Debug().Str("user", name).Str("database", database).Msg("mysql: user reconciled")
	return nil
}

func (p *Provider) deleteUser(ctx context.Context, req orktypes.DeleteRequest, decl orktypes.ProviderDeclaration) error {
	name := decl.Field("name", "")
	host := decl.Field("host", "%")
	if name == "" {
		return nil
	}

	db, err := p.open()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt := fmt.Sprintf("DROP USER IF EXISTS '%s'@'%s'",
		mysqlEscapeStr(name), mysqlEscapeStr(host))
	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("dropping user %q@%q: %w", name, host, err)
	}

	req.Logger.Info().Str("user", name).Str("host", host).Msg("mysql: user dropped")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// mysqlEscapeIdent escapes a MySQL identifier (backtick-quoted).
// Backticks inside the name are doubled.
func mysqlEscapeIdent(name string) string {
	return strings.ReplaceAll(name, "`", "``")
}

// mysqlEscapeStr escapes a MySQL string literal (single-quoted).
// Single quotes inside the string are doubled.
func mysqlEscapeStr(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
