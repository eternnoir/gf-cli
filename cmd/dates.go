// Package cmd — dates.go
//
// Provides the `dates` subcommand for finding the cheapest flight across a
// date range.
//
// Usage:
//
//	gf-cli dates [ORIGIN] [DESTINATION] --from YYYY-MM-DD --to YYYY-MM-DD [flags]
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/eternnoir/gf-cli/internal/formatter"
	"github.com/eternnoir/gf-cli/internal/model"
	"github.com/eternnoir/gf-cli/internal/scraper"
)

var (
	flagFromDate      string
	flagToDate        string
	flagDatesClass    string
	flagDatesAdults   int
	flagDatesChildren int
	flagDatesJSON     bool
	flagStayDays      int
)

var datesCmd = &cobra.Command{
	Use:   "dates [ORIGIN] [DESTINATION]",
	Short: "Find cheapest flights across a date range",
	Long: `Search for the cheapest flight option for each day in a date range.
Useful for flexible travel planning — quickly see which dates are cheapest.

Use --stay N to search round-trip prices (N = number of days at destination).
Without --stay, each date is searched as a one-way flight.

Maximum range: 61 days.

Example:
  gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31
  gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31 --stay 7
  gf-cli dates JFK LHR --from 2026-06-01 --to 2026-06-30 --class business --json`,
	Args: cobra.ExactArgs(2),
	RunE: runDateRange,
}

func init() {
	datesCmd.Flags().StringVar(&flagFromDate, "from", "", "Start date of range (YYYY-MM-DD) [required]")
	datesCmd.Flags().StringVar(&flagToDate, "to", "", "End date of range (YYYY-MM-DD) [required]")
	datesCmd.Flags().StringVarP(&flagDatesClass, "class", "C", "economy", "Seat class (economy|premium-economy|business|first)")
	datesCmd.Flags().IntVarP(&flagDatesAdults, "adults", "a", 1, "Number of adults")
	datesCmd.Flags().IntVarP(&flagDatesChildren, "children", "c", 0, "Number of children")
	datesCmd.Flags().BoolVar(&flagDatesJSON, "json", false, "Output as JSON")
	datesCmd.Flags().IntVar(&flagStayDays, "stay", 0, "Round-trip: number of days to stay at destination (0 = one-way)")

	_ = datesCmd.MarkFlagRequired("from")
	_ = datesCmd.MarkFlagRequired("to")

	rootCmd.AddCommand(datesCmd)
}

func runDateRange(cmd *cobra.Command, args []string) error {
	origin := args[0]
	destination := args[1]

	// Validate dates
	from, err := time.Parse(dateLayout, flagFromDate)
	if err != nil {
		return fmt.Errorf("invalid --from format %q; expected YYYY-MM-DD", flagFromDate)
	}
	if from.Before(time.Now().Truncate(24 * time.Hour)) {
		return fmt.Errorf("from-date %s is in the past", flagFromDate)
	}
	to, err := time.Parse(dateLayout, flagToDate)
	if err != nil {
		return fmt.Errorf("invalid --to format %q; expected YYYY-MM-DD", flagToDate)
	}
	if !to.After(from) && flagToDate != flagFromDate {
		return fmt.Errorf("--to date must be on or after --from date")
	}

	// Validate seat class
	seatClass, err := formatter.ValidateSeatClass(flagDatesClass)
	if err != nil {
		return err
	}

	if flagStayDays < 0 {
		return fmt.Errorf("--stay must be 0 or greater")
	}

	params := model.DateRangeParams{
		Origin:      origin,
		Destination: destination,
		FromDate:    flagFromDate,
		ToDate:      flagToDate,
		Adults:      flagDatesAdults,
		Children:    flagDatesChildren,
		Class:       seatClass,
		StayDays:    flagStayDays,
	}

	s := scraper.NewGoogleFlightsScraper()
	result, err := s.SearchDateRange(params)
	if err != nil {
		return fmt.Errorf("date range search failed: %w", err)
	}

	if flagDatesJSON {
		return formatter.PrintDateRangeJSON(result)
	}
	formatter.PrintDateRangeText(result)
	return nil
}
