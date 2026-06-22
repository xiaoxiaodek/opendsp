package mockiqiyi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	iqiyipb "github.com/opendsp/opendsp/gen/platform/iqiyi"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	mu              sync.RWMutex
	advertisers     map[string]*AdvertiserRecord
	creatives       map[string]*CreativeRecord
	dspToken        string
	bidHandler      http.HandlerFunc
}

type AdvertiserRecord struct {
	AdID     string
	Name     string
	Status   string
	Reason   string
	Files    []string
	UploadAt time.Time
}

type CreativeRecord struct {
	MID      string
	TVID     string
	AdID     string
	Status   string
	Reason   string
	Files    []string
	UploadAt time.Time
}

func NewServer(dspToken string) *Server {
	s := &Server{
		advertisers: make(map[string]*AdvertiserRecord),
		creatives:   make(map[string]*CreativeRecord),
		dspToken:    dspToken,
	}
	return s
}

func (s *Server) SetBidHandler(h http.HandlerFunc) {
	s.bidHandler = h
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/rtb/iqiyi" || path == "/dsp/bid":
		if s.bidHandler != nil {
			s.bidHandler(w, r)
		} else {
			s.handleBid(w, r)
		}
	case path == "/upload/advertiser":
		s.handleAdvertiserUpload(w, r)
	case path == "/upload/api/advertiser":
		s.handleAdvertiserQuery(w, r)
	case path == "/upload/api/batchAdvertiser":
		s.handleAdvertiserBatchQuery(w, r)
	case path == "/upload/post":
		s.handleCreativeUpload(w, r)
	case path == "/upload/api/query":
		s.handleCreativeQuery(w, r)
	case path == "/upload/api/batchQuery":
		s.handleCreativeBatchQuery(w, r)
	case path == "/upload/api/queryByStatus":
		s.handleCreativeQueryByStatus(w, r)
	case path == "/upload/api/offline/query":
		s.handleOfflineQuery(w, r)
	default:
		http.Error(w, "not found", 404)
	}
}

func (s *Server) handleBid(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", 400)
		return
	}

	req := &iqiyipb.BidRequest{}
	if err := proto.Unmarshal(body, req); err != nil {
		log.Printf("mock iqiyi: parse bid request: %v", err)
		http.Error(w, "parse failed", 400)
		return
	}

	if req.GetIsPing() {
		w.WriteHeader(204)
		return
	}

	log.Printf("mock iqiyi: received bid request id=%s imps=%d", req.GetId(), len(req.GetImp()))

	impCount := len(req.GetImp())
	if impCount == 0 {
		w.WriteHeader(204)
		return
	}

	resp := &iqiyipb.BidResponse{
		Id:               req.Id,
		ProcessingTimeMs: proto.Int32(50),
	}

	for _, imp := range req.GetImp() {
		price := int32(500)
		if imp.GetBidfloor() > 0 {
			price = int32(imp.GetBidfloor() * 1.2 * 100)
		}

		adm := buildMockVast(imp.GetId(), imp.GetVideo())

		seatBid := &iqiyipb.Seatbid{
			Bid: []*iqiyipb.Bid{{
				Id:                     proto.String(fmt.Sprintf("mock-bid-%s", imp.GetId())),
				Impid:                  imp.Id,
				Price:                  proto.Int32(price),
				Adm:                    proto.String(adm),
				Crid:                   proto.String("mock-creative-001"),
				IsPrecisionAdvertising: proto.Bool(true),
			}},
		}
		resp.Seatbid = append(resp.Seatbid, seatBid)
	}

	out, err := proto.Marshal(resp)
	if err != nil {
		http.Error(w, "marshal failed", 500)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Write(out)
}

func buildMockVast(impID string, video *iqiyipb.Video) string {
	duration := 15
	if video != nil && video.GetMaxduration() > 0 {
		duration = int(video.GetMaxduration())
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<VAST version="3.0">
  <Ad id="%s">
    <InLine>
      <AdSystem>MockDSP</AdSystem>
      <AdTitle>Mock Ad</AdTitle>
      <Impression><![CDATA[https://mock.iqiyi.com/track/imp?id=%s]]></Impression>
      <Creatives>
        <Creative>
          <Linear>
            <Duration>00:00:%02d</Duration>
            <MediaFiles>
              <MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">
                <![CDATA[https://mock.iqiyi.com/creatives/test.mp4]]>
              </MediaFile>
            </MediaFiles>
          </Linear>
        </Creative>
      </Creatives>
    </InLine>
  </Ad>
</VAST>`, impID, impID, duration)
}

func (s *Server) handleAdvertiserUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	token := r.Header.Get("dsp_token")
	if token != s.dspToken {
		writeJSON(w, 403, map[string]interface{}{"code": 1, "msg": "invalid dsp_token"})
		return
	}

	adID := r.Header.Get("ad_id")
	name := r.Header.Get("ad_name")
	if adID == "" || name == "" {
		writeJSON(w, 400, map[string]interface{}{"code": 1, "msg": "missing ad_id or ad_name"})
		return
	}

	if err := r.ParseMultipartForm(200 << 20); err != nil {
		writeJSON(w, 400, map[string]interface{}{"code": 1, "msg": "parse multipart failed"})
		return
	}

	var files []string
	for _, fhs := range r.MultipartForm.File {
		for _, fh := range fhs {
			files = append(files, fh.Filename)
		}
	}

	s.mu.Lock()
	s.advertisers[adID] = &AdvertiserRecord{
		AdID:     adID,
		Name:     name,
		Status:   "PENDING",
		Files:    files,
		UploadAt: time.Now(),
	}
	s.mu.Unlock()

	log.Printf("mock iqiyi: advertiser uploaded ad_id=%s name=%s files=%v", adID, name, files)

	go func() {
		time.Sleep(2 * time.Second)
		s.mu.Lock()
		if rec, ok := s.advertisers[adID]; ok && rec.Status == "PENDING" {
			rec.Status = "APPROVED"
		}
		s.mu.Unlock()
		log.Printf("mock iqiyi: advertiser auto-approved ad_id=%s", adID)
	}()

	writeJSON(w, 200, map[string]interface{}{"code": 0, "msg": "success"})
}

func (s *Server) handleAdvertiserQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}

	token := r.URL.Query().Get("dsp_token")
	adID := r.URL.Query().Get("ad_id")
	if token != s.dspToken || adID == "" {
		writeJSON(w, 400, map[string]interface{}{"code": 1, "msg": "invalid params"})
		return
	}

	s.mu.RLock()
	rec, ok := s.advertisers[adID]
	s.mu.RUnlock()

	if !ok {
		writeJSON(w, 200, map[string]interface{}{"code": 1, "msg": "not found"})
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"code":   0,
		"ad_id":  rec.AdID,
		"status": rec.Status,
		"reason": rec.Reason,
	})
}

func (s *Server) handleAdvertiserBatchQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}

	token := r.URL.Query().Get("dsp_token")
	batch := r.URL.Query().Get("batch")
	if token != s.dspToken || batch == "" {
		writeJSON(w, 400, map[string]interface{}{"code": 1, "msg": "invalid params"})
		return
	}

	ids := strings.Split(batch, ",")
	var results []map[string]interface{}

	s.mu.RLock()
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if rec, ok := s.advertisers[id]; ok {
			results = append(results, map[string]interface{}{
				"ad_id":  rec.AdID,
				"status": rec.Status,
				"reason": rec.Reason,
			})
		}
	}
	s.mu.RUnlock()

	writeJSON(w, 200, map[string]interface{}{"code": 0, "data": results})
}

func (s *Server) handleCreativeUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	token := r.Header.Get("dsp_token")
	if token != s.dspToken {
		writeJSON(w, 403, map[string]interface{}{"code": 1, "msg": "invalid dsp_token"})
		return
	}

	adID := r.Header.Get("ad_id")
	fileName := r.Header.Get("file_name")
	if adID == "" || fileName == "" {
		writeJSON(w, 400, map[string]interface{}{"code": 1, "msg": "missing ad_id or file_name"})
		return
	}

	if err := r.ParseMultipartForm(200 << 20); err != nil {
		writeJSON(w, 400, map[string]interface{}{"code": 1, "msg": "parse multipart failed"})
		return
	}

	var files []string
	for _, fhs := range r.MultipartForm.File {
		for _, fh := range fhs {
			files = append(files, fh.Filename)
		}
	}

	mID := fmt.Sprintf("%d", time.Now().UnixNano()/1000)
	tvID := fmt.Sprintf("tv_%s", mID)

	s.mu.Lock()
	s.creatives[mID] = &CreativeRecord{
		MID:      mID,
		TVID:     tvID,
		AdID:     adID,
		Status:   "PENDING",
		Files:    files,
		UploadAt: time.Now(),
	}
	s.mu.Unlock()

	log.Printf("mock iqiyi: creative uploaded m_id=%s ad_id=%s files=%v", mID, adID, files)

	go func() {
		time.Sleep(3 * time.Second)
		s.mu.Lock()
		if rec, ok := s.creatives[mID]; ok && rec.Status == "PENDING" {
			rec.Status = "COMPLETE"
		}
		s.mu.Unlock()
		log.Printf("mock iqiyi: creative auto-completed m_id=%s tv_id=%s", mID, tvID)
	}()

	writeJSON(w, 200, map[string]interface{}{
		"code": 0,
		"m_id": mID,
		"msg":  "success",
	})
}

func (s *Server) handleCreativeQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}

	token := r.URL.Query().Get("dsp_token")
	mID := r.URL.Query().Get("m_id")
	if token != s.dspToken || mID == "" {
		writeJSON(w, 400, map[string]interface{}{"code": 1, "msg": "invalid params"})
		return
	}

	s.mu.RLock()
	rec, ok := s.creatives[mID]
	s.mu.RUnlock()

	if !ok {
		writeJSON(w, 200, map[string]interface{}{"code": 1, "msg": "not found"})
		return
	}

	resp := map[string]interface{}{
		"code":   0,
		"m_id":   rec.MID,
		"status": rec.Status,
	}
	if rec.Status == "COMPLETE" {
		resp["tv_id"] = rec.TVID
	}
	if rec.Reason != "" {
		resp["reason"] = rec.Reason
	}

	writeJSON(w, 200, resp)
}

func (s *Server) handleCreativeBatchQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}

	token := r.URL.Query().Get("dsp_token")
	batch := r.URL.Query().Get("batch")
	if token != s.dspToken || batch == "" {
		writeJSON(w, 400, map[string]interface{}{"code": 1, "msg": "invalid params"})
		return
	}

	ids := strings.Split(batch, ",")
	var results []map[string]interface{}

	s.mu.RLock()
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if rec, ok := s.creatives[id]; ok {
			r := map[string]interface{}{
				"m_id":   rec.MID,
				"status": rec.Status,
			}
			if rec.Status == "COMPLETE" {
				r["tv_id"] = rec.TVID
			}
			results = append(results, r)
		}
	}
	s.mu.RUnlock()

	writeJSON(w, 200, map[string]interface{}{"code": 0, "data": results})
}

func (s *Server) handleCreativeQueryByStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}

	token := r.URL.Query().Get("dsp_token")
	status := r.URL.Query().Get("status")
	if token != s.dspToken || status == "" {
		writeJSON(w, 400, map[string]interface{}{"code": 1, "msg": "invalid params"})
		return
	}

	var mids []string
	s.mu.RLock()
	for _, rec := range s.creatives {
		if rec.Status == status {
			mids = append(mids, rec.MID)
		}
	}
	s.mu.RUnlock()

	writeJSON(w, 200, map[string]interface{}{"code": 0, "data": mids})
}

func (s *Server) handleOfflineQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}

	token := r.URL.Query().Get("dsp_token")
	if token != s.dspToken {
		writeJSON(w, 403, map[string]interface{}{"code": 1, "msg": "invalid dsp_token"})
		return
	}

	var offline []map[string]interface{}
	s.mu.RLock()
	for _, rec := range s.creatives {
		if rec.Status == "OFFLINE" {
			offline = append(offline, map[string]interface{}{
				"m_id":   rec.MID,
				"tv_id":  rec.TVID,
				"status": rec.Status,
				"reason": rec.Reason,
			})
		}
	}
	s.mu.RUnlock()

	writeJSON(w, 200, map[string]interface{}{"code": 0, "data": offline})
}

func (s *Server) GetAdvertiser(adID string) *AdvertiserRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.advertisers[adID]
}

func (s *Server) GetCreative(mID string) *CreativeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.creatives[mID]
}

func (s *Server) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advertisers = make(map[string]*AdvertiserRecord)
	s.creatives = make(map[string]*CreativeRecord)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func init() {
	_ = strconv.Itoa
}
