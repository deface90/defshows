// Command backfill-notifications seeds "already handled" markers for episodes that have
// already aired, so turning on a previously-broken delivery channel (e.g. APNs, whose
// rows were never created due to a bad column mapping) does not fire a burst of catch-up
// pushes for every user's existing backlog on first run. Markers carry no deliveries (so
// nothing is pushed) and reserve the scanner's dedupe keys. It is idempotent and safe to
// re-run.
//
// The window matches the notifier's NOTIFIER_LOOKBACK (the scanner never looks back
// further), so seeding that window is enough. To cover everything, run it once with a
// large NOTIFIER_LOOKBACK.
package main

import (
	"context"
	"log"

	"github.com/deface90/defshows/backend/internal/service/repository"
	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/migrations"
	"github.com/deface90/defshows/backend/pkg/config"
	"github.com/deface90/defshows/backend/pkg/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("backfill-notifications: config: %v", err)
	}
	gdb, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("backfill-notifications: db: %v", err)
	}
	if err := db.RunMigrations(gdb, migrations.FS, migrations.Dir); err != nil {
		log.Fatalf("backfill-notifications: migrate: %v", err)
	}

	repo := repository.NewNotificationRepository(gdb)
	n, err := usecase.NewNotificationBackfill(repo).Run(context.Background(), cfg.Notifier.Lookback)
	if err != nil {
		log.Fatalf("backfill-notifications: run: %v", err)
	}
	log.Printf("backfill-notifications: seeded %d marker(s) within the %s lookback window", n, cfg.Notifier.Lookback)
}
