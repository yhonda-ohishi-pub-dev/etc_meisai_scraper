package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/grpc"
	"github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/handlers"
	"github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/services"
)

func main() {
	// コマンドラインフラグ
	var (
		useGRPC    = flag.Bool("grpc", true, "Use gRPC server (default: true)")
		grpcPort   = flag.String("grpc-port", "50052", "gRPC server port for etc_meisai_scraper")
		httpPort   = flag.String("http-port", "8080", "HTTP server port (legacy mode)")
		showHelp   = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	// 環境変数を優先
	if envPort := os.Getenv("GRPC_PORT"); envPort != "" {
		grpcPort = &envPort
	}
	if envPort := os.Getenv("HTTP_PORT"); envPort != "" {
		httpPort = &envPort
	}

	// ロガー設定
	logger := log.New(os.Stdout, "[ETC-MEISAI] ", log.LstdFlags)

	// DB接続は不要（スクレイピング専用サービス）
	var db *sql.DB

	if *useGRPC {
		// gRPCサーバーモード（推奨）
		logger.Println("Starting in gRPC server mode (recommended for desktop-server integration)")
		runGRPCServer(db, logger, *grpcPort)
	} else {
		// HTTPサーバーモード（レガシー）
		logger.Println("Starting in HTTP server mode (legacy)")
		runHTTPServer(db, logger, *httpPort)
	}
}

func printHelp() {
	log.Println("ETC Meisai Scraper - Standalone gRPC Server")
	log.Println()
	log.Println("Usage:")
	log.Println("  etc_meisai_scraper.exe [options]")
	log.Println()
	log.Println("Options:")
	flag.PrintDefaults()
	log.Println()
	log.Println("Examples:")
	log.Println("  # Start as gRPC server (default)")
	log.Println("  etc_meisai_scraper.exe")
	log.Println()
	log.Println("  # Start with custom port")
	log.Println("  etc_meisai_scraper.exe --grpc-port 50052")
	log.Println()
	log.Println("  # Start as HTTP server (legacy)")
	log.Println("  etc_meisai_scraper.exe --grpc=false --http-port 8080")
	log.Println()
	log.Println("Integration with desktop-server:")
	log.Println("  This service is designed to run as a separate process and be called")
	log.Println("  by desktop-server via gRPC. See README.md for integration details.")
}

func runGRPCServer(db *sql.DB, logger *log.Logger, port string) {
	server := grpc.NewServer(db, logger)

	// シグナルハンドリング
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		server.Stop()
		os.Exit(0)
	}()

	if err := server.Start(port); err != nil {
		logger.Fatalf("Failed to start gRPC server: %v", err)
	}
}

func runHTTPServer(db *sql.DB, logger *log.Logger, port string) {
	// ダウンロードサービス初期化
	downloadService := services.NewDownloadService(db, logger)

	// ハンドラー初期化
	downloadHandler := handlers.NewDownloadHandler(downloadService)

	// ルーティング設定
	http.HandleFunc("/api/download/sync", downloadHandler.DownloadSync)
	http.HandleFunc("/api/download/async", downloadHandler.DownloadAsync)
	http.HandleFunc("/api/download/status", downloadHandler.GetDownloadStatus)

	logger.Printf("Starting HTTP server on port %s", port)
	logger.Printf("GitHub repository: https://github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper")
	logger.Printf("Download endpoints:")
	logger.Printf("  POST /api/download/sync  - 同期ダウンロード")
	logger.Printf("  POST /api/download/async - 非同期ダウンロード")
	logger.Printf("  GET  /api/download/status?job_id={id} - ステータス確認")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Fatalf("HTTP server failed to start: %v", err)
	}
}