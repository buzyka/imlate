package isams

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type RegistrationStatus struct {
	SchoolID              string     `json:"schoolId"`
	RegistrationPeriodID  int32      `json:"registrationPeriodId"`

	IsRegistered          int32      `json:"isRegistered"`
	IsPresent             bool       `json:"isPresent"`
	IsLate                bool       `json:"isLate"`
	IsOutOfSchool         bool       `json:"isOutOfSchool"`
	IsFutureAbsence       bool       `json:"isFutureAbsence"`

	NumberOfMinutesLate   int32      `json:"numberOfMinutesLate"`

	LeavingOrLeftDateTime       *string 	 `json:"leavingOrLeftDateTime"`

	PresentCodeID         *int32     `json:"presentCodeId"`
	AbsenceCodeID         *int32     `json:"absenceCodeId"`

	RegistrationComment   *string    `json:"registrationComment"`

	AlertSent             bool       `json:"alertSent"`
	ParentNotificationSent bool      `json:"parentNotificationSent"`
}

type RegistrationStatusRequest struct {
	IsPresent bool `json:"isPresent"`
	IsLate    bool `json:"isLate"`

	PresentCodeID *int32 `json:"presentCodeId,omitempty"`
	AbsenceCodeID *int32 `json:"absenceCodeId,omitempty"`

	LeavingOrLeftDateTime *string `json:"leavingOrLeftDateTime,omitempty"`

	NumberOfMinutesLate int32 `json:"numberOfMinutesLate,omitempty"`

	RegistrationComment *string `json:"registrationComment,omitempty"`
}

func (c *Client) GetRegistrationStatusForStudent(studentSchoolID string, periodID int32) (*RegistrationStatus, error) {
	url := RegistrationStatusEndpoint
	url = strings.Replace(url, "{registrationPeriodId}", fmt.Sprintf("%d", periodID), 1)
	url = strings.Replace(url, "{schoolId}", studentSchoolID, 1)
	req, err := http.NewRequest("GET", c.BaseURL+url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get registration status: %s", resp.Status)
	}
	status := &RegistrationStatus{}
	err = json.NewDecoder(resp.Body).Decode(status)
	if err != nil {
		return nil, err
	}
	return status, nil
}

func (c *Client) PutRegistration(schoolID string, periodID int32, request RegistrationStatusRequest) error {
	url := RegistrationStatusEndpoint
	url = strings.Replace(url, "{registrationPeriodId}", fmt.Sprintf("%d", periodID), 1)
	url = strings.Replace(url, "{schoolId}", schoolID, 1)
	reqBody, err := json.Marshal(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", c.BaseURL+url, strings.NewReader(string(reqBody)))
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update registration status: %s", resp.Status)
	}
	return nil
}
