package isams

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Division struct {
	ID          int32    `json:"id"`
	Code        string `json:"code"`
	LastUpdated string `json:"lastUpdated"`
	Name        string `json:"name"`
	Order       int32    `json:"order"`
}

type YearGroupsDivisionsResponse struct {
	Divisions []Division `json:"divisions"`
}

func (c *Client) GetYearGroupDivisions(yearGroupID int32) (*YearGroupsDivisionsResponse, error) {
	url := strings.Replace(YearGroupsDivisionsEndpoint, "{yearGroupId}", fmt.Sprintf("%d", yearGroupID), 1)
	req, err := http.NewRequest("GET", c.BaseURL+url, nil)
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
		return nil, fmt.Errorf("failed to get year group divisions: %s", resp.Status)
	}

	var payload YearGroupsDivisionsResponse
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}
