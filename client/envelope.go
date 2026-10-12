package client

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// OkResponse is the {"Ok", "Data", "ErrorCode", "ErrorMessage"} envelope the
// newer endpoints answer with. Data may be null even when Ok is true, use a
// pointer or json.RawMessage for T when that matters
type OkResponse[T any] struct {
	Ok           bool    `json:"Ok"`
	Data         T       `json:"Data"`
	ErrorCode    *string `json:"ErrorCode"`
	ErrorMessage *string `json:"ErrorMessage"`
}

// Err turns Ok=false into an *APIError, nil when Ok is true
func (r *OkResponse[T]) Err() error {
	if r.Ok {
		return nil
	}
	e := &APIError{}
	if r.ErrorCode != nil {
		e.Code = cleanString(*r.ErrorCode)
	}
	if r.ErrorMessage != nil {
		e.Message = cleanString(*r.ErrorMessage)
	}
	return e
}

// APIError is a request SOLO.ro answered with Ok=false
type APIError struct {
	Code    string
	Message string
}

func (e *APIError) Error() string {
	switch {
	case e.Message != "" && e.Code != "":
		return fmt.Sprintf("%s (%s)", e.Message, e.Code)
	case e.Message != "":
		return e.Message
	case e.Code != "":
		return "SOLO.ro rejected the request: " + e.Code
	}
	return "SOLO.ro rejected the request"
}

// StatusResponse is the {"Status": "..."} shape of pre-cancel, cancel,
// manual-send, queue-status and the alert endpoints
type StatusResponse struct {
	Status string `json:"Status"`
}

// doOk runs doJSON on an Ok envelope, returns *APIError when Ok is false and
// decodes Data into out (left untouched when out is nil or Data is null)
func (c *Client) doOk(method, path, referer string, reqBody, out any) error {
	var env OkResponse[json.RawMessage]
	if err := c.doJSON(method, path, referer, reqBody, &env); err != nil {
		return err
	}
	if err := env.Err(); err != nil {
		return err
	}
	if out == nil || len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return err
	}
	scrubValue(reflect.ValueOf(out))
	return nil
}
