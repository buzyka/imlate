package isams

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/buzyka/imlate/internal/isb/entity"
)

type RegistrationStatus struct {
	AbsenceCodeID          *int32       `json:"absenceCodeId"`          // can be null
	AlertSent              bool       `json:"alertSent"`
	IsFutureAbsence        bool       `json:"isFutureAbsence"`
	IsLate                 bool       `json:"isLate"`
	IsOutOfSchool          bool       `json:"isOutOfSchool"`
	IsPresent              bool       `json:"isPresent"`
	IsRegistered           int32        `json:"isRegistered"`
	LeavingOrLeftDateTime  *time.Time `json:"leavingOrLeftDateTime"`  // can be null
	NumberOfMinutesLate    *int32        `json:"numberOfMinutesLate"`
	ParentNotificationSent bool       `json:"parentNotificationSent"`
	PresentCodeID          *int32       `json:"presentCodeId"`          // can be null
	RegistrationComment    *string    `json:"registrationComment"`    // can be null
	RegistrationPeriodID   int32        `json:"registrationPeriodId"`
	SchoolID               string     `json:"schoolId"`
}

func (c *Client) GetRegistrationStatusesForVisitor(visitor *entity.Visitor) ([]*RegistrationStatus, error) {
	periods, err := c.GetRegistrationPeriods()
	if err != nil {
		return nil, err
	}

	var statuses []*RegistrationStatus
	for _, period := range periods {
		status, err := c.GetRegistrationStatusForVisitor(period, visitor)
		if err != nil {
			return nil, err
		}
		if status != nil {
			statuses = append(statuses, status)
		}
	}
	return statuses, nil
}

func (c *Client) GetRegistrationStatusForVisitor(period *RegistrationPeriod, visitor *entity.Visitor) (*RegistrationStatus, error) {
	url := c.BaseURL + strings.Replace(RegistrationStatusEndpoint, "{registrationPeriodId}", fmt.Sprintf("%d", period.ID), 1)
	url = strings.Replace(url, "{schoolId}", fmt.Sprintf("%d", visitor.IsamsSchoolId), 1)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get registration status: %s", resp.Status)
	}

	var payload RegistrationStatus
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}
