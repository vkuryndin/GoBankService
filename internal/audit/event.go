package audit

import "context"

const (
	StatusSuccess = "success"
	StatusFailed  = "failed"
	StatusBlocked = "blocked"
)

type Event struct {
	UserID       *int64
	Action       string
	ResourceType string
	ResourceID   *int64
	Status       string
	IPAddress    string
	UserAgent    string
	Details      map[string]any
}

type Recorder interface {
	Record(ctx context.Context, event Event)
}

func Int64Ptr(value int64) *int64 {
	return &value
}
