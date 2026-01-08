package gocontainer

import (
	"context"
	"database/sql"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/infrastructure/db"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/infrastructure/logging"
	"github.com/buzyka/imlate/internal/infrastructure/repository"
	"github.com/buzyka/imlate/internal/usecase/tracking"
	"github.com/golobby/container/v3"
	"go.uber.org/zap"
)

type ERPFactory struct {
	f *isams.ClientFactory
}

func (ef *ERPFactory) NewClient(ctx context.Context) (erp.Client, error) {
	return ef.f.NewClient(ctx)
}

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

	container.MustSingleton(container.Global, func () provider.VisitorRepository {
		return &repository.Visitor{
			Connection: connection,
		}
	})

	container.MustSingleton(container.Global, func () provider.VisitorTrackRepository {
		return &repository.VisitorTrack{
			Connection: connection,
		}
	})

	container.MustSingleton(container.Global, func () erp.Factory {
		f := &isams.ClientFactory{
			BaseURL:    cfg.ISAMSBaseURL,
			ClientID:   cfg.ISAMSAPIClientID,
			ClientSecret:     cfg.ISAMSAPIClientSecret,
		}
		return &ERPFactory{f: f}
	})

	container.MustSingleton(container.Global, func () provider.VisitorRepository {
		return &repository.Visitor{
			Connection: connection,
		}
	})

	container.MustSingleton(container.Global, func () *tracking.StudentTracker {
		tracker := &tracking.StudentTracker{}
		container.MustFill(container.Global, tracker)
		return tracker
	})
}
