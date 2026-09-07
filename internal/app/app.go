package app

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zeldojov/gopost/internal/database"
	"github.com/zeldojov/gopost/internal/session"
)

type Config struct {
	DBPath  string
	Address string
	Session session.CookieConfig
}

type App struct {
	config       Config
	db           *sql.DB
	sessionStore *session.Store
	logger       *log.Logger
}

func New(config Config) (*App, error) {
	a := &App{
		config: config,
		logger: log.Default(),
	}

	if err := a.openDB(); err != nil {
		return nil, err
	}

	sessionRepository := database.NewSessionRepository(a.db)
	a.sessionStore = session.NewStore(sessionRepository)

	return a, nil
}

func (a *App) openDB() error {
	db, err := database.OpenDB(a.config.DBPath)
	if err != nil {
		return err
	}

	if err := database.CreateSessionsTable(db); err != nil {
		db.Close()
		return err
	}

	a.db = db

	return nil
}

func (a *App) Close() error {
	return a.db.Close()
}

func (a *App) Run(handler http.Handler) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	return a.run(ctx, handler)
}

func (a *App) run(ctx context.Context, handler http.Handler) error {
	server := &http.Server{
		Addr:    a.config.Address,
		Handler: handler,
	}

	serverErr := make(chan error, 1)

	go func() {
		a.logger.Printf(
			"Running server on http://%s",
			a.config.Address,
		)

		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil

	case <-ctx.Done():
		a.logger.Print("Shutting down server...")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		return server.Shutdown(shutdownCtx)
	}
}
