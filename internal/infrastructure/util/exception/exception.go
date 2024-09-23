package exception

const (
	ValidationType                      = "validation"
	ExternalServiceErrorType            = "external_service_error"
	ExternalServiceWarningType          = "external_service_warning"
	ExternalResponseProcessingErrorType = "external_respons_processiong_error"
)

type Exception struct {
	Type  string
	Error error
	Code  string
}

func Validation(err error) *Exception {
	return CreateException(ValidationType, err)
}

func ExternalServiceError(err error) *Exception {
	return CreateException(ExternalServiceErrorType, err)
}

func ExternalServiceWarning(err error) *Exception {
	return CreateException(ExternalServiceWarningType, err)
}

func ExternalResponseProcessingError(err error) *Exception {
	return CreateException(ExternalResponseProcessingErrorType, err)
}

func CreateException(t string, err error) *Exception {
	return &Exception{
		Type:  t,
		Error: err,
	}
}

func (e *Exception) SetCode(code string) *Exception {
	e.Code = code
	return e
}
