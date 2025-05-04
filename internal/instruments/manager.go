package instruments

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sabarim/kitedata/internal/config"
)

// InstrumentManager manages instruments data
type InstrumentManager struct {
	config      *config.Config
	instruments map[string]Instrument
}

// NewInstrumentManager creates a new instrument manager
func NewInstrumentManager(config *config.Config) *InstrumentManager {
	return &InstrumentManager{
		config:      config,
		instruments: make(map[string]Instrument),
	}
}

// DownloadInstruments downloads instruments data from the broker
func (im *InstrumentManager) DownloadInstruments() error {
	log.Println("Downloading instruments data...")
	
	// Download NSE equity instruments
	if err := im.downloadAndLoadNSE(); err != nil {
		return fmt.Errorf("failed to download NSE instruments: %w", err)
	}
	
	return nil
}

// downloadAndLoadNSE downloads NSE instruments and loads them into memory
func (im *InstrumentManager) downloadAndLoadNSE() error {
	log.Println("Downloading NSE instruments...")

	// Download the CSV file
	resp, err := http.Get(im.config.Broker.InstrumentsNSEURL)
	if err != nil {
		return fmt.Errorf("failed to download NSE instruments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download NSE instruments, status code: %d", resp.StatusCode)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(im.config.Historical.InstrumentsPath), 0755); err != nil {
		return fmt.Errorf("failed to create instruments directory: %w", err)
	}

	// Create a file to save the CSV
	file, err := os.Create(im.config.Historical.InstrumentsPath)
	if err != nil {
		return fmt.Errorf("failed to create instruments file: %w", err)
	}
	defer file.Close()

	// Copy the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save NSE instruments: %w", err)
	}

	// Rewind the file for reading
	_, err = file.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("failed to rewind file: %w", err)
	}

	// Parse the CSV file
	reader := csv.NewReader(file)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Map header columns to indices
	columns := make(map[string]int)
	for i, col := range header {
		columns[col] = i
	}

	// Read and parse rows
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV record: %w", err)
		}

		// Only process NSE instruments
		if record[columns["exchange"]] != "NSE" {
			continue
		}

		// Parse instrument data
		instrumentToken := parseIntOrZero(record[columns["instrument_token"]])
		exchangeToken := parseIntOrZero(record[columns["exchange_token"]])
		tradingSymbol := record[columns["tradingsymbol"]]
		name := record[columns["name"]]
		lastPrice := parseFloatOrZero(record[columns["last_price"]])
		tickSize := parseFloatOrZero(record[columns["tick_size"]])
		expiry := record[columns["expiry"]]
		instrumentType := record[columns["instrument_type"]]
		segment := record[columns["segment"]]
		exchange := record[columns["exchange"]]
		strike := parseFloatOrZero(record[columns["strike"]])
		lotSize := parseIntOrZero(record[columns["lot_size"]])

		// Create and store the instrument
		instrument := Instrument{
			InstrumentToken: instrumentToken,
			ExchangeToken:   exchangeToken,
			TradingSymbol:   tradingSymbol,
			Name:            name,
			LastPrice:       lastPrice,
			TickSize:        tickSize,
			Expiry:          expiry,
			InstrumentType:  instrumentType,
			Segment:         segment,
			Exchange:        exchange,
			StrikePrice:     strike,
			LotSize:         lotSize,
		}

		im.instruments[tradingSymbol] = instrument
		count++
	}

	log.Printf("Loaded %d NSE instruments", count)
	return nil
}

// GetInstrumentBySymbol returns an instrument by its trading symbol
func (im *InstrumentManager) GetInstrumentBySymbol(symbol string) (Instrument, error) {
	instrument, ok := im.instruments[symbol]
	if !ok {
		return Instrument{}, fmt.Errorf("instrument not found: %s", symbol)
	}
	return instrument, nil
}

// GetInstrumentsForSymbols returns instruments for a list of trading symbols
func (im *InstrumentManager) GetInstrumentsForSymbols(symbols []string) ([]Instrument, error) {
	var instruments []Instrument
	for _, symbol := range symbols {
		instrument, err := im.GetInstrumentBySymbol(symbol)
		if err != nil {
			log.Printf("Warning: %v", err)
			continue
		}
		instruments = append(instruments, instrument)
	}
	return instruments, nil
}

// Helper functions for parsing CSV values
func parseIntOrZero(s string) int64 {
	if s == "" {
		return 0
	}
	var val int64
	_, err := fmt.Sscanf(s, "%d", &val)
	if err != nil {
		return 0
	}
	return val
}

func parseFloatOrZero(s string) float64 {
	if s == "" {
		return 0
	}
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	if err != nil {
		return 0
	}
	return val
}