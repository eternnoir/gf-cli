// Package scraper handles fetching flight data from Google Flights.
//
// NOTE FOR KAI: This is the stub you need to implement.
// The interface is defined — please fill in SearchFlights().
//
// Suggested approach (same as the Python fast-flights library):
//   1. Build a tfs (Travel Form Spec) encoded protobuf query string
//   2. POST to https://www.google.com/travel/flights/search
//      with the encoded query as ?tfs= parameter
//   3. Parse the HTML/JSON response to extract flight data
//
// Reference: https://github.com/AWeirdDev/flights (Python impl)
// The key URL pattern is:
//   https://www.google.com/travel/flights/search?tfs=<base64-protobuf>
//
// Libraries you may find useful:
//   - net/http  (standard HTTP client)
//   - golang.org/x/net/html  (HTML parsing)
//   - google.golang.org/protobuf  (protobuf encoding, if needed)
//   OR just reverse-engineer the JSON API endpoint directly.
package scraper

import (
	"fmt"

	"github.com/eternnoir/gf-cli/internal/model"
)

// Searcher defines the contract for flight search implementations.
// This allows easy swapping/mocking in tests.
type Searcher interface {
	Search(params model.SearchParams) (*model.SearchResult, error)
}

// GoogleFlightsScraper implements Searcher using Google Flights web scraping.
type GoogleFlightsScraper struct{}

// NewGoogleFlightsScraper creates a new scraper instance.
func NewGoogleFlightsScraper() *GoogleFlightsScraper {
	return &GoogleFlightsScraper{}
}

// Search performs a flight search with the given parameters.
// TODO (Kai): Implement the actual Google Flights scraping logic here.
func (s *GoogleFlightsScraper) Search(params model.SearchParams) (*model.SearchResult, error) {
	return nil, fmt.Errorf("Search not yet implemented — Kai, this is yours! See package doc for hints")
}
