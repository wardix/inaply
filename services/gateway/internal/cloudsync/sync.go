package cloudsync

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Delta struct {
	Seq                int64    `json:"seq"`
	Ads                []string `json:"ads"`
	Messages           []string `json:"messages"`
	ForceCheckoutRooms []string `json:"force_checkout_rooms"`
}

// Client connects outbound from the On-Premise Gateway to Cloud CMS.
type Client struct {
	baseURL    string
	propertyID string
	token      string
	httpClient *http.Client
}

func NewClient(baseURL, propertyID, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:    baseURL,
		propertyID: propertyID,
		token:      token,
		httpClient: httpClient,
	}
}

func (c *Client) PullDelta(sinceSeq int64) (*Delta, error) {
	url := fmt.Sprintf("%s/api/v1/sync/delta?since=%d", c.baseURL, sinceSeq)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Gateway-Token", c.token)
	req.Header.Set("X-Property-ID", c.propertyID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cloud sync failed with status %d", resp.StatusCode)
	}

	var d Delta
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, err
	}
	return &d, nil
}
