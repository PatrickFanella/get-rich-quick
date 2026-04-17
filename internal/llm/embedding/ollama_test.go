package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaProvider_Embed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/embed" {
			t.Fatalf("expected /api/embed, got %s", r.URL.Path)
		}

		var req embedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "nomic-embed-text" {
			t.Fatalf("expected model nomic-embed-text, got %s", req.Model)
		}
		if len(req.Input) != 1 {
			t.Fatalf("expected 1 input, got %d", len(req.Input))
		}

		resp := embedResponse{
			Model:      req.Model,
			Embeddings: [][]float64{make([]float64, 768)},
		}
		// Set first dim to known value for assertion.
		resp.Embeddings[0][0] = 0.42

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	p := NewOllamaProvider(OllamaConfig{
		BaseURL: srv.URL,
	})

	vec, err := p.Embed(context.Background(), "test input")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 768 {
		t.Fatalf("expected 768 dims, got %d", len(vec))
	}
	if vec[0] != float32(0.42) {
		t.Fatalf("expected vec[0]=0.42, got %v", vec[0])
	}
}

func TestOllamaProvider_EmbedBatch(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var req embedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		embs := make([][]float64, len(req.Input))
		for i := range embs {
			embs[i] = make([]float64, 768)
			embs[i][0] = float64(i + callCount*100)
		}
		resp := embedResponse{
			Model:      req.Model,
			Embeddings: embs,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	p := NewOllamaProvider(OllamaConfig{
		BaseURL:   srv.URL,
		BatchSize: 2, // Force multiple batches.
	})

	texts := []string{"a", "b", "c"}
	vecs, err := p.EmbedBatch(context.Background(), texts)
	if err != nil {
		t.Fatalf("EmbedBatch: %v", err)
	}
	if len(vecs) != 3 {
		t.Fatalf("expected 3 vectors, got %d", len(vecs))
	}
	// First batch (2 items) → callCount=1, second batch (1 item) → callCount=2.
	if callCount != 2 {
		t.Fatalf("expected 2 HTTP calls, got %d", callCount)
	}
	for _, v := range vecs {
		if len(v) != 768 {
			t.Fatalf("expected 768 dims, got %d", len(v))
		}
	}
}

func TestOllamaProvider_EmbedBatch_Empty(t *testing.T) {
	p := NewOllamaProvider(OllamaConfig{BaseURL: "http://unused"})
	vecs, err := p.EmbedBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("EmbedBatch(nil): %v", err)
	}
	if vecs != nil {
		t.Fatalf("expected nil, got %v", vecs)
	}
}

func TestOllamaProvider_EmbedServerError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"model not found"}`))
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	p := NewOllamaProvider(OllamaConfig{BaseURL: srv.URL})
	_, err := p.Embed(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}

func TestOllamaProvider_Defaults(t *testing.T) {
	p := NewOllamaProvider(OllamaConfig{})
	if p.baseURL != DefaultBaseURL {
		t.Fatalf("expected default base URL %q, got %q", DefaultBaseURL, p.baseURL)
	}
	if p.model != DefaultModel {
		t.Fatalf("expected default model %q, got %q", DefaultModel, p.model)
	}
	if p.batchSize != DefaultBatchSize {
		t.Fatalf("expected default batch size %d, got %d", DefaultBatchSize, p.batchSize)
	}
}
