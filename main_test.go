package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestCache() *HybridCache {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	cache, _ := ristretto.NewCache(&ristretto.Config{
		MaxCost:     1000,
		NumCounters: 10000,
		BufferItems: 64,
	})

	return &HybridCache{
		InMemoryCache:   cache,
		Redis:           redisClient,
		Prefix:          "test",
		ExpiresInMemory: 5 * time.Minute,
		ExpiresRedis:    10 * time.Minute,
	}
}

func TestSetAndGet(t *testing.T) {
	cache := setupTestCache()
	defer cache.Close()

	key := "testKey"
	value := []byte("testValue")

	cache.Set(key, value)

	// Test in-memory cache
	inMemValue, found := cache.GetFromInMemoryCache(key)
	assert.True(t, found)
	assert.Equal(t, value, inMemValue)

	// Test Redis cache
	redisValue, found := cache.GetFromRedis(context.Background(), key)
	assert.True(t, found)
	assert.Equal(t, value, redisValue)

	// Test hybrid cache
	cachedValue, found := cache.Get(key)
	assert.True(t, found)
	assert.Equal(t, value, cachedValue)
}

func TestDelete(t *testing.T) {
	cache := setupTestCache()
	defer cache.Close()

	key := "testKey"
	value := []byte("testValue")

	cache.Set(key, value)
	cache.Del(key)

	// Test in-memory cache
	_, found := cache.GetFromInMemoryCache(key)
	assert.False(t, found)

	// Test Redis cache
	_, found = cache.GetFromRedis(context.Background(), key)
	assert.False(t, found)
}

func TestDeleteKeysByPatternFromRedis(t *testing.T) {
	cache := setupTestCache()
	defer cache.Close()

	key1 := "testKey1"
	key2 := "testKey2"
	value := []byte("testValue")

	cache.Set(key1, value)
	cache.Set(key2, value)

	err := cache.DeleteKeysByPatternFromRedis(context.Background(), "test*")
	assert.NoError(t, err)

	// Test Redis cache
	_, found := cache.GetFromRedis(context.Background(), key1)
	assert.False(t, found)

	_, found = cache.GetFromRedis(context.Background(), key2)
	assert.False(t, found)
}

func TestGetCacheKey(t *testing.T) {
	cache := setupTestCache()
	defer cache.Close()

	data := "some data"
	expectedPrefix := "test:"

	cacheKey := cache.GetCacheKey(data)
	assert.True(t, len(cacheKey) > len(expectedPrefix))
	assert.Equal(t, expectedPrefix, cacheKey[:len(expectedPrefix)])
}
