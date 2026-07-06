package cron

import (
	"context"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/infrastructure/util"
	"github.com/buzyka/imlate/internal/usecase/synchroniser"
	"github.com/buzyka/imlate/internal/usecase/tracking"
	"github.com/go-co-op/gocron/v2"
	"github.com/golobby/container/v3"
	"go.uber.org/zap"
)

type JobFunction func()

type StopCronFunc = func()

func RunCron(cfg *config.Config) (StopCronFunc, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	// Core jobs run regardless of ERP integration.
	if err := registerJobs(s, cfg); err != nil {
		return nil, err
	}

	// ERP-specific jobs run only when the ERP integration is enabled.
	if cfg.IsERPIntegrated() {
		if err := registerERPJobs(s, cfg); err != nil {
			return nil, err
		}
	}

	go s.Start()

	return func() {
		_ = s.Shutdown()
	}, nil
}

// registerJobs registers core jobs that run regardless of ERP integration.
// Add future non-ERP jobs here.
func registerJobs(s gocron.Scheduler, cfg *config.Config) error {
	// Visit report finalization job
	_, err := s.NewJob(
		gocron.CronJob(
			cfg.CronFinalizeReports, // default: at 00:30 every day
			false,
		),
		gocron.NewTask(FinalizeReportsFunc()),
	)
	return err
}

func registerERPJobs(s gocron.Scheduler, cfg *config.Config) error {
	// Student data sync job
	_, err := s.NewJob(
		gocron.CronJob(
			cfg.CronStudentSync, // default: every 2 hours from 7am to 5pm on weekdays
			false,
		),
		gocron.NewTask(StudentSyncFunc()),
	)
	if err != nil {
		return err
	}

	// Photos sync job
	_, err = s.NewJob(
		gocron.CronJob(
			cfg.CronPhotoSync, // default: at 5am on weekdays
			false,
		),
		gocron.NewTask(StudentPhotosSyncFunc()),
	)
	if err != nil {
		return err
	}

	// Registration codes sync job
	_, err = s.NewJob(
		gocron.CronJob(
			cfg.CronRegistrationCodesSync, // default: every hour from 7am to 5pm on weekdays
			false,
		),
		gocron.NewTask(RegistrationCodesSyncFunc()),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		return err
	}

	// Mark not registered students as absent job
	_, err = s.NewJob(
		gocron.CronJob(
			cfg.CronMarkAbsent, // default: every hour from 8:10am to 12:10pm on weekdays
			false,
		),
		gocron.NewTask(MarkNotRegisteredStudentsAsAbsent()),
	)
	if err != nil {
		return err
	}

	// You can register more cron jobs here
	// ...

	// one-time startup run, sequential (only if forceERPSyncOnStart is enabled)
	if cfg.ForceERPSyncOnStart {
		go func() {
			StudentSyncFunc()()
			StudentPhotosSyncFunc()()
		}()
	}

	return nil
}

func StudentSyncFunc() JobFunction {
	return func() {
		var log *zap.SugaredLogger
		container.MustResolve(container.Global, &log)
		log.Infof("Starting Students data sync... TIME: %s", time.Now().Format(time.RFC3339))
		sync := synchroniser.StudentSync{}
		container.MustFill(container.Global, &sync)
		if err := sync.SyncAllStudents(); err != nil {
			log.Errorw("Error during Students data sync", "error", err)
		}
		log.Infof("Finish Students data sync... TIME: %s", time.Now().Format(time.RFC3339))
	}
}

func RegistrationCodesSyncFunc() JobFunction {
	return func() {
		var log *zap.SugaredLogger
		container.MustResolve(container.Global, &log)
		log.Infof("Starting registration codes data sync... TIME: %s", time.Now().Format(time.RFC3339))
		sync := synchroniser.StudentSync{}
		container.MustFill(container.Global, &sync)
		if err := sync.SyncRegistrationCodesDictionaries(); err != nil {
			log.Errorf("Error during registration codes data sync: %v\n", err)
		}
	}
}

func StudentPhotosSyncFunc() JobFunction {
	return func() {
		var log *zap.SugaredLogger
		container.MustResolve(container.Global, &log)
		log.Infof("Starting student photos data sync... TIME: %s", time.Now().Format(time.RFC3339))
		sync := synchroniser.StudentSync{}
		container.MustFill(container.Global, &sync)
		if err := sync.SyncStudentPhotos(); err != nil {
			log.Errorf("Error during student photos data sync: %v\n", err)
		}
	}
}

func FinalizeReportsFunc() JobFunction {
	return func() {
		var log *zap.SugaredLogger
		container.MustResolve(container.Global, &log)
		var repo provider.VisitDailyReportRepository
		container.MustResolve(container.Global, &repo)

		log.Infof("Starting visit report finalization... TIME: %s", time.Now().Format(time.RFC3339))

		// util.Now() already carries the configured app-local timezone.
		now := util.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		days, err := repo.UnfinalizedDaysBefore(today)
		if err != nil {
			log.Errorf("Error finding unfinalized report days: %v\n", err)
			return
		}

		for _, day := range days {
			if err := repo.FinalizeDay(day); err != nil {
				log.Errorf("Error finalizing report for day %s: %v\n", day.Format("2006-01-02"), err)
			}
		}
		log.Infof("Finish visit report finalization... finalized %d day(s)", len(days))
	}
}

func MarkNotRegisteredStudentsAsAbsent() JobFunction {
	return func() {
		var log *zap.SugaredLogger
		container.MustResolve(container.Global, &log)
		var cfg *config.Config
		container.MustResolve(container.Global, &cfg)
		log.Infof("Starting marking not registered students as absent... TIME: %s", time.Now().Format(time.RFC3339))
		st := tracking.StudentTracker{}
		container.MustFill(container.Global, &st)
		var opts []tracking.StudentFilterOption
		if len(cfg.AutoRegistrationYearGroups) > 0 {
			opts = append(opts, tracking.WithYearGroups(cfg.AutoRegistrationYearGroups))
		}
		if err := st.TrackUntrackedStudentsAsAbsence(context.Background(), opts...); err != nil {
			log.Errorf("Error during marking not registered students as absent: %v\n", err)
		}
	}
}
