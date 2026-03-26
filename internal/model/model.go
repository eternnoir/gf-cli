package model

// SeatClass represents the cabin class for a flight.
type SeatClass string

const (
	SeatClassEconomy        SeatClass = "economy"
	SeatClassPremiumEconomy SeatClass = "premium-economy"
	SeatClassBusiness       SeatClass = "business"
	SeatClassFirst          SeatClass = "first"
)

// OutputFormat defines how results are presented to the user.
type OutputFormat string

const (
	OutputFormatText OutputFormat = "text"
	OutputFormatJSON OutputFormat = "json"
)

// SearchParams holds all parameters for a flight search request.
type SearchParams struct {
	Origin      string
	Destination string
	Date        string // YYYY-MM-DD
	ReturnDate  string // YYYY-MM-DD, empty for one-way
	Adults      int
	Children    int
	Class       SeatClass
	Limit       int
}

// Flight represents a single flight option returned from the search.
type Flight struct {
	Airline       string  `json:"airline"`
	DepartureTime string  `json:"departure_time"`
	ArrivalTime   string  `json:"arrival_time"`
	Duration      string  `json:"duration"`
	Stops         int     `json:"stops"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	ArrivalDate   string  `json:"arrival_date,omitempty"` // if arrives next day etc.
}

// SearchResult aggregates all flight results for a query.
type SearchResult struct {
	Origin      string   `json:"origin"`
	Destination string   `json:"destination"`
	Date        string   `json:"date"`
	ReturnDate  string   `json:"return_date,omitempty"`
	PriceTrend  string   `json:"price_trend,omitempty"` // e.g. "low", "typical", "high"
	Flights     []Flight `json:"flights"`
}

// DatePrice represents the cheapest flight found for a specific date.
type DatePrice struct {
	Date       string  `json:"date"`
	ReturnDate string  `json:"return_date,omitempty"` // set when round-trip (--stay N); empty for one-way
	Price      float64 `json:"price"`
	Currency   string  `json:"currency"`
	Airline    string  `json:"airline"`
	Duration   string  `json:"duration"`
	Stops      int     `json:"stops"`
}

// DateRangeParams holds parameters for a date-range price search.
type DateRangeParams struct {
	Origin      string
	Destination string
	FromDate    string // YYYY-MM-DD, start of range
	ToDate      string // YYYY-MM-DD, end of range (inclusive)
	Adults      int
	Children    int
	Class       SeatClass
	StayDays    int // if > 0, search round-trip with this many days between outbound and return
}

// DateRangeResult holds cheapest-flight data across a date range.
type DateRangeResult struct {
	Origin      string      `json:"origin"`
	Destination string      `json:"destination"`
	FromDate    string      `json:"from_date"`
	ToDate      string      `json:"to_date"`
	Dates       []DatePrice `json:"dates"`
}
