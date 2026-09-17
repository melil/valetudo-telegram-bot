package valetudo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) TriggerCapabilityAction(capability string, action string) error {
	url := c.baseURL + "/capabilities/" + capability
	payload := fmt.Sprintf(`{"action":"%s"}`, action)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) SetPreset(capability string, value string) error {
	url := c.baseURL + "/capabilities/" + capability + "/preset"
	payload := fmt.Sprintf(`{"name":"%s"}`, value)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) TriggerAction(action string) error {
	return c.TriggerCapabilityAction("BasicControlCapability", action)
}

func (c *Client) TriggerLocate() error {
	return c.TriggerCapabilityAction("LocateCapability", "locate")
}

func (c *Client) SetOperationMode(mode string) error {
	return c.SetPreset("OperationModeControlCapability", mode)
}

func (c *Client) CleanSegments(segmentIDs []string, iterations int) error {
	url := c.baseURL + "/capabilities/MapSegmentationCapability"
	payload := CleanSegmentPayload{
		Action:     "start_segment_action",
		SegmentIDs: segmentIDs,
		Iterations: iterations,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) ResetConsumable(cType string, cSubType string) error {
	url := c.baseURL + "/capabilities/ConsumableMonitoringCapability"
	payload := fmt.Sprintf(`{"action":"reset", "type":"%s", "subType":"%s"}`, cType, cSubType)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) GetAttributes() ([]GenericAttribute, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/state/attributes")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var attrs []GenericAttribute
	if err := json.NewDecoder(resp.Body).Decode(&attrs); err != nil {
		return nil, err
	}
	return attrs, nil
}

func (c *Client) GetConsumables() ([]ConsumableItem, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/capabilities/ConsumableMonitoringCapability")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var items []ConsumableItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) GetCurrentSessionStats() (min int, sec int, areaM2 float64) {
	resp, err := c.httpClient.Get(c.baseURL + "/capabilities/CurrentStatisticsCapability")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var points []DataPoint
	if err := json.NewDecoder(resp.Body).Decode(&points); err == nil {
		for _, p := range points {
			switch p.Type {
			case "time":
				min = p.Value / 60
				sec = p.Value % 60
			case "area":
				areaM2 = float64(p.Value) / 10000.0
			}
		}
	}
	return
}

func (c *Client) GetTotalStats() (totalHours int, totalCount int, totalAreaM2 float64) {
	resp, err := c.httpClient.Get(c.baseURL + "/capabilities/TotalStatisticsCapability")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var points []DataPoint
	if err := json.NewDecoder(resp.Body).Decode(&points); err == nil {
		for _, p := range points {
			switch p.Type {
			case "time":
				totalHours = p.Value / 3600
			case "count":
				totalCount = p.Value
			case "area":
				totalAreaM2 = float64(p.Value) / 10000.0
			}
		}
	}
	return
}

func (c *Client) GetMapReader() (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/state/map", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "image/png")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return resp.Body, nil
}
