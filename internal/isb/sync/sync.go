package sync

import (
	"fmt"

	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/isb/entity"
)

type ISAMSSynchronizer struct {
	client *isams.Client
	visitor entity.VisitorRepository
}


func (s *ISAMSSynchronizer) SyncStudents() error {
	// _, err := s.getAllISAMSStudents()
	// if err != nil {
	// 	return err
	// }


	updatedAt, err := s.visitor.GetMaxUpdatedAt()
	if err != nil {
		return err
	}
	fmt.Println("Last updated at: ", updatedAt)

	visitorStudents, err := s.visitor.GetAllStudents()
	if err != nil {
		return err
	}

	fmt.Println("Students in ISAMS: ", len(visitorStudents))

	return nil
}

func (s *ISAMSSynchronizer) GetAllISAMSStudents() ([]isams.Student, error) {
	var allStudents []isams.Student
	var page int32 = 1
	var pageSize int32 = 100

	for {
		resp, err := s.client.GetStudents(page, pageSize)
		if err != nil {
			return nil, err
		}

		allStudents = append(allStudents, resp.Students...)
		
		if page >= resp.TotalPages {
			break
		}
		page++
	}
	return allStudents, nil
}
