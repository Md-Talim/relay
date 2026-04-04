package tasks

import (
	"context"
	"encoding/json"
)

type HandlerFunc func(ctx context.Context, payload json.RawMessage) error

type HandlerRegistry interface {
	Get(taskType string) (HandlerFunc, bool)
}
