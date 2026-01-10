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

// UploadDocument uploads a file to SOLO.ro and confirms it as an expense
// Returns the document filename on success
func (c *Client) UploadDocument(filePath string) (string, error) {
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
	if err := c.confirmUpload(uploadID); err != nil {
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

	return result, nil
}

// confirmUpload confirms the uploaded document as an expense
func (c *Client) confirmUpload(uploadID string) error {
	url := fmt.Sprintf("%s/api/financial-documents/save/expenses/%s", baseURL, uploadID)

	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Referer", baseURL+"/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("confirm failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
