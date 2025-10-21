package isams

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/buzyka/imlate/internal/isb/entity"
)

func (c *Client) Register(visitor *entity.Visitor, period *RegistrationPeriod, status *RegistrationStatus) error {

	url := c.BaseURL + strings.ReplaceAll(RegisterEndpoint, "{registrationPeriodId}", fmt.Sprintf("%d", period.ID))
	url = strings.ReplaceAll(url, "{schoolId}", fmt.Sprintf("%d", visitor.IsamsSchoolId))

	bodyBytes, err := json.Marshal(status)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to register: %s", resp.Status)
	}

	return nil
}
