package tracker

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/buzyka/imlate/internal/infrastructure/util"
	"github.com/buzyka/imlate/internal/isb/entity"
	"github.com/buzyka/imlate/internal/isb/registration"
	"github.com/gin-gonic/gin"
)
type timeStruct struct {
	hour int
	minute int
}

var tmpTimeStr *timeStruct = nil

type Request struct {
	VisitorID int32 `json:"visitor_id"`
	VisitKey  string `json:"visit_key"`
	SignedIn  bool `json:"signed_in"`
}

type TrackerController struct {
	VisitorRepository entity.VisitorRepository `container:"type"`
	TrackRepository entity.VisitorTrackRepository `container:"type"`
	Registrator registration.Registrator `container:"type"`
}

type TrackResponse struct {
	Visitor *entity.Visitor `json:"visitor"`
	TrackType string `json:"track_type"`
	TrackDate string `json:"track_date"`
}

func (tc *TrackerController) ChangeTimeHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var Request struct {
			TimeStr string `json:"time"`
		}
		err := ctx.Bind(&Request)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		re := regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
		if !re.MatchString(Request.TimeStr) {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid time format",
			})
			return
		}
		var hour, minute int
		_, err = fmt.Sscanf(Request.TimeStr, "%d:%d", &hour, &minute)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		tmpTimeStr = &timeStruct{
			hour: hour,
			minute: minute,
		}
		ctx.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Changed time to: %d:%d", tmpTimeStr.hour, tmpTimeStr.minute),
		})
	}
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
			VisitKey: Request.VisitKey,
			SignedIn: Request.SignedIn,
		}
		tc.findAndTrack(ctx, track)

		// track := &entity.VisitTrack{
		// 	VisitorId: Request.VisitorID,
		// 	VisitKey:  Request.VisitKey,
		// 	SignedIn:  Request.SignedIn,
		// }
		// track.Visitor, err = tc.VisitorRepository.FindById(Request.VisitorID)
		// if err != nil {
		// 	ctx.JSON(http.StatusNotFound, gin.H{
		// 		"error": "Visitor not exists",
		// 	})
		// 	return
		// }
		// track, err = tc.TrackRepository.Store(track)
		// if err != nil {
		// 	ctx.JSON(http.StatusInternalServerError, gin.H{
		// 		"error": err.Error(),
		// 	})
		// 	return
		// }
		// ctx.JSON(http.StatusOK, gin.H{
		// 	"message": "tracked",
		// 	"id": Request.VisitorID,
		// 	"vk": Request.VisitKey,
		// 	"tr": Request.SignedIn,
		// 	"cr": track.CreatedAt.Format("2006-01-02 15:04:05"),	
		// })
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
			VisitKey: Request.VisitKey,
			SignedIn: Request.SignedIn,
		}
		tc.findAndTrack(ctx, track)		
	}
}

func (tc *TrackerController) findAndTrack(ctx *gin.Context, track *entity.VisitTrack) {	
		visitDetails, err := tc.VisitorRepository.FindByKey(track.VisitKey)
		if err != nil || visitDetails.Visitor == nil {
			ctx.JSON(http.StatusNotFound, gin.H{
				"error": "Visitor not exists",
			})
			return
		}
		track.Visitor = visitDetails.Visitor
		fmt.Printf("Tracking visitor: %d, key: %s, isams_school_id: %d\n", track.Visitor.Id, track.VisitKey, track.Visitor.IsamsSchoolId)
		track.VisitorId = visitDetails.Visitor.Id
		track, err = tc.TrackRepository.Store(track)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		hours := 0
		minutes := 0
		if tmpTimeStr != nil {
			hours = tmpTimeStr.hour
			minutes = tmpTimeStr.minute
		}
		now := time.Date(track.CreatedAt.Year(), track.CreatedAt.Month(), track.CreatedAt.Day(), hours, minutes, 0, 0, time.Local)

		eType := "sign-in"
		startDate := time.Date(track.CreatedAt.Year(), track.CreatedAt.Month(), track.CreatedAt.Day(), 0, 0, 0, 0, time.Local)
		eCount, err := tc.TrackRepository.CountEventsByVisitorIdSince(track.VisitorId, startDate)
		if err == nil {
			if eCount % 2 == 0 {
				eType = "sign-out"
			}
		}

		t := tc.calcDiffInMinutes(now)
		fmt.Printf("-------Calculated minutes late: %d\n", t)
		mis := &MIS{
			Registrator: tc.Registrator,
		}
		_ = mis.Register(track.Visitor)
		// _ = tc.Registrator.Register(*track.Visitor, t)

		response := TrackResponse{
			Visitor: track.Visitor,
			TrackType: eType,
			TrackDate: track.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		ctx.JSON(http.StatusOK, response)
} 

func (tc *TrackerController) calcDiffInMinutes(now time.Time) int32 {
	today0800 := time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, util.GetLocation())

	diff := now.Sub(today0800)
	return int32(diff.Minutes())
}


type FakeTrackerService struct {}

func (fts FakeTrackerService) Track(id string) error {
	return nil
}
