package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	apiresp "github.com/mllbll/kosmohak_nn/internal/api"
	projectV1 "github.com/mllbll/kosmohak_nn/internal/api/project/v1"
	runV1 "github.com/mllbll/kosmohak_nn/internal/api/run/v1"
	"github.com/mllbll/kosmohak_nn/internal/config"
)

type App struct {
	diContainer *diContainer
	httpServer  *http.Server
}

func New(_ context.Context, cfg config.Config) (*App, error) {
	a := &App{
		diContainer: NewDiContainer(cfg),
	}
	if err := a.initHTTPServer(cfg); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *App) initHTTPServer(cfg config.Config) error {
	projectAPI := projectV1.NewAPI(a.diContainer.ProjectService())
	runAPI := runV1.NewAPI(a.diContainer.RunService())

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Minute))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		apiresp.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Post("/api/projects", projectAPI.Create)
	r.Get("/api/projects/{id}", projectAPI.Get)
	r.Patch("/api/projects/{id}", projectAPI.Patch)
	r.Post("/api/projects/{id}/runs", runAPI.Create)

	r.Get("/api/runs/{id}", runAPI.Get)
	r.Get("/api/runs/{id}/metrics", runAPI.GetMetrics)
	r.Get("/api/runs/{id}/snapshot", runAPI.GetSnapshot)
	r.Get("/api/runs/{id}/export", runAPI.Export)
	r.Post("/api/compare", runAPI.Compare)

	a.httpServer = &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return nil
}

func (a *App) Run(_ context.Context) error {
	log.Printf("http server listening on %s", a.httpServer.Addr)
	err := a.httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.httpServer.Shutdown(ctx)
}
