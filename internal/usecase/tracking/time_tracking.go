package tracking

import (
	"context"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/infrastructure/util"
)

type StudentFilterConfig struct {
	YearGroup []*int32
}

func (f *StudentFilterConfig) YearGroupSlice() []int32 {
	result := make([]int32, 0, len(f.YearGroup))
	for _, yg := range f.YearGroup {
		if yg != nil {
			result = append(result, *yg)
		}
	}
	return result
}

type StudentFilterOption func(*StudentFilterConfig)

func WithYearGroups(yearGroups []int32) StudentFilterOption {
	return func(cfg *StudentFilterConfig) {
		cfg.YearGroup = make([]*int32, 0, len(yearGroups))
		for _, yg := range yearGroups {
			ygCopy := yg
			cfg.YearGroup = append(cfg.YearGroup, &ygCopy)
		}
	}
}

func (s *StudentTracker) TrackUntrackedStudentsAsAbsence(ctx context.Context, opts ...StudentFilterOption) error {
	cfg := &StudentFilterConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.YearGroup == nil {
		cfg.YearGroup = []*int32{}
	}
	
	students, err := s.VisitorRepo.FindAll(provider.WithERPYearGroups(cfg.YearGroupSlice()))
	if err != nil {
		return err
	}

	for _, student := range students {
		if err := s.trackAbsenceForNotRegisteredStudent(ctx, student); err != nil {
			s.Logger.Errorf("Error tracking absence for student %d: %v", student.Id, err)
		}
	}

	return nil
}

func (s *StudentTracker) trackAbsenceForNotRegisteredStudent(ctx context.Context, student *entity.Visitor) error {
	erpClient, err := s.ERPFactory.NewClient(ctx)
	if err != nil {
		s.Logger.Errorf("Error creating ERP client for student %d: %v", student.Id, err)
		return err
	}

	// Get periods for visitor for today
	schedule, err := s.getPeriods(erpClient, student.ErpDivisions)
	if err != nil {
		s.Logger.Errorf("Error getting periods for student %d: %v", student.Id, err)
		return err
	}

	studentAttendance := entity.NewStudentAttendance(student, schedule)
	s.fillAttendanceInfo(erpClient, studentAttendance)

	n := util.Now()

	na, shouldUpdate, err := studentAttendance.MarkNotRegisteredAsAbsent(n)
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
			s.Logger.Errorf("AutoAbsence: Error updating registration for student %d, period %d: %v", student.Id, na.Period.ID, err)
			return err
		}
	}

	return nil
}
