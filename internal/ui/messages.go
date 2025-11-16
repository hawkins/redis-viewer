package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/hawkins/redis-viewer/internal/redis"
)

// Error message
type ErrMsg struct {
	Err error
}

// Scan messages
type ScanMsg struct {
	Items        []list.Item
	IsComplete   bool
	TotalScanned int
}

type ScanBatchMsg struct {
	Batch      []list.Item
	IsComplete bool
}

type ScanProgressMsg struct {
	ProcessedCount int
}

type ScanStartedMsg struct{}

// Value loading message
type LoadValueMsg struct {
	Key        string
	KeyType    string
	Val        string
	Err        error
	TTLSeconds int64
}

// Count message
type CountMsg struct {
	Count int
}

// Clock tick message
type TickMsg struct {
	T string
}

// Delete message
type DeleteMsg struct {
	Key string
	Err error
}

// TTL message
type SetTTLMsg struct {
	Key string
	TTL int64
	Err error
}

// Purge message
type PurgeMsg struct {
	DB  int
	Err error
}

// Switch database message
type SwitchDBMsg struct {
	DB     int
	NewRdb interface{}
	Err    error
}

// Stats messages
type StatsMsg struct {
	ServerStats *redis.ServerStats
	DBStats     []*redis.DatabaseStats
	Err         error
}

// Edit key messages
type EditKeyMsg struct {
	Key     string
	TmpFile string
	Err     error
}

type EditKeyResultMsg struct {
	Key string
	Err error
}

type EditorFinishedMsg struct {
	TmpFile string
	Err     error
}

// Create key messages
type CreateKeyMsg struct {
	Key     string
	TmpFile string
	Err     error
}

type CreateKeyResultMsg struct {
	Key string
	Err error
}
