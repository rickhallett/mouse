package approvals

import (
	"encoding/json"
	"net/http"
	"strings"

	"mouse/internal/logging"
)

type Handler struct {
	logger *logging.Logger
}

type approveRequest struct {
	ID string `json:"id"`
}

type approveResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func NewHandler(logger *logging.Logger) *Handler {
	return &Handler{logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req approveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, approveResponse{OK: false, Error: "invalid json"})
		return
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, approveResponse{OK: false, Error: "id is required"})
		return
	}
	if h.logger != nil {
		h.logger.Info("approval received", map[string]string{"id": id})
	}
	writeJSON(w, http.StatusOK, approveResponse{OK: true})
}

func writeJSON(w http.ResponseWriter, status int, payload approveResponse) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
