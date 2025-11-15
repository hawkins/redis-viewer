// @description: commands
// @file: command.go
// @date: 2022/02/07

package tui

import (
	"context"
	"fmt"
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

		// Apply fuzzy filtering if fuzzy filter is set
		var filteredKeys []string
		if m.fuzzyFilter != "" {
			matches := fuzzy.Find(m.fuzzyFilter, allKeys)
			for _, match := range matches {
				filteredKeys = append(filteredKeys, match.Str)
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
