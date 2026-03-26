# gf-cli

A Go CLI tool to search Google Flights directly from your terminal — no API key required.

> Go rewrite of [flight-search](https://github.com/Olafs-World/flight-search) by the SRT Team.

## Usage

```bash
gf-cli [ORIGIN] [DESTINATION] --date YYYY-MM-DD [options]
```

### Examples

```bash
# One-way, economy
gf-cli TPE LAX --date 2026-05-01

# Round trip, business class
gf-cli TPE NRT --date 2026-05-01 --return 2026-05-10 --class business

# JSON output
gf-cli JFK LHR --date 2026-06-15 --output json

# 2 adults, limit 5 results
gf-cli SFO ORD --date 2026-07-04 --adults 2 --limit 5
```

### Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--date` | `-d` | *required* | Departure date (YYYY-MM-DD) |
| `--return` | `-r` | — | Return date for round trips (YYYY-MM-DD) |
| `--adults` | `-a` | 1 | Number of adults |
| `--children` | `-c` | 0 | Number of children |
| `--class` | `-C` | economy | Seat class: economy, premium-economy, business, first |
| `--limit` | `-l` | 10 | Maximum results to show |
| `--output` | `-o` | text | Output format: text, json |

## Development

### Project Structure

```
gf-cli/
├── main.go                   # Entry point
├── cmd/
│   └── root.go               # Cobra CLI definition, flag parsing, validation
├── internal/
│   ├── model/
│   │   └── model.go          # Data structures (Flight, SearchResult, etc.)
│   ├── scraper/
│   │   └── scraper.go        # Google Flights HTTP scraper (Kai)
│   ├── proto/                # Protobuf query encoding (Kai)
│   └── formatter/
│       └── formatter.go      # text / JSON output rendering (Elena)
└── go.mod
```

### Build

```bash
go build -o gf-cli .
```

### Test

```bash
go test ./...
```
