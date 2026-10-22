# Ad Performance Aggregator

CLI tool that aggregates advertising performance data from a large CSV file and
produces ranked reports for top campaigns by CTR and CPA.

## Build & Run

```bash
go build -o aggregator .
./aggregator --input ad_data.csv --output results/
```

### Docker

```bash
docker build -t aggregator .

# Mount the input file and an output directory:
docker run --rm \
  -v "$PWD/ad_data.csv:/data/ad_data.csv:ro" \
  -v "$PWD/results:/results" \
  aggregator --input /data/ad_data.csv --output /results
```

## Output

- `top10_ctr.csv` — 10 campaigns with the highest click-through rate.
- `top10_cpa.csv` — 10 campaigns with the lowest cost per acquisition
  (only campaigns with conversions > 0).

## Libraries

Standard library only (`encoding/csv`, `bufio`, `flag`, `sort`, `strconv`,
`time`, `runtime`). No external dependencies.

## Design Decisions

- **CTR when impressions = 0:** treated as 0. Consistent, avoids null handling
  in output.
- **Tie-breaking:** campaigns with equal CTR or CPA are ordered by `campaign_id`
  ascending (lexicographic). Deterministic across runs.
- **Malformed rows:** rows with wrong column count, non-numeric values, negative
  values, empty campaign ID, or unparseable date are skipped. A count of skipped
  rows is logged to stderr. The run continues.
- **Memory:** bounded by the number of distinct campaigns, not file size. The CSV
  is read one record at a time via a buffered reader.
- **Sorting vs heap:** campaigns are sorted after the full stream is consumed.
  Campaign count is small relative to row count, so a full sort is simpler and
  fast enough.
- **Float precision:** `spend` is formatted to 2 decimal places. `ctr` and `cpa`
  are formatted to 6 decimal places.

## Tests

```bash
go test -v ./...
```

Tests cover: aggregation sums, CTR/CPA computation, ranking with tie-breaking,
malformed row handling, empty/header-only files, missing input file.

## Performance

Tested on the 1 GB input file:

```
$ ./aggregator --input ad_data.csv --output results/
completed in 13.664737267s, heap alloc: 2.9 MB, total alloc: 3578.3 MB
```

- **Wall-clock time:** ~14s
- **Heap alloc:** 2.9 MB
- **Total alloc:** 3578.3 MB
