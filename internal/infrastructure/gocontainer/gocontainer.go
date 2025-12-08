package gocontainer

import (
	"context"
	"database/sql"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/infrastructure/db"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/infrastructure/logging"
	"github.com/buzyka/imlate/internal/infrastructure/repository"
	"github.com/buzyka/imlate/internal/isb/entity"
	"github.com/buzyka/imlate/internal/isb/registration"
	"github.com/golobby/container/v3"
	"go.uber.org/zap"
)

func Build(cfg *config.Config) {
	logger := logging.NewLogger(true)
	connection, err := db.Open(cfg.DatabaseEngine, cfg.DatabaseURL, logger)
	if err != nil {
		panic(err.Error())
	}

	container.MustSingleton(container.Global, func () *config.Config {
		return cfg		
	})

	container.MustSingleton(container.Global, func () *zap.SugaredLogger {
		return logger
	})

	container.MustSingleton(container.Global, func() *sql.DB {		
		return connection
	})

	container.MustSingleton(container.Global, func () entity.VisitorRepository {
		return &repository.Visitor{
			Connection: connection,
		}
	})

	container.MustSingleton(container.Global, func () entity.VisitorTrackRepository {
		return &repository.VisitorTrack{
			Connection: connection,
		}
	})
	container.MustSingleton(container.Global, func () registration.Registrator {
		f := &isams.ClientFactory{
			BaseURL:      "https://developerdemo.isams.cloud",
			ClientID:     "FA6B1B76-46CF-4B75-A29D-70C327A33ED2",
			ClientSecret: "9D904BDB-C8B0-4BEA-97C0-2EE2EC3AC680",
		}
		ctx := context.Background()
		c, err := f.NewClient(ctx)
		if err != nil {
			panic(err)
		}
		return c
	})
}
