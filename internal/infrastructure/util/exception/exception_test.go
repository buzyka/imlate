package exception

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExternalServiceError(t *testing.T) {
	expectedError := fmt.Errorf("Test Error")
	var tests = []struct {
		name          string
		exception     *Exception
		exceptionCode string
		expectedType  string
	}{
		{
			name:          "validation exception",
			exception:     Validation(expectedError),
			exceptionCode: "INCORRECT_REQUEST",
			expectedType:  ValidationType,
		},
		{
			name:          "external service error",
			exception:     ExternalServiceError(expectedError),
			exceptionCode: "EXTERNAL_SERVICE_ERROR",
			expectedType:  ExternalServiceErrorType,
		},
		{
			name:          "external service warning",
			exception:     ExternalServiceWarning(expectedError),
			exceptionCode: "EXTERNAL_SERVICE_WARNING",
			expectedType:  ExternalServiceWarningType,
		},
		{
			name:          "external response processing error",
			exception:     ExternalResponseProcessingError(expectedError),
			exceptionCode: "RESPONSE_PROCESSING_ERROR",
			expectedType:  ExternalResponseProcessingErrorType,
		},
		{
			name:         "empty code",
			exception:    Validation(expectedError),
			expectedType: ValidationType,
		},
	}
	for _, testData := range tests {
		t.Run(testData.name, func(t *testing.T) {
			e := testData.exception
			expectedCode := ""
			if testData.exceptionCode != "" {
				expectedCode = testData.exceptionCode
				e.SetCode(testData.exceptionCode)
			}
			assert.Equal(t, testData.expectedType, e.Type)
			assert.Equal(t, expectedCode, e.Code)
			assert.Equal(t, "Test Error", e.Error.Error())
		})
	}
}
