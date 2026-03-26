// Package scraper — query.go
//
// Builds the f.req JSON payload for Google Flights' internal RPC endpoint.
//
// Reference: https://github.com/punitarani/fli (Python)
// The endpoint accepts a POST with body: f.req=<url_encoded_json>
// where the JSON encodes flight search parameters as a deeply-nested array.
package scraper

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/eternnoir/gf-cli/internal/model"
)

// Seat type values (Google Flights internal API)
const (
	apiSeatEconomy        = 1
	apiSeatPremiumEconomy = 2
	apiSeatBusiness       = 3
	apiSeatFirst          = 4
)

// Trip type values
const (
	apiTripRoundTrip = 1
	apiTripOneWay    = 2
)

// Sort by values
const (
	apiSortByBestFlights = 0
	apiSortByPrice       = 1
	apiSortByDuration    = 2
)

// buildFReq constructs the URL-encoded f.req parameter for the POST API.
//
// The structure mirrors FlightSearchFilters.format() + encode() from fli:
//
//	[None, json.dumps(filters)]  →  url_encode
//
// Where filters is:
//
//	[[], [None, None, trip_type, None, [], seat, passengers, None×5, segments, None×3, 1], sort, 0, 0, 2]
func buildFReq(params model.SearchParams) (string, error) {
	seat := seatValue(params.Class)
	tripType := apiTripOneWay
	if params.ReturnDate != "" {
		tripType = apiTripRoundTrip
	}

	adults := params.Adults
	if adults == 0 {
		adults = 1
	}

	// Build segments
	var segments []interface{}
	outbound := buildSegment(params.Origin, params.Destination, params.Date)
	segments = append(segments, outbound)
	if params.ReturnDate != "" {
		ret := buildSegment(params.Destination, params.Origin, params.ReturnDate)
		segments = append(segments, ret)
	}

	passengers := []int{adults, params.Children, 0, 0}

	filters := []interface{}{
		[]interface{}{},
		[]interface{}{
			nil, nil,
			tripType,
			nil,
			[]interface{}{},
			seat,
			passengers,
			nil, nil, nil, nil, nil, nil,
			segments,
			nil, nil, nil,
			1,
		},
		apiSortByBestFlights,
		0, 0, 2,
	}

	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return "", fmt.Errorf("marshal filters: %w", err)
	}

	outer := []interface{}{nil, string(filtersJSON)}
	outerJSON, err := json.Marshal(outer)
	if err != nil {
		return "", fmt.Errorf("marshal outer: %w", err)
	}

	return url.QueryEscape(string(outerJSON)), nil
}

// buildSegment creates one flight segment entry.
//
// Segment structure (15 elements):
//   [0] departure airport filter (3-level nested)
//   [1] arrival airport filter (3-level nested)
//   [2] time restrictions (nil)
//   [3] stops (0 = any)
//   [4] airlines filter (nil)
//   [5] placeholder (nil)
//   [6] travel date string
//   [7] max duration (nil)
//   [8] selected flight (nil, used for round-trip return leg only)
//   [9] layover airports (nil)
//   [10] placeholder (nil)
//   [11] placeholder (nil)
//   [12] layover duration (nil)
//   [13] emissions (nil)
//   [14] constant 3
func buildSegment(from, to, date string) []interface{} {
	depFilter := [][][]string{{{from, from}}}
	arrFilter := [][][]string{{{to, to}}}
	return []interface{}{
		depFilter, arrFilter,
		nil, 0, nil, nil,
		date,
		nil, nil, nil, nil, nil, nil, nil,
		3,
	}
}

func seatValue(class model.SeatClass) int {
	switch class {
	case model.SeatClassPremiumEconomy:
		return apiSeatPremiumEconomy
	case model.SeatClassBusiness:
		return apiSeatBusiness
	case model.SeatClassFirst:
		return apiSeatFirst
	default:
		return apiSeatEconomy
	}
}
