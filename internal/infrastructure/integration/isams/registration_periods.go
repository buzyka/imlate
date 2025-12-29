package isams

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type RegistrationPeriodsResponse struct {
	RegistrationPeriods []RegistrationPeriod `json:"registrationPeriods"`
}

type RegistrationPeriod struct {
	ID                 int32      `json:"id"`
	Divisions          []Division `json:"divisions"`
	Finish             string     `json:"finish"`
	FriendlyName       string     `json:"friendlyName"`
	NumberOfStudents   int32      `json:"numberOfStudents"`
	RegistrationRuleID int32      `json:"registrationRuleId"`
	RegistrationType   string     `json:"registrationType"`
	Start              string     `json:"start"`
	Time               string     `json:"time"`
}

func (c *Client) GetCurrentRegistrationPeriodsForDivision(divisionID int32) (*RegistrationPeriodsResponse, error) {
	url := RegistrationPeriodsEndpoint

	req, err := http.NewRequest("GET", c.BaseURL+url, nil)
	if err != nil {
		return nil, err
	}
	
	query := req.URL.Query()
	query.Set("divisionId", fmt.Sprintf("%d", divisionID))
	req.URL.RawQuery = query.Encode()

	req.Header.Set("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get registration periods: %s", resp.Status)
	}
	var payload RegistrationPeriodsResponse
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}
