package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"log"
	"time"

	inMemCLib "github.com/dgraph-io/ristretto"
	"github.com/redis/go-redis/v9"
)

// HybridCache is a cache that will use multiple cache sources
type HybridCache struct {
	Prefix          string
	InMemoryCache   *inMemCLib.Cache
	Redis           *redis.Client
	ExpiresInMemory time.Duration
	ExpiresRedis    time.Duration
}

type HybridCacheOption struct {
	ExpiresInMemory time.Duration
	ExpiresRedis    time.Duration
	Prefix          string
	Redis           *redis.Client
	MaxCost         int64
}

// Initialize a new HybridCache
func NewHybridCache(option HybridCacheOption) *HybridCache {
	// inMemCLib
	maxCost := option.MaxCost
	if maxCost == 0 {
		maxCost = 100
	}
	cache, err := inMemCLib.NewCache(&inMemCLib.Config{
		MaxCost:     maxCost,
		NumCounters: maxCost * 10,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatalf("Error creating ristretto cache: %v", err)
	}
	return &HybridCache{
		InMemoryCache:   cache,
		Redis:           option.Redis,
		Prefix:          option.Prefix,
		ExpiresInMemory: option.ExpiresInMemory,
		ExpiresRedis:    option.ExpiresRedis}
}

// Set value in cache
func (hc *HybridCache) Set(key string, value []byte) {
	hc.SetInMemoryCache(key, value)
	hc.SetInRedis(context.Background(), key, value, hc.ExpiresRedis)
}

// SetWithTTL set value in cache with expiration
func (hc *HybridCache) SetWithTTL(key string, value []byte, expiration time.Duration) {
	hc.SetInMemoryCache(key, value)
	hc.SetInRedis(context.Background(), key, value, expiration)
}

// Set value in in-memory cache
func (hc *HybridCache) SetInMemoryCache(key string, value []byte) {
	hc.InMemoryCache.SetWithTTL(key, value, 1, hc.ExpiresInMemory) // set value with cost 1
	hc.InMemoryCache.Wait()                                        // wait for value to pass through buffers
}

// SetInRedis set value in redis
func (rc *HybridCache) SetInRedis(ctx context.Context, key string, value []byte, expiration time.Duration) {
	rc.Redis.Set(ctx, key, value, expiration)
}

// get value from cache
func (hc *HybridCache) Get(key string) ([]byte, bool) {
	value, found := hc.GetFromInMemoryCache(key)
	if !found {
		value, found = hc.GetFromRedis(context.Background(), key)
		if found {
			hc.SetInMemoryCache(key, value)
		}
	}
	return value, found
}

// get value from in memory cache
func (hc *HybridCache) GetFromInMemoryCache(key string) ([]byte, bool) {
	value, found := hc.InMemoryCache.Get(key)
	if !found {
		return nil, false
	}
	return value.([]byte), true
}

// get value from redis
func (rc *HybridCache) GetFromRedis(ctx context.Context, key string) ([]byte, bool) {
	data, err := rc.Redis.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}
	return []byte(data), true
}

// del value from cache
func (hc *HybridCache) Del(key string) {
	hc.DelFromInMemoryCache(key)
	hc.DelFromRedis(context.Background(), key)
}

// del value from im memory cache
func (hc *HybridCache) DelFromInMemoryCache(key string) {
	hc.InMemoryCache.Del(key)
}

// del value from redis
func (hc *HybridCache) DelFromRedis(ctx context.Context, key string) {
	hc.Redis.Del(ctx, key)
}

// del multiple value from cache
func (hc *HybridCache) DelMultipleKeysFromRedis(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	pipe := hc.Redis.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// DeleteKeysByPatternFromRedis deletes all keys matching the provided pattern from Redis.
func (hc *HybridCache) DeleteKeysByPatternFromRedis(ctx context.Context, pattern string) error {
	keys := []string{}
	cursor := uint64(0)

	for {
		result := hc.Redis.Scan(ctx, cursor, pattern, 100)
		scannedKeys, nextCursor, err := result.Result()
		if err != nil {
			return err
		}

		// Accumulate the keys found
		keys = append(keys, scannedKeys...)

		// Update the cursor for the next iteration
		cursor = nextCursor

		// If cursor is 0, we are done scanning
		if cursor == 0 {
			break
		}
	}

	// If no keys found, return nil
	if len(keys) == 0 {
		return nil
	}

	// Delete keys found by pattern using a pipeline
	pipe := hc.Redis.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

// GetCacheKey generates a cache key for the provided data
func (hc *HybridCache) GetCacheKey(data string) string {
	// Create a new SHA256 hash
	var hasher hash.Hash = sha256.New()
	hasher.Write([]byte(data))
	// Convert the hash to a hexadecimal string
	return fmt.Sprintf("%s:%s", hc.Prefix, hex.EncodeToString(hasher.Sum(nil)))
}

// Close the cache
func (hc *HybridCache) Close() {
	hc.InMemoryCache.Close()
}
