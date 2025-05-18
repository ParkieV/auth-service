package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ParkieV/auth-service/internal/infrastructure/cache"
)

func TestSetGetDelete(t *testing.T) {
	rdb := cache.NewRedisCache(RedisConfig)

	// Set
	require.NoError(t, rdb.Set("foo", "bar", 5*time.Second))

	// Get
	val, err := rdb.Get("foo")
	require.NoError(t, err)
	require.Equal(t, "bar", val)

	// Delete
	require.NoError(t, rdb.Delete("foo"))
	_, err = rdb.Get("foo")
	require.Error(t, err)
}
