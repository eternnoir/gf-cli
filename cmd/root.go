// Package cmd provides the CLI entry point for gf-cli.
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/eternnoir/gf-cli/internal/formatter"
	"github.com/eternnoir/gf-cli/internal/model"
	"github.com/eternnoir/gf-cli/internal/scraper"
)

const dateLayout = "2006-01-02"

var rootCmd = &cobra.Command{
	Use:   "gf-cli [ORIGIN] [DESTINATION]",
	Short: "Search Google Flights from the command line",
	Long: `gf-cli lets you search Google Flights without an API key.
Retrieve prices, times, and airline info straight from your terminal.`,
	Args: cobra.ExactArgs(2),
	RunE: runSearch,
}

// flags
var (
	flagDate       string
	flagReturn     string
	flagAdults     int
	flagChildren   int
	flagClass      string
	flagLimit      int
	flagOutputFmt  string
)

func init() {
	rootCmd.Flags().StringVarP(&flagDate, "date", "d", "", "Departure date (YYYY-MM-DD) [required]")
	rootCmd.Flags().StringVarP(&flagReturn, "return", "r", "", "Return date for round trips (YYYY-MM-DD)")
	rootCmd.Flags().IntVarP(&flagAdults, "adults", "a", 1, "Number of adults")
	rootCmd.Flags().IntVarP(&flagChildren, "children", "c", 0, "Number of children")
	rootCmd.Flags().StringVarP(&flagClass, "class", "C", "economy", "Seat class (economy|premium-economy|business|first)")
	rootCmd.Flags().IntVarP(&flagLimit, "limit", "l", 10, "Maximum number of results")
	rootCmd.Flags().StringVarP(&flagOutputFmt, "output", "o", "text", "Output format (text|json)")

	_ = rootCmd.MarkFlagRequired("date")
}

// Execute is the entry point called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runSearch(cmd *cobra.Command, args []string) error {
	origin := args[0]
	destination := args[1]

	// Validate date format and ensure it is not in the past.
	departDate, err := time.Parse(dateLayout, flagDate)
	if err != nil {
		return fmt.Errorf("invalid --date format %q; expected YYYY-MM-DD", flagDate)
	}
	if departDate.Before(time.Now().Truncate(24 * time.Hour)) {
		return fmt.Errorf("departure date %s is in the past", flagDate)
	}

	// Validate optional return date.
	if flagReturn != "" {
		returnDate, err := time.Parse(dateLayout, flagReturn)
		if err != nil {
			return fmt.Errorf("invalid --return format %q; expected YYYY-MM-DD", flagReturn)
		}
		if !returnDate.After(departDate) {
			return fmt.Errorf("return date must be after departure date")
		}
	}

	// Validate seat class.
	seatClass, err := formatter.ValidateSeatClass(flagClass)
	if err != nil {
		return err
	}

	// Validate output format.
	if flagOutputFmt != "text" && flagOutputFmt != "json" {
		return fmt.Errorf("invalid --output %q; choose from: text, json", flagOutputFmt)
	}

	params := model.SearchParams{
		Origin:      origin,
		Destination: destination,
		Date:        flagDate,
		ReturnDate:  flagReturn,
		Adults:      flagAdults,
		Children:    flagChildren,
		Class:       seatClass,
		Limit:       flagLimit,
	}

	s := scraper.NewGoogleFlightsScraper()
	result, err := s.Search(params)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	switch model.OutputFormat(flagOutputFmt) {
	case model.OutputFormatJSON:
		return formatter.PrintJSON(result)
	default:
		formatter.PrintText(result)
	}

	return nil
}
