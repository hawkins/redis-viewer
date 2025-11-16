// @description: list item
// @file: item.go
// @date: 2022/02/08

package tui

import (
	"fmt"
)

type item struct {
	keyType string

	key        string
	val        string
	expiration string // Store formatted expiration string
	ttlSeconds int64  // Store TTL in seconds

	err bool
}

func (i item) Title() string { return i.key }

func (i item) Description() string {
	if i.err {
		return "get error: " + i.val
	}
	return fmt.Sprintf("%d: %d bytes", len(i.key), len(i.val))
}

func (i item) FilterValue() string { return i.key }
