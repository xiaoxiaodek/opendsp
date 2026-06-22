package admanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/opendsp/opendsp/internal/biz"
)

type IqiyiClient struct {
	dspToken   string
	baseURL    string
	httpClient *http.Client
}

func NewIqiyiClient(dspToken, baseURL string) *IqiyiClient {
	if baseURL == "" {
		baseURL = "http://creative.iqiyi.com"
	}
	return &IqiyiClient{
		dspToken: dspToken,
		baseURL:  baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *IqiyiClient) Name() string { return "iqiyi" }

func (c *IqiyiClient) UploadCreative(ctx context.Context, params *biz.CreativeUploadParams) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	key := "video1"
	if params.CreativeType == 1 {
		key = "pic1"
	}
	part, err := w.CreateFormFile(key, params.FileName)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(params.FileData); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/upload/post", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("dsp_token", c.dspToken)
	req.Header.Set("ad_id", params.AdvertiserID)
	req.Header.Set("file_name", params.FileName)
	req.Header.Set("creative_type", strconv.Itoa(int(params.CreativeType)))
	req.Header.Set("platform_type", strconv.Itoa(int(params.PlatformType)))
	req.Header.Set("click_url", params.ClickURL)
	if params.DeeplinkURL != "" {
		req.Header.Set("deeplink_url", params.DeeplinkURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload creative: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		MID  string `json:"m_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("iqiyi upload creative failed: code=%d msg=%s", result.Code, result.Msg)
	}
	return result.MID, nil
}

func (c *IqiyiClient) QueryCreativeStatus(ctx context.Context, externalID string) (*biz.SyncStatusInfo, error) {
	url := fmt.Sprintf("%s/upload/api/query?dsp_token=%s&m_id=%s", c.baseURL, c.dspToken, externalID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query creative: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code   int    `json:"code"`
		Status string `json:"status"`
		TvID   string `json:"tv_id"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("query creative status failed: %d", result.Code)
	}

	return &biz.SyncStatusInfo{
		Status: result.Status,
		TVid:   result.TvID,
		Reason: result.Reason,
		Raw:    body,
	}, nil
}

func (c *IqiyiClient) BatchQueryCreativeStatus(ctx context.Context, externalIDs []string) (map[string]*biz.SyncStatusInfo, error) {
	if len(externalIDs) == 0 {
		return map[string]*biz.SyncStatusInfo{}, nil
	}
	batch := strings.Join(externalIDs, ",")
	url := fmt.Sprintf("%s/upload/api/batchQuery?dsp_token=%s&batch=%s", c.baseURL, c.dspToken, batch)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("batch query: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int              `json:"code"`
		Data []statusResult   `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	statuses := make(map[string]*biz.SyncStatusInfo, len(result.Data))
	for _, d := range result.Data {
		statuses[d.MID] = &biz.SyncStatusInfo{
			Status: d.Status,
			TVid:   d.TvID,
			Reason: d.Reason,
		}
	}
	return statuses, nil
}

type statusResult struct {
	MID    string `json:"m_id"`
	Status string `json:"status"`
	TvID   string `json:"tv_id"`
	Reason string `json:"reason"`
}

func (c *IqiyiClient) UploadAdvertiser(ctx context.Context, params *biz.AdvertiserUploadParams) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", params.FileName)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(params.FileData); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/upload/advertiser", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("dsp_token", c.dspToken)
	req.Header.Set("ad_id", strconv.FormatInt(params.AdvertiserID, 10))
	req.Header.Set("ad_name", params.Name)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload advertiser: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("iqiyi upload advertiser failed: code=%d msg=%s", result.Code, result.Msg)
	}
	return strconv.FormatInt(params.AdvertiserID, 10), nil
}

func (c *IqiyiClient) QueryAdvertiserStatus(ctx context.Context, externalAdID string) (*biz.SyncStatusInfo, error) {
	url := fmt.Sprintf("%s/upload/api/advertiser?dsp_token=%s&ad_id=%s", c.baseURL, c.dspToken, externalAdID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query advertiser: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code   int    `json:"code"`
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &biz.SyncStatusInfo{
		Status: result.Status,
		Reason: result.Reason,
		Raw:    body,
	}, nil
}
