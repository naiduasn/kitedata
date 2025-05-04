package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sabarim/kitedata/internal/auth"
	"github.com/sabarim/kitedata/internal/config"
	"github.com/sabarim/kitedata/internal/historical"
	"github.com/sabarim/kitedata/internal/instruments"
	"github.com/spf13/cobra"
)

var (
	configFile     string
	authServiceURL string
	authServiceKey string
	brokerName     string
	apiKey         string
	apiSecret      string
	sessionToken   string
	symbolsStr     string
	symbolFile     string
	fromDate       string
	toDate         string
	days           int
	interval       string
	outputDir      string
	parquetEnabled bool
	parquetDir     string
	requestDelay   int
	maxRetries     int
	verbose        bool
	version        bool
)

var version_string = "0.1.0"

func main() {
	// Define the root command
	rootCmd := &cobra.Command{
		Use:   "kitedata",
		Short: "A utility to download historical market data from Kite/Zerodha",
		Long:  `A standalone utility for downloading historical market data from Kite/Zerodha broker and saving it in CSV or Parquet format.`,
		Run:   runRootCommand,
	}

	// Define flags
	rootCmd.Flags().StringVar(&configFile, "config", "config.yaml", "Path to config file")
	rootCmd.Flags().StringVar(&authServiceURL, "auth-service-url", "", "URL of the auth service")
	rootCmd.Flags().StringVar(&authServiceKey, "auth-service-key", "", "API key for the auth service")
	rootCmd.Flags().StringVar(&brokerName, "broker", "", "Broker name (default is zerodha)")
	rootCmd.Flags().StringVar(&apiKey, "api-key", "", "Broker API key (if not using auth service)")
	rootCmd.Flags().StringVar(&apiSecret, "api-secret", "", "Broker API secret (if not using auth service)")
	rootCmd.Flags().StringVar(&sessionToken, "session-token", "", "Broker session token (if not using auth service)")
	rootCmd.Flags().StringVar(&symbolsStr, "symbols", "", "Comma-separated list of symbols to download")
	rootCmd.Flags().StringVar(&symbolFile, "symbol-file", "", "File containing symbols, one per line")
	rootCmd.Flags().StringVar(&fromDate, "from", "", "Start date (YYYY-MM-DD)")
	rootCmd.Flags().StringVar(&toDate, "to", "", "End date (YYYY-MM-DD)")
	rootCmd.Flags().IntVar(&days, "days", 0, "Number of days to fetch")
	rootCmd.Flags().StringVar(&interval, "interval", "", "Time interval (minute, hour, day)")
	rootCmd.Flags().StringVar(&outputDir, "output-dir", "", "Output directory for CSV files")
	rootCmd.Flags().BoolVar(&parquetEnabled, "parquet", false, "Convert to Parquet format")
	rootCmd.Flags().StringVar(&parquetDir, "parquet-dir", "", "Output directory for Parquet files")
	rootCmd.Flags().IntVar(&requestDelay, "request-delay", 0, "Delay between requests in milliseconds")
	rootCmd.Flags().IntVar(&maxRetries, "max-retries", 0, "Maximum number of retries for failed requests")
	rootCmd.Flags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	rootCmd.Flags().BoolVar(&version, "version", false, "Print version information")

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runRootCommand(cmd *cobra.Command, args []string) {
	// Check for version flag
	if version {
		fmt.Printf("kitedata version %s\n", version_string)
		return
	}

	// Print environment variables for debugging
	fmt.Println("==== Environment Variables ====")
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "HISTORICAL_") {
			fmt.Println(env)
		}
	}
	
	// 1. Load configuration from file and environment
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}
	
	// Print loaded config for debugging
	fmt.Println("==== Loaded Configuration ====")
	fmt.Printf("Auth Service URL: %s\n", cfg.Auth.AuthServiceURL)
	fmt.Printf("Auth Service API Key: %s\n", cfg.Auth.AuthServiceAPIKey)
	fmt.Printf("Broker Name: %s\n", cfg.Auth.BrokerName)
	fmt.Printf("API Key Set: %v\n", cfg.Auth.ApiKey != "")
	fmt.Printf("Session Token Set: %v\n", cfg.Auth.SessionToken != "")
	fmt.Println("==============================")

	// 2. Override configuration with command-line flags
	if authServiceURL != "" {
		cfg.Auth.AuthServiceURL = authServiceURL
	}
	if authServiceKey != "" {
		cfg.Auth.AuthServiceAPIKey = authServiceKey
	}
	if brokerName != "" {
		cfg.Auth.BrokerName = brokerName
	}
	if apiKey != "" {
		cfg.Auth.ApiKey = apiKey
	}
	if apiSecret != "" {
		cfg.Auth.ApiSecret = apiSecret
	}
	if sessionToken != "" {
		cfg.Auth.SessionToken = sessionToken
	}
	if days > 0 {
		cfg.Historical.DaysToFetch = days
	}
	if interval != "" {
		cfg.Historical.Interval = interval
	}
	if outputDir != "" {
		cfg.Historical.OutputDir = outputDir
	}
	if parquetEnabled {
		cfg.Historical.ParquetEnabled = true
	}
	if parquetDir != "" {
		cfg.Historical.ParquetDir = parquetDir
	}
	if requestDelay > 0 {
		cfg.Historical.RequestDelay = requestDelay
	}
	if maxRetries > 0 {
		cfg.Historical.MaxRetries = maxRetries
	}

	// 3. Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. Handle OS signals
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigchan
		log.Printf("Received signal %v, initiating shutdown...", sig)
		cancel() // Cancel context to initiate shutdown
	}()

	// 5. Initialize authentication
	authManager := auth.NewAuthManager(&cfg)
	
	// 6. Get authenticated client
	fmt.Println("Getting authenticated Kite client...")
	kiteClient, err := authManager.GetClient()
	if err != nil {
		log.Fatalf("Failed to get authenticated client: %v", err)
	}
	fmt.Println("Successfully obtained authenticated Kite client")
	
	// 7. Initialize instrument manager
	instrumentManager := instruments.NewInstrumentManager(&cfg)
	
	// 8. Download instruments data
	if err := instrumentManager.DownloadInstruments(); err != nil {
		log.Fatalf("Failed to download instruments: %v", err)
	}
	
	// 9. Determine symbols to download
	var symbols []string
	
	// First try symbols from command line
	if symbolsStr != "" {
		symbols = strings.Split(symbolsStr, ",")
	} else if symbolFile != "" {
		// Try symbols from file
		content, err := os.ReadFile(symbolFile)
		if err != nil {
			log.Fatalf("Failed to read symbol file: %v", err)
		}
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				symbols = append(symbols, line)
			}
		}
	} else {
		log.Fatalf("No symbols specified. Use --symbols or --symbol-file")
	}
	
	// 10. Get instrument objects for the specified symbols
	instrumentsList, err := instrumentManager.GetInstrumentsForSymbols(symbols)
	if err != nil {
		log.Fatalf("Failed to get instruments: %v", err)
	}
	
	if len(instrumentsList) == 0 {
		log.Fatalf("No valid instruments found for the specified symbols")
	}
	
	log.Printf("Found %d instruments to download", len(instrumentsList))
	
	// 11. Initialize historical downloader
	histDownloader, err := historical.NewHistoricalDownloader(&cfg, kiteClient)
	if err != nil {
		log.Fatalf("Failed to initialize historical downloader: %v", err)
	}
	
	// 12. Download historical data
	if err := histDownloader.DownloadHistoricalData(ctx, instrumentsList); err != nil {
		log.Fatalf("Failed to download historical data: %v", err)
	}
	
	log.Println("Historical data download completed successfully")
}