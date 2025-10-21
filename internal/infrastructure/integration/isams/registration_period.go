package isams

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type RegistrationPeriodsResponse struct {
	RegistrationPeriods []*RegistrationPeriod `json:"registrationPeriods"`
}

type RegistrationPeriod struct {
	ID                  int         `json:"id"`
	Divisions           []Division  `json:"divisions"`
	Finish              time.Time   `json:"finish"`
	FriendlyName        string      `json:"friendlyName"`
	NumberOfStudents    int         `json:"numberOfStudents"`
	RegistrationRuleID  int         `json:"registrationRuleId"`
	RegistrationType    string      `json:"registrationType"`
	Start               time.Time   `json:"start"`
	Time                time.Time   `json:"time"`
}

type Division struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

func (c *Client) GetRegistrationPeriods() ([]*RegistrationPeriod, error) {
	req, err := http.NewRequest("GET", c.BaseURL+RegistrationPeriodsEndpoint, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("divisionId", "2")
	req.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get registration periods: %s", resp.Status)
	}

	var payload RegistrationPeriodsResponse
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}
	fmt.Println("registration periods found: ", len(payload.RegistrationPeriods))

	return payload.RegistrationPeriods, nil
}

func (c *Client) GetRegistrationPeriodByName(name string) (*RegistrationPeriod, error) {
	req, err := http.NewRequest("GET", c.BaseURL+RegistrationPeriodsEndpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get registration period: %s", resp.Status)
	}

	var payload RegistrationPeriodsResponse
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}

	for _, period := range payload.RegistrationPeriods {
		if period.FriendlyName == name {
			return period, nil
		}
	}

	return nil, fmt.Errorf("registration period not found: %s", name)
}
