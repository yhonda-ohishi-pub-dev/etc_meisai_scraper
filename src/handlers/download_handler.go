package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/services"
)

// DownloadHandler はダウンロード関連のハンドラー
type DownloadHandler struct {
	DownloadService services.DownloadServiceInterface
}

// DownloadRequest はダウンロードリクエスト
type DownloadRequest struct {
	Accounts []string `json:"accounts"`
	FromDate string   `json:"from_date"`
	ToDate   string   `json:"to_date"`
	Mode     string   `json:"mode"`
}

// JobStatus はジョブステータス
type JobStatus struct {
	JobID        string     `json:"job_id"`
	Status       string     `json:"status"`
	Progress     int        `json:"progress"`
	TotalRecords int        `json:"total_records"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// NewDownloadHandler creates a new download handler
func NewDownloadHandler(downloadService services.DownloadServiceInterface) *DownloadHandler {
	return &DownloadHandler{
		DownloadService: downloadService,
	}
}

// DownloadSync は同期ダウンロードを実行
func (h *DownloadHandler) DownloadSync(w http.ResponseWriter, r *http.Request) {
	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// パラメータのデフォルト値設定
	if req.FromDate == "" || req.ToDate == "" {
		now := time.Now()
		if req.ToDate == "" {
			req.ToDate = now.Format("2006-01-02")
		}
		if req.FromDate == "" {
			lastMonth := now.AddDate(0, -1, 0)
			req.FromDate = lastMonth.Format("2006-01-02")
		}
	}

	if len(req.Accounts) == 0 {
		h.respondError(w, http.StatusBadRequest, "At least one account is required")
		return
	}

	// TODO: 実際のダウンロード処理を実装
	response := map[string]interface{}{
		"success":      true,
		"record_count": 0,
		"csv_path":     "",
		"records":      []interface{}{},
		"message":      "Download completed successfully",
	}

	h.respondJSON(w, http.StatusOK, response)
}

// DownloadAsync は非同期ダウンロードを開始
func (h *DownloadHandler) DownloadAsync(w http.ResponseWriter, r *http.Request) {
	var req DownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// パラメータのデフォルト値設定
	if req.FromDate == "" || req.ToDate == "" {
		now := time.Now()
		if req.ToDate == "" {
			req.ToDate = now.Format("2006-01-02")
		}
		if req.FromDate == "" {
			lastMonth := now.AddDate(0, -1, 0)
			req.FromDate = lastMonth.Format("2006-01-02")
		}
	}

	if len(req.Accounts) == 0 {
		// デフォルトで全アカウントを使用
		req.Accounts = h.DownloadService.GetAllAccountIDs()
		if len(req.Accounts) == 0 {
			h.respondError(w, http.StatusBadRequest, "No accounts configured")
			return
		}
	}

	// ジョブIDを生成
	jobID := uuid.New().String()

	// 非同期でダウンロード開始
	h.DownloadService.ProcessAsync(jobID, req.Accounts, req.FromDate, req.ToDate)

	response := map[string]interface{}{
		"job_id":  jobID,
		"status":  "pending",
		"message": "Download job started",
	}

	h.respondJSON(w, http.StatusAccepted, response)
}

// GetDownloadStatus はダウンロードステータスを取得
func (h *DownloadHandler) GetDownloadStatus(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		// URLパスから取得を試みる
		// 例: /api/download/status/{jobId}
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) > 0 {
			jobID = parts[len(parts)-1]
		}
	}

	if jobID == "" || jobID == "status" {
		h.respondError(w, http.StatusBadRequest, "Job ID is required")
		return
	}

	// ジョブステータスを取得
	job, exists := h.DownloadService.GetJobStatus(jobID)
	if !exists {
		h.respondError(w, http.StatusNotFound, fmt.Sprintf("Job %s not found", jobID))
		return
	}

	status := JobStatus{
		JobID:        job.ID,
		Status:       job.Status,
		Progress:     job.Progress,
		TotalRecords: job.TotalRecords,
		CompletedAt:  job.CompletedAt,
	}

	if job.ErrorMessage != "" {
		status.ErrorMessage = &job.ErrorMessage
	}

	h.respondJSON(w, http.StatusOK, status)
}

// Helper methods
func (h *DownloadHandler) respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (h *DownloadHandler) respondError(w http.ResponseWriter, code int, message string) {
	h.respondJSON(w, code, map[string]string{"error": message})
}