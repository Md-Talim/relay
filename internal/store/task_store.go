package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pgErrUniqueViolation = "23505"

type Task struct {
	ID             uuid.UUID
	Type           string
	Payload        json.RawMessage
	PayloadHash    []byte
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

type TaskStore interface {
	Create(ctx context.Context, task *Task) (bool, error)
	GetById(ctx context.Context, id string) (*Task, error)
	Cancel(ctx context.Context, id string) (*Task, error)
}

type PostgresTaskStore struct {
	db *pgxpool.Pool
}

func NewTaskStore(db *pgxpool.Pool) *PostgresTaskStore {
	return &PostgresTaskStore{db: db}
}

func (ts *PostgresTaskStore) Create(ctx context.Context, task *Task) (bool, error) {
	payloadHash, err := hashPayload(task.Payload) // new task payload hash
	if err != nil {
		return false, err
	}

	query := `
	INSERT INTO tasks(type, payload, payload_hash, idempotency_key,  priority, max_retries, run_at)
	VALUES($1, $2, $3, $4, $5, $6, $7)
	RETURNING id, status, attempts, created_at, updated_at
	`

	err = ts.db.QueryRow(
		ctx,
		query,
		task.Type,
		task.Payload,
		payloadHash,
		task.IdempotencyKey,
		task.Priority,
		task.MaxRetries,
		task.RunAt,
	).Scan(&task.ID, &task.Status, &task.Attempts, &task.CreatedAt, &task.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			if pgErr.ConstraintName == "tasks_idempotency_type_key" {
				existingTask, err := ts.getByTypeAndIdempotencyKey(ctx, task.Type, *task.IdempotencyKey)
				if err != nil {
					return false, err
				}
				if !bytes.Equal(existingTask.PayloadHash, payloadHash) {
					return false, ErrTaskConflict
				}

				*task = *existingTask
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func (ts *PostgresTaskStore) GetById(ctx context.Context, id string) (*Task, error) {
	query := `
	SELECT
		id, type, payload, idempotency_key, status, priority, attempts, max_retries,
		run_at, started_at, completed_at, last_error, created_at, updated_at
    FROM tasks WHERE id = $1
	`

	task := &Task{}
	err := ts.db.QueryRow(ctx, query, id).Scan(
		&task.ID, &task.Type, &task.Payload, &task.IdempotencyKey, &task.Status, &task.Priority, &task.Attempts,
		&task.MaxRetries, &task.RunAt, &task.StartedAt, &task.CompletedAt, &task.LastError, &task.CreatedAt, &task.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (ts *PostgresTaskStore) Cancel(ctx context.Context, id string) (*Task, error) {
	query := `
		UPDATE tasks SET status = 'CANCELED', updated_at = now()
		WHERE id = $1 AND status = 'PENDING'
		RETURNING id, status, updated_at
	`

	task := &Task{}
	err := ts.db.QueryRow(ctx, query).Scan(&task.ID, &task.Status, &task.UpdatedAt)
	if err == nil {
		// canceled successfully - TODO: log it
		return task, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err // real DB error
	}

	current, err := ts.GetById(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	return current, nil
}

func (ts *PostgresTaskStore) getByTypeAndIdempotencyKey(ctx context.Context, taskType, idempotencyKey string) (*Task, error) {
	query := `
	SELECT
		id, type, payload, payload_hash, idempotency_key, status, priority, attempts, max_retries,
		run_at, started_at, completed_at, last_error, created_at, updated_at
    FROM tasks WHERE type = $1 AND idempotency_key = $2
	`

	task := &Task{}
	err := ts.db.QueryRow(ctx, query, taskType, idempotencyKey).Scan(
		&task.ID, &task.Type, &task.Payload, &task.PayloadHash, &task.IdempotencyKey, &task.Status, &task.Priority, &task.Attempts,
		&task.MaxRetries, &task.RunAt, &task.StartedAt, &task.CompletedAt, &task.LastError, &task.CreatedAt, &task.UpdatedAt,
	)

	return task, err
}
