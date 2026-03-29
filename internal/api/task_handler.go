package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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

	req.normalize()

	if err := req.validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	task := req.toTask()

	created, err := h.taskStore.Create(r.Context(), task)
	if err != nil {
		if errors.Is(err, store.ErrTaskConflict) {
			writeError(w, http.StatusConflict, "idempotency key reused with different payload")
			return
		}
		h.logger.Error("failed to create task", "err", err, "task_type", task.Type)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	writeJSON(w, status, newTaskResponse(task))
}

func (h *TaskHandler) HandleGetTaskById(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.taskStore.GetById(r.Context(), id.String())
	if err != nil {
		h.logger.Error("failed to get task by id", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	writeJSON(w, http.StatusOK, newTaskResponse(task))
}
