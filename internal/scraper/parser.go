// Package scraper — parser.go
//
// Parses Google Flights RPC API JSON responses.
//
// The POST API returns an outer JSON array where parsed[0][2] is itself a
// JSON string (double-encoded). After unwrapping, the inner payload has the
// same structure as what the HTML approach embedded in the ds:1 script tag.
//
// Verified payload structure (debug-confirmed against live response):
//
//	outer JSON          — top-level array
//	outer[0][2]         — JSON string, must be parsed again → inner
//	inner[2][0]         — "best" flight options list
//	inner[3][0]         — additional flight options list
//	k                   — one flight option entry (from either list)
//	  k[0]              — main flight info block
//	    k[0][0]         — airline IATA code (string)
//	    k[0][1]         — []string of airline names, e.g. ["Scoot"]
//	    k[0][2]         — []segments (count - 1 = stops)
//	    k[0][3]         — origin IATA code (string)
//	    k[0][4]         — departure date [year, month, day]
//	    k[0][5]         — departure time [hour, minute]
//	    k[0][6]         — destination IATA code (string)
//	    k[0][7]         — arrival date [year, month, day]
//	    k[0][8]         — arrival time [hour, minute]
//	    k[0][9]         — total duration in minutes (float64)
//	  k[1][0][1]        — price (float64)
package scraper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/eternnoir/gf-cli/internal/model"
)

// parseFlights extracts flight results and a price-trend label from the raw
// (anti-XSSI-stripped) JSON string returned by the POST API.
func parseFlights(raw string) ([]model.Flight, string, error) {
	inner, err := extractInnerPayload(raw)
	if err != nil {
		return nil, "", err
	}

	flights, err := buildFlights(inner)
	if err != nil {
		return nil, "", err
	}

	priceTrend := extractPriceTrend(inner)
	return flights, priceTrend, nil
}

// extractInnerPayload unwraps the double-encoded JSON response.
//
// The outer JSON is an array; outer[0][2] is a JSON string that must be
// parsed a second time to get the actual flight data payload.
func extractInnerPayload(raw string) ([]interface{}, error) {
	var outer []interface{}
	if err := json.Unmarshal([]byte(raw), &outer); err != nil {
		return nil, fmt.Errorf("outer JSON parse error: %w", err)
	}

	if len(outer) == 0 {
		return nil, fmt.Errorf("empty outer response")
	}

	first, ok := outer[0].([]interface{})
	if !ok || len(first) < 3 {
		return nil, fmt.Errorf("unexpected outer[0] structure")
	}

	innerStr, ok := first[2].(string)
	if !ok || innerStr == "" {
		return nil, fmt.Errorf("outer[0][2] is not a string or is empty")
	}

	var inner []interface{}
	if err := json.Unmarshal([]byte(innerStr), &inner); err != nil {
		return nil, fmt.Errorf("inner JSON parse error: %w", err)
	}

	return inner, nil
}

// buildFlights navigates the inner payload and constructs []model.Flight.
//
// Flight options appear at inner[2][0] ("best flights") and inner[3][0]
// ("more flights").  We collect from both lists, deduplicating by
// (airline, departure_time, price).
func buildFlights(inner []interface{}) ([]model.Flight, error) {
	var flights []model.Flight
	seen := make(map[string]bool)

	for _, listIdx := range []int{2, 3} {
		flightList := nestedSlice(inner, listIdx, 0)
		if flightList == nil {
			continue
		}
		for _, raw := range flightList {
			k := toSlice(raw)
			if len(k) == 0 {
				continue
			}

			fi := toSlice(k[0]) // fi = k[0], main flight info block
			if len(fi) < 10 {
				continue
			}

			airline := extractAirlineNames(fi)
			segments := toSlice(fi[2])
			stops := 0
			if len(segments) > 1 {
				stops = len(segments) - 1
			}

			depTime := formatTime(fi, 5)
			arrTime := formatTime(fi, 8)
			duration := formatDuration(fi, 9)
			depDate := formatDate(fi, 4)
			arrDate := formatDate(fi, 7)

			if arrDate == depDate {
				arrDate = ""
			}

			price := extractPrice(k)

			// Deduplicate
			key := fmt.Sprintf("%s|%s|%.0f", airline, depTime, price)
			if seen[key] {
				continue
			}
			seen[key] = true

			flights = append(flights, model.Flight{
				Airline:       airline,
				DepartureTime: depTime,
				ArrivalTime:   arrTime,
				ArrivalDate:   arrDate,
				Duration:      duration,
				Stops:         stops,
				Price:         price,
				Currency:      "USD",
			})
		}
	}

	return flights, nil
}

// extractAirlineNames reads fi[1] which is []string of airline names.
func extractAirlineNames(fi []interface{}) string {
	if len(fi) < 2 {
		return ""
	}
	arr := toSlice(fi[1])
	var names []string
	for _, v := range arr {
		if name, ok := v.(string); ok && name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, " / ")
}

// formatTime reads fi[idx] = [hour, minute] and returns "HH:MM".
func formatTime(fi []interface{}, idx int) string {
	if idx >= len(fi) {
		return ""
	}
	hm := toSlice(fi[idx])
	if len(hm) == 0 {
		return ""
	}
	h, ok := hm[0].(float64)
	if !ok {
		return ""
	}
	m := 0.0
	if len(hm) >= 2 {
		m, _ = hm[1].(float64)
	}
	return fmt.Sprintf("%02d:%02d", int(h), int(m))
}

// formatDate reads fi[idx] = [year, month, day] and returns "YYYY-MM-DD".
func formatDate(fi []interface{}, idx int) string {
	if idx >= len(fi) {
		return ""
	}
	ymd := toSlice(fi[idx])
	if len(ymd) < 3 {
		return ""
	}
	y, okY := ymd[0].(float64)
	mo, okM := ymd[1].(float64)
	d, okD := ymd[2].(float64)
	if !okY || !okM || !okD {
		return ""
	}
	return fmt.Sprintf("%04d-%02d-%02d", int(y), int(mo), int(d))
}

// formatDuration reads fi[idx] = minutes (float64) and returns "Xhr Ymin".
func formatDuration(fi []interface{}, idx int) string {
	if idx >= len(fi) {
		return ""
	}
	mins, ok := fi[idx].(float64)
	if !ok || mins <= 0 {
		return ""
	}
	total := int(mins)
	h, m := total/60, total%60
	if m == 0 {
		return fmt.Sprintf("%dhr", h)
	}
	return fmt.Sprintf("%dhr %dmin", h, m)
}

// extractPrice reads k[1][0][1] (float64).
func extractPrice(k []interface{}) float64 {
	if len(k) < 2 {
		return 0
	}
	l1 := toSlice(k[1])
	if len(l1) == 0 {
		return 0
	}
	l2 := toSlice(l1[0])
	if len(l2) < 2 {
		return 0
	}
	v, _ := l2[1].(float64)
	return v
}

// extractPriceTrend does a best-effort search for a price trend label.
func extractPriceTrend(inner []interface{}) string {
	for _, idx := range []int{0, 3} {
		s := nestedSlice(inner, idx, 3)
		for _, v := range s {
			if t, ok := v.(string); ok {
				lower := strings.ToLower(t)
				if strings.Contains(lower, "low") || strings.Contains(lower, "high") ||
					strings.Contains(lower, "typical") {
					return t
				}
			}
		}
	}
	return ""
}

// ─── helpers ────────────────────────────────────────────────────────────────

func toSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	s, _ := v.([]interface{})
	return s
}

func nestedSlice(root []interface{}, indices ...int) []interface{} {
	cur := root
	for _, idx := range indices {
		if idx >= len(cur) {
			return nil
		}
		cur = toSlice(cur[idx])
		if cur == nil {
			return nil
		}
	}
	return cur
}
