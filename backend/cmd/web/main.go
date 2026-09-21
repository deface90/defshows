// Command web runs the defShows web API (catalog + tracking + auth-protected).
package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	httpapi "github.com/deface90/defshows/backend/internal/gateways/http"
	"github.com/deface90/defshows/backend/internal/gateways/providers/tmdb"
	"github.com/deface90/defshows/backend/internal/service/entity"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/migrations"
	"github.com/deface90/defshows/backend/pkg/auth"
	"github.com/deface90/defshows/backend/pkg/config"
	"github.com/deface90/defshows/backend/pkg/crud"
	"github.com/deface90/defshows/backend/pkg/db"
	"github.com/deface90/defshows/backend/pkg/httpx"
	pkglog "github.com/deface90/defshows/backend/pkg/log"
	"github.com/deface90/defshows/backend/pkg/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("web: config: %v", err)
	}
	if cfg.JWT.Secret == "" {
		log.Fatal("web: JWT_SECRET is required")
	}

	logger := pkglog.New(cfg.Log)
	slog.SetDefault(logger)

	gdb, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("web: db: %v", err)
	}
	if err := db.RunMigrations(gdb, migrations.FS, migrations.Dir); err != nil {
		log.Fatalf("web: migrate: %v", err)
	}

	proxyClient, err := httpx.Client(cfg.Proxy, 15*time.Second)
	if err != nil {
		log.Fatalf("web: proxy: %v", err)
	}
	tmdbProvider, err := tmdb.New("", cfg.TMDB.APIKey, cfg.TMDB.Language, proxyClient)
	if err != nil {
		log.Fatalf("web: tmdb: %v", err)
	}

	// Optional image mirror: copies posters/backdrops into object storage and
	// serves them from /images, so the browser never hits the TMDB image host.
	var imageStore *storage.S3
	if cfg.S3.Enabled() {
		imageStore, err = storage.New(cfg.S3, proxyClient)
		if err != nil {
			log.Fatalf("web: storage: %v", err)
		}
		logger.Info("image mirror enabled", "bucket", cfg.S3.Bucket)
	}

	userRepo := repository.NewUserRepository(gdb)
	catalogRepo := repository.NewCatalogRepository(gdb)
	trackingRepo := repository.NewTrackingRepository(gdb)
	homeRepo := repository.NewHomeRepository(gdb)

	notificationRepo := repository.NewNotificationRepository(gdb)
	notesRepo := repository.NewNotesRepository(gdb)

	jwtMgr := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessTTL)
	authUC := usecase.NewAuthUsecase(userRepo, jwtMgr, cfg.JWT.RefreshTTL)

	// Optionally bootstrap a default admin from env (idempotent).
	if seeded, err := authUC.SeedAdmin(context.Background(), cfg.Seed.AdminEmail, cfg.Seed.AdminPassword); err != nil {
		log.Fatalf("web: seed admin: %v", err)
	} else if seeded {
		logger.Info("seeded default admin", "email", cfg.Seed.AdminEmail)
	}
	catalogUC := usecase.NewCatalogUsecase(catalogRepo, tmdbProvider)
	if imageStore != nil {
		catalogUC.WithImageMirror(imageStore)
	}
	trackingUC := usecase.NewTrackingUsecase(trackingRepo, catalogUC)
	homeUC := usecase.NewHomeUsecase(homeRepo)
	notificationUC := usecase.NewNotificationUsecase(notificationRepo, userRepo, cfg.Telegram.Username, cfg.Notifier.LinkTTL)
	notesUC := usecase.NewNotesUsecase(notesRepo)

	authH := httpapi.NewAuthHandler(authUC).
		WithOAuth(httpapi.BuildOAuthRegistry(cfg.OAuth), cfg.OAuth.RedirectBaseURL, cfg.OAuth.FrontendURL)
	showsH := httpapi.NewShowsHandler(catalogUC)
	trackingH := httpapi.NewTrackingHandler(trackingUC, homeUC, authUC)
	notificationsH := httpapi.NewNotificationsHandler(notificationUC)
	notesH := httpapi.NewNotesHandler(notesUC)
	adminH := httpapi.NewAdminHandler(crud.NewRepository[entity.DubbingStudio](gdb))
	usersH := httpapi.NewUsersHandler(authUC, trackingUC)

	e := httpapi.NewWebRouter(authH, showsH, trackingH, notificationsH, notesH, adminH, usersH, jwtMgr, imageStore, proxyClient, cfg.Server.CORSAllowedOrigins...)
	logger.Info("web service starting", "addr", cfg.Server.Addr)
	if err := e.Start(cfg.Server.Addr); err != nil {
		log.Fatalf("web: server: %v", err)
	}
}
