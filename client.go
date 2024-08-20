package comfyui

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// HandlerType is a ComfyUI API handler type.
type HandlerType string

const (
	// HandlerRawWorkflow is a ComfyUI API handler type for raw workflow.
	HandlerRawWorkflow HandlerType = "RawWorkflow"
)

// Input is a ComfyUI generation API input.
type Input struct {
	Handler   HandlerType     `json:"handler"`
	GCP       *struct{}       `json:"gcp,omitempty"`
	Modifiers *struct{}       `json:"modifiers,omitempty"`
	Workflow  json.RawMessage `json:"workflow_json"`
}

// StatusType is a ComfyUI generation status e.g. pending.
type StatusType string

const (
	// StatusPending is a ComfyUI generation API status pending.
	StatusPending StatusType = "pending"
	// StatusSuccess is a ComfyUI generation API status success.
	StatusSuccess StatusType = "success"
	// TODO: Add more status types.
)

// OutputURLs contains information about the ComfyUI generation API output URLs.
type OutputURLs struct {
	// GCP is a ComfyUI generation API GCP URL. It contains a GET-signed 7-day URL.
	GCP string `json:"gcp_url,omitempty"`
	// S3 is a ComfyUI generation API S3 URL. It contains a GET-signed 7-day URL.
	S3 string `json:"s3_url,omitempty"`
	// URL is a ComfyUI generation API URL. It contains a GET-signed 7-day URL.
	//
	// Deprecated: Use S3 instead.
	URL string `json:"url,omitempty"`
}

// OutputItem contains information about the ComfyUI generation API output.
type OutputItem struct {
	// LocalPath is a ComfyUI generation API local path.
	LocalPath string `json:"local_path,omitempty"`
	OutputURLs
}

// Status is a ComfyUI generation API status.
type Status struct {
	ID              string          `json:"id"`
	Message         string          `json:"message"`
	Status          StatusType      `json:"status"`
	ComfyUIResponse json.RawMessage `json:"comfyui_response"`
	Output          []*OutputItem   `json:"output"`
}

// Client is a ComfyUI API client.
type Client struct {
	// BaseURL is the ComfyUI API base URL. Usually ends with /api.
	BaseURL string
	// APIToken is an optional ComfyUI API token. It is used for Bearer authentication.
	APIToken string
	// Client is an optional HTTP client. If nil, http.DefaultClient is used.
	Client *http.Client
}

// NewClient returns a new ComfyUI API client.
func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL}
}

func client(c *Client) *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return http.DefaultClient
}

func do[T any](c *Client, req *http.Request, v *T) (*T, error) {
	resp, err := client(c).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		details, _ := io.ReadAll(resp.Body)
		return nil, &ClientError{Code: resp.StatusCode, Details: details}
	} else if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return nil, err
	}
	return v, nil
}

type ClientError struct {
	Code    int
	Details []byte
}

// Error implements the error interface.
func (e *ClientError) Error() string { return string(e.Details) }

// StartWorkflowRequest is a ComfyUI API start request.
type StartWorkflowRequest struct {
	Input Input `json:"input"`
}

// NewStartWorkflowRequest returns a new ComfyUI API start request.
func NewStartWorkflowRequest(workflow []byte) *StartWorkflowRequest {
	return &StartWorkflowRequest{
		Input: Input{
			Handler:   HandlerRawWorkflow,
			Workflow:  workflow,
			GCP:       new(struct{}),
			Modifiers: new(struct{}),
		},
	}
}

// StartWorkflow starts a ComfyUI API workflow.
func (c *Client) StartWorkflow(ctx context.Context, prompt *StartWorkflowRequest) (*Status, error) {
	req, err := c.newRequest(ctx, withPath("payload"), withBodyJSON(prompt))
	if err != nil {
		return nil, err
	}
	return do(c, req, new(Status))
}

func (c *Client) WorkflowStatus(ctx context.Context, id string) (*Status, error) {
	req, err := c.newRequest(ctx, withPath("result", id))
	if err != nil {
		return nil, err
	}
	return do(c, req, new(Status))
}

type newRequestOptions struct {
	Method  string
	BaseURL string
	Path    string
	Body    io.Reader
	err     error
}

type newRequestOptionFunc func(*newRequestOptions)

func withPath(path ...string) newRequestOptionFunc {
	return func(o *newRequestOptions) { o.Path, o.err = url.JoinPath(o.BaseURL, path...) }
}

func withBodyJSON(payload any) newRequestOptionFunc {
	return func(o *newRequestOptions) {
		b := new(bytes.Buffer)
		if o.err = json.NewEncoder(b).Encode(payload); o.err == nil {
			o.Body = b
		}
		o.Method = http.MethodPost
	}
}

func (c *Client) newRequest(ctx context.Context, opts ...newRequestOptionFunc) (*http.Request, error) {
	o := newRequestOptions{Method: http.MethodGet, BaseURL: c.BaseURL, Path: c.BaseURL}
	for _, f := range opts {
		f(&o)
		if o.err != nil {
			return nil, o.err
		}
	}
	req, err := http.NewRequestWithContext(ctx, o.Method, o.Path, o.Body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIToken)
	}
	return req, nil
}
