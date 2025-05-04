# KiteData - Historical Market Data Downloader

KiteData is a standalone command-line utility for downloading historical market data from Zerodha/Kite broker API. It provides a flexible interface for fetching historical candle data and saving it in CSV or Parquet format.

## Features

- Download historical candle data for specified symbols
- Support for various intervals (minute, hour, day)
- Automatically handles API limitations (60-day limit for minute data)
- CSV output format with optional Parquet conversion
- Flexible authentication options (auth service, env vars, config file)
- Comprehensive configuration through flags, env vars, or config file

## Installation

### Prerequisites

- Go 1.18 or higher
- An active Zerodha trading account with API access

### Building from source

```bash
# Clone the repository
git clone https://github.com/yourusername/kitedata.git
cd kitedata

# Build the executable
go build -o kitedata ./cmd

# Make it available system-wide (optional)
sudo mv kitedata /usr/local/bin/
```

## Usage

### Basic Usage

```bash
# Download 30 days of minute data for specified symbols
kitedata --symbols NIFTY,BANKNIFTY,RELIANCE

# Use a different output directory
kitedata --symbols NIFTY --output-dir /path/to/data

# Download with specific date range
kitedata --symbols NIFTY --from 2023-01-01 --to 2023-01-31

# Download and convert to Parquet
kitedata --symbols NIFTY --parquet
```

### Using a Config File

```bash
# Create a config file (copy from example)
cp config.yaml.example config.yaml
# Edit the config file with your settings
nano config.yaml
# Run with config file
kitedata --config config.yaml
```

### Authentication Options

The utility supports multiple authentication methods:

1. **Auth Service (recommended)**
   ```bash
   kitedata --auth-service-url http://your-auth-service:8001 --auth-service-key your-key --broker zerodha
   ```

2. **Direct Credentials**
   ```bash
   kitedata --api-key your-api-key --api-secret your-api-secret --session-token your-session-token
   ```

3. **Environment Variables**
   ```bash
   export HISTORICAL_API_KEY=your-api-key
   export HISTORICAL_API_SECRET=your-api-secret
   export HISTORICAL_SESSION_TOKEN=your-session-token
   kitedata --symbols NIFTY
   ```

## Command-line Options

```
Usage: kitedata [options]

Options:
  --config string               Path to config file (default "config.yaml")
  --auth-service-url string     URL of the auth service
  --auth-service-key string     API key for the auth service
  --broker string               Broker name (default "zerodha")
  --api-key string              Broker API key (if not using auth service)
  --api-secret string           Broker API secret (if not using auth service)
  --session-token string        Broker session token (if not using auth service) 
  --symbols strings             Comma-separated list of symbols to download
  --symbol-file string          File containing symbols, one per line
  --from string                 Start date (YYYY-MM-DD)
  --to string                   End date (YYYY-MM-DD)
  --days int                    Number of days to fetch (default 30)
  --interval string             Time interval (minute, hour, day) (default "minute")
  --output-dir string           Output directory for CSV files (default "./historical_data")
  --parquet                     Convert to Parquet format
  --parquet-dir string          Output directory for Parquet files (default "./parquet_data")
  --request-delay int           Delay between requests in milliseconds (default 500)
  --max-retries int             Maximum number of retries for failed requests (default 3)
  --verbose                     Enable verbose logging
  --version                     Print version information
  --help                        Show this help message
```

## Configuration File

The utility supports a YAML configuration file. You can use the `config.yaml.example` as a template:

```yaml
auth:
  # Authentication options
  auth_service_url: "http://example.com:8001"
  auth_service_api_key: "your-api-key"
  broker_name: "zerodha"
  
  # Direct broker credentials (used if auth service unavailable)
  api_key: ""
  api_secret: ""
  session_token: ""

broker:
  # Broker-specific settings
  instruments_nse_url: "https://api.kite.trade/instruments/NSE"

historical:
  # Download parameters
  interval: "minute"  # Can be "minute", "hour", "day"
  days_to_fetch: 30   # How many days of history to fetch
  request_delay: 500  # Milliseconds between requests
  max_retries: 3      # Number of retries for failed requests
  
  # Output options
  output_dir: "./historical_data"
  parquet_enabled: false
  parquet_dir: "./parquet_data"
  
  # Instruments path
  instruments_path: "./instruments.csv"

# List of symbols to download (used if --symbols or --symbol-file not provided)
symbols:
  - "NIFTY"
  - "BANKNIFTY"
  - "RELIANCE"
  - "TCS"
  - "INFY"
```

## Environment Variables

The utility also supports configuration through environment variables:

```
# Authentication
HISTORICAL_AUTH_SERVICE_URL=http://example.com:8001
HISTORICAL_AUTH_SERVICE_KEY=your-api-key
HISTORICAL_BROKER_NAME=zerodha
HISTORICAL_API_KEY=your-api-key
HISTORICAL_API_SECRET=your-api-secret
HISTORICAL_SESSION_TOKEN=your-session-token

# Broker settings
HISTORICAL_INSTRUMENTS_NSE_URL=https://api.kite.trade/instruments/NSE

# Download parameters
HISTORICAL_INTERVAL=minute
HISTORICAL_DAYS=30
HISTORICAL_REQUEST_DELAY=500
HISTORICAL_MAX_RETRIES=3

# Output options
HISTORICAL_OUTPUT_DIR=./historical_data
HISTORICAL_PARQUET_ENABLED=true
HISTORICAL_PARQUET_DIR=./parquet_data

# Instruments 
HISTORICAL_INSTRUMENTS_PATH=./instruments.csv

# Symbols (comma-separated)
HISTORICAL_SYMBOLS=NIFTY,BANKNIFTY,RELIANCE,TCS,INFY
```

## Output Formats

### CSV Format

Historical data is saved in CSV format with the following structure:

```
timestamp,date,open,high,low,close,volume
1622527800,2021-06-01,15435.00,15461.15,15418.35,15435.35,152700
1622527860,2021-06-01,15435.35,15442.40,15435.35,15442.40,27600
...
```

Files are organized by symbol:
```
./historical_data/{symbol}/{symbol}_historical.csv
```

### Parquet Format (Optional)

When Parquet conversion is enabled, data is also saved in Parquet format with the following schema:

- symbol: string
- timestamp: int64
- date: string
- year: int32 
- month: int32
- day: int32
- open: double
- high: double
- low: double
- close: double
- volume: int64

Files are organized by symbol and month:
```
./parquet_data/{symbol}/{symbol}_{year}-{month}.parquet
```

## Handling API Limitations

The Zerodha API has a limitation where it only allows fetching 60 days of minute data in a single request. KiteData automatically handles this limitation by:

1. Detecting when the requested date range exceeds 60 days
2. Breaking the request into multiple 60-day chunks
3. Downloading each chunk separately with appropriate delays
4. Combining the results into a single dataset

This chunking logic ensures that you can request data for any date range without worrying about API limitations.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgements

- Based on the excellent [gokiteconnect](https://github.com/zerodha/gokiteconnect) library by Zerodha
- Uses [parquet-go](https://github.com/xitongsys/parquet-go) for Parquet file handling
- Built with [cobra](https://github.com/spf13/cobra) and [viper](https://github.com/spf13/viper) for a robust CLI interface