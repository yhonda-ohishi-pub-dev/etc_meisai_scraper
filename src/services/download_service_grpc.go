package services

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	pb "github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DownloadServiceGRPC はgRPC対応のダウンロードサービス
type DownloadServiceGRPC struct {
	pb.UnimplementedDownloadServiceServer
	downloadService DownloadServiceInterface
}

// NewDownloadServiceGRPC creates a new gRPC download service
func NewDownloadServiceGRPC(db *sql.DB, logger *log.Logger) *DownloadServiceGRPC {
	return &DownloadServiceGRPC{
		downloadService: NewDownloadService(db, logger),
	}
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
		// デフォルトで全アカウントを使用
		accounts = s.downloadService.GetAllAccountIDs()
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