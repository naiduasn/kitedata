package historical

import "time"

// HistoricalCandle represents a single candlestick
type HistoricalCandle struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int64
}

// HistoricalDataPoint represents a single historical data point for parquet
type HistoricalDataPoint struct {
	Symbol    string  `parquet:"name=symbol, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Timestamp int64   `parquet:"name=timestamp, type=INT64, encoding=DELTA_BINARY_PACKED"`
	Date      string  `parquet:"name=date, type=BYTE_ARRAY, convertedtype=UTF8, encoding=PLAIN_DICTIONARY"`
	Year      int32   `parquet:"name=year, type=INT32, encoding=PLAIN_DICTIONARY"` 
	Month     int32   `parquet:"name=month, type=INT32, encoding=PLAIN_DICTIONARY"`
	Day       int32   `parquet:"name=day, type=INT32, encoding=PLAIN_DICTIONARY"`
	Open      float64 `parquet:"name=open, type=DOUBLE, encoding=PLAIN"`
	High      float64 `parquet:"name=high, type=DOUBLE, encoding=PLAIN"`
	Low       float64 `parquet:"name=low, type=DOUBLE, encoding=PLAIN"`
	Close     float64 `parquet:"name=close, type=DOUBLE, encoding=PLAIN"`
	Volume    int64   `parquet:"name=volume, type=INT64, encoding=DELTA_BINARY_PACKED"`
}