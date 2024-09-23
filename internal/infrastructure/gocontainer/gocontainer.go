package gocontainer

import (
	"database/sql"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/infrastructure/db"
	"github.com/buzyka/imlate/internal/infrastructure/logging"
	"github.com/buzyka/imlate/internal/infrastructure/repository"
	"github.com/buzyka/imlate/internal/isb/entity"
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
}