// Command worker runs the defShows integration worker (catalog refresh).
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/deface90/defshows/backend/internal/gateways/providers/omdb"
	"github.com/deface90/defshows/backend/internal/gateways/providers/tmdb"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/internal/service/workers"
	"github.com/deface90/defshows/backend/migrations"
	"github.com/deface90/defshows/backend/pkg/config"
	"github.com/deface90/defshows/backend/pkg/db"
	"github.com/deface90/defshows/backend/pkg/httpx"
	pkglog "github.com/deface90/defshows/backend/pkg/log"
	"github.com/deface90/defshows/backend/pkg/storage"
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

	proxyClient, err := httpx.Client(cfg.Proxy, 15*time.Second)
	if err != nil {
		log.Fatalf("worker: proxy: %v", err)
	}
	tmdbProvider, err := tmdb.New("", cfg.TMDB.APIKey, cfg.TMDB.Language, proxyClient)
	if err != nil {
		log.Fatalf("worker: tmdb: %v", err)
	}
	catalogRepo := repository.NewCatalogRepository(gdb)
	catalogUC := usecase.NewCatalogUsecase(catalogRepo, tmdbProvider)
	if cfg.S3.Enabled() {
		imageStore, err := storage.New(cfg.S3, proxyClient)
		if err != nil {
			log.Fatalf("worker: storage: %v", err)
		}
		catalogUC.WithImageMirror(imageStore)
		logger.Info("image mirror enabled", "bucket", cfg.S3.Bucket)
	}
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
