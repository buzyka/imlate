package isams

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type RegistrationPresentCodeResponse struct {
	PresentCodes []RegistrationPresentCode `json:"presentCodes"`
}

type RegistrationPresentCode struct {
	ID             int32  `json:"id"`
	Code           string `json:"code"`
	DisplayOrder   int32  `json:"displayOrder"`
	IsOOCActive    bool   `json:"isOOCActive"`
	Name           string `json:"name"`
	ShowOnRegister bool   `json:"showOnRegister"`
}

func (c *Client) GetRegistrationPresentCodes() (*RegistrationPresentCodeResponse, error) {
	url := RegistrationPresentCodesEndpoint
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
		return nil, fmt.Errorf("failed to get registration present codes: %s", resp.Status)
	}
	presentCodes := &RegistrationPresentCodeResponse{}
	err = json.NewDecoder(resp.Body).Decode(presentCodes)
	if err != nil {
		return nil, err
	}
	return presentCodes, nil
}
