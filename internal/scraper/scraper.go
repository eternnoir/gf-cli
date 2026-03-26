// Package scraper handles fetching flight data from Google Flights.
//
// Implementation overview:
//  1. Encode search parameters as a nested JSON array → f.req parameter (see query.go)
//  2. POST to Google Flights' internal RPC endpoint
//  3. Strip the ")]}'" anti-XSSI prefix from the response
//  4. Parse the JSON payload (see parser.go)
//
// This approach is more stable than HTML scraping because it targets the same
// internal API that the browser uses, and the response is structured JSON rather
// than embedded in an HTML page.
//
// Reference: https://github.com/punitarani/fli (Python)
package scraper

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eternnoir/gf-cli/internal/model"
)

const (
	// maxDateRangeDays caps the date range to avoid hammering Google.
	maxDateRangeDays = 61
	// dateRangeDelay is the pause between requests during date-range scans.
	dateRangeDelay = 500 * time.Millisecond
)

const (
	// rpcEndpoint includes curr=USD&hl=en to request USD prices in English.
	// This is belt-and-suspenders alongside the currency extraction in parser.go.
	rpcEndpoint = "https://www.google.com/_/FlightsFrontendUi/data/travel.frontend.flights.FlightsFrontendService/GetShoppingResults?curr=USD&hl=en"
	userAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	timeout     = 30 * time.Second

	// antiXSSIPrefix is prepended to all responses to prevent cross-site script inclusion.
	// It must be stripped before JSON parsing.
	antiXSSIPrefix = ")]}'"
)

// Searcher defines the contract for flight search implementations.
type Searcher interface {
	Search(params model.SearchParams) (*model.SearchResult, error)
}

// GoogleFlightsScraper implements Searcher via Google Flights' internal RPC API.
type GoogleFlightsScraper struct {
	client *http.Client
}

// NewGoogleFlightsScraper creates a new scraper instance.
func NewGoogleFlightsScraper() *GoogleFlightsScraper {
	return &GoogleFlightsScraper{
		client: &http.Client{Timeout: timeout},
	}
}

// Search performs a flight search with the given parameters.
func (s *GoogleFlightsScraper) Search(params model.SearchParams) (*model.SearchResult, error) {
	raw, err := s.fetchFlights(params)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	flights, priceTrend, err := parseFlights(raw)
	if err != nil {
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	if params.Limit > 0 && len(flights) > params.Limit {
		flights = flights[:params.Limit]
	}

	return &model.SearchResult{
		Origin:      params.Origin,
		Destination: params.Destination,
		Date:        params.Date,
		ReturnDate:  params.ReturnDate,
		PriceTrend:  priceTrend,
		Flights:     flights,
	}, nil
}

// SearchDateRange iterates over each date in [params.FromDate, params.ToDate] and
// returns the cheapest flight option per date. Requests are rate-limited by
// dateRangeDelay to be polite to Google's servers.
//
// Maximum range is maxDateRangeDays (61 days). Dates that return no results
// are silently skipped.
func (s *GoogleFlightsScraper) SearchDateRange(params model.DateRangeParams) (*model.DateRangeResult, error) {
	const layout = "2006-01-02"

	from, err := time.Parse(layout, params.FromDate)
	if err != nil {
		return nil, fmt.Errorf("invalid from-date %q: %w", params.FromDate, err)
	}
	to, err := time.Parse(layout, params.ToDate)
	if err != nil {
		return nil, fmt.Errorf("invalid to-date %q: %w", params.ToDate, err)
	}
	if to.Before(from) {
		return nil, fmt.Errorf("to-date must be on or after from-date")
	}
	days := int(to.Sub(from).Hours()/24) + 1
	if days > maxDateRangeDays {
		return nil, fmt.Errorf("date range too large (%d days); maximum is %d", days, maxDateRangeDays)
	}

	result := &model.DateRangeResult{
		Origin:      params.Origin,
		Destination: params.Destination,
		FromDate:    params.FromDate,
		ToDate:      params.ToDate,
	}

	for d := 0; d < days; d++ {
		departDate := from.AddDate(0, 0, d)
		date := departDate.Format(layout)

		searchParams := model.SearchParams{
			Origin:      params.Origin,
			Destination: params.Destination,
			Date:        date,
			Adults:      params.Adults,
			Children:    params.Children,
			Class:       params.Class,
			Limit:       1, // cheapest only
		}
		if params.StayDays > 0 {
			searchParams.ReturnDate = departDate.AddDate(0, 0, params.StayDays).Format(layout)
		}

		sr, err := s.Search(searchParams)
		if err != nil || len(sr.Flights) == 0 {
			// Skip dates with no results or errors (network blip etc.)
			if d < days-1 {
				time.Sleep(dateRangeDelay)
			}
			continue
		}

		cheapest := sr.Flights[0]
		dp := model.DatePrice{
			Date:     date,
			Price:    cheapest.Price,
			Currency: cheapest.Currency,
			Airline:  cheapest.Airline,
			Duration: cheapest.Duration,
			Stops:    cheapest.Stops,
		}
		if params.StayDays > 0 {
			dp.ReturnDate = departDate.AddDate(0, 0, params.StayDays).Format(layout)
		}
		result.Dates = append(result.Dates, dp)

		if d < days-1 {
			time.Sleep(dateRangeDelay)
		}
	}

	return result, nil
}

// fetchFlights sends the POST request and returns the raw (prefix-stripped) JSON string.
func (s *GoogleFlightsScraper) fetchFlights(params model.SearchParams) (string, error) {
	fReq, err := buildFReq(params)
	if err != nil {
		return "", fmt.Errorf("build f.req: %w", err)
	}

	body := strings.NewReader("f.req=" + fReq)
	req, err := http.NewRequest(http.MethodPost, rpcEndpoint, body)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// Exclude brotli — Go stdlib doesn't support it.
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("X-Same-Domain", "1")
	req.Header.Set("Google-Fieldnum", "100")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	var reader io.Reader = resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", fmt.Errorf("gzip reader: %w", err)
		}
		defer gr.Close()
		reader = gr
	}

	raw, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	text := strings.TrimPrefix(string(raw), antiXSSIPrefix)
	text = strings.TrimSpace(text)
	return text, nil
}
