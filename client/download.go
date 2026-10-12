package client

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

const maxDownloadBytes = 200 << 20

// DownloadedFile is a file fetched from SOLO.ro
type DownloadedFile struct {
	Data        []byte
	ContentType string
}

// Extension picks the file extension (without dot) for the download
func (f *DownloadedFile) Extension() string {
	return FileExtension(f.ContentType, f.Data)
}

// download GETs a file from falcon like the web app does. Anything but 200 is
// an error and so is a 200 HTML body, which is an error or login page
func (c *Client) download(path, referer string) (*DownloadedFile, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Referer", baseURL+referer)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	if len(data) > maxDownloadBytes {
		return nil, fmt.Errorf("download larger than %d MB", maxDownloadBytes>>20)
	}
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(strings.ToLower(ct), "html") || looksLikeHTML(data) {
		return nil, fmt.Errorf("got a web page instead of a file, the session may have expired or the document does not exist")
	}
	return &DownloadedFile{Data: data, ContentType: ct}, nil
}

func looksLikeHTML(data []byte) bool {
	head := bytes.ToLower(bytes.TrimLeft(trimBOM(data[:min(len(data), 512)]), " \t\r\n"))
	return bytes.HasPrefix(head, []byte("<!doctype html")) || bytes.HasPrefix(head, []byte("<html"))
}

func trimBOM(b []byte) []byte {
	return bytes.TrimPrefix(b, []byte("\xEF\xBB\xBF"))
}

// FileExtension picks an extension (without dot) from the content type, then
// from the file signature for generic or missing types. Falls back to "dat"
func FileExtension(contentType string, data []byte) string {
	mt, _, _ := mime.ParseMediaType(contentType)
	if mt == "" {
		mt = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	}
	switch {
	case strings.Contains(mt, "pdf"):
		return "pdf"
	case strings.Contains(mt, "png"):
		return "png"
	case strings.Contains(mt, "jpg"), strings.Contains(mt, "jpeg"):
		return "jpg"
	case strings.Contains(mt, "heic"):
		return "heic"
	case strings.Contains(mt, "xml"):
		return "xml"
	case strings.Contains(mt, "zip"):
		return "zip"
	}
	if mt != "" && mt != "application/octet-stream" && mt != "binary/octet-stream" {
		if exts, _ := mime.ExtensionsByType(mt); len(exts) > 0 {
			return strings.TrimPrefix(exts[0], ".")
		}
	}

	switch {
	case bytes.HasPrefix(data, []byte("%PDF")):
		return "pdf"
	case bytes.HasPrefix(data, []byte("PK")):
		return "zip"
	case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}):
		return "png"
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "jpg"
	}
	head := bytes.TrimLeft(trimBOM(data[:min(len(data), 64)]), " \t\r\n")
	for _, prefix := range []string{"<?xml", "<Invoice", "<CreditNote"} {
		if bytes.HasPrefix(head, []byte(prefix)) {
			return "xml"
		}
	}
	return "dat"
}
