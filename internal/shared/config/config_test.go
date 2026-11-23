package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadJWTSecret(t *testing.T) {
	t.Run("development allows a local secret", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "development")
		t.Setenv("JWT_SECRET", "local-test-key")

		cfg, err := Load()
		require.NoError(t, err)
		require.Equal(t, "local-test-key", cfg.JWTSecret)
	})

	t.Run("production rejects the development secret", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "production")
		t.Setenv("JWT_SECRET", "development-jwt-secret")

		_, err := Load()
		require.ErrorContains(t, err, "JWT_SECRET")
	})

	t.Run("production accepts a configured key", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "production")
		t.Setenv("JWT_SECRET", strings.Repeat("x", 32))

		_, err := Load()
		require.NoError(t, err)
	})
}
