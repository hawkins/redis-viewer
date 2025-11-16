// Package rv redis operation.
package rv

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hawkins/redis-viewer/internal/constant"

	"github.com/go-redis/redis/v8"
)

// CountKeys .
func CountKeys(rdb redis.UniversalClient, match string, unlimited bool) (int, error) {
	ctx := context.TODO()

	switch rdb := rdb.(type) {
	case *redis.ClusterClient:
		var count int64

		err := rdb.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			iter := client.Scan(ctx, 0, match, 0).Iterator()
			for iter.Next(ctx) {
				atomic.AddInt64(&count, 1)
				if !unlimited && count > constant.MaxScanCount {
					break
				}
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
			if !unlimited && count > constant.MaxScanCount {
				break
			}
		}
		if err := iter.Err(); err != nil {
			return 0, err
		}

		return count, nil
	}
}

//nolint:govet
type keyMessage struct {
	Key string
	Err error
}

// GetKeys .
func GetKeys(
	rdb redis.UniversalClient,
	cursor uint64,
	match string,
	count int64,
	unlimited bool,
) <-chan keyMessage {
	res := make(chan keyMessage, 1)

	go func() {
		ctx := context.TODO()

		switch rdb := rdb.(type) {
		case *redis.ClusterClient:
			var i int64

			err := rdb.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
				iter := client.Scan(ctx, cursor, match, 0).Iterator()
				for iter.Next(ctx) {
					atomic.AddInt64(&i, 1)
					if !unlimited && i > count {
						break
					}
					res <- keyMessage{iter.Val(), nil}
				}
				if err := iter.Err(); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				res <- keyMessage{"", err}
			}
		default:
			// For non-cluster mode, if unlimited, scan until no more keys
			if unlimited {
				var allKeys []string
				cursor := uint64(0)
				for {
					keys, nextCursor, err := rdb.Scan(ctx, cursor, match, count).Result()
					if err != nil {
						res <- keyMessage{"", err}
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
					res <- keyMessage{key, nil}
				}
			} else {
				keys, _, err := rdb.Scan(ctx, cursor, match, count).Result()
				if err != nil {
					res <- keyMessage{"", err}
				} else {
					for _, key := range keys {
						res <- keyMessage{key, nil}
					}
				}
			}
		}

		close(res)
	}()

	return res
}

func DeleteKey(rdb redis.UniversalClient, key string) error {
	ctx := context.TODO()
	return rdb.Del(ctx, key).Err()
}

func SetKey(rdb redis.UniversalClient, key string, value string) error {
	ctx := context.TODO()
	return rdb.Set(ctx, key, value, 0).Err()
}

func SetKeyTTL(rdb redis.UniversalClient, key string, ttlSeconds int64) error {
	ctx := context.TODO()
	if ttlSeconds <= 0 {
		// Remove TTL (make key persistent)
		return rdb.Persist(ctx, key).Err()
	}
	// Set TTL
	return rdb.Expire(ctx, key, time.Duration(ttlSeconds)*time.Second).Err()
}

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

// Helper functions
func parseInfo(info string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(info, "\n")

	for _, line := range lines {
		// Trim whitespace and carriage returns
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return result
}

func parseInt64(s string) int64 {
	var result int64
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			result = result*10 + int64(s[i]-'0')
		}
	}
	return result
}

func parseFloat64(s string) float64 {
	var result float64
	var fraction float64
	var divisor float64 = 1.0
	inFraction := false

	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			inFraction = true
			continue
		}
		if s[i] >= '0' && s[i] <= '9' {
			digit := float64(s[i] - '0')
			if inFraction {
				divisor *= 10.0
				fraction = fraction*10.0 + digit
			} else {
				result = result*10.0 + digit
			}
		}
	}

	if inFraction && divisor > 1.0 {
		result += fraction / divisor
	}

	return result
}

func formatSeconds(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}

	days := seconds / 86400
	seconds %= 86400
	hours := seconds / 3600
	seconds %= 3600
	minutes := seconds / 60
	seconds %= 60

	var parts []string
	if days > 0 {
		parts = append(parts, formatInt(days)+"d")
	}
	if hours > 0 {
		parts = append(parts, formatInt(hours)+"h")
	}
	if minutes > 0 {
		parts = append(parts, formatInt(minutes)+"m")
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, formatInt(seconds)+"s")
	}

	return joinStrings(parts, " ")
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}

	var digits []byte
	for n > 0 {
		digits = append(digits, byte(n%10)+'0')
		n /= 10
	}

	// Reverse
	for i := 0; i < len(digits)/2; i++ {
		digits[i], digits[len(digits)-1-i] = digits[len(digits)-1-i], digits[i]
	}

	return string(digits)
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}

	totalLen := len(parts) - 1
	for _, p := range parts {
		totalLen += len(p)
	}

	result := make([]byte, 0, totalLen)
	for i, p := range parts {
		if i > 0 {
			result = append(result, sep...)
		}
		result = append(result, p...)
	}

	return string(result)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return formatInt(bytes) + "B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"K", "M", "G", "T", "P", "E"}

	// Calculate the value with 2 decimal places
	val := float64(bytes) / float64(div)

	// Format as string with 2 decimal places
	intPart := int64(val)
	fracPart := int64((val - float64(intPart)) * 100)

	result := formatInt(intPart) + "."
	if fracPart < 10 {
		result += "0"
	}
	result += formatInt(fracPart) + units[exp]

	return result
}
