//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a/internal/shared/database"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	testServiceVersion = "1.0.0-test"
	testEnvironment    = "test"
)

func setupPostgres(t *testing.T) *database.Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	container, err := postgres.Run(ctx,
		"postgres:18.7-alpine3.24",
		postgres.WithDatabase("api_template_test"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)
	require.NoError(t, err, "start test PostgreSQL container")

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "read test connection string")

	db, err := database.New(dsn, true)
	require.NoError(t, err, "connect to test database")
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	require.NoError(t, db.SetConnectionPool(1, 5, 30*time.Minute))
	return db
}
