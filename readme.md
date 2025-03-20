# Go-Cache

Go-Cache is a hybrid caching library that uses both in-memory and Redis for caching. It provides a simple interface to set, get, and delete cache entries.

## Features

- In-memory caching using Ristretto
- Redis caching
- Set and get cache entries
- Delete cache entries
- Delete multiple keys by pattern
- Generate cache keys using SHA256

## Installation

To install Go-Cache, use `go get`:

```sh
go get github.com/xplore-run/go-cache
```

## Usage

### Initialize HybridCache

```go
import (
    "github.com/yourusername/go-cache"
    "github.com/redis/go-redis/v9"
    "time"
)

func main() {
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    cache := cache.NewHybridCache(cache.HybridCacheOption{
        ExpiresInMemory: 5 * time.Minute,
        ExpiresRedis:    10 * time.Minute,
        Prefix:          "myapp",
        Redis:           redisClient,
        MaxCost:         1000,
    })
}
```

### Set and Get Cache Entries

```go
cache.Set("key", []byte("value"))

value, found := cache.Get("key")
if found {
    fmt.Println("Cache hit:", string(value))
} else {
    fmt.Println("Cache miss")
}
```

### Delete Cache Entries

```go
cache.Del("key")
```

### Delete Multiple Keys by Pattern

```go
err := cache.DeleteKeysByPatternFromRedis(context.Background(), "myapp:*")
if err != nil {
    log.Fatalf("Error deleting keys: %v", err)
}
```

### Generate Cache Key

```go
cacheKey := cache.GetCacheKey("some data")
fmt.Println("Generated cache key:", cacheKey)
```

### Close Cache

```go
cache.Close()
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
```

Make sure to replace `github.com/xplore-run/go-cache` with the actual import path of your project.