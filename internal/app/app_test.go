package app

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestApp_Close(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := app.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = app.db.Exec("SELECT 1")
	if err == nil {
		t.Fatal("expected database to be closed")
	}
}

func TestNew_InitializesSessionStore(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	sessionID, err := app.sessionStore.AddSession(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = app.sessionStore.GetSession(sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
func TestNew_InvalidDBPath(t *testing.T) {
	_, err := New(Config{
		DBPath: "/invalid/path/gopost.db",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNew_CreatesSessionsTable(t *testing.T) {
	app, err := New(Config{
		DBPath: ":memory:",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer app.Close()

	var name string

	err = app.db.QueryRow(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'table' AND name = 'sessions'
	`).Scan(&name)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if name != "sessions" {
		t.Fatalf("expected sessions table, got %q", name)
	}
}

func TestApp_Run_ReturnsServerError(t *testing.T) {
	app := &App{
		config: Config{
			Address: "invalid-address",
		},
		logger: log.Default(),
	}

	err := app.run(context.Background(), http.HandlerFunc(func(
		http.ResponseWriter,
		*http.Request,
	) {
	}))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
func TestApp_Run_ShutsDownOnContextCancellation(t *testing.T) {
	app := &App{
		config: Config{
			Address: "127.0.0.1:0",
		},
		logger: log.Default(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() {
		done <- app.run(ctx, http.HandlerFunc(func(
			http.ResponseWriter,
			*http.Request,
		) {
		}))
	}()

	time.Sleep(50 * time.Millisecond)

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

	case <-time.After(time.Second):
		t.Fatal("server did not shut down")
	}
}
