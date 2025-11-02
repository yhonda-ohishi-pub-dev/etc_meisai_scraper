package grpc

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	pb "github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/pb"
	"github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/services"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server はgRPCサーバー
type Server struct {
	grpcServer      *grpc.Server
	downloadService *services.DownloadServiceGRPC
	logger          *log.Logger
	netListener     NetListener
}

// NewServerWithListener creates a new gRPC server with custom NetListener
func NewServerWithListener(db *sql.DB, logger *log.Logger, listener NetListener) *Server {
	if logger == nil {
		logger = log.New(os.Stdout, "[GRPC-SERVER] ", log.LstdFlags|log.Lshortfile)
	}

	grpcServer := grpc.NewServer()
	downloadService := services.NewDownloadServiceGRPC(db, logger)

	// サービスを登録
	pb.RegisterDownloadServiceServer(grpcServer, downloadService)

	// リフレクションを有効化（開発用）
	reflection.Register(grpcServer)

	return &Server{
		grpcServer:      grpcServer,
		downloadService: downloadService,
		logger:          logger,
		netListener:     listener,
	}
}

// NewServer creates a new gRPC server
func NewServer(db *sql.DB, logger *log.Logger) *Server {
	return NewServerWithListener(db, logger, &DefaultNetListener{})
}

// Start はgRPCサーバーを起動
func (s *Server) Start(port string) error {
	if port == "" {
		port = "50051"
	}

	if s.netListener == nil {
		s.netListener = &DefaultNetListener{}
	}
	lis, err := s.netListener.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Printf("Starting gRPC server on port %s", port)
	s.logger.Printf("GitHub repository: https://github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper")
	s.logger.Printf("Available gRPC services:")
	s.logger.Printf("  - DownloadService")
	s.logger.Printf("    * DownloadSync")
	s.logger.Printf("    * DownloadAsync")
	s.logger.Printf("    * GetJobStatus")
	s.logger.Printf("    * GetAllAccountIDs")
	s.logger.Printf("    * GetEnvironmentVariables")
	s.logger.Printf("    * GetServerLogs")

	// ログバッファにサーバー起動メッセージを追加
	s.downloadService.LogMessage(fmt.Sprintf("Starting gRPC server on port %s", port))

	return s.grpcServer.Serve(lis)
}

// Stop はgRPCサーバーを停止
func (s *Server) Stop() {
	s.logger.Println("Stopping gRPC server...")
	s.grpcServer.GracefulStop()
}