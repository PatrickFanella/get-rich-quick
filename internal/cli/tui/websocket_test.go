package tui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	internalapi "github.com/PatrickFanella/get-rich-quick/internal/api"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestConnectWebSocketSubscribesAndStreamsEvents(t *testing.T) {
	t.Parallel()

	upgrader := websocket.Upgrader{}
	runID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		defer conn.Close()

		var command map[string]string
		if err := conn.ReadJSON(&command); err != nil {
			t.Fatalf("read command: %v", err)
		}
		if command["action"] != "subscribe_all" {
			t.Fatalf("action = %q, want subscribe_all", command["action"])
		}

		if err := conn.WriteJSON(map[string]string{"status": "ok", "action": "subscribe_all"}); err != nil {
			t.Fatalf("write ack: %v", err)
		}
		if err := conn.WriteJSON(internalapi.WSMessage{
			Type:      internalapi.EventSignal,
			RunID:     runID,
			Timestamp: time.Now().UTC(),
			Data:      map[string]string{"ticker": "AAPL", "signal": "buy"},
		}); err != nil {
			t.Fatalf("write event: %v", err)
		}
	}))
	defer server.Close()

	source, err := ConnectWebSocket(context.Background(), "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatalf("ConnectWebSocket() error = %v", err)
	}
	defer source.Close()

	select {
	case msg := <-source.Messages():
		raw, err := json.Marshal(msg.Data)
		if err != nil {
			t.Fatalf("marshal msg data: %v", err)
		}
		if msg.RunID != runID {
			t.Fatalf("RunID = %s, want %s", msg.RunID, runID)
		}
		if !strings.Contains(string(raw), "AAPL") {
			t.Fatalf("event data = %s, want ticker payload", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for websocket event")
	}
}
