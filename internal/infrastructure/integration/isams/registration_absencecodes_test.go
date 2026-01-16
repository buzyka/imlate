package isams

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistrationAbsenceCode_Marshaling(t *testing.T) {
	jsonStr := `{
            "id": 1,
            "code": "N",
            "description": "",
            "displayOrder": 1,
            "governmentCode": "N",
            "governmentCodeName": "No Reason Yet Provided For Absence",
            "isOOSActive": false,
            "name": "No Reason Yet Provided For Absence"
        }`

	var code RegistrationAbsenceCode
	err := json.Unmarshal([]byte(jsonStr), &code)
	assert.NoError(t, err)
	assert.Equal(t, int32(1), code.ID)
	assert.Equal(t, "N", code.Code)
	assert.NotNil(t, code.Description)
	assert.Equal(t, "", *code.Description)
	assert.Equal(t, int32(1), code.DisplayOrder)
	assert.Equal(t, "N", code.GovernmentCode)
	assert.Equal(t, "No Reason Yet Provided For Absence", code.GovernmentCodeName)
	assert.False(t, code.IsOOSActive)
	assert.Equal(t, "No Reason Yet Provided For Absence", code.Name)
}

func TestRegistrationAbsenceCode_Marshaling_NullDescription(t *testing.T) {
	jsonStr := `{
            "id": 2,
            "code": "-",
            "description": null,
            "displayOrder": 26,
            "governmentCode": "-",
            "governmentCodeName": "Unknown (Invalid Code)",
            "isOOSActive": true,
            "name": "Unknown"
        }`

	var code RegistrationAbsenceCode
	err := json.Unmarshal([]byte(jsonStr), &code)
	assert.NoError(t, err)
	assert.Equal(t, int32(2), code.ID)
	assert.Nil(t, code.Description)
}

func TestRegistrationAbsenceCodesResponse_Marshaling(t *testing.T) {
	jsonStr := `{
    "absenceCodes": [
        {
            "id": 1,
            "code": "N",
            "description": "",
            "displayOrder": 1,
            "governmentCode": "N",
            "governmentCodeName": "No Reason Yet Provided For Absence",
            "isOOSActive": false,
            "name": "No Reason Yet Provided For Absence"
        }
    ]
}`
	var resp RegistrationAbsenceCodesResponse
	err := json.Unmarshal([]byte(jsonStr), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.AbsenceCodes, 1)
	assert.Equal(t, int32(1), resp.AbsenceCodes[0].ID)
}
