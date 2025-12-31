package entity

import (
	"time"

	"github.com/buzyka/imlate/internal/config"
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

func (sa *StudentAttendance) TrackInMainRegistration(trackTime time.Time) (*StudentAttendanceItem, bool) {
	mainReg, ok := sa.Schedule().GetPeriodByName(config.ERPFirstRegistrationPeriodName())
	if !ok {
		return nil, false
	}

	schedule, ok := (*sa.studentSchedule)[RegistrationPeriodID(mainReg.ID)]
	if !ok {
		return nil, false
	}

	// attendanceItem, ok := sa.attendance[RegistrationPeriodID(mainReg.ID)]
	// if !ok {
	// 	return nil, false
	// }

	defaultPresentCode, ok := GetDefaultPresentCode()
	if !ok {
		return nil, false
	}

	// Student is not yet registered
	if schedule.Attendance.IsRegistered == 0 {
		if  trackTime.Before(mainReg.Finish) {
			schedule.Attendance.IsRegistered = 1
			schedule.Attendance.IsPresent = true
			schedule.Attendance.IsLate = false
			schedule.Attendance.PresentCodeID = &defaultPresentCode.ID

			return schedule, true
		} else {
			schedule.Attendance.IsRegistered = 1
			schedule.Attendance.IsPresent = true
			schedule.Attendance.IsLate = true
			schedule.Attendance.NumberOfMinutesLate = int32(trackTime.Sub(mainReg.Start).Minutes())			
			schedule.Attendance.PresentCodeID = nil
			schedule.Attendance.AbsenceCodeID = nil

			return schedule, true
		}				
	}

	// Student is already registered
	if schedule.Attendance.IsRegistered > 0 {
		// Student is marked as absent - update to present and mark a late time
		if !schedule.Attendance.IsPresent {
			schedule.Attendance.IsRegistered = 1
			schedule.Attendance.IsPresent = true
			schedule.Attendance.IsLate = true
			schedule.Attendance.NumberOfMinutesLate = int32(trackTime.Sub(mainReg.Start).Minutes())			
			schedule.Attendance.PresentCodeID = nil
			schedule.Attendance.AbsenceCodeID = nil

			return schedule, true
		}
	}

	return nil, false
}




