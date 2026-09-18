// Command auth runs the defShows authentication service.
package main

import (
	"log"

	httpapi "github.com/deface90/defshows/backend/internal/gateways/http"
	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/migrations"
	"github.com/deface90/defshows/backend/pkg/auth"
	"github.com/deface90/defshows/backend/pkg/config"
	"github.com/deface90/defshows/backend/pkg/db"
	pkglog "github.com/deface90/defshows/backend/pkg/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("auth: config: %v", err)
	}
	if cfg.JWT.Secret == "" {
		log.Fatal("auth: JWT_SECRET is required")
	}

	logger := pkglog.New(cfg.Log)

	gdb, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("auth: db: %v", err)
	}
	if err := db.RunMigrations(gdb, migrations.FS, migrations.Dir); err != nil {
		log.Fatalf("auth: migrate: %v", err)
	}

	repo := repository.NewUserRepository(gdb)
	jwtMgr := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.AccessTTL)
	authUC := usecase.NewAuthUsecase(repo, jwtMgr, cfg.JWT.RefreshTTL)
	handler := httpapi.NewAuthHandler(authUC).
		WithOAuth(httpapi.BuildOAuthRegistry(cfg.OAuth), cfg.OAuth.RedirectBaseURL, cfg.OAuth.FrontendURL)

	e := httpapi.NewAuthRouter(handler, jwtMgr)
	logger.Info("auth service starting", "addr", cfg.Server.Addr)
	if err := e.Start(cfg.Server.Addr); err != nil {
		log.Fatalf("auth: server: %v", err)
	}
}
