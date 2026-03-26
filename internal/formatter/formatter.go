// Package formatter handles rendering flight search results in different formats.
package formatter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/eternnoir/gf-cli/internal/model"
)

const divider = "─────────────────────────────────────────"

// PrintText renders a SearchResult as human-readable text to stdout.
func PrintText(result *model.SearchResult) {
	fmt.Printf("✈  %s → %s  |  %s", result.Origin, result.Destination, result.Date)
	if result.ReturnDate != "" {
		fmt.Printf(" → %s (round trip)", result.ReturnDate)
	}
	if result.PriceTrend != "" {
		fmt.Printf("  |  Prices: %s", result.PriceTrend)
	}
	fmt.Println()
	fmt.Println(divider)

	if len(result.Flights) == 0 {
		fmt.Println("No flights found.")
		return
	}

	for i, f := range result.Flights {
		stopLabel := formatStops(f.Stops)
		arrivalExtra := ""
		if f.ArrivalDate != "" {
			arrivalExtra = fmt.Sprintf(" (+%s)", f.ArrivalDate)
		}

		fmt.Printf("%d. %s\n", i+1, f.Airline)
		fmt.Printf("   🕐 %s → %s%s  (%s)  |  %s\n",
			f.DepartureTime, f.ArrivalTime, arrivalExtra, f.Duration, stopLabel)
		fmt.Printf("   💰 %s %.0f\n", f.Currency, f.Price)
		fmt.Println(divider)
	}
}

// PrintJSON renders a SearchResult as pretty-printed JSON to stdout.
func PrintJSON(result *model.SearchResult) error {
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode results as JSON: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

// PrintDateRangeText renders a DateRangeResult as a human-readable price table.
func PrintDateRangeText(result *model.DateRangeResult) {
	// Detect round-trip mode from first result that has ReturnDate set.
	isRoundTrip := len(result.Dates) > 0 && result.Dates[0].ReturnDate != ""

	header := fmt.Sprintf("📅 %s → %s  |  %s to %s",
		result.Origin, result.Destination, result.FromDate, result.ToDate)
	if isRoundTrip {
		// Calculate stay length from first entry for the header label.
		header += "  (round trip)"
	}
	fmt.Println(header)
	fmt.Println(divider)

	if len(result.Dates) == 0 {
		fmt.Println("No flights found for any date in range.")
		return
	}

	// Find cheapest date for highlight
	minPrice := result.Dates[0].Price
	for _, dp := range result.Dates[1:] {
		if dp.Price < minPrice {
			minPrice = dp.Price
		}
	}

	for _, dp := range result.Dates {
		tag := ""
		if dp.Price == minPrice {
			tag = " ⭐ cheapest"
		}
		stopLabel := formatStops(dp.Stops)
		dateLabel := dp.Date
		if dp.ReturnDate != "" {
			dateLabel = dp.Date + " → " + dp.ReturnDate
		}
		fmt.Printf("  %s  💰 %s %.0f  ✈ %s  ⏱ %s  %s%s\n",
			dateLabel, dp.Currency, dp.Price, dp.Airline, dp.Duration, stopLabel, tag)
	}
	fmt.Println(divider)
	fmt.Printf("Total: %d dates with available flights\n", len(result.Dates))
}

// PrintDateRangeJSON renders a DateRangeResult as pretty-printed JSON to stdout.
func PrintDateRangeJSON(result *model.DateRangeResult) error {
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode date range results as JSON: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

// formatStops converts a stop count integer to a readable label.
func formatStops(stops int) string {
	switch stops {
	case 0:
		return "Nonstop"
	case 1:
		return "1 stop"
	default:
		return fmt.Sprintf("%d stops", stops)
	}
}

// ValidateSeatClass checks whether the given class string is a recognized value.
func ValidateSeatClass(class string) (model.SeatClass, error) {
	valid := []model.SeatClass{
		model.SeatClassEconomy,
		model.SeatClassPremiumEconomy,
		model.SeatClassBusiness,
		model.SeatClassFirst,
	}
	for _, v := range valid {
		if strings.EqualFold(string(v), class) {
			return v, nil
		}
	}
	return "", fmt.Errorf("invalid seat class %q; choose from: economy, premium-economy, business, first", class)
}
