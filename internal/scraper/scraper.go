// Package scraper handles fetching flight data from Google Flights.
//
// Implementation overview:
//  1. Encode search parameters as proto3 binary (see proto.go)
//  2. Base64-encode the bytes → ?tfs= URL parameter
//  3. GET https://www.google.com/travel/flights/search?tfs=<b64>&hl=en&curr=USD
//  4. Parse the embedded ds:1 JSON script tag (see parser.go)
//
// Reference: https://github.com/AWeirdDev/flights (Python fast-flights)
package scraper

import (
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eternnoir/gf-cli/internal/model"
)

const (
	googleFlightsURL = "https://www.google.com/travel/flights/search"
	// Chrome 120 UA — keeps Google's bot detection happy enough for common queries.
	userAgent      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	requestTimeout = 30 * time.Second
)

// Searcher defines the contract for flight search implementations.
// This allows easy swapping / mocking in tests.
type Searcher interface {
	Search(params model.SearchParams) (*model.SearchResult, error)
}

// GoogleFlightsScraper implements Searcher using Google Flights web scraping.
type GoogleFlightsScraper struct {
	client *http.Client
}

// NewGoogleFlightsScraper creates a new scraper instance.
func NewGoogleFlightsScraper() *GoogleFlightsScraper {
	return &GoogleFlightsScraper{
		client: &http.Client{Timeout: requestTimeout},
	}
}

// Search performs a flight search with the given parameters.
func (s *GoogleFlightsScraper) Search(params model.SearchParams) (*model.SearchResult, error) {
	tfsBytes := buildTFS(params)
	tfs := base64.StdEncoding.EncodeToString(tfsBytes)

	targetURL := googleFlightsURL + "?tfs=" + url.QueryEscape(tfs) + "&hl=en&curr=USD"

	htmlContent, err := s.fetchHTML(targetURL)
	if err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	flights, priceTrend, err := parseFlights(htmlContent)
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

// fetchHTML sends a browser-impersonating GET request and returns the response body.
// Go's standard net/http TLS stack works for most Google Flights queries; if Google
// starts blocking aggressively, replace this client with bogdanfinn/tls-client.
func (s *GoogleFlightsScraper) fetchHTML(targetURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	// Explicitly exclude brotli — we only handle gzip in the response reader below.
	// Sending "br" in Accept-Encoding causes Google to respond with brotli which
	// we cannot decode without an external dependency.
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status %d for %s", resp.StatusCode, targetURL)
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

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read body: %w", err)
	}
	return string(body), nil
}
