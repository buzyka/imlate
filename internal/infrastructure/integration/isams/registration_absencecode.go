package isams

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type AbsenceResponse struct {
	AbsenceCodes []AbsenceCode `json:"absenceCodes"`
}

type AbsenceCode struct {
	ID                 int32   `json:"id"`
	Code               string  `json:"code"`
	Description        *string `json:"description"` // null в JSON -> nil в Go
	DisplayOrder       int32   `json:"displayOrder"`
	GovernmentCode     string  `json:"governmentCode"`
	GovernmentCodeName string  `json:"governmentCodeName"`
	IsOOSActive        bool    `json:"isOOSActive"`
	Name               string  `json:"name"`
}

func (c *Client) GetAbsenceCodes() ([]AbsenceCode, error) {
	req, err := http.NewRequest("GET", c.BaseURL+AbsenceCodesEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get absence codes: %s", resp.Status)
	}

	var payload AbsenceResponse
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}

	return payload.AbsenceCodes, nil
}

func (c *Client) GetAbsenceCodeByCode(code string) (*AbsenceCode, error) {
	absenceCodes, err := c.GetAbsenceCodes()
	if err != nil {
		return nil, err
	}

	for _, ac := range absenceCodes {
		if ac.Code == code {
			return &ac, nil
		}
	}

	return nil, fmt.Errorf("absence code not found: %s", code)
}
