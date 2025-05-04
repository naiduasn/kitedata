package instruments

// Instrument represents a trading instrument
type Instrument struct {
	InstrumentToken int64
	ExchangeToken   int64
	TradingSymbol   string
	Name            string
	LastPrice       float64
	TickSize        float64
	Expiry          string
	InstrumentType  string
	Segment         string
	Exchange        string
	StrikePrice     float64
	LotSize         int64
	Underlying      string
	UnderlyingToken int64
}