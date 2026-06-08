package plane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
)

// ListAttachments returns the attachment metadata for an issue.
func (c *Client) ListAttachments(ctx context.Context, projectID, issueID string) ([]Attachment, error) {
	raw, err := c.doRaw(ctx, http.MethodGet, c.issuePath(projectID, issueID, "issue-attachments"), nil, nil)
	if err != nil {
		return nil, err
	}
	items := extractResults(raw)
	out := make([]Attachment, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}

// AttachmentDetailURL returns the relative URL for a single attachment.
func (c *Client) AttachmentDetailURL(projectID, issueID, attachmentID string) string {
	return c.issuePath(projectID, issueID, "issue-attachments", attachmentID)
}

// InitAttachmentUpload requests upload credentials from Plane. The response
// includes the asset id (under "id" or "asset_id") and an "upload_data" dict
// with the S3 form fields plus the upload URL.
func (c *Client) InitAttachmentUpload(ctx context.Context, projectID, issueID, name string, fileType *string, size *int64) (map[string]any, error) {
	payload := map[string]any{"name": name}
	if fileType != nil {
		payload["type"] = *fileType
	}
	if size != nil {
		payload["size"] = *size
	}
	var out map[string]any
	if err := c.doJSON(ctx, http.MethodPost, c.issuePath(projectID, issueID, "issue-attachments"), nil, payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CompleteAttachmentUpload finalises an attachment upload after the S3
// transfer has succeeded. A 204 response is returned as an empty map.
func (c *Client) CompleteAttachmentUpload(ctx context.Context, projectID, issueID, assetID string) (map[string]any, error) {
	path := c.issuePath(projectID, issueID, "issue-attachments", assetID)
	raw, err := c.doRaw(ctx, http.MethodPatch, path, nil, map[string]any{})
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(trimmed, &out); err != nil {
		return nil, fmt.Errorf("plane: decode PATCH %s: %w", path, err)
	}
	return out, nil
}

// DownloadAttachment fetches the raw bytes for an attachment URL. Same-host
// URLs receive the X-API-Key header; external URLs (S3 redirects) do not.
func (c *Client) DownloadAttachment(ctx context.Context, rawURL string) ([]byte, string, error) {
	target, withAuth := c.attachmentTarget(rawURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", fmt.Errorf("plane: build GET %s: %w", target, err)
	}
	if withAuth {
		req.Header.Set("X-API-Key", c.apiToken)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.download.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("plane: GET %s: %w", target, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("plane: read GET %s: %w", target, err)
	}
	if resp.StatusCode >= 300 {
		trimmed := body
		if len(trimmed) > maxErrorBodySize {
			trimmed = trimmed[:maxErrorBodySize]
		}
		return nil, "", &APIError{StatusCode: resp.StatusCode, Method: http.MethodGet, URL: target, Body: trimmed}
	}
	return body, resp.Header.Get("Content-Type"), nil
}

func (c *Client) attachmentTarget(raw string) (string, bool) {
	switch {
	case strings.HasPrefix(raw, "/"):
		return c.baseURL + raw, true
	case strings.HasPrefix(raw, c.baseURL):
		return raw, true
	default:
		return raw, false
	}
}

// UploadAttachmentBlob uploads file content to the S3-style URL returned by
// InitAttachmentUpload. Fields are written before the file part to match
// what Plane (and S3) expect; a trailing slash is appended when missing.
func (c *Client) UploadAttachmentBlob(ctx context.Context, uploadURL string, uploadFields map[string]string, name string, content io.Reader, mimeType string) error {
	if !strings.HasSuffix(uploadURL, "/") {
		uploadURL += "/"
	}

	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range uploadFields {
		if err := mw.WriteField(k, v); err != nil {
			return fmt.Errorf("plane: multipart field %s: %w", k, err)
		}
	}
	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, name))
	partHeader.Set("Content-Type", mimeType)
	fw, err := mw.CreatePart(partHeader)
	if err != nil {
		return fmt.Errorf("plane: multipart file part: %w", err)
	}
	if _, err := io.Copy(fw, content); err != nil {
		return fmt.Errorf("plane: copy upload body: %w", err)
	}
	if err := mw.Close(); err != nil {
		return fmt.Errorf("plane: close multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &buf)
	if err != nil {
		return fmt.Errorf("plane: build POST %s: %w", uploadURL, err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.download.Do(req)
	if err != nil {
		return fmt.Errorf("plane: POST %s: %w", uploadURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
		return &APIError{StatusCode: resp.StatusCode, Method: http.MethodPost, URL: uploadURL, Body: body}
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
