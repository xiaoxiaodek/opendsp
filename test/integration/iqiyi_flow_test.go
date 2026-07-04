package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/freq"
	"github.com/opendsp/opendsp/internal/index"
	"github.com/opendsp/opendsp/internal/service/adserver"
	iqiyipb "github.com/opendsp/opendsp/gen/platform/iqiyi"
	mockiqiyi "github.com/opendsp/opendsp/test/mock_iqiyi"
	"google.golang.org/protobuf/proto"
)

func TestIqiyiBidFlow(t *testing.T) {
	mockIQiyi := mockiqiyi.NewServer("test-dsp-token-12345")

	idx := index.New()
	freqCtrl := &freq.Controller{}

	ag := &biz.AdGroup{
		ID:         100,
		CampaignID: 10,
		Name:       "Test AdGroup",
		BidType:    biz.BidTypeCPM,
		BidPrice:   5.5,
		DailyBudget: floatPtr(1000),
		Status:     biz.CampaignStatusActive,
		Targeting:  []byte(`{"inventory":{"media":["iqiyi"],"ad_position":["pre_roll"]},"geo":{"city":["110000"]},"device":{"os":["ios"],"device_type":["mobile"]}}`),
	}

	creatives := []biz.Creative{{
		ID:            1,
		AdGroupID:     100,
		Name:          "Test Creative",
		CreativeType:  biz.CreativeTypeVideo,
		AssetURL:      "http://seaweedfs:9000/opendsp-creatives/creatives/ab/abc123.mp4",
		AssetMime:     "video/mp4",
		AssetWidth:    1920,
		AssetHeight:   1080,
		AssetDuration: 15,
		Title:         "Test Ad",
		LandingURL:    "https://example.com/landing",
		DeeplinkURL:   "myapp://open",
		ImpTracker:    "https://track.example.com/imp",
		ClickTracker:  "https://track.example.com/click",
		AuditStatus:   biz.AuditStatusApproved,
	}}

	idx.AddAdGroup(ag, creatives)

	engine := adserver.NewEngine(idx, freqCtrl, nil)
	tracker := adserver.NewTracker(freqCtrl, nil)
	server := adserver.NewServer(engine, tracker)

	mockIQiyi.SetBidHandler(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	})

	t.Run("BidRequest_NoContent_WhenPing", func(t *testing.T) {
		req := &iqiyipb.BidRequest{
			Id:     proto.String("ping-001"),
			IsPing: proto.Bool(true),
		}
		body, _ := proto.Marshal(req)
		resp := sendProtobuf(mockIQiyi, "/rtb/iqiyi", body)
		if resp.StatusCode != 204 {
			t.Fatalf("expected 204 for ping, got %d", resp.StatusCode)
		}
	})

	t.Run("BidRequest_ReturnsBidResponse", func(t *testing.T) {
		req := &iqiyipb.BidRequest{
			Id: proto.String("bid-001"),
			Device: &iqiyipb.Device{
				Os:         proto.String("ios"),
				Ip:         proto.String("1.2.3.4"),
				Idfa:       proto.String("IDFA-TEST-001"),
				PlatformId: proto.String("2"),
				Model:      proto.String("iPhone15"),
				Geo: &iqiyipb.Geo{
					City: proto.String("110000"),
				},
			},
			User: &iqiyipb.User{
				Id: proto.String("user-001"),
			},
			Site: &iqiyipb.Site{
				Content: &iqiyipb.Content{
					Title:       proto.String("Test Show"),
					ChannelId:   proto.String("drama"),
					VideoClipId: proto.String("clip-001"),
				},
			},
			Imp: []*iqiyipb.Impression{{
				Id:       proto.String("imp-001"),
				Bidfloor: proto.Float64(0.5),
				Video: &iqiyipb.Video{
					AdType:      proto.Int32(1),
					W:           proto.Int32(1920),
					H:           proto.Int32(1080),
					Minduration: proto.Int32(5),
					Maxduration: proto.Int32(30),
					AdZoneId:    proto.String("zone-001"),
				},
			}},
		}

		body, _ := proto.Marshal(req)
		resp := sendProtobuf(mockIQiyi, "/rtb/iqiyi", body)

		if resp.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 200, got %d body=%s", resp.StatusCode, string(bodyBytes))
		}

		respBody, _ := io.ReadAll(resp.Body)
		bidResp := &iqiyipb.BidResponse{}
		if err := proto.Unmarshal(respBody, bidResp); err != nil {
			t.Fatalf("unmarshal bid response: %v", err)
		}

		if bidResp.GetId() != "bid-001" {
			t.Errorf("expected id bid-001, got %s", bidResp.GetId())
		}

		if len(bidResp.GetSeatbid()) == 0 {
			t.Fatal("expected at least one seatbid")
		}

		seat := bidResp.GetSeatbid()[0]
		if len(seat.GetBid()) == 0 {
			t.Fatal("expected at least one bid")
		}

		bid := seat.GetBid()[0]
		if bid.GetImpid() != "imp-001" {
			t.Errorf("expected impid imp-001, got %s", bid.GetImpid())
		}

		if bid.GetAdm() == "" {
			t.Error("expected VAST XML in adm")
		}

		if bid.GetPrice() <= 0 {
			t.Error("expected positive price")
		}

		t.Logf("Bid response: id=%s price=%d adm_len=%d",
			bidResp.GetId(), bid.GetPrice(), len(bid.GetAdm()))
	})

	t.Run("BidRequest_NoContent_WhenNoMatch", func(t *testing.T) {
		req := &iqiyipb.BidRequest{
			Id: proto.String("bid-no-match"),
			Device: &iqiyipb.Device{
				Os:   proto.String("android"),
				Ip:   proto.String("1.2.3.4"),
				Imei: proto.String("IMEI-NO-MATCH"),
			},
			Imp: []*iqiyipb.Impression{{
				Id: proto.String("imp-no-match"),
				Video: &iqiyipb.Video{
					AdType:      proto.Int32(99),
					W:           proto.Int32(640),
					H:           proto.Int32(480),
					Minduration: proto.Int32(5),
					Maxduration: proto.Int32(30),
				},
			}},
		}

		body, _ := proto.Marshal(req)
		resp := sendProtobuf(mockIQiyi, "/rtb/iqiyi", body)

		if resp.StatusCode != 204 {
			t.Errorf("expected 204 for no match, got %d", resp.StatusCode)
		}
	})
}

func TestIqiyiAdvertiserUploadFlow(t *testing.T) {
	mockIQiyi := mockiqiyi.NewServer("test-dsp-token-12345")
	srv := httptest.NewServer(mockIQiyi)
	defer srv.Close()

	baseURL := srv.URL

	t.Run("UploadAdvertiser_Success", func(t *testing.T) {
		body := &bytes.Buffer{}
		w := multipartWriter(body)
		w.WriteField("file", "test content", "license.jpg")
		w.Close()

		req, _ := http.NewRequest("POST", baseURL+"/upload/advertiser", body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("dsp_token", "test-dsp-token-12345")
		req.Header.Set("ad_id", "adv-001")
		req.Header.Set("ad_name", "Test Advertiser")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("upload advertiser: %v", err)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if code, _ := result["code"].(float64); code != 0 {
			t.Fatalf("expected code 0, got %v: %v", code, result["msg"])
		}
	})

	t.Run("QueryAdvertiser_Approved", func(t *testing.T) {
		time.Sleep(3 * time.Second)

		resp, err := http.Get(fmt.Sprintf("%s/upload/api/advertiser?dsp_token=test-dsp-token-12345&ad_id=adv-001", baseURL))
		if err != nil {
			t.Fatalf("query advertiser: %v", err)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		status, _ := result["status"].(string)
		if status != "APPROVED" {
			t.Errorf("expected APPROVED, got %s", status)
		}
		t.Logf("Advertiser status: %s", status)
	})

	t.Run("BatchQueryAdvertisers", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/upload/api/batchAdvertiser?dsp_token=test-dsp-token-12345&batch=adv-001", baseURL))
		if err != nil {
			t.Fatalf("batch query: %v", err)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		data, _ := result["data"].([]interface{})
		if len(data) == 0 {
			t.Error("expected at least one result")
		}
		t.Logf("Batch query results: %d", len(data))
	})
}

func TestIqiyiCreativeUploadFlow(t *testing.T) {
	mockIQiyi := mockiqiyi.NewServer("test-dsp-token-12345")
	srv := httptest.NewServer(mockIQiyi)
	defer srv.Close()

	baseURL := srv.URL

	t.Run("UploadCreative_Success", func(t *testing.T) {
		body := &bytes.Buffer{}
		w := multipartWriter(body)
		w.WriteField("video1", "fake video content", "ad.mp4")
		w.Close()

		req, _ := http.NewRequest("POST", baseURL+"/upload/post", body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("dsp_token", "test-dsp-token-12345")
		req.Header.Set("ad_id", "adv-001")
		req.Header.Set("file_name", "ad.mp4")
		req.Header.Set("creative_type", "1")
		req.Header.Set("platform_type", "1")
		req.Header.Set("click_url", "https://example.com")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("upload creative: %v", err)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if code, _ := result["code"].(float64); code != 0 {
			t.Fatalf("expected code 0, got %v: %v", code, result["msg"])
		}

		mID, _ := result["m_id"].(string)
		if mID == "" {
			t.Fatal("expected m_id in response")
		}
		t.Logf("Creative m_id: %s", mID)

		t.Run("QueryCreative_Complete", func(t *testing.T) {
			time.Sleep(4 * time.Second)

			resp, err := http.Get(fmt.Sprintf("%s/upload/api/query?dsp_token=test-dsp-token-12345&m_id=%s", baseURL, mID))
			if err != nil {
				t.Fatalf("query creative: %v", err)
			}

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()

			status, _ := result["status"].(string)
			if status != "COMPLETE" {
				t.Errorf("expected COMPLETE, got %s", status)
			}

			tvID, _ := result["tv_id"].(string)
			if tvID == "" {
				t.Error("expected tv_id when COMPLETE")
			}
			t.Logf("Creative status: %s tv_id: %s", status, tvID)
		})
	})
}

func floatPtr(v float64) *float64 { return &v }

func TestAttributionFlow(t *testing.T) {
	mockIQiyi := mockiqiyi.NewServer("test-dsp-token-12345")

	idx := index.New()
	freqCtrl := &freq.Controller{}

	ag := &biz.AdGroup{
		ID:         200,
		CampaignID: 20,
		Name:       "Attribution AdGroup",
		BidType:    biz.BidTypeCPM,
		BidPrice:   10.0,
		Status:     biz.CampaignStatusActive,
		Targeting:  []byte(`{"inventory":{"media":["iqiyi"],"ad_position":["pre_roll"]},"device":{"os":["ios"]}}`),
	}

	creatives := []biz.Creative{{
		ID:            2,
		AdGroupID:     200,
		Name:          "Attribution Creative",
		CreativeType:  biz.CreativeTypeVideo,
		AssetURL:      "http://seaweedfs:9000/opendsp-creatives/creatives/cd/def456.mp4",
		AssetMime:     "video/mp4",
		AssetWidth:    1920,
		AssetHeight:   1080,
		AssetDuration: 15,
		Title:         "Attribution Test",
		LandingURL:    "https://example.com/landing",
		ImpTracker:    "https://track.example.com/imp",
		ClickTracker:  "https://track.example.com/click",
		AuditStatus:   biz.AuditStatusApproved,
	}}

	idx.AddAdGroup(ag, creatives)

	engine := adserver.NewEngine(idx, freqCtrl, nil)
	tracker := adserver.NewTracker(freqCtrl, nil)
	server := adserver.NewServer(engine, tracker)

	mockIQiyi.SetBidHandler(func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	})

	t.Run("BidResponse_ContainsClickID", func(t *testing.T) {
		req := &iqiyipb.BidRequest{
			Id: proto.String("attr-001"),
			Device: &iqiyipb.Device{
				Os:   proto.String("ios"),
				Ip:   proto.String("1.2.3.4"),
				Idfa: proto.String("IDFA-ATTR-001"),
			},
			Imp: []*iqiyipb.Impression{{
				Id: proto.String("imp-attr-001"),
				Video: &iqiyipb.Video{
					AdType:      proto.Int32(1),
					W:           proto.Int32(1920),
					H:           proto.Int32(1080),
					Minduration: proto.Int32(5),
					Maxduration: proto.Int32(30),
				},
			}},
		}

		body, _ := proto.Marshal(req)
		resp := sendProtobuf(mockIQiyi, "/rtb/iqiyi", body)

		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		respBody, _ := io.ReadAll(resp.Body)
		bidResp := &iqiyipb.BidResponse{}
		proto.Unmarshal(respBody, bidResp)

		adm := bidResp.GetSeatbid()[0].GetBid()[0].GetAdm()
		if adm == "" {
			t.Fatal("expected VAST XML")
		}

		if !strings.Contains(adm, "click_id=") {
			t.Error("VAST should contain click_id parameter")
		}

		t.Logf("VAST contains click_id: %v", strings.Contains(adm, "click_id="))
		t.Logf("VAST adm: %s", adm[:200])
	})

	t.Run("Postback_NoDB_ReturnsOK", func(t *testing.T) {
		clickID := adserver.GenerateClickID()

		req, _ := http.NewRequest("GET",
			fmt.Sprintf("/postback?click_id=%s&event_type=install&revenue=5.5&adgroup_id=200&creative_id=2&campaign_id=20&advertiser_id=1&media_id=1&ad_position_id=1",
				clickID), nil)

		w := httptest.NewRecorder()
		attrTracker := adserver.NewAttributionTracker(nil)
		attrTracker.HandlePostback(w, req)

		if w.Code != 200 {
			t.Errorf("expected 200, got %d body=%s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		json.NewDecoder(w.Body).Decode(&result)
		if result["status"] != "ok" {
			t.Errorf("expected status ok, got %v", result["status"])
		}

		t.Logf("Postback response: %v", result)
	})

	t.Run("Postback_MissingClickID_Returns400", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/postback?event_type=install", nil)
		w := httptest.NewRecorder()
		attrTracker := adserver.NewAttributionTracker(nil)
		attrTracker.HandlePostback(w, req)

		if w.Code != 400 {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("PostbackBatch_JSONBody", func(t *testing.T) {
		events := []map[string]interface{}{
			{"click_id": adserver.GenerateClickID(), "event_type": "install", "revenue": 9.99},
			{"click_id": adserver.GenerateClickID(), "event_type": "purchase", "revenue": 19.99},
		}
		body, _ := json.Marshal(events)

		req, _ := http.NewRequest("POST", "/postback/batch", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		attrTracker := adserver.NewAttributionTracker(nil)
		attrTracker.HandlePostbackBatch(w, req)

		if w.Code != 200 {
			t.Errorf("expected 200, got %d body=%s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		json.NewDecoder(w.Body).Decode(&result)
		if result["status"] != "ok" {
			t.Errorf("expected status ok, got %v", result["status"])
		}
		if result["success"].(float64) != 2 {
			t.Errorf("expected 2 successes, got %v", result["success"])
		}

		t.Logf("Batch postback response: %v", result)
	})

	t.Run("PostbackBatch_EmptyBody_Returns400", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/postback/batch", bytes.NewReader([]byte("[]")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		attrTracker := adserver.NewAttributionTracker(nil)
		attrTracker.HandlePostbackBatch(w, req)

		if w.Code != 400 {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("PostbackBatch_GET_Returns405", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/postback/batch", nil)
		w := httptest.NewRecorder()
		attrTracker := adserver.NewAttributionTracker(nil)
		attrTracker.HandlePostbackBatch(w, req)

		if w.Code != 405 {
			t.Errorf("expected 405, got %d", w.Code)
		}
	})

	t.Run("MMP_Adjust_Postback", func(t *testing.T) {
		clickID := adserver.GenerateClickID()
		url := fmt.Sprintf("/postback/mmp?platform=adjust&click_id=%s&event_name=purchase&revenue=29.99&currency=USD&adgroup_id=200&creative_id=2&campaign_id=20&advertiser_id=1&gps_adid=GAID-TEST&app_id=com.test.app&country=US&os_name=android",
			clickID)

		req, _ := http.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		attrTracker := adserver.NewAttributionTracker(nil)
		attrTracker.HandleMMPPostback(w, req)

		if w.Code != 200 {
			t.Errorf("expected 200, got %d body=%s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		json.NewDecoder(w.Body).Decode(&result)
		if result["platform"] != "adjust" {
			t.Errorf("expected platform adjust, got %v", result["platform"])
		}

		t.Logf("Adjust postback response: %v", result)
	})

	t.Run("MMP_AppsFlyer_Postback", func(t *testing.T) {
		clickID := adserver.GenerateClickID()
		url := fmt.Sprintf("/postback/mmp?platform=appsflyer&click_id=%s&event_name=af_purchase&event_revenue=49.99&adgroup_id=200&creative_id=2&campaign_id=20&advertiser_id=1&advertising_id=IDFA-TEST&app_id=id123456789&media_source=iqiyi&platform=ios&country_code=CN",
			clickID)

		req, _ := http.NewRequest("GET", url, nil)
		w := httptest.NewRecorder()
		attrTracker := adserver.NewAttributionTracker(nil)
		attrTracker.HandleMMPPostback(w, req)

		if w.Code != 200 {
			t.Errorf("expected 200, got %d body=%s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		json.NewDecoder(w.Body).Decode(&result)
		if result["platform"] != "appsflyer" {
			t.Errorf("expected platform appsflyer, got %v", result["platform"])
		}

		t.Logf("AppsFlyer postback response: %v", result)
	})

	t.Run("MMP_UnknownPlatform_Returns400", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/postback/mmp?platform=unknown&click_id=test", nil)
		w := httptest.NewRecorder()
		attrTracker := adserver.NewAttributionTracker(nil)
		attrTracker.HandleMMPPostback(w, req)

		if w.Code != 400 {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("ClickID_IsUnique", func(t *testing.T) {
		id1 := adserver.GenerateClickID()
		id2 := adserver.GenerateClickID()
		if id1 == id2 {
			t.Error("click IDs should be unique")
		}
		if len(id1) != 32 {
			t.Errorf("expected 32 hex chars, got %d", len(id1))
		}
		t.Logf("Click IDs: %s / %s", id1, id2)
	})

	t.Run("AttributionWindow_Default", func(t *testing.T) {
		tracker := adserver.NewAttributionTracker(nil)
		if tracker == nil {
			t.Fatal("expected non-nil tracker")
		}
		t.Logf("Attribution tracker created with default windows")
	})

	t.Run("AttributionWindow_Custom", func(t *testing.T) {
		tracker := adserver.NewAttributionTrackerWithWindows(nil, 24*time.Hour, 1*time.Hour)
		if tracker == nil {
			t.Fatal("expected non-nil tracker")
		}
		t.Logf("Attribution tracker created with custom windows")
	})
}

func sendProtobuf(handler http.Handler, path string, body []byte) *http.Response {
	req := httptest.NewRequest("POST", path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Result()
}

type multipartWriterHelper struct {
	buf      *bytes.Buffer
	boundary string
}

func multipartWriter(buf *bytes.Buffer) *multipartWriterHelper {
	return &multipartWriterHelper{buf: buf, boundary: "test-boundary-12345"}
}

func (w *multipartWriterHelper) WriteField(fieldname, content, filename string) {
	w.buf.WriteString(fmt.Sprintf("--%s\r\n", w.boundary))
	w.buf.WriteString(fmt.Sprintf("Content-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\n", fieldname, filename))
	w.buf.WriteString("Content-Type: application/octet-stream\r\n\r\n")
	w.buf.WriteString(content)
	w.buf.WriteString("\r\n")
}

func (w *multipartWriterHelper) Close() {
	w.buf.WriteString(fmt.Sprintf("--%s--\r\n", w.boundary))
}

func (w *multipartWriterHelper) FormDataContentType() string {
	return fmt.Sprintf("multipart/form-data; boundary=%s", w.boundary)
}
