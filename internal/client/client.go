package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PedroEvaldt/recall/internal/api"
	"github.com/google/uuid"
)

var (
	// ErrNotFound marks a server response where the requested document does not exist.
	ErrNotFound = errors.New("document not found")
	// ErrEmptyURL marks an empty server URL configuration.
	ErrEmptyURL = errors.New("server URL is empty")
	// ErrInvalidURL marks a server URL that cannot be parsed.
	ErrInvalidURL = errors.New("server URL is invalid")
	// ErrMissingPart marks a parsed URL without both scheme and host.
	ErrMissingPart = errors.New("server URL must include scheme and host")
)

// IsNotFound reports whether err wraps ErrNotFound.
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// Client sends authenticated requests to a recall server.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	authToken  string
}

// New validates baseURL and returns a Client configured with request timeout
// and bearer-token authentication.
func New(baseURL string, timeout time.Duration, token string) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("%w (set --server or RECALL_SERVER)", ErrEmptyURL)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("%w (got %q)", ErrMissingPart, baseURL)
	}
	return &Client{
		baseURL: u,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		authToken: token,
	}, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.authToken)
}

// ListDocuments fetches document metadata from the server, optionally filtering
// by query and limiting the number of results.
func (c *Client) ListDocuments(ctx context.Context, query, limit string) ([]api.DocumentResponse, error) {
	u := c.baseURL.JoinPath("/documents")
	q := u.Query()
	if query != "" {
		q.Set("q", query)
	}
	if limit != "" {
		q.Set("limit", limit)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server returned %s: %s", resp.Status, errResp.Error)
		}
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	var docs []api.DocumentResponse
	if err := json.NewDecoder(resp.Body).Decode(&docs); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return docs, nil
}

// GetContent downloads the raw document content stream for id.
func (c *Client) GetContent(ctx context.Context, id uuid.UUID) (io.ReadCloser, error) {
	u := c.baseURL.JoinPath("/documents", id.String(), "/content")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server returned %s: %s", resp.Status, errResp.Error)
		}
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	return resp.Body, nil
}

// GetMeta fetches document metadata without downloading the document content.
func (c *Client) GetMeta(ctx context.Context, id uuid.UUID) (api.MetaResponse, error) {
	u := c.baseURL.JoinPath("/documents", id.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return api.MetaResponse{}, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return api.MetaResponse{}, fmt.Errorf("do request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == http.StatusNotFound {
			return api.MetaResponse{}, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return api.MetaResponse{}, fmt.Errorf("server returned %s: %s", resp.Status, errResp.Error)
		}
		return api.MetaResponse{}, fmt.Errorf("server returned %s", resp.Status)
	}
	var docMeta api.MetaResponse
	if err := json.NewDecoder(resp.Body).Decode(&docMeta); err != nil {
		return api.MetaResponse{}, fmt.Errorf("decode response: %w", err)
	}
	return docMeta, nil
}

// PostDocument uploads a document as multipart form data and returns the
// metadata created by the server.
func (c *Client) PostDocument(ctx context.Context, title, filename string, body io.Reader, tags []string) (*api.DocumentResponse, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	err := writer.WriteField("title", title)
	if err != nil {
		return nil, fmt.Errorf("write title field: %w", err)
	}
	if len(tags) > 0 {
		err = writer.WriteField("tags", strings.Join(tags, ","))
		if err != nil {
			return nil, fmt.Errorf("write tags field: %w", err)
		}
	}

	form, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form: %w", err)
	}
	_, err = io.Copy(form, body)
	if err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	u := c.baseURL.JoinPath("/documents")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("server returned %s: %s", resp.Status, errResp.Error)
		}
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}

	var doc *api.DocumentResponse
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return doc, nil
}
