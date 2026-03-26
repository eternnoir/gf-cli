// Package scraper — parser.go
//
// Parses Google Flights HTML responses.
//
// Google embeds flight data as JSON inside a <script class="ds:1"> tag.
// The content looks like:  AF_initDataCallback({key:"ds:1", ... data:[...], ...})
// We extract the array after "data:" and navigate its nested structure.
//
// Array path reference (reverse-engineered from fast-flights Python parser):
//   payload[3][0]          — list of flight options
//   k[0]                   — flight info for option k
//     flight[1]            — airline names ([]interface{})
//     flight[2]            — segments ([]interface{})
//       segment[3]         — origin IATA code
//       segment[6]         — destination IATA code
//       segment[8]         — departure time string
//       segment[10]        — arrival time string
//       segment[11]        — duration string
//       segment[20]        — departure date
//       segment[21]        — arrival date
//   k[1][0][1]             — price (float64)
//   payload[7][1][1]       — airline lookup table
package scraper

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/net/html"

	"github.com/eternnoir/gf-cli/internal/model"
)

// parseFlights extracts flight results and a price-trend label from raw HTML.
func parseFlights(htmlContent string) ([]model.Flight, string, error) {
	scriptText, err := extractDS1Script(htmlContent)
	if err != nil {
		return nil, "", err
	}

	payload, err := extractPayload(scriptText)
	if err != nil {
		return nil, "", err
	}

	flights, err := buildFlights(payload)
	if err != nil {
		return nil, "", err
	}

	priceTrend := extractPriceTrend(payload)
	return flights, priceTrend, nil
}

// extractDS1Script finds <script class="ds:1"> and returns its text content.
func extractDS1Script(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("HTML parse error: %w", err)
	}

	var content string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if content != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, a := range n.Attr {
				if a.Key == "class" {
					// class may be exactly "ds:1" or contain it
					for _, cls := range strings.Fields(a.Val) {
						if cls == "ds:1" {
							if n.FirstChild != nil {
								content = n.FirstChild.Data
							}
							return
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if content == "" {
		return "", fmt.Errorf("script tag 'ds:1' not found — Google may have blocked the request or changed their page structure")
	}
	return content, nil
}

// extractPayload parses the JSON array that follows "data:" inside the script text.
func extractPayload(scriptText string) ([]interface{}, error) {
	idx := strings.Index(scriptText, "data:")
	if idx < 0 {
		return nil, fmt.Errorf("'data:' key not found in script content")
	}

	rest := strings.TrimSpace(scriptText[idx+5:])
	if len(rest) == 0 || rest[0] != '[' {
		return nil, fmt.Errorf("expected JSON array after 'data:', got: %.40s", rest)
	}

	end := findClosingBracket(rest)
	if end < 0 {
		return nil, fmt.Errorf("unmatched brackets in data payload")
	}

	var payload []interface{}
	if err := json.Unmarshal([]byte(rest[:end+1]), &payload); err != nil {
		return nil, fmt.Errorf("JSON unmarshal error: %w", err)
	}
	return payload, nil
}

// findClosingBracket returns the index of the bracket that closes rest[0].
func findClosingBracket(s string) int {
	depth := 0
	inStr := false
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inStr {
			escaped = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		switch c {
		case '[', '{':
			depth++
		case ']', '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// buildFlights navigates the payload and constructs []model.Flight.
func buildFlights(payload []interface{}) ([]model.Flight, error) {
	// payload[3][0] = flight results list
	flightList := nestedSlice(payload, 3, 0)
	if flightList == nil {
		return nil, fmt.Errorf("flight data section not found in payload (payload[3][0])")
	}

	var flights []model.Flight
	for _, raw := range flightList {
		k, ok := raw.([]interface{})
		if !ok || len(k) == 0 {
			continue
		}

		// k[0] = flight info block
		flightInfo := toSlice(k[0])
		if flightInfo == nil || len(flightInfo) < 3 {
			continue
		}

		airline := extractAirlineName(flightInfo)
		segments := toSlice(flightInfo[2])
		if len(segments) == 0 {
			continue
		}

		firstSeg := toSlice(segments[0])
		lastSeg := toSlice(segments[len(segments)-1])
		if firstSeg == nil || lastSeg == nil {
			continue
		}

		depTime := strAt(firstSeg, 8)
		arrTime := strAt(lastSeg, 10)
		arrDate := strAt(lastSeg, 21)
		duration := totalDuration(segments)
		stops := len(segments) - 1
		price := extractPrice(k)

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
	return flights, nil
}

// extractAirlineName returns a comma-joined list of airlines for a flight block.
// flightInfo[1] contains airline name entries.
func extractAirlineName(flightInfo []interface{}) string {
	if len(flightInfo) < 2 {
		return ""
	}
	airlines := toSlice(flightInfo[1])
	var names []string
	for _, a := range airlines {
		as := toSlice(a)
		if len(as) > 0 {
			if name, ok := as[0].(string); ok && name != "" {
				names = append(names, name)
			}
		}
	}
	result := strings.Join(names, ", ")
	if result == "" {
		// fallback: direct string
		if s, ok := flightInfo[1].(string); ok {
			return s
		}
	}
	return result
}

// extractPrice reads k[1][0][1] and returns it as a float64.
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
	switch v := l2[1].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
}

// totalDuration sums segment durations (segment[11]).
// If segment[11] is already a formatted string like "2 hr 30 min", return the
// overall trip duration from the first segment's parent info if available;
// otherwise, just return the last segment's duration as a best-effort value.
func totalDuration(segments []interface{}) string {
	if len(segments) == 0 {
		return ""
	}
	// Use the last segment's duration as proxy for the full trip leg duration
	// since Google sometimes stores per-leg total there. Return segment[11] of
	// the first segment which often holds the connection duration for multi-leg.
	last := toSlice(segments[len(segments)-1])
	if d := strAt(last, 11); d != "" {
		return d
	}
	first := toSlice(segments[0])
	return strAt(first, 11)
}

// extractPriceTrend reads a price-trend label (e.g. "Low", "Typical") from the
// payload if present.  Location varies; we do a best-effort search.
func extractPriceTrend(payload []interface{}) string {
	// Common location: payload[0][3] or payload[3][3]
	for _, idx := range []int{0, 3} {
		s := nestedSlice(payload, idx, 3)
		if s != nil {
			for _, v := range s {
				if t, ok := v.(string); ok && t != "" {
					lower := strings.ToLower(t)
					if strings.Contains(lower, "low") || strings.Contains(lower, "high") ||
						strings.Contains(lower, "typical") {
						return t
					}
				}
			}
		}
	}
	return ""
}

// ─── helpers ────────────────────────────────────────────────────────────────

// toSlice casts interface{} to []interface{}, returning nil on failure.
func toSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	s, _ := v.([]interface{})
	return s
}

// nestedSlice walks a chain of integer indices into a []interface{} tree.
func nestedSlice(root []interface{}, indices ...int) []interface{} {
	cur := root
	for _, idx := range indices {
		if idx >= len(cur) {
			return nil
		}
		next := toSlice(cur[idx])
		if next == nil {
			return nil
		}
		cur = next
	}
	return cur
}

// strAt safely returns arr[idx] as a string, or "" on any failure.
func strAt(arr []interface{}, idx int) string {
	if arr == nil || idx >= len(arr) {
		return ""
	}
	s, _ := arr[idx].(string)
	return s
}
