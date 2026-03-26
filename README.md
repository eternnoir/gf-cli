# gf-cli

Search Google Flights directly from your terminal — no API key required.

## Installation

```bash
go install github.com/eternnoir/gf-cli@latest
```

Or build from source:

```bash
git clone https://github.com/eternnoir/gf-cli.git
cd gf-cli
go build -o gf-cli .
```

## Usage

### Search flights

```bash
gf-cli [ORIGIN] [DESTINATION] --date YYYY-MM-DD [flags]
```

```bash
# One-way, economy
gf-cli TPE NRT --date 2026-05-01

# Round trip, business class
gf-cli JFK LHR --date 2026-06-15 --return 2026-06-22 --class business

# Multiple passengers
gf-cli TPE NRT --date 2026-05-01 --adults 2 --children 2

# JSON output (Agent/script friendly)
gf-cli TPE NRT --date 2026-05-01 --json

# Limit results
gf-cli SFO ORD --date 2026-07-04 --limit 5
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--date` | `-d` | *required* | Departure date (YYYY-MM-DD) |
| `--return` | `-r` | — | Return date for round trips (YYYY-MM-DD) |
| `--adults` | `-a` | 1 | Number of adults |
| `--children` | `-c` | 0 | Number of children |
| `--class` | `-C` | economy | Seat class: `economy` \| `premium-economy` \| `business` \| `first` |
| `--limit` | `-l` | 10 | Maximum results to show |
| `--output` | `-o` | text | Output format: `text` \| `json` |
| `--json` | — | false | Shorthand for `--output json` |

### Find cheapest dates

Search across a date range to find the cheapest day to fly:

```bash
gf-cli dates [ORIGIN] [DESTINATION] --from YYYY-MM-DD --to YYYY-MM-DD [flags]
```

```bash
# Find cheapest day in May
gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31

# JSON output
gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31 --json
```

Maximum range: 61 days.

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--from` | *required* | Start date (YYYY-MM-DD) |
| `--to` | *required* | End date (YYYY-MM-DD) |
| `--class` | economy | Seat class |
| `--adults` | 1 | Number of adults |
| `--children` | 0 | Number of children |
| `--json` | false | JSON output |

## Example Output

```
✈  TPE → NRT  |  2026-05-01
─────────────────────────────────────────
1. Scoot
   🕐 15:30 → 20:00  (3hr 30min)  |  Nonstop
   💰 USD 151
─────────────────────────────────────────
2. Tigerair Taiwan
   🕐 14:25 → 18:35  (3hr 10min)  |  Nonstop
   💰 USD 166
─────────────────────────────────────────
```

```
📅 TPE → NRT  |  2026-05-01 to 2026-05-07
─────────────────────────────────────────
  2026-05-01  💰 USD 151  ✈ Scoot          ⏱ 3hr 30min  Nonstop
  2026-05-02  💰 USD 126  ✈ Jetstar        ⏱ 3hr 25min  Nonstop
  2026-05-03  💰 USD 107  ✈ Jetstar        ⏱ 3hr 20min  Nonstop ⭐ cheapest
```

## Project Structure

```
gf-cli/
├── main.go
├── cmd/
│   ├── root.go          # Flight search CLI (flags, validation)
│   └── dates.go         # Date-range search subcommand
├── internal/
│   ├── model/
│   │   └── model.go     # Data structures
│   ├── scraper/
│   │   ├── scraper.go   # Google Flights RPC client
│   │   ├── query.go     # f.req payload builder
│   │   └── parser.go    # JSON response parser
│   └── formatter/
│       └── formatter.go # text / JSON output
└── go.mod
```

## License

[MIT](LICENSE)
