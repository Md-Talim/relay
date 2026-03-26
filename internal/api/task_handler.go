package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

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

	if err := h.taskStore.Create(r.Context(), task); err != nil {
		h.logger.Error("failed to create task", "err", err, "task_type", task.Type)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	resp := newTaskResponse(task)
	writeJSON(w, http.StatusCreated, resp)
}
