package adminapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupFireAlarmController() (*entity.FireAlarmState, *AdminAPIController) {
	gin.SetMode(gin.TestMode)
	alarm := &entity.FireAlarmState{}
	return alarm, &AdminAPIController{FireAlarm: alarm, Logger: zap.NewNop().Sugar()}
}

func postFireAlarm(controller *AdminAPIController, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/fire-alarm", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	controller.SetFireAlarmHandler()(c)
	return w
}

func getFireAlarm(controller *AdminAPIController) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin-api/fire-alarm", nil)

	controller.GetFireAlarmHandler()(c)
	return w
}

func decodeFireAlarm(t *testing.T, w *httptest.ResponseRecorder) FireAlarmResponse {
	t.Helper()
	var resp FireAlarmResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func TestSetFireAlarmHandler_Enables(t *testing.T) {
	alarm, controller := setupFireAlarmController()

	w := postFireAlarm(controller, `{"enabled":true,"default_duration":3600}`)

	require.Equal(t, http.StatusOK, w.Code)
	resp := decodeFireAlarm(t, w)
	assert.True(t, resp.Enabled)
	assert.Equal(t, 3600, resp.DefaultDuration)
	assert.Positive(t, resp.RemainingSeconds)
	require.NotNil(t, resp.StartedAt)
	assert.True(t, alarm.IsActive(), "the shared state, not just the response, must be on")
}

// Regression guard for the *bool in SetFireAlarmRequest: with a plain bool,
// binding:"required" rejects false and the alarm could never be switched off.
func TestSetFireAlarmHandler_DisablesWithFalse(t *testing.T) {
	alarm, controller := setupFireAlarmController()
	require.NoError(t, alarm.Enable(time.Hour))

	w := postFireAlarm(controller, `{"enabled":false}`)

	require.Equal(t, http.StatusOK, w.Code)
	resp := decodeFireAlarm(t, w)
	assert.False(t, resp.Enabled)
	assert.Nil(t, resp.StartedAt)
	assert.Equal(t, 0, resp.RemainingSeconds)
	assert.False(t, alarm.IsActive())
}

func TestSetFireAlarmHandler_Validation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "enabled without a duration", body: `{"enabled":true}`},
		{name: "zero duration", body: `{"enabled":true,"default_duration":0}`},
		{name: "negative duration", body: `{"enabled":true,"default_duration":-60}`},
		{name: "one second over 24 hours", body: `{"enabled":true,"default_duration":86401}`},
		{name: "enabled field missing entirely", body: `{"default_duration":3600}`},
		{name: "malformed json", body: `{invalid`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alarm, controller := setupFireAlarmController()

			w := postFireAlarm(controller, tt.body)

			require.Equal(t, http.StatusBadRequest, w.Code)
			var errResp map[string]string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
			assert.NotEmpty(t, errResp["error"])
			assert.False(t, alarm.IsActive(), "a rejected request must not open access")
		})
	}
}

// Exactly 24 hours is the documented maximum and has to be accepted.
func TestSetFireAlarmHandler_AcceptsTheMaximumDuration(t *testing.T) {
	_, controller := setupFireAlarmController()

	w := postFireAlarm(controller, `{"enabled":true,"default_duration":86400}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, decodeFireAlarm(t, w).Enabled)
}

func TestGetFireAlarmHandler_ReportsInactiveByDefault(t *testing.T) {
	_, controller := setupFireAlarmController()

	w := getFireAlarm(controller)

	require.Equal(t, http.StatusOK, w.Code)
	resp := decodeFireAlarm(t, w)
	assert.False(t, resp.Enabled)
	assert.Nil(t, resp.StartedAt)
	assert.Equal(t, 0, resp.DefaultDuration)
	assert.Equal(t, 0, resp.RemainingSeconds)
}

func TestGetFireAlarmHandler_ReportsActive(t *testing.T) {
	alarm, controller := setupFireAlarmController()
	require.NoError(t, alarm.Enable(time.Hour))

	resp := decodeFireAlarm(t, getFireAlarm(controller))

	assert.True(t, resp.Enabled)
	assert.Equal(t, 3600, resp.DefaultDuration)
	assert.Positive(t, resp.RemainingSeconds)
}

// The GET must evaluate expiry itself, not report a stale "on".
func TestGetFireAlarmHandler_ReportsExpiredAsInactive(t *testing.T) {
	_, controller := setupFireAlarmController()
	require.NoError(t, controller.FireAlarm.Enable(time.Millisecond))

	time.Sleep(5 * time.Millisecond)

	resp := decodeFireAlarm(t, getFireAlarm(controller))
	assert.False(t, resp.Enabled)
}

// The controller is filled by the DI container, but a nil logger must never
// turn switching the alarm into a panic.
func TestSetFireAlarmHandler_NilLoggerDoesNotPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &AdminAPIController{FireAlarm: &entity.FireAlarmState{}}

	assert.NotPanics(t, func() {
		w := postFireAlarm(controller, `{"enabled":true,"default_duration":60}`)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// The audit line names the admin who opened access; reading the identity must
// not write a second body onto the already-sent 200.
func TestSetFireAlarmHandler_LogsAdminIdentityWithoutCorruptingTheResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	alarm := &entity.FireAlarmState{}
	controller := &AdminAPIController{FireAlarm: alarm, Logger: zap.NewNop().Sugar()}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/fire-alarm",
		bytes.NewBufferString(`{"enabled":true,"default_duration":60}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", &entity.User{UserName: "headteacher"})

	controller.SetFireAlarmHandler()(c)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, decodeFireAlarm(t, w).Enabled)
}

// A context holding something that is not a *entity.User must be tolerated:
// the log falls back to "unknown" rather than panicking or 500-ing.
func TestSetFireAlarmHandler_ToleratesUnexpectedIdentityType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &AdminAPIController{FireAlarm: &entity.FireAlarmState{}, Logger: zap.NewNop().Sugar()}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin-api/fire-alarm",
		bytes.NewBufferString(`{"enabled":false}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", "not-a-user-struct")

	assert.NotPanics(t, func() { controller.SetFireAlarmHandler()(c) })
	assert.Equal(t, http.StatusOK, w.Code)
}
