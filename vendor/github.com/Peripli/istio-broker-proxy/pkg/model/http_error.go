package model

import (
	"encoding/json"
)

type HttpError struct {
	ErrorMsg    string `json:"error"`
	Description string `json:"description"`
	StatusCode  int    `json:"-"`
}

func HttpErrorFromError(err error, statusCode int) *HttpError {
	switch t := err.(type) {
	case *HttpError:
		return t
	case HttpError:
		return &t
	default:
		return &HttpError{err.Error(), "", statusCode}
	}
}

func HttpErrorFromResponse(statusCode int, body []byte) error {
	okResponse := statusCode/100 == 2
	if !okResponse {
		var httpError HttpError
		err := json.Unmarshal(body, &httpError)
		if err != nil {
			return &HttpError{StatusCode: statusCode, ErrorMsg: string(body)}
		}
		httpError.StatusCode = statusCode
		return &httpError
	}
	return nil
}

func (e HttpError) Error() string {
	return e.ErrorMsg
}
