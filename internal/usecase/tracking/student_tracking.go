package tracking

import (
	"context"
	"fmt"

	"github.com/buzyka/imlate/internal/config"
	"github.com/buzyka/imlate/internal/domain/entity"
	"github.com/buzyka/imlate/internal/domain/erp"
	"github.com/buzyka/imlate/internal/infrastructure/integration/isams"
	"github.com/buzyka/imlate/internal/infrastructure/util"
)

type StudentTracker struct {
	cfg 				*config.Config 	`container:"type"`
	ERPFactory          erp.Factory     `container:"type"`	
}

func (s *StudentTracker) Track(ctx context.Context, visitor *entity.Visitor) error {
	// Get periods for visitor for today
	schedule, err := s.getPeriods(ctx, visitor.ErpDivisions)
	if err != nil {
		return err
	}

	// Get today student registration info

	// Is it first login of the day? (set status for AM)

	// is he late? (set status for AM and late time and set to the specific period) 

	// Is he already registered,  than it is log out (set status for PM)

	// TODO: implement track logic

	fmt.Printf("---Tracking periods: %d\n", len(schedule.Periods))
	
	return nil
}

func (s *StudentTracker) getPeriods(ctx context.Context, divisions []int32) (*entity.Schedule, error) {
	erpClient, err := s.ERPFactory.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	schedule := &entity.Schedule{}
	for _, division := range divisions {
		resp, err := erpClient.GetCurrentRegistrationPeriodsForDivision(division)
		if err != nil {
			// TODO: log error
			continue
		}
		s.addPeriodsFromResponse(schedule, resp)
	}

	return schedule, nil
}

func (s *StudentTracker) addPeriodsFromResponse(schedule *entity.Schedule, resp *isams.RegistrationPeriodsResponse) {
	for _, period := range resp.RegistrationPeriods {
		startDate, err := util.ParseTimeStrToLocal(period.Start)
		if err != nil {
			continue
		}
		finishDate, err := util.ParseTimeStrToLocal(period.Finish)
		if err != nil {
			continue
		}
		timeDate, err := util.ParseTimeStrToLocal(period.Time)
		if err != nil {
			continue
		}
		period := &entity.RegistrationPeriod{
			ID:        period.ID,
			Name:      period.FriendlyName,
			Time:      timeDate,
			Start: startDate,
			Finish:   finishDate,
		}
		schedule.AddPeriod(period)
	}
}
