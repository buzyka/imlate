package adminapi

import (
	"net/http"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/gin-gonic/gin"
)

// ManualTrackRequest defines the payload for manual visitor tracking.
type ManualTrackRequest struct {
	VisitorID   int32   `json:"visitor_id" binding:"required"`
	SignedIn    bool    `json:"signed_in"`
	Description *string `json:"description"`
}

// ManualTrackResponse defines the response payload for manual visitor tracking.
type ManualTrackResponse struct {
	Visitor   *entity.Visitor `json:"visitor"`
	TrackType string          `json:"track_type"`
	TrackDate string          `json:"track_date"`
}

// ManualTrackHandler godoc
// @Summary      Manual visitor track
// @Description  Manually records a visitor sign-in/sign-out event on behalf of a visitor by an admin.
// @Description  Used when a visitor forgot their key, lost it, or is otherwise unable to use it.
// @Tags         admin-tracking
// @Accept       json
// @Produce      json
// @Param        request  body      ManualTrackRequest  true  "Track request"
// @Success      200      {object}  ManualTrackResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Failure      500      {object}  ErrorResponse
// @Security     ApiKeyAuth
// @Router       /admin-api/track/visit [post]
func (ac *AdminAPIController) ManualTrackHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		adminUser, ok := currentAdminUser(c)
		if !ok {
			return
		}

		req := ManualTrackRequest{SignedIn: true}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		visitor, err := ac.AdminAPI.VisitorRepo.FindById(req.VisitorID)
		if err != nil || visitor == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "visitor not found"})
			return
		}

		track := &entity.VisitTrack{
			VisitorId:   visitor.Id,
			SignedIn:    req.SignedIn,
			Visitor:     visitor,
			AdminID:     &adminUser.ID,
			Description: req.Description,
		}

		track, err = ac.TrackRepo.Store(track)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ac.Aggregator.Enqueue(track.VisitorId, track.CreatedAt)

		eType := "sign-in"
		startDate := time.Date(track.CreatedAt.Year(), track.CreatedAt.Month(), track.CreatedAt.Day(), 0, 0, 0, 0, time.Local)
		eCount, err := ac.TrackRepo.CountEventsByVisitorIdSince(track.VisitorId, startDate)
		if err == nil && eCount%2 == 0 {
			eType = "sign-out"
		}

		if track.Visitor.IsStudent {
			if err := ac.StudentTracker.Track(c, track.Visitor); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		c.JSON(http.StatusOK, ManualTrackResponse{
			Visitor:   track.Visitor,
			TrackType: eType,
			TrackDate: track.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
}
