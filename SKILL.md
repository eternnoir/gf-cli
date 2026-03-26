# gf-cli Skill

**gf-cli** is a command-line tool that searches Google Flights from the terminal. No API key required.

## When to use

Use this tool when asked to:
- Find flights between two airports on a specific date
- Compare flight prices, airlines, and durations
- Search round-trip fares with a return date
- Find the cheapest day to fly within a date range
- Output flight data as JSON for further processing

## Commands

### Search flights on a specific date

```bash
gf-cli [ORIGIN] [DESTINATION] --date YYYY-MM-DD [flags]
```

Key flags: `--date` (required), `--return` (round trip), `--adults`, `--children`, `--class` (economy/premium-economy/business/first), `--limit`, `--json`

### Find cheapest date in a range

```bash
gf-cli dates [ORIGIN] [DESTINATION] --from YYYY-MM-DD --to YYYY-MM-DD [flags]
```

Maximum range: 61 days. Use `--json` for machine-readable output.

## Examples

```bash
# Cheapest one-way flights TPE→NRT on May 1
gf-cli TPE NRT --date 2026-05-01

# Business class round trip JFK→LHR
gf-cli JFK LHR --date 2026-06-15 --return 2026-06-22 --class business

# JSON output for scripting or Agent use
gf-cli TPE NRT --date 2026-05-01 --json

# Find cheapest day to fly in May
gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31

# JSON date range (Agent friendly)
gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31 --json
```

## JSON output schema

### Flight search (`gf-cli ... --json`)

```json
{
  "origin": "TPE",
  "destination": "NRT",
  "date": "2026-05-01",
  "flights": [
    {
      "airline": "Scoot",
      "departure_time": "15:30",
      "arrival_time": "20:00",
      "duration": "3hr 30min",
      "stops": 0,
      "price": 151,
      "currency": "USD"
    }
  ]
}
```

### Date range search (`gf-cli dates ... --json`)

```json
{
  "origin": "TPE",
  "destination": "NRT",
  "from_date": "2026-05-01",
  "to_date": "2026-05-07",
  "dates": [
    {
      "date": "2026-05-03",
      "price": 107,
      "currency": "USD",
      "airline": "Jetstar",
      "duration": "3hr 20min",
      "stops": 0
    }
  ]
}
```

## Notes

- Prices are fetched live from Google Flights; results may vary by time of request
- Airport codes follow IATA format (e.g. TPE, NRT, JFK, LHR)
- `arrival_date` field appears in JSON only when the flight arrives on a different day
