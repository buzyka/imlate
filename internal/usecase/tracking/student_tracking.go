package tracking

import (
	"context"
	"fmt"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/infrastructure/util"
	"go.uber.org/zap"
)

type StudentTracker struct {
	Cfg 				*config.Config 	`container:"type"`
	ERPFactory          erp.Factory     `container:"type"`
	VisitorRepo         provider.VisitorRepository  `container:"type"`
	Logger				*zap.SugaredLogger `container:"type"`
}

func (s *StudentTracker) Track(ctx context.Context, visitor *entity.Visitor) error {
	erpClient, err := s.ERPFactory.NewClient(ctx)
	if err != nil {
		s.Logger.Errorf("StudentTracker: Error creating ERP client for student %d: %v", visitor.Id, err)
		return err
	}

	// Get periods for visitor for today
	schedule, err := s.getPeriods(erpClient, visitor.ErpDivisions)
	if err != nil {
		s.Logger.Errorf("StudentTracker: Error getting periods for student %d: %v", visitor.Id, err)
		return err
	}

	studentAttendance := entity.NewStudentAttendance(visitor, schedule)
	s.fillAttendanceInfo(erpClient, studentAttendance)

	n := util.Now()
	na, shouldUpdate, err := studentAttendance.TrackInMainRegistration(n)
	if err != nil {
		return err
	}
	if shouldUpdate {
		// Update ERP with new attendance info
		err = erpClient.PutRegistration(
			studentAttendance.Student().ErpSchoolID, 
			int32(na.Period.ID),
			s.PrepareRegistrationStatusRequest(na.Attendance),
		)
		if err != nil {
			s.Logger.Errorf("StudentTracker: Error updating registration for student %d, period %d: %v", visitor.Id, na.Period.ID, err)
			return err
		}

		naList, shouldUpdate, err := studentAttendance.TrackForbyPeriodsForPresent(n)
		if err != nil {
			return err
		}
		if shouldUpdate {
			for _, na := range naList {
				// Update ERP with new attendance info
				err = erpClient.PutRegistration(
					studentAttendance.Student().ErpSchoolID, 
					int32(na.Period.ID),
					s.PrepareRegistrationStatusRequest(na.Attendance),
				)
				if err != nil {
					s.Logger.Errorf("StudentTracker: Error updating registration for student %d, period %d: %v", visitor.Id, na.Period.ID, err)
					return err
				}
			}
		}
	}
	
	return nil
}

func (s *StudentTracker) PrepareRegistrationStatusRequest(item *entity.AttendanceItem) isams.RegistrationStatusRequest {
	var leavingDateTime *string
	if item.LeavingOrLeftDateTime != nil {
		leavingDateTimeStr := util.FromLocalTimeToTimeStr(*item.LeavingOrLeftDateTime, s.Cfg.ERPTimeLocation())
		leavingDateTime = &leavingDateTimeStr
	}
	req := isams.RegistrationStatusRequest{
		IsPresent: item.IsPresent,
		IsLate: item.IsLate,
		PresentCodeID: item.PresentCodeID,
		AbsenceCodeID:          item.AbsenceCodeID,
		LeavingOrLeftDateTime:  leavingDateTime,
		NumberOfMinutesLate:    item.NumberOfMinutesLate,
		RegistrationComment:    item.RegistrationComment,
	}
	return req
}

func (s *StudentTracker) getPeriods(erpClient erp.Client, divisions []int32) (*entity.Schedule, error) {
	if erpClient == nil {
		return nil, fmt.Errorf("erp client is nil")
	}
	schedule := &entity.Schedule{}
	for _, division := range divisions {
		if division <= 0 {
			continue
		}
		resp, err := erpClient.GetCurrentRegistrationPeriodsForDivision(division)
		if err != nil {
			s.Logger.Errorf("StudentTracker: Error getting registration periods for division %d: %v", division, err)
			continue
		}
		s.addPeriodsFromResponse(schedule, resp)
	}

	return schedule, nil
}

func (s *StudentTracker) addPeriodsFromResponse(schedule *entity.Schedule, resp *isams.RegistrationPeriodsResponse) {
	for _, period := range resp.RegistrationPeriods {
		startDate, err := util.ParseTimeStrToLocal(period.Start)
		if err != nil {
			continue
		}
		finishDate, err := util.ParseTimeStrToLocal(period.Finish)
		if err != nil {
			continue
		}
		timeDate, err := util.ParseTimeStrToLocal(period.Time)
		if err != nil {
			continue
		}
		period := &entity.RegistrationPeriod{
			ID:        period.ID,
			Name:      period.FriendlyName,
			Type:      period.RegistrationType,
			Time:      timeDate,
			Start: startDate,
			Finish:   finishDate,
		}
		schedule.AddPeriod(period)
	}
}

func (s *StudentTracker) fillAttendanceInfo(erpClient erp.Client, studentAttendance *entity.StudentAttendance) {
	for _, period := range studentAttendance.Schedule().Periods {
		status, err := erpClient.GetRegistrationStatusForStudent(studentAttendance.Student().ErpSchoolID, period.ID)
		if err != nil {
			continue
		}

		var leaveTime *time.Time
		if status.LeavingOrLeftDateTime != nil && *status.LeavingOrLeftDateTime != "" {			
			parsedTime, err := util.ParseTimeStrToLocal(*status.LeavingOrLeftDateTime)
			if err == nil {
				leaveTime = &parsedTime
			}
		}

		studentAttendance.SetAttendanceStatus( &entity.AttendanceItem{
			AbsenceCodeID:          status.AbsenceCodeID,
			AlertSent:              status.AlertSent,
			IsFutureAbsence:        status.IsFutureAbsence,
			IsLate:                 status.IsLate,
			IsOutOfSchool:          status.IsOutOfSchool,
			IsPresent:              status.IsPresent,
			IsRegistered:           status.IsRegistered,
			LeavingOrLeftDateTime:  leaveTime,
			NumberOfMinutesLate:    status.NumberOfMinutesLate,
			ParentNotificationSent: status.ParentNotificationSent,
			PresentCodeID:          status.PresentCodeID,
			RegistrationComment:    status.RegistrationComment,
			RegistrationPeriodID:   entity.RegistrationPeriodID(period.ID),
			SchoolID:               status.SchoolID,
		})
	}
}
