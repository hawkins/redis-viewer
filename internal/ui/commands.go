package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	redisv8 "github.com/go-redis/redis/v8"
	"github.com/hawkins/redis-viewer/internal/redis"
	"github.com/hawkins/redis-viewer/internal/ui/components/keylist"
	"github.com/hawkins/redis-viewer/internal/util"
	"github.com/sahilm/fuzzy"
	"github.com/spf13/cast"
)

const scanBatchSize = 50

// scanCmd initiates a key scan operation
func (a App) scanCmd() tea.Cmd {
	return a.scanStreamCmd()
}

// scanStreamCmd performs streaming key scan for better performance
func (a App) scanStreamCmd() tea.Cmd {
	return func() tea.Msg {
		keyMessages := redis.GetKeys(a.rdb, cast.ToUint64(a.offset*a.limit), "", a.limit)

		// Quickly collect all key names (no TYPE/TTL - much faster!)
		var allKeys []string
		var processedCount int

		for keyMessage := range keyMessages {
			if keyMessage.Err != nil {
				return ErrMsg{Err: keyMessage.Err}
			}

			processedCount++
			allKeys = append(allKeys, keyMessage.Key)
		}

		// Apply filtering
		filteredKeys := a.applyFilter(allKeys)

		// Create items WITHOUT fetching TYPE/TTL - this makes scanning MUCH faster
		var items []list.Item
		for _, key := range filteredKeys {
			items = append(items, keylist.Item{
				KeyType:    "", // Will be fetched on demand
				Key:        key,
				Val:        "",
				Err:        false,
				TTLSeconds: -1, // -1 means not yet fetched
				Loaded:     false,
			})
		}

		// Return items immediately - TYPE/TTL will be fetched lazily when selected
		return ScanMsg{
			Items:        items,
			IsComplete:   true,
			TotalScanned: processedCount,
		}
	}
}

// applyFilter applies fuzzy or strict filtering to keys
func (a App) applyFilter(allKeys []string) []string {
	if a.fuzzyFilter == "" {
		return allKeys
	}

	var filteredKeys []string
	if a.fuzzyStrict {
		filterLower := strings.ToLower(a.fuzzyFilter)
		for _, key := range allKeys {
			if strings.Contains(strings.ToLower(key), filterLower) {
				filteredKeys = append(filteredKeys, key)
			}
		}
	} else {
		matches := fuzzy.Find(a.fuzzyFilter, allKeys)
		for _, match := range matches {
			filteredKeys = append(filteredKeys, match.Str)
		}
	}
	return filteredKeys
}

// loadValueCmd loads the value for a specific key
func (a App) loadValueCmd(key string, keyType string, ttlSeconds int64) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var (
			val interface{}
			err error
		)

		// Fetch TYPE if not already known
		if keyType == "" {
			keyType = a.rdb.Type(ctx, key).Val()
		}

		// Fetch TTL if not already known
		if ttlSeconds == -1 {
			ttl, ttlErr := a.rdb.TTL(ctx, key).Result()
			if ttlErr == nil && ttl > 0 {
				ttlSeconds = int64(ttl.Seconds())
			} else {
				ttlSeconds = 0 // No TTL or error
			}
		}

		// Fetch the value based on key type
		switch keyType {
		case "string":
			val, err = a.rdb.Get(ctx, key).Result()
		case "list":
			val, err = a.rdb.LRange(ctx, key, 0, -1).Result()
		case "set":
			val, err = a.rdb.SMembers(ctx, key).Result()
		case "zset":
			val, err = a.rdb.ZRange(ctx, key, 0, -1).Result()
		case "hash":
			val, err = a.rdb.HGetAll(ctx, key).Result()
		default:
			val = ""
			err = fmt.Errorf("unsupported type: %s", keyType)
		}

		var itemValue string
		if err != nil {
			itemValue = err.Error()
		} else {
			if keyType == "string" {
				itemValue = cast.ToString(val)
			} else {
				valBts, _ := util.JsonMarshalIndent(val)
				itemValue = string(valBts)
			}
		}

		return LoadValueMsg{
			Key:        key,
			KeyType:    keyType,
			Val:        itemValue,
			Err:        err,
			TTLSeconds: ttlSeconds,
		}
	}
}

// displayBatchCmd displays keys in batches for better UX
func (a App) displayBatchCmd() tea.Cmd {
	return func() tea.Msg {
		const batchSize = 50

		if a.pendingScanIndex >= len(a.pendingScanItems) {
			// No more items to display
			return ScanBatchMsg{Batch: nil, IsComplete: true}
		}

		// Calculate batch range
		start := a.pendingScanIndex
		end := start + batchSize
		if end > len(a.pendingScanItems) {
			end = len(a.pendingScanItems)
		}

		batchItems := a.pendingScanItems[start:end]
		isComplete := end >= len(a.pendingScanItems)

		// Convert to []list.Item
		batch := make([]list.Item, len(batchItems))
		for i, item := range batchItems {
			batch[i] = item
		}

		return ScanBatchMsg{
			Batch:      batch,
			IsComplete: isComplete,
		}
	}
}

// countCmd counts matching keys
func (a App) countCmd() tea.Cmd {
	return func() tea.Msg {
		count, err := redis.CountKeys(a.rdb, "")
		if err != nil {
			return ErrMsg{Err: err}
		}

		return CountMsg{Count: count}
	}
}

// tickCmd provides clock updates
func (a App) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return TickMsg{T: time.Now().Format("2006-01-02 15:04:05")}
	})
}

// deleteCmd deletes a key
func (a App) deleteCmd(key string) tea.Cmd {
	return func() tea.Msg {
		err := redis.DeleteKey(a.rdb, key)
		return DeleteMsg{Key: key, Err: err}
	}
}

// setTTLCmd sets TTL for a key
func (a App) setTTLCmd(key string, ttlSeconds int64) tea.Cmd {
	return func() tea.Msg {
		err := redis.SetKeyTTL(a.rdb, key, ttlSeconds)
		return SetTTLMsg{Key: key, TTL: ttlSeconds, Err: err}
	}
}

// purgeCmd flushes the current database
func (a App) purgeCmd() tea.Cmd {
	return func() tea.Msg {
		err := redis.FlushDB(a.rdb)
		return PurgeMsg{DB: a.db, Err: err}
	}
}

// switchDBCmd switches to a different database
func (a App) switchDBCmd(db int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Create new options with the new database
		newOpts := *a.redisOpts
		newOpts.DB = db

		// Create new client with the new database
		newRdb := redisv8.NewUniversalClient(&newOpts)

		// Test the connection
		_, err := newRdb.Ping(ctx).Result()
		if err != nil {
			newRdb.Close()
			return SwitchDBMsg{DB: db, NewRdb: nil, Err: err}
		}

		return SwitchDBMsg{DB: db, NewRdb: newRdb, Err: nil}
	}
}

// statsCmd loads server statistics
func (a App) statsCmd() tea.Cmd {
	return func() tea.Msg {
		// Get server stats
		serverStats, err := redis.GetServerStats(a.rdb)
		if err != nil {
			return StatsMsg{Err: err}
		}

		// Get stats for multiple databases (0-15 for standard Redis)
		var dbStats []*redis.DatabaseStats
		maxDB := 16 // Standard Redis has 16 databases (0-15)

		// For cluster mode, only DB 0 is available
		if _, ok := a.rdb.(*redisv8.ClusterClient); ok {
			maxDB = 1
		}

		for i := 0; i < maxDB; i++ {
			// Create a client for this specific database
			opts := *a.redisOpts
			opts.DB = i

			dbClient := redisv8.NewUniversalClient(&opts)
			defer dbClient.Close()

			// Test if this database is accessible
			ctx := context.Background()
			_, err := dbClient.Ping(ctx).Result()
			if err != nil {
				// Skip databases that are not accessible
				continue
			}

			// Get stats for this database (sample 10 keys for TTL average)
			stats, err := redis.GetDatabaseStats(dbClient, i, 10)
			if err == nil && stats.Keys > 0 {
				// Only include databases that have keys
				dbStats = append(dbStats, stats)
			}
		}

		return StatsMsg{
			ServerStats: serverStats,
			DBStats:     dbStats,
			Err:         nil,
		}
	}
}

// editKeyCmd prepares a key for editing
func (a App) editKeyCmd(key string, currentValue string) tea.Cmd {
	return func() tea.Msg {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("redis-viewer-%s-*.txt", sanitizeFilename(key)))
		if err != nil {
			return EditKeyMsg{Key: key, Err: fmt.Errorf("failed to create temp file: %w", err)}
		}

		// Write current value to temp file
		if _, err := tmpFile.WriteString(currentValue); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
			return EditKeyMsg{Key: key, Err: fmt.Errorf("failed to write to temp file: %w", err)}
		}
		_ = tmpFile.Close()

		// Return message with temp file path to trigger editor
		return EditKeyMsg{Key: key, TmpFile: tmpFile.Name(), Err: nil}
	}
}

// openEditorCmd opens an external editor
func (a App) openEditorCmd(tmpFilePath string) tea.Cmd {
	// Get editor from environment or use default
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // fallback to vi
	}

	c := exec.Command(editor, tmpFilePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{TmpFile: tmpFilePath, Err: err}
	})
}

// processEditedKeyCmd processes the edited key content
func (a App) processEditedKeyCmd(key string, tmpFilePath string) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			_ = os.Remove(tmpFilePath)
		}()

		// Read the modified content
		content, err := os.ReadFile(tmpFilePath)
		if err != nil {
			return EditKeyResultMsg{Key: key, Err: fmt.Errorf("failed to read temp file: %w", err)}
		}

		// Update Redis key
		if err := redis.SetKey(a.rdb, key, string(content)); err != nil {
			return EditKeyResultMsg{Key: key, Err: fmt.Errorf("failed to update key: %w", err)}
		}

		return EditKeyResultMsg{Key: key, Err: nil}
	}
}

// createKeyCmd prepares to create a new key
func (a App) createKeyCmd(keyName string) tea.Cmd {
	return func() tea.Msg {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("redis-viewer-new-%s-*.txt", sanitizeFilename(keyName)))
		if err != nil {
			return CreateKeyMsg{Key: keyName, Err: fmt.Errorf("failed to create temp file: %w", err)}
		}

		_ = tmpFile.Close()

		// Return message with temp file path to trigger editor
		return CreateKeyMsg{Key: keyName, TmpFile: tmpFile.Name(), Err: nil}
	}
}

// processCreatedKeyCmd processes the created key content
func (a App) processCreatedKeyCmd(key string, tmpFilePath string) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			_ = os.Remove(tmpFilePath)
		}()

		// Read the content
		content, err := os.ReadFile(tmpFilePath)
		if err != nil {
			return CreateKeyResultMsg{Key: key, Err: fmt.Errorf("failed to read temp file: %w", err)}
		}

		// Create Redis key
		if err := redis.SetKey(a.rdb, key, string(content)); err != nil {
			return CreateKeyResultMsg{Key: key, Err: fmt.Errorf("failed to create key: %w", err)}
		}

		return CreateKeyResultMsg{Key: key, Err: nil}
	}
}

// sanitizeFilename removes characters that might be problematic in filenames
func sanitizeFilename(s string) string {
	// Replace problematic characters with underscores
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	result := replacer.Replace(s)

	// Limit length to avoid filesystem issues
	if len(result) > 50 {
		result = result[:50]
	}

	return result
}
