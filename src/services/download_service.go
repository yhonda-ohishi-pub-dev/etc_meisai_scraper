package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/scraper"
)

// DownloadService はダウンロード処理を管理
type DownloadService struct {
	db             *sql.DB
	logger         *log.Logger
	jobs           map[string]*DownloadJob
	jobMutex       sync.RWMutex
	scraperFactory ScraperFactory
	logCallback    func(string) // ログコールバック関数
}

// DownloadJob はダウンロードジョブの状態
type DownloadJob struct {
	ID           string
	Status       string
	Progress     int
	TotalRecords int
	ErrorMessage string
	StartedAt    time.Time
	CompletedAt  *time.Time
}

// DownloadServiceInterface はダウンロードサービスのインターフェース
type DownloadServiceInterface interface {
	GetAllAccountIDs() []string
	GetAllAccountsWithCredentials() []string
	ProcessAsync(jobID string, accounts []string, fromDate, toDate string)
	GetJobStatus(jobID string) (*DownloadJob, bool)
	SetLogCallback(callback func(string))
}

// NewDownloadService creates a new download service
func NewDownloadService(db *sql.DB, logger *log.Logger) *DownloadService {
	return NewDownloadServiceWithFactory(db, logger, NewDefaultScraperFactory())
}

// NewDownloadServiceWithFactory creates a new download service with a custom scraper factory
func NewDownloadServiceWithFactory(db *sql.DB, logger *log.Logger, factory ScraperFactory) *DownloadService {
	return &DownloadService{
		db:             db,
		logger:         logger,
		jobs:           make(map[string]*DownloadJob),
		scraperFactory: factory,
	}
}

// parseAccountsString はアカウント文字列をパース（JSON配列またはカンマ区切り文字列に対応）
func parseAccountsString(accountsStr string) []string {
	if accountsStr == "" {
		return nil
	}

	var accounts []string

	// JSON配列形式かチェック（desktop-server形式: ["user1:pass1","user2:pass2"]）
	if strings.HasPrefix(strings.TrimSpace(accountsStr), "[") {
		var jsonAccounts []string
		if err := json.Unmarshal([]byte(accountsStr), &jsonAccounts); err == nil {
			accounts = jsonAccounts
		} else {
			// JSONパースエラーの場合はカンマ区切りとして扱う
			accounts = strings.Split(accountsStr, ",")
		}
	} else {
		// カンマ区切り文字列形式（従来形式: "user1:pass1,user2:pass2"）
		accounts = strings.Split(accountsStr, ",")
	}

	return accounts
}

// GetAllAccountsWithCredentials は設定されているすべてのアカウント情報（ID:パスワード形式）を取得
func (s *DownloadService) GetAllAccountsWithCredentials() []string {
	// ETC_CORP_ACCOUNTS (推奨) - JSON配列またはカンマ区切り文字列に対応
	corpAccounts := os.Getenv("ETC_CORP_ACCOUNTS")
	if corpAccounts != "" {
		return parseAccountsString(corpAccounts)
	}

	// 後方互換性のため ETC_CORPORATE_ACCOUNTS と ETC_PERSONAL_ACCOUNTS もサポート
	var allAccounts []string

	corporateAccounts := os.Getenv("ETC_CORPORATE_ACCOUNTS")
	if corporateAccounts != "" {
		allAccounts = append(allAccounts, parseAccountsString(corporateAccounts)...)
	}

	personalAccounts := os.Getenv("ETC_PERSONAL_ACCOUNTS")
	if personalAccounts != "" {
		allAccounts = append(allAccounts, parseAccountsString(personalAccounts)...)
	}

	return allAccounts
}

// GetAllAccountIDs は設定されているすべてのアカウントIDを取得
func (s *DownloadService) GetAllAccountIDs() []string {
	var accountIDs []string

	// GetAllAccountsWithCredentials を使用して完全なアカウント情報を取得
	accounts := s.GetAllAccountsWithCredentials()
	for _, accountStr := range accounts {
		parts := strings.Split(strings.TrimSpace(accountStr), ":")
		if len(parts) >= 1 {
			accountIDs = append(accountIDs, parts[0])
		}
	}

	return accountIDs
}

// ProcessAsync は非同期でダウンロードを実行
func (s *DownloadService) ProcessAsync(jobID string, accounts []string, fromDate, toDate string) {
	s.jobMutex.Lock()
	job := &DownloadJob{
		ID:        jobID,
		Status:    "processing",
		Progress:  0,
		StartedAt: time.Now(),
	}
	s.jobs[jobID] = job
	s.jobMutex.Unlock()

	// ダウンロード処理をシミュレート
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if s.logger != nil {
					s.logger.Printf("Panic in download job %s: %v", jobID, r)
				}
				s.updateJobStatus(jobID, "failed", 0, fmt.Sprintf("Internal error: %v", r))
			}
		}()

		s.logMessage("Starting download job %s for %d accounts from %s to %s",
			jobID, len(accounts), fromDate, toDate)

		// Create a shared session folder for all accounts in this job
		sessionFolder := fmt.Sprintf("./downloads/%s", time.Now().Format("20060102_150405"))

		// 各アカウントを処理
		totalAccounts := len(accounts)
		for i, account := range accounts {
			// 進捗更新
			progress := int(float64(i+1) / float64(totalAccounts) * 100)
			s.updateJobProgress(jobID, progress)

			// 実際のダウンロード処理（セッションフォルダを渡す）
			if err := s.downloadAccountData(account, fromDate, toDate, sessionFolder); err != nil {
				s.logMessage("Error downloading data for account %s: %v", account, err)
				// エラーがあってもほかのアカウントの処理は続ける
			}

			// レート制限のため少し待機
			time.Sleep(time.Second)
		}

		// 完了
		now := time.Now()
		s.jobMutex.Lock()
		if job, exists := s.jobs[jobID]; exists {
			job.Status = "completed"
			job.Progress = 100
			job.CompletedAt = &now
		}
		s.jobMutex.Unlock()

		s.logMessage("Completed download job %s", jobID)
	}()
}

// downloadAccountData は単一アカウントのデータをダウンロード
func (s *DownloadService) downloadAccountData(accountID, fromDate, toDate, sessionFolder string) error {
	// アカウント情報の解析（accountID:password形式）
	parts := strings.Split(accountID, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid account format: %s (expected accountID:password)", accountID)
	}

	userID := parts[0]
	password := parts[1]

	// スクレイパーの設定
	config := &scraper.ScraperConfig{
		UserID:        userID,
		Password:      password,
		DownloadPath:  "./downloads",
		SessionFolder: sessionFolder, // Use shared session folder
		Headless:      getHeadlessMode(),
		Timeout:       30000,
		RetryCount:    3,
	}

	// スクレイパー作成
	etcScraper, err := s.scraperFactory.CreateScraper(config, s.logger)
	if err != nil {
		return fmt.Errorf("failed to create scraper: %w", err)
	}
	defer etcScraper.Close()

	// Playwright初期化
	if err := etcScraper.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize scraper: %w", err)
	}

	// ログイン
	if err := etcScraper.Login(); err != nil {
		return fmt.Errorf("login failed for account %s: %w", userID, err)
	}

	// データダウンロード
	csvPath, err := etcScraper.DownloadMeisai(fromDate, toDate)
	if err != nil {
		return fmt.Errorf("download failed for account %s: %w", userID, err)
	}

	s.logMessage("Successfully downloaded data for account %s: %s", userID, csvPath)

	// TODO: CSVファイルをパースしてDBに保存

	return nil
}

// updateJobProgress はジョブの進捗を更新
func (s *DownloadService) updateJobProgress(jobID string, progress int) {
	s.jobMutex.Lock()
	defer s.jobMutex.Unlock()

	if job, exists := s.jobs[jobID]; exists {
		job.Progress = progress
	}
}

// updateJobStatus はジョブのステータスを更新
func (s *DownloadService) updateJobStatus(jobID string, status string, progress int, errorMsg string) {
	s.jobMutex.Lock()
	defer s.jobMutex.Unlock()

	if job, exists := s.jobs[jobID]; exists {
		job.Status = status
		job.Progress = progress
		if errorMsg != "" {
			job.ErrorMessage = errorMsg
		}
		if status == "completed" || status == "failed" {
			now := time.Now()
			job.CompletedAt = &now
		}
	}
}

// GetJobStatus はジョブのステータスを取得
func (s *DownloadService) GetJobStatus(jobID string) (*DownloadJob, bool) {
	s.jobMutex.RLock()
	defer s.jobMutex.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, false
	}

	// コピーを返す
	jobCopy := *job
	return &jobCopy, true
}

// GetHeadlessMode は環境変数からHeadlessモードの設定を取得
// ETC_HEADLESS=false でブラウザを表示、未設定またはtrueでHeadlessモード（デフォルト）
func GetHeadlessMode() bool {
	headlessEnv := os.Getenv("ETC_HEADLESS")
	if headlessEnv == "" {
		log.Println("[Headless] ETC_HEADLESS not set, using default: true (Headless mode)")
		return true // デフォルトはHeadlessモード
	}

	// "false", "0", "no" の場合は非Headlessモード（ブラウザ表示）
	headless, err := strconv.ParseBool(headlessEnv)
	if err != nil {
		// パースエラーの場合もデフォルトのHeadlessモード
		log.Printf("[Headless] Invalid ETC_HEADLESS value %q, using default: true (Headless mode)", headlessEnv)
		return true
	}

	if headless {
		log.Printf("[Headless] ETC_HEADLESS=%s -> Headless mode (browser not visible)", headlessEnv)
	} else {
		log.Printf("[Headless] ETC_HEADLESS=%s -> VISIBLE mode (browser will appear)", headlessEnv)
	}

	return headless
}

// getHeadlessMode は後方互換性のため維持（非推奨）
func getHeadlessMode() bool {
	return GetHeadlessMode()
}

// SetLogCallback はログコールバック関数を設定
func (s *DownloadService) SetLogCallback(callback func(string)) {
	s.logCallback = callback
}

// logMessage はログメッセージを記録
func (s *DownloadService) logMessage(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if s.logger != nil {
		s.logger.Println(msg)
	}
	if s.logCallback != nil {
		s.logCallback(msg)
	}
}