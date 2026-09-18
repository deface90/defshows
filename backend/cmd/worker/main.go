// Command worker runs the defShows integration worker (catalog refresh).
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/deface90/defshows/backend/internal/gateways/providers/omdb"
	"github.com/deface90/defshows/backend/internal/gateways/providers/tmdb"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/internal/service/workers"
	"github.com/deface90/defshows/backend/migrations"
	"github.com/deface90/defshows/backend/pkg/config"
	"github.com/deface90/defshows/backend/pkg/db"
	pkglog "github.com/deface90/defshows/backend/pkg/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("worker: config: %v", err)
	}
	logger := pkglog.New(cfg.Log)

	gdb, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("worker: db: %v", err)
	}
	if err := db.RunMigrations(gdb, migrations.FS, migrations.Dir); err != nil {
		log.Fatalf("worker: migrate: %v", err)
	}

	tmdbProvider, err := tmdb.New("", cfg.TMDB.APIKey, cfg.TMDB.Language, nil)
	if err != nil {
		log.Fatalf("worker: tmdb: %v", err)
	}
	catalogRepo := repository.NewCatalogRepository(gdb)
	catalogUC := usecase.NewCatalogUsecase(catalogRepo, tmdbProvider)
	syncer := workers.NewCatalogSyncer(catalogUC, logger, cfg.Worker.StaleAge, cfg.Worker.Throttle, cfg.Worker.Batch)

	if cfg.OMDb.APIKey != "" {
		omdbProvider, err := omdb.New("", cfg.OMDb.APIKey, nil)
		if err != nil {
			log.Fatalf("worker: omdb: %v", err)
		}
		syncer = syncer.WithRatings(usecase.NewRatingsUsecase(omdbProvider, catalogRepo))
	} else {
		logger.Warn("OMDB_API_KEY not set — ratings refresh disabled")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("worker starting", "interval", cfg.Worker.SyncInterval.String())
	syncer.Run(ctx, cfg.Worker.SyncInterval)
	logger.Info("worker stopped")
}
