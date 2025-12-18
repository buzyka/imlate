package synchroniser

import (
	"context"
	"fmt"
	"time"

	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/domain/provider"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/isb/entity"
)

const PageSize = 100

type StudentSync struct {
	ERPFactory      erp.Factory                `container:"type"`
	VisitorRepo     provider.VisitorRepository `container:"type"`
	currentVisitors []*entity.Visitor
}

func (s *StudentSync) SyncAllStudents() error {
	ctx := context.Background()
	ERPClient, err := s.ERPFactory.NewClient(ctx)
	if err != nil {
		return err
	}

	s.currentVisitors, err = s.VisitorRepo.GetAll()
	if err != nil {
		return err
	}

	var pageNumber int32 = 1

	for {
		resp, err := ERPClient.GetStudents(pageNumber, PageSize)
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

	var grade int
	if student.YearGroup != nil {
		grade = *student.YearGroup
	}

	var UpdatedAt time.Time
	if student.LastUpdated != nil {
		parsedTime, err := time.Parse(time.RFC3339, *student.LastUpdated)
		if err == nil {
			UpdatedAt = parsedTime.Truncate(time.Second)
		}
	}

	visitor := &entity.Visitor{
		Name:        "",
		Surname:     fullName,
		IsStudent:   true,
		Grade:       grade,
		ErpID:       student.ID,
		ErpSchoolID: student.SchoolID,
		UpdatedAt:   UpdatedAt,
	}

	// Check if visitor needs to be updated or added
	// and set visitor.Id if exists
	if !s.IsUpToDate(visitor) {
		// TODO remove it after testing
		if student.FullName != nil {
			fmt.Println(*student.FullName)
		}
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
			case currentVisitor.UpdatedAt.Equal(newVisitor.UpdatedAt):
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
