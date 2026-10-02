// Command backfill-activity seeds the activity_events feed from pre-existing
// tracking state (shows added, episodes watched) so the social feed has history
// from before event emission was wired into the tracking transactions. It is
// idempotent and safe to re-run.
package main

import (
	"context"
	"log"

	"github.com/deface90/defshows/backend/internal/service/usecase"
	"github.com/deface90/defshows/backend/migrations"
	"github.com/deface90/defshows/backend/pkg/config"
	"github.com/deface90/defshows/backend/pkg/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("backfill-activity: config: %v", err)
	}
	gdb, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("backfill-activity: db: %v", err)
	}
	if err := db.RunMigrations(gdb, migrations.FS, migrations.Dir); err != nil {
		log.Fatalf("backfill-activity: migrate: %v", err)
	}

	n, err := usecase.NewActivityBackfill(gdb).Run(context.Background())
	if err != nil {
		log.Fatalf("backfill-activity: run: %v", err)
	}
	log.Printf("backfill-activity: inserted %d event(s)", n)
}
