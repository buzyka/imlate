package tracker

import (
	"fmt"
	"net/http"
	"time"

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

type TrackResponse struct {
	Visitor *entity.Visitor `json:"visitor"`
	TrackType string `json:"track_type"`
	TrackDate string `json:"track_date"`
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

func (tc *TrackerController) FindAndTrackHandler() gin.HandlerFunc {
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
		if err != nil || track.Visitor.Id == "" {
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

		eType := "sign-in"
		startDate := time.Date(track.CreatedAt.Year(), track.CreatedAt.Month(), track.CreatedAt.Day(), 0, 0, 0, 0, time.Local)
		eCount, err := tc.TrackRepository.CountEventsByVisitorIdSince(Request.VisitorID, startDate)
		if err == nil {
			if eCount % 2 == 0 {
				eType = "sign-out"
			}
		}

		response := TrackResponse{
			Visitor: track.Visitor,
			TrackType: eType,
			TrackDate: track.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		ctx.JSON(http.StatusOK, response)
	}
}

type FakeTrackerService struct {}

func (fts FakeTrackerService) Track(id string) error {
	return nil
}