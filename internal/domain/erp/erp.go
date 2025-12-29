package erp

import (
	"context"

	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
)

type Factory interface {
	NewClient(ctx context.Context) (Client, error)
}

type Client interface {
	GetStudents(page, pageSize int32) (*isams.StudentsResponse, error)
	GetYearGroupDivisions(yearGroupID int32) (*isams.YearGroupsDivisionsResponse, error)
	GetCurrentRegistrationPeriodsForDivision(divisionID int32) (*isams.RegistrationPeriodsResponse, error)
	GetRegistrationStatusForStudent(studentSchoolID string, periodID int32) (*isams.RegistrationStatus, error)
	GetRegistrationAbsenceCodes() (*isams.RegistrationAbsenceCodesResponse, error)
	GetRegistrationPresentCodes() (*isams.RegistrationPresentCodeResponse, error)
}
