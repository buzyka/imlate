package cron

import (
	"context"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/usecase/synchroniser"
	"github.com/buzyka/imlate/internal/usecase/tracking"
	"github.com/go-co-op/gocron/v2"
	"github.com/golobby/container/v3"
	"go.uber.org/zap"
)

type JobFunction func()

type StopCronFunc = func()

func RunCron(cfg *config.Config) (StopCronFunc, error) {
	if !cfg.IsERPIntegrated() {
		return func() {}, nil
	}

	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	err = registerJobs(s, cfg.ForceERPSyncOnStart)
	if err != nil {
		return nil, err
	}

	go s.Start()

	return func() {
		_ = s.Shutdown()
	}, nil
}

func registerJobs(s gocron.Scheduler, forceERPSyncOnStart bool) error {
	// Student data sync job
	_, err := s.NewJob(
		gocron.CronJob(
			"0 7-17/2 * * 1-5", // every 2 hours from 7am to 5pm on weekdays
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
			"0 5 * * 1-5", // at 5am on weekdays
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
			"0 7-17/1 * * 1-5", // every hour from 7am to 5pm on weekdays
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
			"10 8-12/1 * * 1-5", // every hour from 8:10am to 12:10pm on weekdays
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
	if forceERPSyncOnStart {
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
