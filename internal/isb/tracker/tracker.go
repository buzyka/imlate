package tracker

import (
	"net/http"

	"github.com/buzyka/imlate/internal/isb/entity"
	"github.com/gin-gonic/gin"
)



type Request struct {
	VisitorID string `json:"visitor_id"`
	SignedIn bool `json:"signed_in"`
}

type TrackerController struct {
	VisitorRepository entity.VisitorRepository `container:"type"`
	TrackRepository entity.VisitorTrackRepository `container:"type"`
}

func (tc *TrackerController) TrackHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var Request Request
		err := ctx.Bind(&Request);
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		track := &entity.VisitTrack{
			VisitorId: Request.VisitorID,
			SignedIn: Request.SignedIn,
		}
		track.Visitor, err = tc.VisitorRepository.FindById(Request.VisitorID)
		if err != nil {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Visitor not exists",
			})
			return
		}
		track, err = tc.TrackRepository.Store(track)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"message": "tracked",
			"id": Request.VisitorID,
			"tr": Request.SignedIn,
			"cr": track.CreatedAt.Format("2006-01-02 15:04:05"),	
		})
	}
}

type FakeTrackerService struct {}

func (fts FakeTrackerService) Track(id string) error {
	return nil
}