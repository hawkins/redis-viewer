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
	ttlSeconds int64

	err    bool
	loaded bool // indicates if value has been fetched from Redis
}

func (i item) Title() string { return i.key }

func (i item) Description() string {
	if i.err {
		return "get error: " + i.val
	}
	if !i.loaded {
		return fmt.Sprintf("%s (not loaded)", i.keyType)
	}
	return fmt.Sprintf("%s: %d bytes", i.keyType, len(i.val))
}

func (i item) FilterValue() string { return i.key }
