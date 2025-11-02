package services

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	pb "github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DownloadServiceGRPC はgRPC対応のダウンロードサービス
type DownloadServiceGRPC struct {
	pb.UnimplementedDownloadServiceServer
	downloadService DownloadServiceInterface
	logBuffer       *LogBuffer
}

// LogBuffer はログを保持するリングバッファ
type LogBuffer struct {
	lines    []string
	maxLines int
	mu       sync.RWMutex
}

// NewLogBuffer creates a new log buffer
func NewLogBuffer(maxLines int) *LogBuffer {
	return &LogBuffer{
		lines:    make([]string, 0, maxLines),
		maxLines: maxLines,
	}
}

// Add adds a log line to the buffer
func (lb *LogBuffer) Add(line string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.lines = append(lb.lines, line)
	if len(lb.lines) > lb.maxLines {
		lb.lines = lb.lines[1:]
	}
}

// GetTail returns the last N lines
func (lb *LogBuffer) GetTail(n int) []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if n <= 0 || n > len(lb.lines) {
		n = len(lb.lines)
	}

	start := len(lb.lines) - n
	if start < 0 {
		start = 0
	}

	result := make([]string, n)
	copy(result, lb.lines[start:])
	return result
}

// GetAll returns all lines
func (lb *LogBuffer) GetAll() []string {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	result := make([]string, len(lb.lines))
	copy(result, lb.lines)
	return result
}

// NewDownloadServiceGRPC creates a new gRPC download service
func NewDownloadServiceGRPC(db *sql.DB, logger *log.Logger) *DownloadServiceGRPC {
	grpcService := &DownloadServiceGRPC{
		downloadService: NewDownloadService(db, logger),
		logBuffer:       NewLogBuffer(1000), // 最大1000行保持
	}

	// ログコールバックを設定
	grpcService.downloadService.SetLogCallback(func(msg string) {
		grpcService.logBuffer.Add(msg)
	})

	return grpcService
}

// NewDownloadServiceGRPCWithMock creates a new gRPC download service with a custom download service
func NewDownloadServiceGRPCWithMock(downloadService DownloadServiceInterface) *DownloadServiceGRPC {
	return &DownloadServiceGRPC{
		downloadService: downloadService,
	}
}

// DownloadSync は同期ダウンロードを実行
func (s *DownloadServiceGRPC) DownloadSync(ctx context.Context, req *pb.DownloadRequest) (*pb.DownloadResponse, error) {
	// パラメータのデフォルト値設定
	fromDate, toDate := s.setDefaultDates(req.FromDate, req.ToDate)

	// TODO: 実際のダウンロード処理を実装
	// ここで fromDate と toDate を使用してダウンロード処理を行う
	_ = fromDate
	_ = toDate

	response := &pb.DownloadResponse{
		Success:     true,
		RecordCount: 0,
		CsvPath:     "",
		Records:     []*pb.ETCMeisaiRecord{},
	}

	return response, nil
}

// DownloadAsync は非同期でダウンロードを開始
func (s *DownloadServiceGRPC) DownloadAsync(ctx context.Context, req *pb.DownloadRequest) (*pb.DownloadJobResponse, error) {
	// パラメータのデフォルト値設定
	fromDate, toDate := s.setDefaultDates(req.FromDate, req.ToDate)

	accounts := req.Accounts
	if len(accounts) == 0 {
		// デフォルトで全アカウントを使用（ID:パスワード形式）
		// GetAllAccountsWithCredentials() を使用して完全な認証情報を取得
		accounts = s.downloadService.GetAllAccountsWithCredentials()
		if len(accounts) == 0 {
			return &pb.DownloadJobResponse{
				JobId:   "",
				Status:  "failed",
				Message: "No accounts configured",
			}, nil
		}
	}

	// ジョブIDを生成
	jobID := uuid.New().String()

	// 非同期でダウンロード開始
	s.downloadService.ProcessAsync(jobID, accounts, fromDate, toDate)

	return &pb.DownloadJobResponse{
		JobId:   jobID,
		Status:  "pending",
		Message: "Download job started",
	}, nil
}

// GetJobStatus はジョブのステータスを取得
func (s *DownloadServiceGRPC) GetJobStatus(ctx context.Context, req *pb.GetJobStatusRequest) (*pb.JobStatus, error) {
	job, exists := s.downloadService.GetJobStatus(req.JobId)
	if !exists {
		return nil, nil
	}

	status := &pb.JobStatus{
		JobId:        job.ID,
		Status:       job.Status,
		Progress:     int32(job.Progress),
		TotalRecords: int32(job.TotalRecords),
		ErrorMessage: job.ErrorMessage,
		StartedAt:    timestamppb.New(job.StartedAt),
	}

	if job.CompletedAt != nil {
		status.CompletedAt = timestamppb.New(*job.CompletedAt)
	}

	return status, nil
}

// GetAllAccountIDs は設定されている全アカウントIDを取得
func (s *DownloadServiceGRPC) GetAllAccountIDs(ctx context.Context, req *pb.GetAllAccountIDsRequest) (*pb.GetAllAccountIDsResponse, error) {
	accountIDs := s.downloadService.GetAllAccountIDs()
	return &pb.GetAllAccountIDsResponse{
		AccountIds: accountIDs,
	}, nil
}

// GetEnvironmentVariables は環境変数を取得（デバッグ用）
func (s *DownloadServiceGRPC) GetEnvironmentVariables(ctx context.Context, req *pb.GetEnvironmentVariablesRequest) (*pb.GetEnvironmentVariablesResponse, error) {
	return &pb.GetEnvironmentVariablesResponse{
		EtcCorpAccounts:       maskAccountString(os.Getenv("ETC_CORP_ACCOUNTS")),
		EtcHeadless:           os.Getenv("ETC_HEADLESS"),
		GrpcPort:              os.Getenv("GRPC_PORT"),
		HttpPort:              os.Getenv("HTTP_PORT"),
		EtcCorporateAccounts:  maskAccountString(os.Getenv("ETC_CORPORATE_ACCOUNTS")),
		EtcPersonalAccounts:   maskAccountString(os.Getenv("ETC_PERSONAL_ACCOUNTS")),
	}, nil
}

// GetServerLogs はサーバーログを取得（デバッグ用）
func (s *DownloadServiceGRPC) GetServerLogs(ctx context.Context, req *pb.GetServerLogsRequest) (*pb.GetServerLogsResponse, error) {
	tailLines := int(req.TailLines)
	if tailLines <= 0 {
		tailLines = 100 // デフォルト100行
	}

	var logLines []string
	if s.logBuffer != nil {
		logLines = s.logBuffer.GetTail(tailLines)
	} else {
		logLines = []string{"Log buffer not initialized"}
	}

	return &pb.GetServerLogsResponse{
		LogLines:   logLines,
		TotalLines: int32(len(logLines)),
	}, nil
}

// LogMessage はログメッセージをバッファに追加（外部から呼び出し可能）
func (s *DownloadServiceGRPC) LogMessage(message string) {
	if s.logBuffer != nil {
		s.logBuffer.Add(message)
	}
}

// maskAccountString はアカウント文字列をマスク（パスワード部分を隠す）
func maskAccountString(accountStr string) string {
	if accountStr == "" {
		return ""
	}

	accounts := strings.Split(accountStr, ",")
	maskedAccounts := make([]string, len(accounts))

	for i, account := range accounts {
		parts := strings.Split(account, ":")
		if len(parts) >= 2 {
			// userid:******* の形式にマスク
			maskedAccounts[i] = parts[0] + ":*******"
		} else {
			maskedAccounts[i] = account
		}
	}

	return strings.Join(maskedAccounts, ",")
}

// setDefaultDates はデフォルトの日付を設定
func (s *DownloadServiceGRPC) setDefaultDates(fromDate, toDate string) (string, string) {
	now := time.Now()
	if toDate == "" {
		toDate = now.Format("2006-01-02")
	}
	if fromDate == "" {
		lastMonth := now.AddDate(0, -1, 0)
		fromDate = lastMonth.Format("2006-01-02")
	}
	return fromDate, toDate
}