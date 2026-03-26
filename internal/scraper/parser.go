// Package scraper — parser.go
//
// Parses Google Flights HTML responses.
//
// Google embeds flight data as JSON inside a <script class="ds:1"> tag.
// The content looks like:  AF_initDataCallback({key:"ds:1", ... data:[...], ...})
// We extract the array after "data:" and navigate its nested structure.
//
// Verified payload structure (debug-confirmed against live Google Flights response):
//
//   payload[3][0]          — list of flight options, each element is k
//   k[0]                   — main flight info block (length ~25)
//     k[0][1]              — []string of airline names (e.g. ["Scoot"])
//     k[0][2]              — []segments (count - 1 = stops)
//     k[0][3]              — origin IATA code (string)
//     k[0][4]              — departure date [year, month, day]
//     k[0][5]              — departure time [hour, minute]
//     k[0][6]              — destination IATA code (string)
//     k[0][7]              — arrival date [year, month, day]
//     k[0][8]              — arrival time [hour, minute]
//     k[0][9]              — total duration in minutes (float64)
//   k[1][0][1]             — price (float64, USD)
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
//
// Key structure (verified against live response):
//
//	payload[3][0] → slice of flight options k
//	k[0]          → flight info block
//	k[1][0][1]    → price (float64)
func buildFlights(payload []interface{}) ([]model.Flight, error) {
	flightList := nestedSlice(payload, 3, 0)
	if flightList == nil {
		return nil, fmt.Errorf("flight data section not found (payload[3][0])")
	}

	var flights []model.Flight
	for _, raw := range flightList {
		k := toSlice(raw)
		if len(k) == 0 {
			continue
		}

		fi := toSlice(k[0]) // fi = k[0], main flight info
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

		// Suppress arrDate if it equals depDate (same-day arrival is the norm)
		if arrDate == depDate {
			arrDate = ""
		}

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

// extractAirlineNames reads fi[1] which is []string of airline names.
// Example: fi[1] = ["Scoot"] or ["Delta", "KLM"] for codeshare.
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
// Google omits the minute element when it is 0, so [9] means 09:00.
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
func extractPriceTrend(payload []interface{}) string {
	for _, idx := range []int{0, 3} {
		s := nestedSlice(payload, idx, 3)
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
