package admanager

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/opendsp/opendsp/internal/biz"
	"github.com/opendsp/opendsp/internal/middleware"
	pb "github.com/opendsp/opendsp/gen/filegateway/v1"
)

type UploadHandler struct {
	fileGateway     pb.FileGatewayClient
	proofMaterialUC *biz.ProofMaterialUseCase
}

func NewUploadHandler(fg pb.FileGatewayClient, pmUC *biz.ProofMaterialUseCase) *UploadHandler {
	return &UploadHandler{
		fileGateway:     fg,
		proofMaterialUC: pmUC,
	}
}

func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}

	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tokenStr == "" {
		writeJSON(w, 401, map[string]string{"error": "missing authorization"})
		return
	}
	claims, err := middleware.ParseToken(tokenStr)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "invalid token"})
		return
	}
	_ = claims

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/upload")

	switch {
	case strings.HasPrefix(path, "/creative"):
		h.handleCreativeUpload(w, r)
	case strings.HasPrefix(path, "/proof"):
		h.handleProofUpload(w, r)
	default:
		http.Error(w, "unknown upload type", 400)
	}
}

func (h *UploadHandler) handleCreativeUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(200 << 20); err != nil {
		writeJSON(w, 400, map[string]string{"error": "parse form: " + err.Error()})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "missing file"})
		return
	}
	defer file.Close()

	if err := validateCreativeFile(header); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	mimeType := detectMimeType(header.Filename)

	resp, err := h.fileGateway.CreateUploadURL(r.Context(), &pb.CreateUploadURLReq{
		Namespace:   "creative",
		Filename:    header.Filename,
		ContentType: mimeType,
		FileSize:    header.Size,
	})
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "create upload: " + err.Error()})
		return
	}

	// Forward file directly to file-gateway HTTP endpoint (bypass presigned URL)
	fileGateHost := os.Getenv("FILE_GATEWAY_HTTP_ADDR")
	if fileGateHost == "" {
		fileGateHost = "file-gateway:9001"
	}
	putURL := fmt.Sprintf("http://%s/upload/%s/%s", fileGateHost, "creative", resp.FileId)

	putReq, err := http.NewRequestWithContext(r.Context(), "PUT", putURL, file)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "forward upload: " + err.Error()})
		return
	}
	putReq.ContentLength = header.Size
	putReq.Header.Set("Content-Type", mimeType)

	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "forward upload: " + err.Error()})
		return
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("upload: %s", string(body))})
		return
	}

	pubURL := os.Getenv("ASSETS_PUBLIC_URL") + resp.PublicPath
	writeJSON(w, 200, map[string]interface{}{
		"file_id":      resp.FileId,
		"public_path":  pubURL,
		"filename":     header.Filename,
		"size":         header.Size,
		"content_type": mimeType,
	})
}

func (h *UploadHandler) handleProofUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeJSON(w, 400, map[string]string{"error": "parse form: " + err.Error()})
		return
	}

	advertiserIDStr := r.FormValue("advertiser_id")
	materialTypeStr := r.FormValue("material_type")

	advertiserID, err := strconv.ParseInt(advertiserIDStr, 10, 64)
	if err != nil || advertiserID <= 0 {
		writeJSON(w, 400, map[string]string{"error": "invalid advertiser_id"})
		return
	}
	materialType, err := strconv.Atoi(materialTypeStr)
	if err != nil || materialType < 1 || materialType > 4 {
		writeJSON(w, 400, map[string]string{"error": "invalid material_type"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "missing file"})
		return
	}
	defer file.Close()

	if err := validateProofFile(header); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	resp, err := h.fileGateway.CreateUploadURL(r.Context(), &pb.CreateUploadURLReq{
		Namespace:   "proof",
		Filename:    header.Filename,
		ContentType: "application/octet-stream",
		FileSize:    header.Size,
	})
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "create upload: " + err.Error()})
		return
	}

	// Forward file directly to file-gateway HTTP endpoint
	fileGateHost := os.Getenv("FILE_GATEWAY_HTTP_ADDR")
	if fileGateHost == "" {
		fileGateHost = "file-gateway:9001"
	}
	putURL := fmt.Sprintf("http://%s/upload/%s/%s", fileGateHost, "proof", resp.FileId)

	putReq, err := http.NewRequestWithContext(r.Context(), "PUT", putURL, file)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "forward upload: " + err.Error()})
		return
	}
	putReq.ContentLength = header.Size
	putReq.Header.Set("Content-Type", "application/octet-stream")

	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "forward upload: " + err.Error()})
		return
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(putResp.Body)
		writeJSON(w, 500, map[string]string{"error": fmt.Sprintf("upload: %s", string(body))})
		return
	}

	fileSize := int32(header.Size)
	fileName := header.Filename
	m := &biz.ProofMaterial{
		AdvertiserID: advertiserID,
		MaterialType: int16(materialType),
		FileURL:      resp.PublicPath,
		FileName:     &fileName,
		FileSize:     &fileSize,
	}
	if err := h.proofMaterialUC.Upload(r.Context(), m); err != nil {
		writeJSON(w, 500, map[string]string{"error": "save proof failed: " + err.Error()})
		return
	}

	writeJSON(w, 200, map[string]interface{}{
		"file_id":      resp.FileId,
		"public_path":  resp.PublicPath,
		"filename":     header.Filename,
		"size":         header.Size,
		"material_id":  m.ID,
	})
}

var creativeImageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
}

var creativeVideoExts = map[string]bool{
	".mp4": true, ".flv": true, ".webm": true, ".mov": true,
}

var proofAllowedExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".bmp": true,
	".pdf": true, ".zip": true, ".rar": true,
}

const (
	maxImageSize = 10 << 20
	maxVideoSize = 200 << 20
	maxProofSize = 50 << 20
)

func validateCreativeFile(header *multipart.FileHeader) error {
	ext := strings.ToLower(filepath.Ext(header.Filename))

	if creativeImageExts[ext] {
		if header.Size > maxImageSize {
			return fmt.Errorf("image too large: %d bytes (max %d)", header.Size, maxImageSize)
		}
		return nil
	}

	if creativeVideoExts[ext] {
		if header.Size > maxVideoSize {
			return fmt.Errorf("video too large: %d bytes (max %d)", header.Size, maxVideoSize)
		}
		return nil
	}

	return fmt.Errorf("file extension not allowed: %s", ext)
}

func validateProofFile(header *multipart.FileHeader) error {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !proofAllowedExts[ext] {
		return fmt.Errorf("file extension not allowed: %s", ext)
	}
	if header.Size > maxProofSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", header.Size, maxProofSize)
	}
	return nil
}

func detectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".flv":
		return "video/x-flv"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	default:
		return "application/octet-stream"
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
