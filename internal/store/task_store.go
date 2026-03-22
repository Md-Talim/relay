package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Task struct {
	ID             uuid.UUID
	Type           string
	Payload        json.RawMessage
	IdempotencyKey *string
	Status         string
	Priority       int
	Attempts       int
	MaxRetries     int
	RunAt          time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
	LockedAt       *time.Time
	LockedBy       *string
	LastError      *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TaskStore struct {
	db *pgxpool.Pool
}

func NewTaskStore(db *pgxpool.Pool) *TaskStore {
	return &TaskStore{db: db}
}

func (ts *TaskStore) Create(ctx context.Context, task *Task) error {
	query := `
	INSERT INTO tasks(type, payload, idempotency_key,  priority, max_retries, run_at)
	VALUES($1, $2, $3, $4, $5, $6)
	RETURNING id, status, attempts, created_at, updated_at
	`

	return ts.db.QueryRow(
		ctx,
		query,
		task.Type,
		task.Payload,
		task.IdempotencyKey,
		task.Priority,
		task.MaxRetries,
		task.RunAt,
	).Scan(&task.ID, &task.Status, &task.Attempts, &task.CreatedAt, &task.UpdatedAt)
}
