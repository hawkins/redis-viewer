// @description: commands
// @file: command.go
// @date: 2022/02/07

package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sahilm/fuzzy"
	"github.com/saltfishpr/redis-viewer/internal/rv"
	"github.com/saltfishpr/redis-viewer/internal/util"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cast"
)

type errMsg struct {
	err error
}

type scanMsg struct {
	items []list.Item
}

func (m model) scanCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var (
			val   interface{}
			err   error
			items []list.Item
		)

		keyMessages := rv.GetKeys(m.rdb, cast.ToUint64(m.offset*m.limit), m.searchValue, m.limit)

		// Collect all keys for potential fuzzy filtering
		var allKeys []string
		keyDataMap := make(map[string]struct {
			keyType    string
			val        string
			err        bool
			expiration string
		})

		for keyMessage := range keyMessages {
			if keyMessage.Err != nil {
				return errMsg{err: keyMessage.Err}
			}

			allKeys = append(allKeys, keyMessage.Key)
			kt := m.rdb.Type(ctx, keyMessage.Key).Val()
			ttl, ttlErr := m.rdb.TTL(ctx, keyMessage.Key).Result()
			var expirationStr string
			if ttlErr == nil && ttl > 0 {
				expirationStr = formatDuration(ttl)
			}
			switch kt {
			case "string":
				val, err = m.rdb.Get(ctx, keyMessage.Key).Result()
			case "list":
				val, err = m.rdb.LRange(ctx, keyMessage.Key, 0, -1).Result()
			case "set":
				val, err = m.rdb.SMembers(ctx, keyMessage.Key).Result()
			case "zset":
				val, err = m.rdb.ZRange(ctx, keyMessage.Key, 0, -1).Result()
			case "hash":
				val, err = m.rdb.HGetAll(ctx, keyMessage.Key).Result()
			default:
				val = ""
				err = fmt.Errorf("unsupported type: %s", kt)
			}

			var itemValue string
			var hasErr bool
			if err != nil {
				itemValue = err.Error()
				hasErr = true
			} else {
				if kt == "string" {
					itemValue = cast.ToString(val)
				} else {
					valBts, _ := util.JsonMarshalIndent(val)
					itemValue = string(valBts)
				}
			}

			keyDataMap[keyMessage.Key] = struct {
				keyType    string
				val        string
				err        bool
				expiration string
			}{keyType: kt, val: itemValue, err: hasErr, expiration: expirationStr}
		}

		// Apply fuzzy or strict filtering if fuzzy filter is set
		var filteredKeys []string
		if m.fuzzyFilter != "" {
			if m.fuzzyStrict {
				// Strict mode: case-insensitive substring matching
				filterLower := strings.ToLower(m.fuzzyFilter)
				for _, key := range allKeys {
					if strings.Contains(strings.ToLower(key), filterLower) {
						filteredKeys = append(filteredKeys, key)
					}
				}
			} else {
				// Fuzzy mode: fuzzy matching
				matches := fuzzy.Find(m.fuzzyFilter, allKeys)
				for _, match := range matches {
					filteredKeys = append(filteredKeys, match.Str)
				}
			}
		} else {
			filteredKeys = allKeys
		}

		// Build items list from filtered keys
		for _, key := range filteredKeys {
			data := keyDataMap[key]
			items = append(items, item{
				keyType:    data.keyType,
				key:        key,
				val:        data.val,
				err:        data.err,
				expiration: data.expiration,
			})
		}

		return scanMsg{items: items}
	}
}

type countMsg struct {
	count int
}

func (m model) countCmd() tea.Cmd {
	return func() tea.Msg {
		count, err := rv.CountKeys(m.rdb, m.searchValue)
		if err != nil {
			return errMsg{err: err}
		}

		return countMsg{count: count}
	}
}

type tickMsg struct {
	t string
}

func (m model) tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return tickMsg{t: time.Now().Format("2006-01-02 15:04:05")}
	})
}

type deleteMsg struct {
	key string
	err error
}

func (m model) deleteCmd(key string) tea.Cmd {
	return func() tea.Msg {
		err := rv.DeleteKey(m.rdb, key)
		return deleteMsg{key: key, err: err}
	}
}

type purgeMsg struct {
	db  int
	err error
}

func (m model) purgeCmd() tea.Cmd {
	return func() tea.Msg {
		err := rv.FlushDB(m.rdb)
		return purgeMsg{db: m.db, err: err}
	}
}

type switchDBMsg struct {
	db     int
	newRdb interface{}
	err    error
}

func (m model) switchDBCmd(db int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Create new options with the new database
		newOpts := *m.redisOpts
		newOpts.DB = db

		// Create new client with the new database
		newRdb := redis.NewUniversalClient(&newOpts)

		// Test the connection
		_, err := newRdb.Ping(ctx).Result()
		if err != nil {
			newRdb.Close()
			return switchDBMsg{db: db, newRdb: nil, err: err}
		}

		return switchDBMsg{db: db, newRdb: newRdb, err: nil}
	}
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "" // No expiration or already expired
	}

	days := d / (24 * time.Hour)
	remaining := d % (24 * time.Hour)
	hours := remaining / time.Hour
	remaining %= time.Hour
	minutes := remaining / time.Minute
	seconds := remaining % time.Minute / time.Second

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || len(parts) > 0 {
		parts = append(parts, fmt.Sprintf("%02dh", hours))
	}
	if minutes > 0 || len(parts) > 0 {
		parts = append(parts, fmt.Sprintf("%02dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%02ds", seconds))
	}

	return strings.Join(parts, " ")
}

type statsMsg struct {
	serverStats *rv.ServerStats
	dbStats     []*rv.DatabaseStats
	err         error
}

func (m model) statsCmd() tea.Cmd {
	return func() tea.Msg {
		// Get server stats
		serverStats, err := rv.GetServerStats(m.rdb)
		if err != nil {
			return statsMsg{err: err}
		}

		// Get stats for multiple databases (0-15 for standard Redis)
		var dbStats []*rv.DatabaseStats
		maxDB := 16 // Standard Redis has 16 databases (0-15)

		// For cluster mode, only DB 0 is available
		if _, ok := m.rdb.(*redis.ClusterClient); ok {
			maxDB = 1
		}

		for i := 0; i < maxDB; i++ {
			// Create a client for this specific database
			opts := *m.redisOpts
			opts.DB = i

			dbClient := redis.NewUniversalClient(&opts)
			defer dbClient.Close()

			// Test if this database is accessible
			ctx := context.Background()
			_, err := dbClient.Ping(ctx).Result()
			if err != nil {
				// Skip databases that are not accessible
				continue
			}

			// Get stats for this database (sample 10 keys for TTL average)
			stats, err := rv.GetDatabaseStats(dbClient, i, 10)
			if err == nil && stats.Keys > 0 {
				// Only include databases that have keys
				dbStats = append(dbStats, stats)
			}
		}

		return statsMsg{
			serverStats: serverStats,
			dbStats:     dbStats,
			err:         nil,
		}
	}
}

type editKeyMsg struct {
	key     string
	tmpFile string
	err     error
}

type editKeyResultMsg struct {
	key string
	err error
}

func (m model) editKeyCmd(key string, currentValue string) tea.Cmd {
	return func() tea.Msg {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("redis-viewer-%s-*.txt", sanitizeFilename(key)))
		if err != nil {
			return editKeyMsg{key: key, err: fmt.Errorf("failed to create temp file: %w", err)}
		}

		// Write current value to temp file
		if _, err := tmpFile.WriteString(currentValue); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpFile.Name())
			return editKeyMsg{key: key, err: fmt.Errorf("failed to write to temp file: %w", err)}
		}
		_ = tmpFile.Close()

		// Return message with temp file path to trigger editor
		return editKeyMsg{key: key, tmpFile: tmpFile.Name(), err: nil}
	}
}

func (m model) openEditorCmd(tmpFilePath string) tea.Cmd {
	// Get editor from environment or use default
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi" // fallback to vi
	}

	c := exec.Command(editor, tmpFilePath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{tmpFile: tmpFilePath, err: err}
	})
}

type editorFinishedMsg struct {
	tmpFile string
	err     error
}

func (m model) processEditedKeyCmd(key string, tmpFilePath string) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			_ = os.Remove(tmpFilePath)
		}()

		// Read the modified content
		content, err := os.ReadFile(tmpFilePath)
		if err != nil {
			return editKeyResultMsg{key: key, err: fmt.Errorf("failed to read temp file: %w", err)}
		}

		// Update Redis key
		if err := rv.SetKey(m.rdb, key, string(content)); err != nil {
			return editKeyResultMsg{key: key, err: fmt.Errorf("failed to update key: %w", err)}
		}

		return editKeyResultMsg{key: key, err: nil}
	}
}

type createKeyMsg struct {
	key     string
	tmpFile string
	err     error
}

type createKeyResultMsg struct {
	key string
	err error
}

func (m model) createKeyCmd(keyName string) tea.Cmd {
	return func() tea.Msg {
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("redis-viewer-new-%s-*.txt", sanitizeFilename(keyName)))
		if err != nil {
			return createKeyMsg{key: keyName, err: fmt.Errorf("failed to create temp file: %w", err)}
		}

		_ = tmpFile.Close()

		// Return message with temp file path to trigger editor
		return createKeyMsg{key: keyName, tmpFile: tmpFile.Name(), err: nil}
	}
}

func (m model) processCreatedKeyCmd(key string, tmpFilePath string) tea.Cmd {
	return func() tea.Msg {
		defer func() {
			_ = os.Remove(tmpFilePath)
		}()

		// Read the content
		content, err := os.ReadFile(tmpFilePath)
		if err != nil {
			return createKeyResultMsg{key: key, err: fmt.Errorf("failed to read temp file: %w", err)}
		}

		// Create Redis key
		if err := rv.SetKey(m.rdb, key, string(content)); err != nil {
			return createKeyResultMsg{key: key, err: fmt.Errorf("failed to create key: %w", err)}
		}

		return createKeyResultMsg{key: key, err: nil}
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
