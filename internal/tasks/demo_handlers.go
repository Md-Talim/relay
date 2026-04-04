package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

func Echo(ctx context.Context, payload json.RawMessage) error {
	slog.Info("echo handler", "payload", string(payload))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
		return nil
	}
}

func SendEmail(ctx context.Context, payload json.RawMessage) error {
	var p struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	slog.Info("sending email", "to", p.To, "subject", p.Subject)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
		return nil
	}
}

func AlwaysFails(ctx context.Context, payload json.RawMessage) error {
	return errors.New("intentional failure for testing retry behavior")
}

func SlowTask(ctx context.Context, payload json.RawMessage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(10 * time.Minute):
		return nil
	}
}
