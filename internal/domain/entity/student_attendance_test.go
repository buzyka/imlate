package entity

import (
	"testing"
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/golobby/container/v3"
	"github.com/stretchr/testify/assert"
)

type saEnv struct {
	student  *Visitor
	schedule *Schedule
}

func TestNewStudentAttendance(t *testing.T) {
	_, saEnv := prepareTestEnv()
	sa := NewStudentAttendance(saEnv.student, saEnv.schedule)

	assert.Equal(t, saEnv.student, sa.Student())
	assert.Equal(t, saEnv.schedule, sa.Schedule())
	assert.Equal(t, 3, len(*sa.studentSchedule))
	assert.Equal(t, 0, len(sa.attendance))
	for key, period := range saEnv.schedule.Periods {
		item, exists := (*sa.studentSchedule)[RegistrationPeriodID(key)]
		assert.True(t, exists)
		assert.Equal(t, period, item.Period)
		assert.Nil(t, item.Attendance)
	}

	ai := &AttendanceItem{
		RegistrationPeriodID: RegistrationPeriodID(100),
		SchoolID:             saEnv.student.ErpSchoolID,
		IsFutureAbsence:      false,
		IsPresent:            false,
		IsLate:               false,
		IsOutOfSchool:        false,
		IsRegistered:         0,
	}
	sa.SetAttendanceStatus(ai)

	assert.Equal(t, 1, len(sa.attendance))
	savedItem, exists := sa.attendance[RegistrationPeriodID(100)]
	assert.True(t, exists)
	assert.Equal(t, ai, savedItem)
}

func TestSetAttendanceStatusWillUpdateExisting(t *testing.T) {
	sa, _ := prepareTestEnv()

	item, ok := (*sa.studentSchedule)[RegistrationPeriodID(1)]
	assert.True(t, ok)
	assert.NotNil(t, item.Attendance)
	assert.False(t, item.Attendance.IsPresent)
	assert.Equal(t, item.Attendance.IsRegistered, int32(0))

	ai := &AttendanceItem{
		RegistrationPeriodID: RegistrationPeriodID(1),
		SchoolID:             sa.student.ErpSchoolID,
		IsFutureAbsence:      true,
		IsPresent:            true,
		IsLate:               false,
		IsOutOfSchool:        false,
		IsRegistered:         1,
	}
	sa.SetAttendanceStatus(ai)

	item, ok = (*sa.studentSchedule)[RegistrationPeriodID(1)]
	assert.True(t, ok)
	assert.NotNil(t, item.Attendance)
	assert.True(t, item.Attendance.IsPresent)
	assert.Equal(t, item.Attendance.IsRegistered, int32(1))
}

func TestTrackInMainRegistrationWillUpdateAttendanceItem(t *testing.T) {
	now := time.Now()
	presentCode := int32(1)
	absenceCode := int32(11)
	var tests = []struct {
		name                  string
		trackTime             time.Time
		attendance            *AttendanceItem
		exIsLate              bool
		exIsPresent           bool
		exPresentCode         *int32
		exAbsenceCode         *int32
		exNumberOfMinutesLate int32
	}{
		{
			name:      "Before AM period",
			trackTime: time.Date(now.Year(), now.Month(), now.Day(), 7, 35, 0, 0, time.UTC),
			attendance: &AttendanceItem{
				RegistrationPeriodID: RegistrationPeriodID(100),
				SchoolID:             "S123",
				IsFutureAbsence:      false,
				IsPresent:            false,
				IsLate:               false,
				IsOutOfSchool:        false,
				IsRegistered:         0,
			},
			exIsLate:      false,
			exIsPresent:   true,
			exPresentCode: &presentCode,
		},
		{
			name:      "During AM period",
			trackTime: time.Date(now.Year(), now.Month(), now.Day(), 7, 45, 0, 0, time.UTC),
			attendance: &AttendanceItem{
				RegistrationPeriodID: RegistrationPeriodID(100),
				SchoolID:             "S123",
				IsFutureAbsence:      false,
				IsPresent:            false,
				IsLate:               false,
				IsOutOfSchool:        false,
				IsRegistered:         0,
			},
			exIsLate:      false,
			exIsPresent:   true,
			exPresentCode: &presentCode,
		},
		{
			name:      "Execly in the finish time AM period",
			trackTime: time.Date(now.Year(), now.Month(), now.Day(), 7, 55, 0, 0, time.UTC),
			attendance: &AttendanceItem{
				RegistrationPeriodID: RegistrationPeriodID(100),
				SchoolID:             "S123",
				IsFutureAbsence:      false,
				IsPresent:            false,
				IsLate:               false,
				IsOutOfSchool:        false,
				IsRegistered:         0,
			},
			exIsLate:      false,
			exIsPresent:   true,
			exPresentCode: &presentCode,
		},
		{
			name:      "After AM period",
			trackTime: time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC),
			attendance: &AttendanceItem{
				RegistrationPeriodID: RegistrationPeriodID(100),
				SchoolID:             "S123",
				IsFutureAbsence:      false,
				IsPresent:            false,
				IsLate:               false,
				IsOutOfSchool:        false,
				IsRegistered:         0,
			},
			exIsLate:              true,
			exIsPresent:           true,
			exPresentCode:         nil,
			exNumberOfMinutesLate: 20,
		},
		{
			name:      "After AM period for 1 second",
			trackTime: time.Date(now.Year(), now.Month(), now.Day(), 7, 55, 1, 0, time.UTC),
			attendance: &AttendanceItem{
				RegistrationPeriodID: RegistrationPeriodID(100),
				SchoolID:             "S123",
				IsFutureAbsence:      false,
				IsPresent:            false,
				IsLate:               false,
				IsOutOfSchool:        false,
				IsRegistered:         0,
			},
			exIsLate:              true,
			exIsPresent:           true,
			exPresentCode:         nil,
			exNumberOfMinutesLate: 15,
		},
		{
			name:      "After AM period and already registered as absence",
			trackTime: time.Date(now.Year(), now.Month(), now.Day(), 8, 10, 0, 0, time.UTC),
			attendance: &AttendanceItem{
				RegistrationPeriodID: RegistrationPeriodID(100),
				AbsenceCodeID:        &absenceCode,
				SchoolID:             "S123",
				IsFutureAbsence:      false,
				IsPresent:            false,
				IsLate:               false,
				IsOutOfSchool:        false,
				IsRegistered:         1,
			},
			exIsLate:              true,
			exIsPresent:           true,
			exPresentCode:         nil,
			exAbsenceCode:         nil,
			exNumberOfMinutesLate: 30,
		},
	}

	prepareConfig(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldPCD, oldACD := preparePresentsCodeDictionary(t)
			defer restoreCodesDictionaries(oldPCD, oldACD)

			sa, _ := prepareTestEnv()

			sa.SetAttendanceStatus(tt.attendance)
			a, u, err := sa.TrackInMainRegistration(tt.trackTime)
			assert.NoError(t, err)
			assert.True(t, u)
			assert.NotNil(t, a)

			assert.Equal(t, int32(1), a.Attendance.IsRegistered)
			assert.Equal(t, tt.exIsPresent, a.Attendance.IsPresent)
			assert.Equal(t, tt.exIsLate, a.Attendance.IsLate)
			assert.Equal(t, tt.exNumberOfMinutesLate, a.Attendance.NumberOfMinutesLate)
			if tt.exPresentCode == nil {
				assert.Nil(t, a.Attendance.PresentCodeID)
			} else {
				assert.Equal(t, *tt.exPresentCode, *a.Attendance.PresentCodeID)
			}
		})
	}
}

func TestTrackInMainRegistrationWithAlreadyRegisteredStudentWillNotUpdate(t *testing.T) {
	oldPCD, oldACD := preparePresentsCodeDictionary(t)
	defer restoreCodesDictionaries(oldPCD, oldACD)

	prepareConfig(t)

	presentCode := int32(1)
	sa, _ := prepareTestEnv()

	ai := &AttendanceItem{
		RegistrationPeriodID: RegistrationPeriodID(100),
		PresentCodeID:        &presentCode,
		SchoolID:             "S123",
		IsFutureAbsence:      false,
		IsPresent:            true,
		IsLate:               false,
		IsOutOfSchool:        false,
		IsRegistered:         1,
	}
	sa.SetAttendanceStatus(ai)

	trackTime := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 8, 10, 0, 0, time.UTC)
	a, u, err := sa.TrackInMainRegistration(trackTime)
	assert.NoError(t, err)
	assert.False(t, u)
	assert.Nil(t, a)
}

func TestTrackInMainRegistrationWillReturnError(t *testing.T) {

	t.Run("Period not exists", func(t *testing.T) {
		oldPCD, oldACD := preparePresentsCodeDictionary(t)
		defer restoreCodesDictionaries(oldPCD, oldACD)

		cfg := &config.Config{
			ERPFirstRegistrationPeriodName: "TT",
		}
		err := container.Singleton(func() *config.Config {
			return cfg
		})
		assert.NoError(t, err)

		sa, _ := prepareTestEnv()

		a, u, err := sa.TrackInMainRegistration(time.Now())
		assert.Error(t, err)
		assert.False(t, u)
		assert.Nil(t, a)

		assert.ErrorIs(t, err, ErrorMainPeriodNotFound)
		assert.Contains(t, err.Error(), "expected default period TT")
	})

	t.Run("Student Schedule not exists", func(t *testing.T) {
		oldPCD, oldACD := preparePresentsCodeDictionary(t)
		defer restoreCodesDictionaries(oldPCD, oldACD)

		prepareConfig(t)

		sa, _ := prepareTestEnv()
		// Remove AM period from student schedule
		delete(*sa.studentSchedule, RegistrationPeriodID(100))

		a, u, err := sa.TrackInMainRegistration(time.Now())
		assert.Error(t, err)
		assert.False(t, u)
		assert.Nil(t, a)

		assert.ErrorIs(t, err, ErrorStudentSchedulePeriodNotFound)
		assert.Contains(t, err.Error(), "expected default period AM")
	})

	t.Run("Default Present Code not found", func(t *testing.T) {
		oldPresentsCodeDictionary := GetPresentsCodeDictionary()
		SetPresentsCodeDictionary(nil)
		defer SetPresentsCodeDictionary(oldPresentsCodeDictionary)

		prepareConfig(t)

		sa, _ := prepareTestEnv()

		a, u, err := sa.TrackInMainRegistration(time.Now())
		assert.Error(t, err)
		assert.False(t, u)
		assert.Nil(t, a)

		assert.ErrorIs(t, err, ErrorDefaultPresentCodeNotFound)
	})
}

func TestTrackForbyPeriodsForPresentWillReturnError(t *testing.T) {

	t.Run("Period not exists", func(t *testing.T) {
		oldPCD, oldACD := preparePresentsCodeDictionary(t)
		defer restoreCodesDictionaries(oldPCD, oldACD)

		cfg := &config.Config{
			ERPFirstRegistrationPeriodName: "TT",
		}
		err := container.Singleton(func() *config.Config {
			return cfg
		})
		assert.NoError(t, err)

		sa, _ := prepareTestEnv()

		a, u, err := sa.TrackForbyPeriodsForPresent(time.Now())
		assert.Error(t, err)
		assert.False(t, u)
		assert.Nil(t, a)

		assert.ErrorIs(t, err, ErrorMainPeriodNotFound)
		assert.Contains(t, err.Error(), "expected default period TT")
	})

	t.Run("Default Absence Code not found", func(t *testing.T) {
		oldPresentsCodeDictionary := GetPresentsCodeDictionary()
		SetPresentsCodeDictionary(nil)
		defer SetPresentsCodeDictionary(oldPresentsCodeDictionary)

		prepareConfig(t)

		sa, _ := prepareTestEnv()

		a, u, err := sa.TrackForbyPeriodsForPresent(time.Now())
		assert.Error(t, err)
		assert.False(t, u)
		assert.Nil(t, a)

		assert.ErrorIs(t, err, ErrorDefaultLessonAbsenceCodeNotFound)
	})
}

func TestTrackForbyPeriodsForPresentWithLateForSecondPeriodWillAbsenceOnFirstAndLateOnSecond(t *testing.T) {
	oldPCD, oldACD := preparePresentsCodeDictionary(t)
	defer restoreCodesDictionaries(oldPCD, oldACD)

	prepareConfig(t)

	sa, _ := prepareTestEnv()

	now := time.Now()

	trackTime := time.Date(now.Year(), now.Month(), now.Day(), 9, 10, 0, 0, time.UTC)

	ui, u, err := sa.TrackForbyPeriodsForPresent(trackTime)
	assert.NoError(t, err)
	assert.True(t, u)
	assert.Equal(t, 2, len(ui))

	var item1, item2 *StudentAttendanceItem
	for _, saItem := range ui {
		switch saItem.Period.ID {
		case 1:
			item1 = saItem
		case 2:
			item2 = saItem
		}
	}

	assert.Equal(t, int32(1), item1.Attendance.IsRegistered)
	assert.False(t, item1.Attendance.IsPresent)
	assert.False(t, item1.Attendance.IsLate)
	assert.Nil(t, item1.Attendance.PresentCodeID)
	assert.NotNil(t, item1.Attendance.AbsenceCodeID)

	assert.Equal(t, int32(1), item2.Attendance.IsRegistered)
	assert.True(t, item2.Attendance.IsPresent)
	assert.True(t, item2.Attendance.IsLate)
	assert.Nil(t, item2.Attendance.PresentCodeID)
	assert.Nil(t, item2.Attendance.AbsenceCodeID)
	assert.Equal(t, int32(10), item2.Attendance.NumberOfMinutesLate)
}



func TestTrackForbyPeriodsForPresentWithLateForFirstPeriodWillLateOnFirst(t *testing.T) {

	oldPCD, oldACD := preparePresentsCodeDictionary(t)
	defer restoreCodesDictionaries(oldPCD, oldACD)

	prepareConfig(t)

	sa, _ := prepareTestEnv()

	now := time.Now()

	trackTime := time.Date(now.Year(), now.Month(), now.Day(), 8, 15, 0, 0, time.UTC)

	ui, u, err := sa.TrackForbyPeriodsForPresent(trackTime)
	assert.NoError(t, err)
	assert.True(t, u)
	assert.Equal(t, 1, len(ui))

	item := ui[0]
	assert.Equal(t, int32(1), item.Period.ID)
	assert.Equal(t, int32(1), item.Attendance.IsRegistered)
	assert.True(t, item.Attendance.IsPresent)
	assert.True(t, item.Attendance.IsLate)
	assert.Nil(t, item.Attendance.PresentCodeID)
	assert.Nil(t, item.Attendance.AbsenceCodeID)
	assert.Equal(t, int32(15), item.Attendance.NumberOfMinutesLate)
}

func TestTrackForbyPeriodsForPresentWithOnTimeForSecondPeriodWillLateOnFirst(t *testing.T) {
	var tests = []struct {
		name      string
		trackTime time.Time
	}{
		{
			name:      "On time for second period",
			trackTime: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 9, 0, 0, 0, time.UTC),
		},
		{
			name:      "Exactly in the end time for first period",
			trackTime: time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 8, 59, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldPCD, oldACD := preparePresentsCodeDictionary(t)
			defer restoreCodesDictionaries(oldPCD, oldACD)

			prepareConfig(t)

			sa, _ := prepareTestEnv()

			ui, u, err := sa.TrackForbyPeriodsForPresent(tt.trackTime)
			assert.NoError(t, err)
			assert.True(t, u)
			assert.Equal(t, 1, len(ui))

			item := ui[0]
			assert.Equal(t, int32(1), item.Period.ID)
			assert.Equal(t, int32(1), item.Attendance.IsRegistered)
			assert.False(t, item.Attendance.IsPresent)
			assert.False(t, item.Attendance.IsLate)
			assert.Nil(t, item.Attendance.PresentCodeID)
			assert.NotNil(t, item.Attendance.AbsenceCodeID)
			assert.Equal(t, int32(11), *item.Attendance.AbsenceCodeID)
		})
	}
}

func preparePresentsCodeDictionary(t *testing.T) (oldPresentsCodeDictionary *RegistrationCodeDictionary, oldAbsenceCodeDictionary *RegistrationCodeDictionary) {
	t.Helper()
	oldPresentsCodeDictionary = GetPresentsCodeDictionary()
	presentsCode := &RegistrationCode{
		ID:            1,
		Code:          "/",
		Name:          "Present",
		IsAbsenceCode: false,
	}
	newDictionary := &RegistrationCodeDictionary{
		Codes: map[int32]*RegistrationCode{
			presentsCode.ID: presentsCode,
		},
		UploadedAt: time.Now(),
	}
	SetPresentsCodeDictionary(newDictionary)

	oldAbsenceCodeDictionary = GetAbsenceCodeDictionary()
	absenceCode := &RegistrationCode{
		ID:            11,
		Code:          "C",
		Name:          "Lesson absence",
		IsAbsenceCode: true,
	}
	newAbsenceDictionary := &RegistrationCodeDictionary{
		Codes: map[int32]*RegistrationCode{
			absenceCode.ID: absenceCode,
		},
		UploadedAt: time.Now(),
	}
	SetAbsenceCodeDictionary(newAbsenceDictionary)
	return oldPresentsCodeDictionary, oldAbsenceCodeDictionary
}

func restoreCodesDictionaries(oldPresentsCodeDictionary *RegistrationCodeDictionary, oldAbsenceCodeDictionary *RegistrationCodeDictionary) {
	SetPresentsCodeDictionary(oldPresentsCodeDictionary)
	SetAbsenceCodeDictionary(oldAbsenceCodeDictionary)
}

func prepareConfig(t *testing.T) {
	cfg := &config.Config{
		ERPFirstRegistrationPeriodName:  "AM",
		ERPDefaultLessonAbsenceCodeName: "C",
		ERPDefaultPresentCodeName:       "/",
	}
	err := container.Singleton(func() *config.Config {
		return cfg
	})
	assert.NoError(t, err)
}

func prepareTestEnv() (*StudentAttendance, *saEnv) {
	student := &Visitor{
		Id:             1,
		ErpID:          12345,
		ErpSchoolID:    "S123",
		ErpYearGroupID: 10,
		ErpDivisions:   []int32{101, 102},
	}

	now := time.Now()

	periodAM := &RegistrationPeriod{
		ID:     100,
		Name:   "AM",
		Start:  time.Date(now.Year(), now.Month(), now.Day(), 7, 37, 0, 0, time.UTC),
		Time:   time.Date(now.Year(), now.Month(), now.Day(), 7, 40, 0, 0, time.UTC),
		Finish: time.Date(now.Year(), now.Month(), now.Day(), 7, 55, 0, 0, time.UTC),
	}

	period1 := &RegistrationPeriod{
		ID:     1,
		Name:   "Period 1",
		Start:  time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC),
		Time:   time.Date(now.Year(), now.Month(), now.Day(), 8, 0, 0, 0, time.UTC),
		Finish: time.Date(now.Year(), now.Month(), now.Day(), 8, 59, 0, 0, time.UTC),
	}

	period2 := &RegistrationPeriod{
		ID:     2,
		Name:   "Period 2",
		Start:  time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.UTC),
		Time:   time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.UTC),
		Finish: time.Date(now.Year(), now.Month(), now.Day(), 9, 59, 0, 0, time.UTC),
	}

	schedule := &Schedule{
		Periods: map[int32]*RegistrationPeriod{
			periodAM.ID: periodAM,
			period1.ID:  period1,
			period2.ID:  period2,
		},
	}

	sa := NewStudentAttendance(student, schedule)

	for _, period := range schedule.Periods {
		ai := &AttendanceItem{
			RegistrationPeriodID: RegistrationPeriodID(period.ID),
			SchoolID:             student.ErpSchoolID,
			IsFutureAbsence:      false,
			IsPresent:            false,
			IsLate:               false,
			IsOutOfSchool:        false,
			IsRegistered:         0,
		}
		sa.SetAttendanceStatus(ai)
	}

	return sa, &saEnv{
		student:  student,
		schedule: schedule,
	}
}
