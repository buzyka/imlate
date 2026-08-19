package main

import (
	"net"
	"net/http"
	"syscall"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/provider/providertest"
	"github.com/buzyka/imlate/internal/usecase/reportaggregator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// On SIGTERM the HTTP server must be shut down, the cron stopped, and the
// aggregator drained. Before graceful shutdown existed the process died
// immediately and everything still queued in the aggregator was lost on every
// deploy.
func TestWaitForShutdown_StopsEverythingAndDrainsAggregator(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &http.Server{Handler: http.NewServeMux()}
	go func() { _ = srv.Serve(listener) }()

	cronStopped := false
	stopCron := func() { cronStopped = true }

	repo := new(providertest.VisitDailyReportRepositoryMock)
	day := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	repo.On("EnsureDayRows", mock.Anything).Return(nil)
	repo.On("RecalculateVisitorDay", int32(42), day).Return(nil)

	// A long flush interval guarantees the ticker never fires: the pending job
	// can only reach the repository through the shutdown drain.
	aggregator := &reportaggregator.Aggregator{Repo: repo, FlushInterval: time.Hour}
	aggregator.Start()
	aggregator.Enqueue(42, day)

	done := make(chan struct{})
	go func() {
		waitForShutdown(srv, stopCron, aggregator)
		close(done)
	}()

	// Give waitForShutdown a moment to install its signal handler.
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGTERM))

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("waitForShutdown did not return after SIGTERM")
	}

	assert.True(t, cronStopped, "cron should be stopped")
	repo.AssertExpectations(t) // the queued job was drained, not dropped

	// The server no longer accepts connections.
	_, err = http.Get("http://" + listener.Addr().String())
	assert.Error(t, err)
}
