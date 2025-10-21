package isams

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (c *Client) GetStudents(page, pageSize int32) (*StudentsResponse, error) {
	req, err := http.NewRequest("GET", c.BaseURL+StudentsEndpoint, nil)
	if err != nil {
		return nil, err
	}

	query := req.URL.Query()
	query.Set("page", fmt.Sprintf("%d", page))
	query.Set("pageSize", fmt.Sprintf("%d", pageSize))
	req.URL.RawQuery = query.Encode()

	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get students: %s", resp.Status)
	}

	var payload StudentsResponse
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}

func (c *Client) GetStudentByID(id int32) (*Student, error) {
	url := c.BaseURL + strings.Replace(StudentByIDEndpoint, "{schoolId}", fmt.Sprintf("%d", id), 1)
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
		return nil, fmt.Errorf("failed to get student: %s", resp.Status)
	}

	var payload Student
	err = json.NewDecoder(resp.Body).Decode(&payload)
	if err != nil {
		return nil, err
	}

	return &payload, nil
}

