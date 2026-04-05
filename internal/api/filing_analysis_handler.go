package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/automation"
	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

type analyzeFilingRequest struct {
	Symbol string `json:"symbol"`
	Form   string `json:"form"`
	URL    string `json:"url"`
}

func (s *Server) handleAnalyzeFiling(w http.ResponseWriter, r *http.Request) {
	if s.llmProvider == nil {
		respondError(w, http.StatusNotImplemented, "LLM provider not configured", ErrCodeNotImplemented)
		return
	}

	var req analyzeFilingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", ErrCodeBadRequest)
		return
	}

	if req.Symbol == "" {
		respondError(w, http.StatusBadRequest, "symbol is required", ErrCodeValidation)
		return
	}
	if req.Form == "" {
		respondError(w, http.StatusBadRequest, "form is required", ErrCodeValidation)
		return
	}
	if req.URL == "" {
		respondError(w, http.StatusBadRequest, "url is required", ErrCodeValidation)
		return
	}

	filing := domain.SECFiling{
		Symbol:    req.Symbol,
		Form:      req.Form,
		URL:       req.URL,
		FiledDate: time.Now().UTC(),
	}

	analysis, err := automation.AnalyzeFiling(r.Context(), s.llmProvider, "", filing, "", s.logger)
	if err != nil {
		s.logger.Error("filing analysis failed", "symbol", req.Symbol, "error", err)
		respondError(w, http.StatusInternalServerError, "filing analysis failed", ErrCodeInternal)
		return
	}

	respondJSON(w, http.StatusOK, analysis)
}
