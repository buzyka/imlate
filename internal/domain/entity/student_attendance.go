package entity

import (
	"errors"
	"fmt"
	"time"

	"github.com/buzyka/imlate/internal/config"
)

var (
	ErrorMainPeriodNotFound = errors.New("main registration period not found")
	ErrorStudentSchedulePeriodNotFound = errors.New("student schedule not found")
	ErrorDefaultPresentCodeNotFound = errors.New("default present code not found")
)

type RegistrationPeriodID int32

type AttendanceItem struct {
	AbsenceCodeID          *int32 
	AlertSent              bool
	IsFutureAbsence        bool
	IsLate                 bool
	IsOutOfSchool          bool
	IsPresent              bool
	IsRegistered           int32
	LeavingOrLeftDateTime  *time.Time
	NumberOfMinutesLate    int32
	ParentNotificationSent bool
	PresentCodeID          *int32
	RegistrationComment    *string
	RegistrationPeriodID   RegistrationPeriodID
	SchoolID               string
}

type AttendanceList map[RegistrationPeriodID]*AttendanceItem

type StudentAttendanceItem struct {
	Period   *RegistrationPeriod
	Attendance *AttendanceItem
}
type StudentAttendanceItems map[RegistrationPeriodID]*StudentAttendanceItem


func NewStudentAttendance(student *Visitor, schedule *Schedule) *StudentAttendance {
	items := make(StudentAttendanceItems)
	for _, period := range schedule.Periods {
		items[RegistrationPeriodID(period.ID)] = &StudentAttendanceItem{
			Period: period,
		}
	}	
	return &StudentAttendance{
		student: student,
		studentSchedule: &items,
		schedule: schedule,
		attendance: make(AttendanceList),
	}
}

type StudentAttendance struct {
	student *Visitor
	studentSchedule *StudentAttendanceItems
	schedule *Schedule
	attendance AttendanceList
}

func (sa *StudentAttendance) Schedule() *Schedule {
	return sa.schedule
}

func (sa *StudentAttendance) Student() *Visitor {
	return sa.student
}

func (sa *StudentAttendance) SetAttendanceStatus(item *AttendanceItem) {
	sa.attendance[item.RegistrationPeriodID] = item
	studentAttendanceItem, ok := (*sa.studentSchedule)[item.RegistrationPeriodID]
	if ok {
		studentAttendanceItem.Attendance = item
	}
}

func (sa *StudentAttendance) TrackInMainRegistration(trackTime time.Time) (*StudentAttendanceItem, bool, error) {
	mainReg, ok := sa.Schedule().GetPeriodByName(config.ERPFirstRegistrationPeriodName())
	if !ok {
		return nil, false, fmt.Errorf("%w: expected default period %s", ErrorMainPeriodNotFound, config.ERPFirstRegistrationPeriodName())
	}

	schedule, ok := (*sa.studentSchedule)[RegistrationPeriodID(mainReg.ID)]
	if !ok {
		return nil, false, fmt.Errorf("%w: expected default period %s", ErrorStudentSchedulePeriodNotFound, config.ERPFirstRegistrationPeriodName())
	}

	defaultPresentCode, ok := GetDefaultPresentCode()
	if !ok {
		return nil, false, ErrorDefaultPresentCodeNotFound
	}

	// Student is not yet registered
	if schedule.Attendance.IsRegistered == 0 {
		if trackTime.Before(mainReg.Finish) || trackTime.Equal(mainReg.Finish) {
			schedule.Attendance.IsRegistered = 1
			schedule.Attendance.IsPresent = true
			schedule.Attendance.IsLate = false
			schedule.Attendance.PresentCodeID = &defaultPresentCode.ID

			return schedule, true, nil
		} else {
			schedule.Attendance.IsRegistered = 1
			schedule.Attendance.IsPresent = true
			schedule.Attendance.IsLate = true
			schedule.Attendance.NumberOfMinutesLate = int32(trackTime.Sub(mainReg.Time).Minutes())
			schedule.Attendance.PresentCodeID = nil
			schedule.Attendance.AbsenceCodeID = nil

			return schedule, true, nil
		}
	}

	// Student is already registered
	if schedule.Attendance.IsRegistered > 0 {
		// Student is marked as absent - update to present and mark a late time
		if !schedule.Attendance.IsPresent {
			schedule.Attendance.IsRegistered = 1
			schedule.Attendance.IsPresent = true
			schedule.Attendance.IsLate = true
			schedule.Attendance.NumberOfMinutesLate = int32(trackTime.Sub(mainReg.Time).Minutes())			
			schedule.Attendance.PresentCodeID = nil
			schedule.Attendance.AbsenceCodeID = nil

			return schedule, true, nil
		}
	}

	return nil, false, nil
}

func (sa *StudentAttendance) TrackForbyPeriodsForPresent(trackTime time.Time) (updatedItems []*StudentAttendanceItem, updateRequired bool) {
	updateRequired = false
	mainReg, ok := sa.Schedule().GetPeriodByName(config.ERPFirstRegistrationPeriodName())
	if !ok {
		return nil, false
	}

	defaultLessonAbsenceCode, ok := GetDefaultLessonAbsenceCode()
	if !ok {
		return nil, false
	}

	for key, scheduleItem := range *sa.studentSchedule {
		if key == RegistrationPeriodID(mainReg.ID) {
			continue
		}

		//period forby student marked like absent
		if trackTime.After(scheduleItem.Period.Finish) {
			if scheduleItem.Attendance.IsRegistered == 0 {
				fmt.Println("--- Forby Period - Mark Absent")
				scheduleItem.Attendance.IsRegistered = 1
				scheduleItem.Attendance.IsPresent = false
				scheduleItem.Attendance.IsLate = false
				scheduleItem.Attendance.PresentCodeID = nil
				scheduleItem.Attendance.AbsenceCodeID = &defaultLessonAbsenceCode.ID
			}
			updatedItems = append(updatedItems, scheduleItem)

			updateRequired = true
			continue
		}

		// late for the period
		if trackTime.After(scheduleItem.Period.Time) && trackTime.Before(scheduleItem.Period.Finish) {
			if scheduleItem.Attendance.IsRegistered == 0 && scheduleItem.Attendance.IsPresent == false {
				fmt.Println("--- Forby Period - Late Registration")
				scheduleItem.Attendance.IsRegistered = 1
				scheduleItem.Attendance.IsPresent = true
				scheduleItem.Attendance.IsLate = true
				scheduleItem.Attendance.NumberOfMinutesLate = int32(trackTime.Sub(scheduleItem.Period.Start).Minutes())			
				scheduleItem.Attendance.PresentCodeID = nil
				scheduleItem.Attendance.AbsenceCodeID = nil
				updatedItems = append(updatedItems, scheduleItem)

				updateRequired = true
				continue
			}
		}		
	}
	return updatedItems, updateRequired
}




