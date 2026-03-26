# gf-cli Skill

**gf-cli** is a command-line tool that searches Google Flights from the terminal. No API key required.

## When to use

Use this tool when asked to:
- Find flights between two airports on a specific date (one-way or round trip)
- Search round-trip fares — specify departure date + return date, or use date-range to find cheapest departure day with a fixed stay length
- Find the cheapest day to fly within a date range (one-way or round trip with `--stay`)
- Compare flight prices, airlines, and durations
- Output flight data as JSON for further processing or analysis

> **Round trip is the most common use case.** Always use `--return` (specific dates) or `--stay` (date range) unless the user explicitly wants one-way.

## Commands

### 1. Search flights on a specific date

```bash
gf-cli [ORIGIN] [DESTINATION] --date YYYY-MM-DD [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--date` | `-d` | required | Departure date (YYYY-MM-DD) |
| `--return` | `-r` | — | **Return date for round trip (YYYY-MM-DD).** When set, returns combined outbound + return price. Omit for one-way. |
| `--adults` | `-a` | 1 | Number of adults |
| `--children` | `-c` | 0 | Number of children |
| `--class` | `-C` | economy | Seat class: `economy` / `premium-economy` / `business` / `first` |
| `--limit` | `-l` | 10 | Max results |
| `--json` | — | false | JSON output (recommended for programmatic use) |

### 2. Find cheapest departure date across a range

```bash
gf-cli dates [ORIGIN] [DESTINATION] --from YYYY-MM-DD --to YYYY-MM-DD [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--from` | required | Start of departure date range (YYYY-MM-DD) |
| `--to` | required | End of departure date range (YYYY-MM-DD, max 61 days from `--from`) |
| `--stay` | 0 (one-way) | **Number of days to stay at destination for round-trip search.** Each departure date is automatically paired with `departure + N days` as the return date. Set to 0 or omit for one-way. |
| `--adults` | 1 | Number of adults |
| `--children` | 0 | Number of children |
| `--class` | economy | Seat class |
| `--json` | false | JSON output |

## Examples

```bash
# Round trip on specific dates (most common)
gf-cli TPE NRT --date 2026-05-01 --return 2026-05-08

# Round trip, business class, 2 adults
gf-cli JFK LHR --date 2026-06-15 --return 2026-06-22 --class business --adults 2

# Round trip + JSON output (for Agent use)
gf-cli TPE NRT --date 2026-05-01 --return 2026-05-08 --json

# One-way (explicit)
gf-cli TPE NRT --date 2026-05-01

# Find cheapest round-trip departure in May, staying 7 days (most common dates use case)
gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31 --stay 7

# Same, JSON output for Agent
gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31 --stay 7 --json

# Find cheapest one-way day in May (no --stay)
gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31
```

## JSON output schema

### Flight search (`gf-cli ... --json`)

```json
{
  "origin": "TPE",
  "destination": "NRT",
  "date": "2026-05-01",
  "return_date": "2026-05-08",
  "flights": [
    {
      "airline": "Scoot",
      "departure_time": "15:30",
      "arrival_time": "20:00",
      "duration": "3hr 30min",
      "stops": 0,
      "price": 312,
      "currency": "USD",
      "arrival_date": ""
    }
  ]
}
```

> `return_date` is present when `--return` was used. `arrival_date` is non-empty only when the flight arrives on a different calendar day.

### Date range search (`gf-cli dates ... --json`)

```json
{
  "origin": "TPE",
  "destination": "NRT",
  "from_date": "2026-05-01",
  "to_date": "2026-05-31",
  "dates": [
    {
      "date": "2026-05-03",
      "price": 261,
      "currency": "USD",
      "airline": "Jetstar",
      "duration": "3hr 20min",
      "stops": 0
    }
  ]
}
```

> When `--stay` is used, `price` reflects the combined round-trip fare (outbound + return). The `date` field is the **departure date**; return date = `date + stay days`.

## Notes

- Prices are fetched live from Google Flights; results may vary by time of request
- Airport codes follow IATA format (e.g. TPE, NRT, JFK, LHR, SFO, SIN)
- `dates` command makes one API call per day in the range — for a 31-day range, ~31 requests with 500ms spacing (~15 seconds total)
- If Google rate-limits a request (HTTP 429), that date is silently skipped in the results
