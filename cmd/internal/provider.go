package internal

import (
	"context"
	"os"

	"github.com/ialexeze/orkestra/pkg/logger"
	awsprovider "github.com/ialexeze/orkestra/pkg/provider/aws"
	// mongoprovider "github.com/ialexeze/orkestra/pkg/provider/mongo"
	orktypes "github.com/ialexeze/orkestra/pkg/types"
)

// loadProviders registers all external providers and returns the registry.
// Returns a NoOpProviderRegistry on any failure — the operator still starts,
// but provider blocks in the Katalog will be skipped with a warning.
//
// Call in konstructOrkestra before building the DependencyKontroller,
// then pass the registry to NewDependencyKontroller.
func loadProviders(ctx context.Context) orktypes.ProviderRegistry {
	registry := orktypes.NewProviderRegistry()

	// ── AWS ───────────────────────────────────────────────────────────────────
	// NewFromContext loads credentials from the standard chain:
	//   1. AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY env vars
	//   2. ~/.aws/credentials
	//   3. EC2 instance profile / ECS task role
	aws, err := awsprovider.NewFromContext(ctx)
	if err != nil {
		logger.Warn().Err(err).
			Msg("AWS provider not registered — aws: blocks in Katalog will be skipped. " +
				"Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY to enable.")
	} else {
		registry.Register(aws)
		logger.Info().
			Str("provider", "aws").
			Msg("AWS provider registered")
	}

	// ── MongoDB ───────────────────────────────────────────────────────────────
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	// mongo, err := mongoprovider.NewFromURI(ctx, mongoURI)
	// if err != nil {
	// 	logger.Warn().Err(err).
	// 		Str("uri", mongoURI).
	// 		Msg("MongoDB provider not registered — mongodb: blocks in Katalog will be skipped. " +
	// 			"Set MONGO_URI to enable.")
	// } else {
	// 	registry.Register(mongo)
	// 	logger.Info().
	// 		Str("provider", "mongodb").
	// 		Str("uri", mongoURI).
	// 		Msg("MongoDB provider registered")
	// }

	return registry
}
