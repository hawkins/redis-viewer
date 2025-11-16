package keylist

import "fmt"

// Item represents a Redis key in the list
type Item struct {
	KeyType string

	Key        string
	Val        string
	TTLSeconds int64

	Err    bool
	Loaded bool // indicates if value has been fetched from Redis
}

// Title implements list.Item
func (i Item) Title() string { return i.Key }

// Description implements list.Item
func (i Item) Description() string {
	if i.Err {
		return "get error: " + i.Val
	}
	if !i.Loaded {
		return fmt.Sprintf("%s (not loaded)", i.KeyType)
	}
	return fmt.Sprintf("%s: %d bytes", i.KeyType, len(i.Val))
}

// FilterValue implements list.Item
func (i Item) FilterValue() string { return i.Key }
