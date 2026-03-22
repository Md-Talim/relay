package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/md-talim/relay/internal/store"
)

type TaskHandler struct {
	taskStore *store.TaskStore
	logger    *slog.Logger
}

func NewTaskHandler(taskStore *store.TaskStore, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{taskStore: taskStore, logger: logger}
}

func (h *TaskHandler) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("decode create task request", "err", err)
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	req.setDefaults()

	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	task := &store.Task{
		Type:           req.Type,
		Payload:        req.Payload,
		IdempotencyKey: req.IdempotencyKey,
		Priority:       *req.Priority,
		MaxRetries:     *req.MaxRetries,
		RunAt:          *req.RunAt,
	}

	if err := h.taskStore.Create(r.Context(), task); err != nil {
		h.logger.Error("failed to create task", "err", err, "task_type", task.Type)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	resp := newTaskResponse(task)
	writeJSON(w, http.StatusCreated, resp)
}

type createTaskRequest struct {
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey *string         `json:"idempotency_key"`
	Priority       *int            `json:"priority"`
	MaxRetries     *int            `json:"max_retries"`
	RunAt          *time.Time      `json:"run_at"`
}

func (r *createTaskRequest) setDefaults() {
	if r.Payload == nil {
		r.Payload = json.RawMessage(`{}`)
	}
	if r.Priority == nil {
		defaultPriority := 0
		r.Priority = &defaultPriority
	}
	if r.MaxRetries == nil {
		defaultRetries := 5
		r.MaxRetries = &defaultRetries
	}
	if r.RunAt == nil {
		now := time.Now()
		r.RunAt = &now
	}
}

func (r *createTaskRequest) validate() error {
	if r.Type == "" {
		return errors.New("type is required")
	}
	if len(r.Type) > 100 {
		return errors.New("type must be 100 characters or fewer")
	}
	if *r.Priority < 0 || *r.Priority > 100 {
		return errors.New("priority must be between 0 and 100")
	}
	if *r.MaxRetries < 0 || *r.MaxRetries > 20 {
		return errors.New("max_retries must be between 0 and 20")
	}
	if r.IdempotencyKey != nil && len(*r.IdempotencyKey) > 255 {
		return errors.New("idempotency_key must be 255 characters or fewer")
	}
	if len(r.Payload) > 64*1024 {
		return errors.New("payload must not exceed 64KB")
	}
	if r.RunAt.Before(time.Now().Add(-5 * time.Minute)) {
		return errors.New("run_at cannot be in the past")
	}
	if r.RunAt.After(time.Now().Add(30 * 24 * time.Hour)) {
		return errors.New("run_at cannot be more than 30 days in the future")
	}

	return nil
}

type taskResponse struct {
	ID          uuid.UUID  `json:"id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	Priority    int        `json:"priority"`
	Attempts    int        `json:"attempts"`
	MaxRetries  int        `json:"max_retries"`
	RunAt       time.Time  `json:"run_at"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	LastError   *string    `json:"last_error"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func newTaskResponse(t *store.Task) taskResponse {
	r := taskResponse{
		ID:          t.ID,
		Type:        t.Type,
		Status:      t.Status,
		Attempts:    t.Attempts,
		MaxRetries:  t.MaxRetries,
		RunAt:       t.RunAt,
		StartedAt:   t.StartedAt,
		CompletedAt: t.CompletedAt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}

	if t.LastError != nil {
		switch t.Status {
		case "DEAD", "RUNNING":
			r.LastError = t.LastError
		case "PENDING":
			if t.Attempts > 0 {
				r.LastError = t.LastError
			}
		}
	}

	return r
}
