package indexer

import (
	"encoding/json"
	"net/http"
	"strconv"

	"mouse/internal/logging"
)

type Handler struct {
	indexer *Indexer
	logger  *logging.Logger
}

type ReindexHandler struct {
	indexer *Indexer
	logger  *logging.Logger
}

type searchResponse struct {
	Matches []Match `json:"matches"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler(indexer *Indexer, logger *logging.Logger) *Handler {
	return &Handler{indexer: indexer, logger: logger}
}

func NewReindexHandler(indexer *Indexer, logger *logging.Logger) *ReindexHandler {
	return &ReindexHandler{indexer: indexer, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.indexer == nil {
		writeError(w, http.StatusServiceUnavailable, "indexer not configured")
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "q is required")
		return
	}
	limit := 5
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if val, err := strconv.Atoi(raw); err == nil && val > 0 {
			limit = val
		}
	}
	matches, err := h.indexer.Search(r.Context(), query, limit)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("index search failed", map[string]string{
				"error": err.Error(),
			})
		}
		writeError(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, searchResponse{Matches: matches})
}

func (h *ReindexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if h.indexer == nil {
		writeError(w, http.StatusServiceUnavailable, "indexer not configured")
		return
	}
	if err := h.indexer.ScanOnce(r.Context()); err != nil {
		if h.logger != nil {
			h.logger.Error("index reindex failed", map[string]string{
				"error": err.Error(),
			})
		}
		writeError(w, http.StatusInternalServerError, "reindex failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
