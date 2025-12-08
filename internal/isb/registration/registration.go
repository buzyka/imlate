package registration

import (
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/isb/entity"
)

type Registrator interface {
	// Register(visitor *entity.Visitor, numberOfMinutesLate int32) error
	Register(visitor *entity.Visitor, period *isams.RegistrationPeriod, status *isams.RegistrationStatus) error
	GetRegistrationPeriodByName(name string) (*isams.RegistrationPeriod, error)
	GetRegistrationPeriods() ([]*isams.RegistrationPeriod, error)
	GetRegistrationStatusForVisitor(period *isams.RegistrationPeriod, visitor *entity.Visitor) (*isams.RegistrationStatus, error)
	GetRegistrationStatusesForVisitor(visitor *entity.Visitor) ([]*isams.RegistrationStatus, error)
}
