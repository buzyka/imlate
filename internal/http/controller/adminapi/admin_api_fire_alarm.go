package adminapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/gin-gonic/gin"
)

// SetFireAlarmRequest is the body of POST /admin-api/fire-alarm.
//
// Both fields are pointers on purpose. binding:"required" on a plain bool
// rejects false, which would make it impossible to switch the alarm off; on a
// *bool the same tag means "the field was sent", so false passes. And
// DefaultDuration cannot be unconditionally required, because switching the
// alarm off does not need one — it is validated below, only when enabling.
type SetFireAlarmRequest struct {
	Enabled         *bool `json:"enabled" binding:"required"`
	DefaultDuration *int  `json:"default_duration"`
}

// FireAlarmResponse is the body of both fire alarm endpoints.
type FireAlarmResponse struct {
	Enabled          bool       `json:"enabled"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	DefaultDuration  int        `json:"default_duration"`
	RemainingSeconds int        `json:"remaining_seconds"`
}

// fireAlarmResponse renders a snapshot. An inactive alarm reports zeroes and
// omits started_at rather than echoing stale timing back to the caller.
func fireAlarmResponse(snapshot entity.FireAlarmSnapshot) FireAlarmResponse {
	if !snapshot.Enabled {
		return FireAlarmResponse{}
	}
	startedAt := snapshot.StartedAt
	return FireAlarmResponse{
		Enabled:          true,
		StartedAt:        &startedAt,
		DefaultDuration:  int(snapshot.Duration.Seconds()),
		RemainingSeconds: int(snapshot.Remaining.Seconds()),
	}
}

// GetFireAlarmHandler godoc
// @Summary      Get fire alarm status
// @Description  Returns whether a fire alarm is running, when it started, how long it lasts and
// @Description  how much of that window is left. Expiration is evaluated before the status is
// @Description  returned, so an elapsed alarm is never reported as active.
// @Tags         admin-fire-alarm
// @Produce      json
// @Success      200  {object}  FireAlarmResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/fire-alarm [get]
func (ac *AdminAPIController) GetFireAlarmHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, fireAlarmResponse(ac.FireAlarm.Snapshot()))
	}
}

// SetFireAlarmHandler godoc
// @Summary      Set fire alarm status
// @Description  Switches the fire alarm on for a bounded window or off immediately. While it is
// @Description  on, the public /firelist/{grade} page serves evacuation rosters; otherwise that
// @Description  page returns 403. default_duration is in seconds, is mandatory when enabling,
// @Description  and may not exceed 24 hours.
// @Tags         admin-fire-alarm
// @Accept       json
// @Produce      json
// @Param        request  body      SetFireAlarmRequest  true  "Fire alarm request"
// @Success      200      {object}  FireAlarmResponse
// @Failure      400      {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/fire-alarm [post]
func (ac *AdminAPIController) SetFireAlarmHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SetFireAlarmRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if !*req.Enabled {
			ac.FireAlarm.Disable()
			ac.logFireAlarmChange(c, "disabled", 0)
			c.JSON(http.StatusOK, fireAlarmResponse(ac.FireAlarm.Snapshot()))
			return
		}

		if req.DefaultDuration == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": entity.ErrDurationRequired.Error()})
			return
		}

		duration := time.Duration(*req.DefaultDuration) * time.Second
		if err := ac.FireAlarm.Enable(duration); err != nil {
			if errors.Is(err, entity.ErrDurationRequired) || errors.Is(err, entity.ErrDurationTooLong) {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ac.logFireAlarmChange(c, "enabled", *req.DefaultDuration)
		c.JSON(http.StatusOK, fireAlarmResponse(ac.FireAlarm.Snapshot()))
	}
}

// logFireAlarmChange records who opened or closed public access to the
// rosters. Switching this on exposes the names of every child in the school
// for the duration, so the change needs to be attributable after the fact.
func (ac *AdminAPIController) logFireAlarmChange(c *gin.Context, action string, durationSeconds int) {
	if ac.Logger == nil {
		return
	}
	// Read the identity directly instead of via currentAdminUser: that helper
	// writes a 401/500 body when the context is missing a user, which on this
	// path would append a second payload to an already-written 200.
	username := "unknown"
	if data, exists := c.Get("id"); exists {
		if admin, ok := data.(*entity.User); ok {
			username = admin.UserName
		}
	}
	ac.Logger.Infow("fire alarm "+action,
		"admin", username,
		"duration_seconds", durationSeconds,
	)
}
