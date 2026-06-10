package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"unicode"
)

// listRequest is the common JSON body accepted by all SOLO.ro list endpoints
type listRequest struct {
	SearchText string `json:"SearchText"`
	StartIndex int    `json:"StartIndex"`
	MaxResults int    `json:"MaxResults"`
	SortBy     string `json:"SortBy"`
	SortAsc    bool   `json:"SortAsc"`
}

func newListRequest(startIndex, maxResults int, search string) listRequest {
	return listRequest{SearchText: search, StartIndex: startIndex, MaxResults: maxResults, SortAsc: true}
}

// doJSON performs an API request with the browser-mimicking headers SOLO.ro
// expects and decodes the JSON response into out. reqBody and out may be nil
func (c *Client) doJSON(method, path, referer string, reqBody, out any) error {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Referer", baseURL+referer)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	}
	if method != http.MethodGet {
		req.Header.Set("Origin", baseURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return err
		}
		// Single choke point: API strings end up in terminal output, so
		// strip control characters (escape injection) from every response
		scrubValue(reflect.ValueOf(out))
	}
	return nil
}

// cleanString drops terminal control characters from a server-supplied
// string, keeping printable text and turning whitespace controls into spaces
func cleanString(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t' || r == '\r':
			return ' '
		case unicode.IsPrint(r):
			return r
		default:
			return -1
		}
	}, s)
}

// scrubValue walks a decoded response and sanitizes every string in place
func scrubValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.String:
		if v.CanSet() {
			v.SetString(cleanString(v.String()))
		}
	case reflect.Ptr, reflect.Interface:
		if !v.IsNil() {
			scrubValue(v.Elem())
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			scrubValue(v.Field(i))
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			scrubValue(v.Index(i))
		}
	}
}
