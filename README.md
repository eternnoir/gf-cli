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

## Why LLM Agents Love This Tool

`gf-cli` is designed to be **agent-friendly out of the box**:

- **`--json` flag on every command** — structured output, no screen-scraping needed
- **Round-trip search built-in** — `--return` for single date, `--stay N` for date-range sweep
- **Predictable JSON schema** — stable field names (`origin`, `destination`, `flights[]`, `dates[]`)
- **Non-interactive** — fully scriptable, no prompts or confirmations

### Quick start for Agents

```bash
# Find cheapest round-trip in May, staying 7 days — JSON output
gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31 --stay 7 --json

# Single round-trip query with full details
gf-cli TPE NRT --date 2026-05-10 --return 2026-05-17 --json
```

Output is valid JSON — pipe directly into `jq` or parse in your agent tool call.
See [SKILL.md](SKILL.md) for the full schema reference and agent integration guide.

## Usage

### Search flights on a specific date

```bash
gf-cli [ORIGIN] [DESTINATION] --date YYYY-MM-DD [flags]
```

> **Round trip is the most common use case.** Add `--return` to get a combined outbound + return price.

```bash
# Round trip (most common) — outbound May 1, return May 8
gf-cli TPE NRT --date 2026-05-01 --return 2026-05-08

# Round trip, business class, 2 adults
gf-cli JFK LHR --date 2026-06-15 --return 2026-06-22 --class business --adults 2

# One-way
gf-cli TPE NRT --date 2026-05-01

# Multiple passengers (2 adults + 2 children), round trip
gf-cli TPE NRT --date 2026-05-01 --return 2026-05-08 --adults 2 --children 2

# JSON output (Agent/script friendly)
gf-cli TPE NRT --date 2026-05-01 --return 2026-05-08 --json

# Limit results
gf-cli SFO ORD --date 2026-07-04 --return 2026-07-11 --limit 5
```

**Flags:**

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--date` | `-d` | *required* | Departure date (YYYY-MM-DD) |
| `--return` | `-r` | — | **Return date for round trips (YYYY-MM-DD).** When set, the search returns combined outbound + return prices. Omit for one-way. |
| `--adults` | `-a` | 1 | Number of adults |
| `--children` | `-c` | 0 | Number of children |
| `--class` | `-C` | economy | Seat class: `economy` \| `premium-economy` \| `business` \| `first` |
| `--limit` | `-l` | 10 | Maximum results to show |
| `--output` | `-o` | text | Output format: `text` \| `json` |
| `--json` | — | false | Shorthand for `--output json` |

### Find cheapest dates across a range

Search a range of departure dates to find the cheapest day to fly. Maximum range: 61 days.

```bash
gf-cli dates [ORIGIN] [DESTINATION] --from YYYY-MM-DD --to YYYY-MM-DD [flags]
```

> **Round trip with `--stay N`** is the most common use case — finds which departure date gives the cheapest combined outbound + return fare when you stay N days.

```bash
# Round trip: find cheapest May departure, staying 7 days (most common)
gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31 --stay 7

# Round trip: business class, 2 adults, stay 5 days
gf-cli dates JFK LHR --from 2026-06-01 --to 2026-06-30 --stay 5 --class business --adults 2

# Round trip: JSON output for scripting/Agent use
gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31 --stay 7 --json

# One-way: find cheapest single day (no --stay)
gf-cli dates TPE NRT --from 2026-05-01 --to 2026-05-31
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--from` | *required* | Start of departure date range (YYYY-MM-DD) |
| `--to` | *required* | End of departure date range (YYYY-MM-DD) |
| `--stay` | 0 (one-way) | **Number of days to stay at destination for round-trip search.** Each departure date is paired with `departure + N days` as the return date. Set to 0 or omit for one-way. |
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
📅 TPE → NRT  |  2026-05-01 to 2026-05-07  (round trip, 7 days)
─────────────────────────────────────────
  2026-05-01 → 2026-05-08  💰 USD 312  ✈ Scoot    ⏱ 3hr 30min  Nonstop
  2026-05-02 → 2026-05-09  💰 USD 284  ✈ Jetstar  ⏱ 3hr 25min  Nonstop
  2026-05-03 → 2026-05-10  💰 USD 261  ✈ Jetstar  ⏱ 3hr 20min  Nonstop ⭐ cheapest
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
