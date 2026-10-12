package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// UploadCategory is where an uploaded document lands for the accountant
type UploadCategory string

const (
	UploadExpenses UploadCategory = "expenses"
	UploadRevenues UploadCategory = "revenues"
)

// referer is the page the web uploader lives on for the save step
func (u UploadCategory) referer() string {
	if u == UploadRevenues {
		return "/revenues"
	}
	return "/"
}

// UploadDocument uploads a file to SOLO.ro and saves it as an expense or
// revenue document. Returns the server filename on success
func (c *Client) UploadDocument(filePath string, category UploadCategory) (string, error) {
	if category != UploadExpenses && category != UploadRevenues {
		return "", fmt.Errorf("unknown upload category %q", category)
	}

	// Generate unique ID for this upload
	uploadID := uuid.New().String()
	uploadID = fmt.Sprintf("%s%s%s%s%s",
		uploadID[0:8], uploadID[9:13], uploadID[14:18], uploadID[19:23], uploadID[24:36])

	// Step 1: Upload the file
	filename, err := c.uploadFile(uploadID, filePath)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	// Step 2: Confirm the upload
	if err := c.confirmUpload(uploadID, category); err != nil {
		return "", fmt.Errorf("confirm failed: %w", err)
	}

	return filename, nil
}

// uploadFile performs the multipart file upload
func (c *Client) uploadFile(uploadID, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add empty JSON metadata field
	metaField, err := writer.CreateFormField("filepond")
	if err != nil {
		return "", err
	}
	metaField.Write([]byte("{}"))

	// Add file field
	filename := filepath.Base(filePath)
	fileField, err := writer.CreateFormFile("filepond", filename)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fileField, file); err != nil {
		return "", err
	}

	writer.Close()

	// Make request
	url := fmt.Sprintf("%s/api/local-storage/upload/%s", baseURL, uploadID)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Response is quoted string, e.g. "filename.pdf"
	var result string
	if err := json.Unmarshal(body, &result); err != nil {
		// If not JSON, use raw response
		result = string(body)
	}

	return cleanString(result), nil
}

// confirmUpload saves the staged upload. The web reads the answer as the
// number of saved documents and treats <= 0 as a failure, older answers
// ({} or an empty body) count as success
func (c *Client) confirmUpload(uploadID string, category UploadCategory) error {
	path := fmt.Sprintf("/api/financial-documents/save/%s/%s", category, uploadID)
	body, err := c.doRaw("POST", path, category.referer(), struct{}{})
	if err != nil {
		return err
	}
	var saved int
	if json.Unmarshal(body, &saved) == nil && saved <= 0 {
		return fmt.Errorf("SOLO.ro saved no documents")
	}
	return nil
}
