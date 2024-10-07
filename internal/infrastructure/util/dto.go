package util

type FailureResponse struct {
	Error string `json:"error"`
}

type ExtendedFailureResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}
