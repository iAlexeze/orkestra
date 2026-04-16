package internal

import (
	"context"

	"github.com/orkspace/orkestra/pkg/katalog"
	"github.com/orkspace/orkestra/pkg/logger"
	awsprovider "github.com/orkspace/orkestra/pkg/provider/aws"
	azureprovider "github.com/orkspace/orkestra/pkg/provider/azure"
	googleprovider "github.com/orkspace/orkestra/pkg/provider/google"
	mongoprovider "github.com/orkspace/orkestra/pkg/provider/mongo"
	mysqlprovider "github.com/orkspace/orkestra/pkg/provider/mysql"
	pgsprovider "github.com/orkspace/orkestra/pkg/provider/postgres"
	redisprovider "github.com/orkspace/orkestra/pkg/provider/redis"
	orktypes "github.com/orkspace/orkestra/pkg/types"
)

// loadProviders registers providers declared at the top-level providers[] block.
//
// Rules:
//   - Only providers declared in providers[] are registered. A CRD's
//     operatorBox.providers block for an undeclared provider is skipped with a warning.
//   - Credentials come from the katalog auth block (with $ENV_VAR expansion).
//     If the auth block is absent, the provider's default credential chain is used
//     (env vars, ~/.aws/credentials, instance profile, etc.).
//   - required: true → fatal log when the provider cannot be initialised.
//   - required: false → warning only; the operator continues without that provider.
func loadProviders(ctx context.Context, kat *katalog.Katalog) orktypes.ProviderRegistry {
	registry := orktypes.NewProviderRegistry()

	for _, req := range kat.Providers {
		auth := req.ResolvedAuth()
		switch req.Name {

		// ── AWS ────────────────────────────────────────────────────────────────
		case "aws":
			var (
				p   *awsprovider.Provider
				err error
			)
			if len(auth) > 0 {
				p, err = awsprovider.NewFromAuth(ctx, auth)
			} else {
				p, err = awsprovider.NewFromContext(ctx)
			}
			if err != nil {
				if req.Required {
					logger.Fatal().Err(err).
						Str("provider", "aws").
						Msg("required AWS provider failed to initialise — cannot start")
				}
				logger.Warn().Err(err).
					Msg("AWS provider not registered — aws: blocks in Katalog will be skipped")
				continue
			}
			registry.Register(p)
			logger.Info().Str("provider", "aws").Msg("AWS provider registered")

		// ── Google Cloud ───────────────────────────────────────────────────────
		case "google":
			p, err := googleprovider.NewFromAuth(ctx, auth)
			if err != nil {
				if req.Required {
					logger.Fatal().Err(err).
						Str("provider", "google").
						Msg("required Google Cloud provider failed to initialise — cannot start")
				}
				logger.Warn().Err(err).
					Msg("Google Cloud provider not registered — google: blocks in Katalog will be skipped")
				continue
			}
			registry.Register(p)
			logger.Info().Str("provider", "google").Msg("Google Cloud provider registered")

		// ── Azure ──────────────────────────────────────────────────────────────
		case "azure":
			p, err := azureprovider.NewFromAuth(ctx, auth)
			if err != nil {
				if req.Required {
					logger.Fatal().Err(err).
						Str("provider", "azure").
						Msg("required Azure provider failed to initialise — cannot start")
				}
				logger.Warn().Err(err).
					Msg("Azure provider not registered — azure: blocks in Katalog will be skipped")
				continue
			}
			registry.Register(p)
			logger.Info().Str("provider", "azure").Msg("Azure provider registered")

			// ── MongoDB ────────────────────────────────────────────────────────────
		case "mongodb":
			mongoURI := auth["mongoUri"]
			if mongoURI == "" {
				mongoURI = auth["uri"]
			}
			if mongoURI == "" {
				if req.Required {
					logger.Fatal().
						Str("provider", "mongodb").
						Msg("required MongoDB provider has no mongoUri in auth — cannot start")
				}
				logger.Warn().
					Msg("MongoDB provider not registered — set auth.mongoUri or $MONGODB_URL")
				continue
			}
			p, err := mongoprovider.NewFromURI(ctx, mongoURI)
			if err != nil {
				if req.Required {
					logger.Fatal().Err(err).
						Str("provider", "mongodb").
						Msg("required MongoDB provider failed to initialise — cannot start")
				}
				logger.Warn().Err(err).
					Msg("MongoDB provider not registered — mongodb: blocks in Katalog will be skipped")
				continue
			}
			registry.Register(p)
			logger.Info().Str("provider", "mongodb").Msg("MongoDB provider registered")

		// ── PostgreSQL ─────────────────────────────────────────────────────────
		case "postgres":
			p, err := pgsprovider.NewFromAuth(ctx, auth)
			if err != nil {
				if req.Required {
					logger.Fatal().Err(err).
						Str("provider", "postgres").
						Msg("required PostgreSQL provider failed to initialise — cannot start")
				}
				logger.Warn().Err(err).
					Msg("PostgreSQL provider not registered — postgres: blocks in Katalog will be skipped")
				continue
			}
			registry.Register(p)
			logger.Info().Str("provider", "postgres").Msg("PostgreSQL provider registered")

		// ── Redis (cache) ──────────────────────────────────────────────────────
		case "cache":
			p, err := redisprovider.NewFromAuth(ctx, auth)
			if err != nil {
				if req.Required {
					logger.Fatal().Err(err).
						Str("provider", "cache").
						Msg("required Redis cache provider failed to initialise — cannot start")
				}
				logger.Warn().Err(err).
					Msg("Redis cache provider not registered — cache: blocks in Katalog will be skipped")
				continue
			}
			registry.Register(p)
			logger.Info().Str("provider", "cache").Msg("Redis cache provider registered")

		// ── MySQL ──────────────────────────────────────────────────────────────
		case "mysql":
			p, err := mysqlprovider.NewFromAuth(ctx, auth)
			if err != nil {
				if req.Required {
					logger.Fatal().Err(err).
						Str("provider", "mysql").
						Msg("required MySQL provider failed to initialise — cannot start")
				}
				logger.Warn().Err(err).
					Msg("MySQL provider not registered — mysql: blocks in Katalog will be skipped")
				continue
			}
			registry.Register(p)
			logger.Info().Str("provider", "mysql").Msg("MySQL provider registered")

		default:
			logger.Warn().
				Str("provider", req.Name).
				Msg("unknown provider name in providers — no built-in handler; skipping")
		}
	}

	return registry
}
