package historical

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sabarim/kitedata/internal/config"
	"github.com/sabarim/kitedata/internal/instruments"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

// HistoricalDownloader manages historical data downloading and processing
type HistoricalDownloader struct {
	config      *config.Config
	kiteConnect *kiteconnect.Client
}

// NewHistoricalDownloader creates a new historical data downloader
func NewHistoricalDownloader(config *config.Config, kiteConnect *kiteconnect.Client) (*HistoricalDownloader, error) {
	// Ensure output directories exist
	if err := os.MkdirAll(config.Historical.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	if config.Historical.ParquetEnabled {
		if err := os.MkdirAll(config.Historical.ParquetDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create parquet directory: %w", err)
		}
	}

	return &HistoricalDownloader{
		config:      config,
		kiteConnect: kiteConnect,
	}, nil
}

// DownloadHistoricalData downloads historical data for specified instruments
func (hd *HistoricalDownloader) DownloadHistoricalData(ctx context.Context, instruments []instruments.Instrument) error {
	log.Println("Downloading historical data...")

	// Calculate from and to dates
	to := time.Now()
	from := to.AddDate(0, 0, -hd.config.Historical.DaysToFetch)

	// Parse interval
	var interval string
	switch hd.config.Historical.Interval {
	case "minute":
		interval = "minute"
	case "hour":
		interval = "60minute"
	case "day":
		interval = "day"
	default:
		return fmt.Errorf("invalid interval: %s", hd.config.Historical.Interval)
	}

	// Download data for each instrument
	for _, instrument := range instruments {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue processing
		}

		log.Printf("Downloading historical data for %s (%s)...", instrument.Name, instrument.TradingSymbol)

		// Download data with retry and chunking for 60-day limit
		candles, err := hd.downloadWithRetry(instrument.InstrumentToken, from, to, interval)
		if err != nil {
			log.Printf("Error downloading data for %s: %v, skipping...", instrument.Name, err)
			continue
		}

		// Save data to CSV
		if err := hd.saveToCSV(instrument, candles); err != nil {
			log.Printf("Error saving data for %s: %v", instrument.Name, err)
			continue
		}

		// Convert to Parquet if enabled
		if hd.config.Historical.ParquetEnabled {
			if err := hd.convertToParquet(instrument, candles); err != nil {
				log.Printf("Error converting data to Parquet for %s: %v", instrument.Name, err)
				continue
			}
		}

		// Respect rate limits
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(hd.config.Historical.RequestDelay) * time.Millisecond):
			// Continue after delay
		}
	}

	log.Println("Historical data download completed")
	return nil
}

// downloadWithRetry attempts to download historical data with retries
// This function handles the 60-day limit for minute data by chunking requests
func (hd *HistoricalDownloader) downloadWithRetry(instrumentToken int64, from, to time.Time, interval string) ([]HistoricalCandle, error) {
	var allCandles []HistoricalCandle

	// For minute interval, Zerodha limits API calls to 60 days
	// We need to chunk the requests if the total duration is more than 60 days
	isMinuteInterval := interval == "minute" ||
		(strings.HasSuffix(interval, "minute") && interval != "day")

	// Calculate total duration
	totalDuration := to.Sub(from)

	// If using minute intervals and duration > 60 days, we need to chunk
	if isMinuteInterval && totalDuration > 60*24*time.Hour {
		log.Printf("Duration (%v days) exceeds Zerodha's 60-day limit for minute data, chunking requests",
			totalDuration.Hours()/24)

		// Process in 60-day chunks
		currentFrom := from
		for currentFrom.Before(to) {
			// Calculate end of chunk (max 60 days)
			currentTo := currentFrom.Add(60 * 24 * time.Hour)
			if currentTo.After(to) {
				currentTo = to
			}

			log.Printf("Downloading chunk from %s to %s (%v days)",
				currentFrom.Format("2006-01-02"),
				currentTo.Format("2006-01-02"),
				currentTo.Sub(currentFrom).Hours()/24)

			// Download this chunk
			chunkCandles, err := hd.downloadChunk(instrumentToken, currentFrom, currentTo, interval)
			if err != nil {
				return nil, fmt.Errorf("error downloading chunk from %s to %s: %w",
					currentFrom.Format("2006-01-02"),
					currentTo.Format("2006-01-02"),
					err)
			}

			// Add chunk candles to all candles
			allCandles = append(allCandles, chunkCandles...)

			// Move to next chunk
			currentFrom = currentTo.Add(time.Second)

			// Add a delay between chunks to avoid rate limiting
			time.Sleep(time.Duration(hd.config.Historical.RequestDelay) * time.Millisecond)
		}

		return allCandles, nil
	}

	// For non-minute intervals or short durations, download normally
	return hd.downloadChunk(instrumentToken, from, to, interval)
}

// downloadChunk attempts to download a single chunk of historical data with retries
func (hd *HistoricalDownloader) downloadChunk(instrumentToken int64, from, to time.Time, interval string) ([]HistoricalCandle, error) {
	var candles []HistoricalCandle

	for i := 0; i < hd.config.Historical.MaxRetries; i++ {
		// Try to get historical data for this chunk
		historicalData, err := hd.kiteConnect.GetHistoricalData(
			int(instrumentToken),
			interval,
			from,
			to,
			false,
			false,
		)

		if err == nil {
			// Convert the data to our own format
			for _, data := range historicalData {
				candle := HistoricalCandle{
					Timestamp: data.Date.Time,
					Open:      data.Open,
					High:      data.High,
					Low:       data.Low,
					Close:     data.Close,
					Volume:    int64(data.Volume),
				}

				candles = append(candles, candle)
			}

			return candles, nil
		}

		log.Printf("Retry %d: Error downloading chunk data: %v", i+1, err)

		// Check for specific error about interval exceeding max limit
		if err != nil && (strings.Contains(err.Error(), "interval exceeds max limit: 60 days") ||
			strings.Contains(err.Error(), "too many candles requested")) {
			// If we're already trying with a small date range and still getting this error,
			// there might be another issue (like the interval being too small)
			if to.Sub(from) <= 5*24*time.Hour {
				return nil, fmt.Errorf("even a small date range failed: %w", err)
			}

			// Reduce the chunk size by half and try again
			mid := from.Add(to.Sub(from) / 2)
			log.Printf("Reducing chunk size: splitting at %s", mid.Format("2006-01-02"))

			// Download the first half
			firstHalf, err := hd.downloadChunk(instrumentToken, from, mid, interval)
			if err != nil {
				return nil, err
			}

			// Add delay between requests
			time.Sleep(time.Duration(hd.config.Historical.RequestDelay) * time.Millisecond)

			// Download the second half
			secondHalf, err := hd.downloadChunk(instrumentToken, mid.Add(time.Second), to, interval)
			if err != nil {
				return nil, err
			}

			// Combine the results
			return append(firstHalf, secondHalf...), nil
		}

		// If we've hit a rate limit, wait longer before retrying
		if i < hd.config.Historical.MaxRetries-1 {
			time.Sleep(time.Duration(hd.config.Historical.RequestDelay*2) * time.Millisecond)
		}
	}

	return nil, fmt.Errorf("failed to download chunk after %d retries", hd.config.Historical.MaxRetries)
}

// saveToCSV saves historical data to a CSV file
func (hd *HistoricalDownloader) saveToCSV(instrument instruments.Instrument, candles []HistoricalCandle) error {
	// Create output directory if it doesn't exist
	outputDir := filepath.Join(hd.config.Historical.OutputDir, instrument.TradingSymbol)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	filename := filepath.Join(outputDir, fmt.Sprintf("%s_historical.csv", instrument.TradingSymbol))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Write header
	_, err = file.WriteString("timestamp,date,open,high,low,close,volume\n")
	if err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write data
	for _, candle := range candles {
		line := fmt.Sprintf("%d,%s,%.2f,%.2f,%.2f,%.2f,%d\n",
			candle.Timestamp.Unix(),
			candle.Timestamp.Format("2006-01-02"),
			candle.Open,
			candle.High,
			candle.Low,
			candle.Close,
			candle.Volume,
		)
		_, err = file.WriteString(line)
		if err != nil {
			return fmt.Errorf("failed to write data: %w", err)
		}
	}

	log.Printf("Saved %d data points to %s", len(candles), filename)
	return nil
}

// convertToParquet converts historical data to Parquet format
func (hd *HistoricalDownloader) convertToParquet(instrument instruments.Instrument, candles []HistoricalCandle) error {
	if len(candles) == 0 {
		log.Printf("No candles to convert for %s", instrument.TradingSymbol)
		return nil
	}

	// Group candles by month to create separate files
	// This helps with both organization and query performance
	candlesByYearMonth := make(map[string][]HistoricalCandle)

	for _, candle := range candles {
		yearMonth := candle.Timestamp.Format("2006-01")
		candlesByYearMonth[yearMonth] = append(candlesByYearMonth[yearMonth], candle)
	}

	// Process each month group separately
	for yearMonth, monthCandles := range candlesByYearMonth {
		// Get the first candle to determine year and month for directory
		firstCandle := monthCandles[0]
		year := firstCandle.Timestamp.Year()
		month := firstCandle.Timestamp.Month()

		// Create directory for the symbol
		dirPath := filepath.Join(hd.config.Historical.ParquetDir, instrument.TradingSymbol)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory structure: %w", err)
		}

		// Create parquet file with year-month in the filename
		filename := filepath.Join(dirPath, fmt.Sprintf("%s_%d-%02d.parquet",
			instrument.TradingSymbol, year, month))

		// Convert historical candles to parquet format
		if err := writeCandles(filename, instrument.TradingSymbol, monthCandles); err != nil {
			return fmt.Errorf("failed to write parquet file: %w", err)
		}

		log.Printf("Converted %d data points to parquet for %s in %s: %s",
			len(monthCandles), instrument.TradingSymbol, yearMonth, filename)
	}

	return nil
}