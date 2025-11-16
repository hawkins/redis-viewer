package redis

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
)

// KeyMessage represents a key returned from scanning
type KeyMessage struct {
	Key string
	Err error
}

// CountKeys counts all keys matching the given pattern
func CountKeys(rdb redis.UniversalClient, match string) (int, error) {
	ctx := context.TODO()

	switch rdb := rdb.(type) {
	case *redis.ClusterClient:
		var count int64

		err := rdb.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			iter := client.Scan(ctx, 0, match, 0).Iterator()
			for iter.Next(ctx) {
				atomic.AddInt64(&count, 1)
			}
			if err := iter.Err(); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return 0, err
		}

		return int(count), nil
	default:
		var count int

		iter := rdb.Scan(ctx, 0, match, 0).Iterator()
		for iter.Next(ctx) {
			count++
		}
		if err := iter.Err(); err != nil {
			return 0, err
		}

		return count, nil
	}
}

// GetKeys scans and retrieves keys asynchronously via a channel
func GetKeys(
	rdb redis.UniversalClient,
	cursor uint64,
	match string,
	count int64,
) <-chan KeyMessage {
	res := make(chan KeyMessage, 1)

	go func() {
		ctx := context.TODO()

		switch rdb := rdb.(type) {
		case *redis.ClusterClient:
			// Collect all keys from all masters
			var allKeys []string
			var keysMutex sync.Mutex

			err := rdb.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
				cursor := uint64(0)
				for {
					keys, nextCursor, err := client.Scan(ctx, cursor, match, 0).Result()
					if err != nil {
						return err
					}
					if len(keys) > 0 {
						keysMutex.Lock()
						allKeys = append(allKeys, keys...)
						keysMutex.Unlock()
					}
					cursor = nextCursor
					if cursor == 0 {
						break
					}
				}
				return nil
			})
			if err != nil {
				res <- KeyMessage{"", err}
			} else {
				for _, key := range allKeys {
					res <- KeyMessage{key, nil}
				}
			}
		default:
			// Scan until no more keys
			var allKeys []string
			cursor := uint64(0)
			for {
				keys, nextCursor, err := rdb.Scan(ctx, cursor, match, count).Result()
				if err != nil {
					res <- KeyMessage{"", err}
					break
				}
				for _, key := range keys {
					allKeys = append(allKeys, key)
				}
				cursor = nextCursor
				if cursor == 0 {
					break
				}
			}
			for _, key := range allKeys {
				res <- KeyMessage{key, nil}
			}
		}

		close(res)
	}()

	return res
}

// DeleteKey deletes a single key
func DeleteKey(rdb redis.UniversalClient, key string) error {
	ctx := context.TODO()
	return rdb.Del(ctx, key).Err()
}

// SetKey sets a key to a value
func SetKey(rdb redis.UniversalClient, key string, value string) error {
	ctx := context.TODO()
	return rdb.Set(ctx, key, value, 0).Err()
}

// SetKeyTTL sets or removes TTL for a key
func SetKeyTTL(rdb redis.UniversalClient, key string, ttlSeconds int64) error {
	ctx := context.TODO()
	if ttlSeconds <= 0 {
		// Remove TTL (make key persistent)
		return rdb.Persist(ctx, key).Err()
	}
	// Set TTL
	return rdb.Expire(ctx, key, time.Duration(ttlSeconds)*time.Second).Err()
}

// FlushDB flushes the current database
func FlushDB(rdb redis.UniversalClient) error {
	ctx := context.TODO()

	switch rdb := rdb.(type) {
	case *redis.ClusterClient:
		// For cluster mode, flush each master node
		return rdb.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			return client.FlushDB(ctx).Err()
		})
	default:
		// For standalone and sentinel modes
		return rdb.FlushDB(ctx).Err()
	}
}
