package studentvisit

import (
	"time"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/isb/entity"
	"github.com/buzyka/imlate/internal/isb/registration"
)

type StudentVisit struct {
	Cfg 		*config.ISAMSConfig 
	Registrator registration.Registrator
	CurrentTime	time.Time // in ISAMS Timezone
	Visitor *entity.Visitor
	RegistrationStatuses []*isams.RegistrationStatus // TODO: remove it later
	regStatuses map [int32]*isams.RegistrationStatus
}

func (sv *StudentVisit) SetRegistrationStatuses(statuses []*isams.RegistrationStatus) {
	sv.RegistrationStatuses = statuses
	sv.regStatuses = make(map[int32]*isams.RegistrationStatus)
	for _, status := range statuses {
		sv.regStatuses[status.RegistrationPeriodID] = status
	}
}

func (sv *StudentVisit) FirstEntryIsRegistered() *isams.RegistrationStatus {
	return nil
}








