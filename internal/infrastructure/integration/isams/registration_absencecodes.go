package isams

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type RegistrationAbsenceCodesResponse struct {
	AbsenceCodes []RegistrationAbsenceCode `json:"absenceCodes"`
}

type RegistrationAbsenceCode struct {
	ID                 int32   `json:"id"`
	Code               string  `json:"code"`
	Description        *string `json:"description"`
	DisplayOrder       int32   `json:"displayOrder"`
	GovernmentCode     string  `json:"governmentCode"`
	GovernmentCodeName string  `json:"governmentCodeName"`
	IsOOSActive        bool    `json:"isOOSActive"`
	Name               string  `json:"name"`
}

func (c *Client) GetRegistrationAbsenceCodes() (*RegistrationAbsenceCodesResponse, error) {
	url := AbsenceCodesEndpoint
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
		return nil, fmt.Errorf("failed to get registration absence codes: %s", resp.Status)
	}
	absenceCodes := &RegistrationAbsenceCodesResponse{}
	err = json.NewDecoder(resp.Body).Decode(absenceCodes)
	if err != nil {
		return nil, err
	}
	return absenceCodes, nil
}
