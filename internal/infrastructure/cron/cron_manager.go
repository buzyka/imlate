package cron

import (
	"time"

	"github.com/buzyka/imlate/internal/usecase/synchroniser"
	"github.com/go-co-op/gocron/v2"
	"github.com/golobby/container/v3"
	"go.uber.org/zap"
)

type StopCronFunc = func()

func RunCron() (StopCronFunc, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	err = registerJobs(s)
	if err != nil {
		return nil, err
	}

	go s.Start()

	return func() {
		_ = s.Shutdown()
	}, nil
}

func registerJobs(s gocron.Scheduler) error {

	// Student data sync job
	_, err := s.NewJob(
		gocron.CronJob(
			"0 7-17/2 * * 1-5", // every 2 hours from 7am to 5pm on weekdays
			false,
		),
		gocron.NewTask(
			func() {
				var log *zap.SugaredLogger
				container.MustResolve(container.Global, &log)
				log.Infof("Starting ERP data sync... TIME: %s", time.Now().Format(time.RFC3339))
				sync := synchroniser.StudentSync{}
				container.MustFill(container.Global, &sync)
				if err := sync.SyncAllStudents(); err != nil {
					log.Errorf("Error during ERP data sync: %v\n", err)
				}
			},
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
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
		gocron.NewTask(
			func() {
				var log *zap.SugaredLogger
				container.MustResolve(container.Global, &log)
				log.Infof("Starting registration codes data sync... TIME: %s", time.Now().Format(time.RFC3339))
				sync := synchroniser.StudentSync{}
				container.MustFill(container.Global, &sync)
				if err := sync.SyncRegistrationCodesDictionaries(); err != nil {
					log.Errorf("Error during registration codes data sync: %v\n", err)
				}
			},
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		return err
	}

	// Registration codes sync job
	_, err = s.NewJob(
		gocron.CronJob(
			"0 5 * * 1-5", // at 5am on weekdays
			false,
		),
		gocron.NewTask(
			func() {
				var log *zap.SugaredLogger
				container.MustResolve(container.Global, &log)
				log.Infof("Starting student photos data sync... TIME: %s", time.Now().Format(time.RFC3339))
				sync := synchroniser.StudentSync{}
				container.MustFill(container.Global, &sync)
				if err := sync.SyncStudentPhotos(); err != nil {
					log.Errorf("Error during student photos data sync: %v\n", err)
				}
			},
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	)
	if err != nil {
		return err
	}

	// You can register more cron jobs here

	return nil
}
