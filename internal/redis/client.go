// Package redis provides Redis client operations
package redis

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/go-redis/redis/v8"
)

// DatabaseStats contains statistics for a single database
type DatabaseStats struct {
	DB         int
	Keys       int64
	AvgTTL     string
	SampleSize int
}

// ServerStats contains overall Redis server statistics
type ServerStats struct {
	Version                string
	UptimeSeconds          int64
	UsedMemory             string
	UsedMemoryPeak         string
	MemFragmentationRatio  float64
	ConnectedClients       int64
	TotalCommandsProcessed int64
	OpsPerSec              int64
	EvictedKeys            int64
	ExpiredKeys            int64
}

// GetServerStats retrieves server-level statistics from Redis INFO command
func GetServerStats(rdb redis.UniversalClient) (*ServerStats, error) {
	ctx := context.TODO()

	// Get all INFO sections to ensure we get memory stats
	info, err := rdb.Info(ctx).Result()
	if err != nil {
		return nil, err
	}

	stats := &ServerStats{}

	// Parse INFO output
	lines := parseInfo(info)

	// DEBUG: Write parsed info to file for inspection
	if debugFile, err := os.Create("/tmp/redis-viewer-debug.txt"); err == nil {
		fmt.Fprintf(debugFile, "=== Raw INFO Output ===\n%s\n\n", info)
		fmt.Fprintf(debugFile, "=== Parsed Lines ===\n")
		for k, v := range lines {
			fmt.Fprintf(debugFile, "%s: %s\n", k, v)
		}
		debugFile.Close()
	}

	// Server info
	if val, ok := lines["redis_version"]; ok {
		stats.Version = val
	}
	if val, ok := lines["uptime_in_seconds"]; ok {
		stats.UptimeSeconds = parseInt64(val)
	}

	// Memory info
	if val, ok := lines["used_memory_human"]; ok && val != "" {
		stats.UsedMemory = val
	} else if val, ok := lines["used_memory"]; ok {
		// Fallback: format bytes to human readable
		stats.UsedMemory = formatBytes(parseInt64(val))
	}
	if val, ok := lines["used_memory_peak_human"]; ok && val != "" {
		stats.UsedMemoryPeak = val
	} else if val, ok := lines["used_memory_peak"]; ok {
		// Fallback: format bytes to human readable
		stats.UsedMemoryPeak = formatBytes(parseInt64(val))
	}
	if val, ok := lines["mem_fragmentation_ratio"]; ok {
		stats.MemFragmentationRatio = parseFloat64(val)
	}

	// Client info
	if val, ok := lines["connected_clients"]; ok {
		stats.ConnectedClients = parseInt64(val)
	}

	// Stats info
	if val, ok := lines["total_commands_processed"]; ok {
		stats.TotalCommandsProcessed = parseInt64(val)
	}
	if val, ok := lines["instantaneous_ops_per_sec"]; ok {
		stats.OpsPerSec = parseInt64(val)
	}
	if val, ok := lines["evicted_keys"]; ok {
		stats.EvictedKeys = parseInt64(val)
	}
	if val, ok := lines["expired_keys"]; ok {
		stats.ExpiredKeys = parseInt64(val)
	}

	return stats, nil
}

// GetDatabaseStats retrieves statistics for a specific database
func GetDatabaseStats(rdb redis.UniversalClient, db int, sampleSize int) (*DatabaseStats, error) {
	ctx := context.TODO()

	stats := &DatabaseStats{
		DB:         db,
		SampleSize: sampleSize,
	}

	// Get key count for current database
	switch rdb := rdb.(type) {
	case *redis.ClusterClient:
		var totalKeys int64
		err := rdb.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			size, err := client.DBSize(ctx).Result()
			if err != nil {
				return err
			}
			atomic.AddInt64(&totalKeys, size)
			return nil
		})
		if err != nil {
			return nil, err
		}
		stats.Keys = totalKeys
	default:
		size, err := rdb.DBSize(ctx).Result()
		if err != nil {
			return nil, err
		}
		stats.Keys = size
	}

	// Calculate average TTL by sampling keys
	if stats.Keys > 0 && sampleSize > 0 {
		avgTTL, err := calculateAverageTTL(rdb, sampleSize)
		if err == nil {
			stats.AvgTTL = avgTTL
		}
	}

	return stats, nil
}

// calculateAverageTTL samples random keys and calculates their average TTL
func calculateAverageTTL(rdb redis.UniversalClient, sampleSize int) (string, error) {
	ctx := context.TODO()

	var totalTTL int64
	var keysWithTTL int64

	switch rdb := rdb.(type) {
	case *redis.ClusterClient:
		err := rdb.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			for i := 0; i < sampleSize; i++ {
				key, err := client.RandomKey(ctx).Result()
				if err != nil || key == "" {
					continue
				}

				ttl, err := client.TTL(ctx, key).Result()
				if err != nil || ttl <= 0 {
					continue
				}

				atomic.AddInt64(&totalTTL, int64(ttl.Seconds()))
				atomic.AddInt64(&keysWithTTL, 1)
			}
			return nil
		})
		if err != nil {
			return "N/A", err
		}
	default:
		for i := 0; i < sampleSize; i++ {
			key, err := rdb.RandomKey(ctx).Result()
			if err != nil || key == "" {
				continue
			}

			ttl, err := rdb.TTL(ctx, key).Result()
			if err != nil || ttl <= 0 {
				continue
			}

			totalTTL += int64(ttl.Seconds())
			keysWithTTL++
		}
	}

	if keysWithTTL == 0 {
		return "No TTL", nil
	}

	avgSeconds := totalTTL / keysWithTTL
	return formatSeconds(avgSeconds), nil
}
