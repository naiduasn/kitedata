package historical

import (
	"fmt"
	"log"
	"os"

	"github.com/xitongsys/parquet-go-source/local"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/writer"
)

// LocalFileWriter is a wrapper for a local file for parquet writer
type LocalFileWriter struct {
	file *os.File
}

// NewLocalFileWriter creates a new local file writer
func NewLocalFileWriter(name string) (*LocalFileWriter, error) {
	file, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	return &LocalFileWriter{file: file}, nil
}

// Write writes data to the file
func (fw *LocalFileWriter) Write(p []byte) (int, error) {
	return fw.file.Write(p)
}

// Close closes the file
func (fw *LocalFileWriter) Close() error {
	return fw.file.Close()
}

// Write candles to parquet file
func writeCandles(filename string, symbol string, candles []HistoricalCandle) error {
	// Create parquet file with proper ParquetFile interface
	fw, err := local.NewLocalFileWriter(filename)
	if err != nil {
		return fmt.Errorf("failed to create parquet file: %w", err)
	}
	defer fw.Close()

	// Define parquet schema - use optimized row groups for better compression
	pw, err := writer.NewParquetWriter(fw, new(HistoricalDataPoint), 4)
	if err != nil {
		return fmt.Errorf("failed to create parquet writer: %w", err)
	}

	// Configure compression with better ratio
	pw.CompressionType = parquet.CompressionCodec_GZIP

	// Configure row group size and page size for better compression
	// A larger row group allows better compression
	pw.RowGroupSize = 128 * 1024 * 1024 // 128MB row groups
	pw.PageSize = 8 * 1024             // 8KB pages

	// Write data for this month
	for _, candle := range candles {
		// Extract date components for partitioning
		candleYear := candle.Timestamp.Year()
		candleMonth := candle.Timestamp.Month()
		candleDay := candle.Timestamp.Day()

		point := HistoricalDataPoint{
			Symbol:    symbol,
			Timestamp: candle.Timestamp.Unix(),
			Date:      candle.Timestamp.Format("2006-01-02"),
			Year:      int32(candleYear),
			Month:     int32(candleMonth),
			Day:       int32(candleDay),
			Open:      candle.Open,
			High:      candle.High,
			Low:       candle.Low,
			Close:     candle.Close,
			Volume:    candle.Volume,
		}

		if err := pw.Write(point); err != nil {
			return fmt.Errorf("failed to write parquet data: %w", err)
		}
	}

	// Flush and close the writer
	if err := pw.WriteStop(); err != nil {
		return fmt.Errorf("failed to finalize parquet file: %w", err)
	}

	log.Printf("Successfully wrote %d candles to %s", len(candles), filename)
	return nil
}