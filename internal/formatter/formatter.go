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
	enc := json.NewEncoder(nil)
	_ = enc
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode results as JSON: %w", err)
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
