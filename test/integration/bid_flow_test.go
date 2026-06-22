package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

const baseURL = "http://localhost:8080"

func TestBidFlow(t *testing.T) {
	t.Skip("requires running services - run 'docker compose up -d' first")

	// Wait for server to be ready
	time.Sleep(5 * time.Second)

	// Test 1: Health check
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health check returned %d", resp.StatusCode)
	}
	t.Log("health check: PASS")

	// Test 2: OpenRTB bid request
	bidReq := map[string]interface{}{
		"id": "test-request-1",
		"imp": []map[string]interface{}{
			{
				"id":       "imp-1",
				"bidfloor": 0.05,
				"video": map[string]interface{}{
					"w":           1920,
					"h":           1080,
					"minduration": 5,
					"maxduration": 30,
				},
			},
		},
		"device": map[string]interface{}{
			"ua":         "Mozilla/5.0",
			"ip":         "1.2.3.4",
			"os":         "android",
			"devicetype": 1,
		},
		"user": map[string]interface{}{
			"buyeruid": "test-user-1",
		},
		"site": map[string]interface{}{
			"domain": "iqiyi",
		},
	}

	body, _ := json.Marshal(bidReq)
	resp, err = http.Post(baseURL+"/rtb/openrtb", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("bid request failed: %v", err)
	}
	defer resp.Body.Close()

	t.Logf("bid response status: %d", resp.StatusCode)
	if resp.StatusCode == http.StatusNoContent {
		t.Log("no bid (expected if no ads configured)")
	} else if resp.StatusCode == http.StatusOK {
		var bidResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&bidResp)
		t.Logf("bid response: %+v", bidResp)
	}

	// Test 3: Impression tracking
	impURL := fmt.Sprintf("%s/track/impression?adgroup_id=1&creative_id=1&price=0.05&uid=test-user-1", baseURL)
	resp, err = http.Get(impURL)
	if err != nil {
		t.Fatalf("impression tracking failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("impression tracking returned %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType != "image/gif" {
		t.Fatalf("expected image/gif, got %s", contentType)
	}
	t.Log("impression tracking: PASS")

	// Test 4: Click tracking
	clickURL := fmt.Sprintf("%s/track/click?adgroup_id=1&creative_id=1&url=https://example.com", baseURL)
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err = client.Get(clickURL)
	if err != nil {
		t.Fatalf("click tracking failed: %v", err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("click tracking returned %d, expected 302", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location != "https://example.com" {
		t.Fatalf("expected redirect to https://example.com, got %s", location)
	}
	t.Log("click tracking: PASS")
}
