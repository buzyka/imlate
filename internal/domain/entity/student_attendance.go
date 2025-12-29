package entity

import "time"

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


func NewStudentAttendance(student *Visitor, schedule *Schedule) *StudentAttendance {
	return &StudentAttendance{
		student: student,
		schedule: schedule,
		attendance: make(AttendanceList),
	}
}

type StudentAttendance struct {
	student *Visitor
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
}


