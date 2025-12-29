package synchroniser

import (
	"context"
	"time"

	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
)

const PageSize = 100

type StudentSync struct {
	ERPFactory         erp.Factory                `container:"type"`
	VisitorRepo        provider.VisitorRepository `container:"type"`
	currentVisitors    []*entity.Visitor
	ctx                context.Context
	currentClient      erp.Client
	yearGroupDivisions map[int32][]int32
}

func (s *StudentSync) cleanUpSyncSession() {
	s.currentClient = nil
	s.ctx = nil
	s.yearGroupDivisions = nil
}

func (s *StudentSync) startSyncSession(ctx context.Context) error {
	ERPClient, err := s.ERPFactory.NewClient(ctx)
	if err != nil {
		return err
	}
	s.currentClient = ERPClient
	s.yearGroupDivisions = make(map[int32][]int32)
	return nil
}

func (s *StudentSync) SyncAllStudents() error {
	ctx := context.Background()
	err := s.startSyncSession(ctx)
	if err != nil {
		return err
	}
	defer s.cleanUpSyncSession()

	s.currentVisitors, err = s.VisitorRepo.GetAll()
	if err != nil {
		return err
	}

	var pageNumber int32 = 1

	for {
		resp, err := s.currentClient.GetStudents(pageNumber, PageSize)
		if err != nil {
			return err
		}

		for _, student := range resp.Students {
			if err = s.SaveStudent(student); err != nil {
				return err
			}
		}
		if resp.TotalPages <= pageNumber {
			break
		}
		pageNumber++
	}
	return nil
}

func (s *StudentSync) SaveStudent(student isams.Student) error {
	var fullName string
	if student.FullName != nil {
		fullName = *student.FullName
	}

	var grade, yearGroup int
	if student.YearGroup != nil {
		grade = *student.YearGroup
		yearGroup = *student.YearGroup
	}

	divisions, err := s.getDivisionsByYearGroup(int32(yearGroup))
	if err != nil {
		return err
	}

	var UpdatedAt time.Time
	if student.LastUpdated != nil {
		parsedTime, err := time.Parse(time.RFC3339, *student.LastUpdated)
		if err == nil {
			UpdatedAt = parsedTime.Truncate(time.Second)
		}
	}

	visitor := &entity.Visitor{
		Name:           "",
		Surname:        fullName,
		IsStudent:      true,
		Grade:          grade,
		ErpID:          student.ID,
		ErpSchoolID:    student.SchoolID,
		ErpYearGroupID: int32(yearGroup),
		ErpDivisions:   divisions,
		UpdatedAt:      UpdatedAt,
	}

	// Check if visitor needs to be updated or added
	// and set visitor.Id if exists
	if !s.IsUpToDate(visitor) {
		return s.VisitorRepo.AddVisitor(visitor)
	}
	return nil
}

func (s *StudentSync) IsUpToDate(newVisitor *entity.Visitor) bool {
	for _, currentVisitor := range s.currentVisitors {
		if currentVisitor.ErpID == newVisitor.ErpID {
			newVisitor.Id = currentVisitor.Id
			switch {
			case newVisitor.UpdatedAt.IsZero():
				return true
			case currentVisitor.UpdatedAt.Equal(newVisitor.UpdatedAt) && currentVisitor.GetSyncHash() == newVisitor.GetSyncHash():
				return true
			default:
				return false
			}
		}
	}
	if newVisitor.UpdatedAt.IsZero() {
		newVisitor.UpdatedAt = time.Now().Truncate(time.Second)
	}
	return false
}

func (s *StudentSync) getDivisionsByYearGroup(erpID int32) ([]int32, error) {
	if divisions, ok := s.yearGroupDivisions[erpID]; ok {
		return divisions, nil
	}

	resp, err := s.currentClient.GetYearGroupDivisions(erpID)
	if err != nil {
		return nil, err
	}
	var divisions []int32
	for _, division := range resp.Divisions {
		divisions = append(divisions, division.ID)
	}
	s.yearGroupDivisions[erpID] = divisions
	return divisions, nil
}
