package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// listRequest is the common JSON body accepted by all SOLO.ro list endpoints
type listRequest struct {
	SearchText string `json:"SearchText"`
	StartIndex int    `json:"StartIndex"`
	MaxResults int    `json:"MaxResults"`
	SortBy     string `json:"SortBy"`
	SortAsc    bool   `json:"SortAsc"`
}

func newListRequest(startIndex, maxResults int) listRequest {
	return listRequest{StartIndex: startIndex, MaxResults: maxResults, SortAsc: true}
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
		return json.Unmarshal(body, out)
	}
	return nil
}
