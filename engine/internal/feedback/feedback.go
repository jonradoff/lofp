package feedback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Client struct {
	url    string
	apiKey string
	http   *http.Client
}

// BatchItem is the feedback request payload sent to VibeCtl.
type BatchItem struct {
	ProjectCode string         `json:"projectCode"`
	RawContent  string         `json:"rawContent"`
	SourceType  string         `json:"sourceType"`
	SubmittedBy string         `json:"submittedBy,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func New(url, apiKey string) *Client {
	if url == "" || apiKey == "" {
		return &Client{}
	}
	return &Client{
		url:    url,
		apiKey: apiKey,
		http:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c.url != "" && c.apiKey != ""
}

func (c *Client) Submit(playerName string, roomNum int, roomName, reportText string) {
	if !c.Enabled() {
		return
	}
	go c.submit(playerName, roomNum, roomName, reportText)
}

func (c *Client) submit(playerName string, roomNum int, roomName, reportText string) {
	req := BatchItem{
		ProjectCode: "LOFP",
		RawContent:  reportText,
		SourceType:  "in_game",
		SubmittedBy: playerName,
		Metadata: map[string]any{
			"roomNum":  roomNum,
			"roomName": roomName,
			"source":   "report_command",
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		log.Printf("feedback: marshal error: %v", err)
		return
	}
	httpReq, err := http.NewRequest("POST", c.url+"/api/v1/feedback", bytes.NewReader(body))
	if err != nil {
		log.Printf("feedback: request error: %v", err)
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		log.Printf("feedback: submit error: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("feedback: vibectl returned %d", resp.StatusCode)
	}
}

// SubmitBatch sends multiple feedback items for backfill purposes (synchronous).
func (c *Client) SubmitBatch(items []BatchItem) error {
	if !c.Enabled() {
		return fmt.Errorf("feedback client not configured")
	}
	body, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	httpReq, err := http.NewRequest("POST", c.url+"/api/v1/feedback/batch", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("submit: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("vibectl returned %d", resp.StatusCode)
	}
	return nil
}

// NewBatchItem creates a request suitable for SubmitBatch.
func NewBatchItem(playerName string, roomNum int, roomName, reportText string) BatchItem {
	return BatchItem{
		ProjectCode: "LOFP",
		RawContent:  reportText,
		SourceType:  "in_game",
		SubmittedBy: playerName,
		Metadata: map[string]any{
			"roomNum":  roomNum,
			"roomName": roomName,
			"source":   "report_command",
		},
	}
}
