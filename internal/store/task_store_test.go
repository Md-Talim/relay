package store_test

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/md-talim/relay/internal/db"
	"github.com/md-talim/relay/internal/db/migrations"
	"github.com/md-talim/relay/internal/store"
)

var testStore *store.TaskStore
var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()

	db, err := setupTestDB(ctx)
	if err != nil {
		log.Fatalf("failed to setup test db: %v", err)
	}

	testStore = store.NewTaskStore(db)
	testDB = db

	code := m.Run()

	dropTables(ctx, db)
	db.Close()

	os.Exit(code)
}

func TestCreateTask_NewTask(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() {
		truncateTables(ctx, testDB)
	})

	task := &store.Task{
		Type:       "send_email",
		Payload:    json.RawMessage(`{}`),
		Priority:   0,
		MaxRetries: 0,
		RunAt:      time.Now().Add(5 * time.Minute),
	}

	created, err := testStore.Create(ctx, task)
	if err != nil {
		t.Fatalf("Create() returned unexpected error: %v", err)
	}
	if !created {
		t.Fatal("Create() returned created=false, want true for new task")
	}

	// Check if DB fields populated
	if task.ID == (uuid.UUID{}) {
		t.Error("Create() did not populate ID")
	}
	if task.Status != "PENDING" {
		t.Errorf("Create() status = %q, want PENDING", task.Status)
	}
	if task.CreatedAt.IsZero() {
		t.Error("Create() did not populate CreatedAt")
	}
	if task.UpdatedAt.IsZero() {
		t.Error("Create() did not populate UpdatedAt")
	}
	if task.Attempts != 0 {
		t.Errorf("Create() attempts = %d, want 0", task.Attempts)
	}
}

func TestCreateTask_DelayedTask(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() {
		truncateTables(ctx, testDB)
	})
	future := time.Now().Add(1 * time.Hour).UTC().Truncate(time.Microsecond)

	task := &store.Task{
		Type:       "send_email",
		Payload:    json.RawMessage(`{}`),
		Priority:   0,
		MaxRetries: 0,
		RunAt:      future,
	}

	_, err := testStore.Create(ctx, task)
	if err != nil {
		t.Fatalf("Create() delayed task returned unexpected error: %v", err)
	}
	if !task.RunAt.Equal(future) {
		t.Errorf("Create() run_at = %v, want %v", task.RunAt, future)
	}
}

func TestCreateTask_IdempotencyKey_ReturnExisting(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() {
		truncateTables(ctx, testDB)
	})

	key := "order_123"
	task1 := &store.Task{
		Type:           "send_email",
		IdempotencyKey: &key,
		Payload:        json.RawMessage(`{}`),
		Priority:       0,
		MaxRetries:     0,
		RunAt:          time.Now().Add(5 * time.Minute),
	}

	task2 := &store.Task{
		Type:           "send_email",
		IdempotencyKey: &key,
		Payload:        json.RawMessage(`{}`),
		Priority:       0,
		MaxRetries:     0,
		RunAt:          time.Now().Add(5 * time.Minute),
	}

	if _, err := testStore.Create(ctx, task1); err != nil {
		t.Fatalf("Create() first insert returned unexpected error: %v", err)
	}

	created, err := testStore.Create(ctx, task2)
	if err != nil {
		t.Log(created)
		t.Fatalf("Create() duplicate idempotency key returned unexpected error: %v", err)
	}
	if created {
		t.Fatal("Create() returned created=true for duplicate idempotency key, want false")
	}
	if task2.ID != task1.ID {
		t.Errorf("Create() returned id=%v for duplicate, want original id=%v", task2.ID, task1.ID)
	}
}

func TestCreateTask_IdempotencyKey_SameKeyDifferentType(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() {
		truncateTables(ctx, testDB)
	})

	key := "order_123"
	task1 := &store.Task{
		Type:           "send_email",
		IdempotencyKey: &key,
		Payload:        json.RawMessage(`{}`),
		Priority:       0,
		MaxRetries:     0,
		RunAt:          time.Now().Add(5 * time.Minute),
	}

	task2 := &store.Task{
		Type:           "process_payment",
		IdempotencyKey: &key,
		Payload:        json.RawMessage(`{}`),
		Priority:       0,
		MaxRetries:     0,
		RunAt:          time.Now().Add(5 * time.Minute),
	}

	if _, err := testStore.Create(ctx, task1); err != nil {
		t.Fatalf("Create() send_email returned unexpected error: %v", err)
	}

	created, err := testStore.Create(ctx, task2)
	if err != nil {
		t.Log(created)
		t.Fatalf("Create() process_payment with same key returned unexpected error: %v", err)
	}
	if !created {
		t.Fatal("Create() returned created=false for same key but different type, want true")
	}
}

func TestCreateTask_NilIdempotencyKey_NoDuplicateConstraint(t *testing.T) {
	ctx := context.Background()
	t.Cleanup(func() {
		truncateTables(ctx, testDB)
	})

	task1 := &store.Task{
		Type:           "send_email",
		IdempotencyKey: nil,
		Payload:        json.RawMessage(`{}`),
		Priority:       0,
		MaxRetries:     0,
		RunAt:          time.Now().Add(5 * time.Minute),
	}
	task2 := &store.Task{
		Type:           "send_email",
		IdempotencyKey: nil,
		Payload:        json.RawMessage(`{}`),
		Priority:       0,
		MaxRetries:     0,
		RunAt:          time.Now().Add(5 * time.Minute),
	}

	if _, err := testStore.Create(ctx, task1); err != nil {
		t.Fatalf("Create() first nil key returned unexpected error: %v", err)
	}
	created, err := testStore.Create(ctx, task2)
	if err != nil {
		t.Fatalf("Create() second nil key returned unexpected error: %v", err)
	}
	if !created {
		t.Fatal("Create() returned created=false for nil idempotency key, want true (nulls are not unique-constrained)")
	}
}

func setupTestDB(ctx context.Context) (*pgxpool.Pool, error) {
	os.Setenv("RELAY_DATABASE_URL", "postgres://relay:relay@localhost:5433/relay_test?sslmode=disable")

	pool, err := db.Open(ctx)
	if err != nil {
		return nil, err
	}

	if err = migrations.RunMigrations(ctx, pool, nil, "../db/migrations"); err != nil {
		return nil, err
	}

	return pool, err
}

func dropTables(ctx context.Context, db *pgxpool.Pool) {
	db.Exec(ctx, `DROP TABLE IF EXISTS dead_letters, task_logs, tasks, schema_migrations CASCADE`)
}

func truncateTables(ctx context.Context, db *pgxpool.Pool) {
	db.Exec(ctx, `TRUNCATE TABLE dead_letters, task_logs, tasks RESTART IDENTITY CASCADE`)
}
