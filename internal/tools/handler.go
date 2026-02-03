package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"mouse/internal/logging"
	"mouse/internal/sandbox"
)

type Handler struct {
	policy *Policy
	runner *sandbox.Runner
	logger *logging.Logger
}

type request struct {
	Tool    string   `json:"tool"`
	Command []string `json:"command"`
}

type response struct {
	OK         bool   `json:"ok"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

func NewHandler(policy *Policy, runner *sandbox.Runner, logger *logging.Logger) *Handler {
	return &Handler{policy: policy, runner: runner, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.runner == nil || h.policy == nil {
		writeError(w, http.StatusServiceUnavailable, "tool runner not configured")
		return
	}
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	tool := strings.TrimSpace(req.Tool)
	if tool == "" {
		writeError(w, http.StatusBadRequest, "tool is required")
		return
	}
	if !h.policy.Allowed(tool) {
		h.logDenied(tool)
		writeError(w, http.StatusForbidden, "tool is not allowed")
		return
	}
	if len(req.Command) == 0 {
		writeError(w, http.StatusBadRequest, "command is required")
		return
	}

	result, err := h.runner.Run(r.Context(), req.Command)
	if err != nil {
		h.logFailure(tool, result, err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if result.ExitCode != 0 {
		h.logFailure(tool, result, fmt.Errorf("exit %d", result.ExitCode))
	} else {
		h.logSuccess(tool, result)
	}
	writeJSON(w, http.StatusOK, response{
		OK:         result.ExitCode == 0,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
		ExitCode:   result.ExitCode,
		DurationMS: result.Duration.Milliseconds(),
	})
}

func (h *Handler) logDenied(tool string) {
	if h.logger == nil {
		return
	}
	h.logger.Warn("tool denied", map[string]string{
		"tool": tool,
	})
}

func (h *Handler) logSuccess(tool string, result sandbox.Result) {
	if h.logger == nil {
		return
	}
	h.logger.Info("tool executed", map[string]string{
		"tool":         tool,
		"exit_code":    strconv.Itoa(result.ExitCode),
		"duration_ms":  strconv.FormatInt(result.Duration.Milliseconds(), 10),
		"stdout_bytes": strconv.Itoa(len(result.Stdout)),
		"stderr_bytes": strconv.Itoa(len(result.Stderr)),
	})
}

func (h *Handler) logFailure(tool string, result sandbox.Result, err error) {
	if h.logger == nil {
		return
	}
	fields := map[string]string{
		"tool":         tool,
		"exit_code":    strconv.Itoa(result.ExitCode),
		"duration_ms":  strconv.FormatInt(result.Duration.Milliseconds(), 10),
		"stdout_bytes": strconv.Itoa(len(result.Stdout)),
		"stderr_bytes": strconv.Itoa(len(result.Stderr)),
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	h.logger.Error("tool failed", fields)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, response{OK: false, Error: message})
}

func writeJSON(w http.ResponseWriter, status int, payload response) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
