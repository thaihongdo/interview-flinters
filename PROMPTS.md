# Implementation Task: Ad Performance Aggregator (CLI, Go)

## Role & Objective

You are building a command-line tool in **Go** that aggregates advertising
performance data from a large CSV file (~1 GB) and produces two ranked report
files. Correctness comes first; after that, optimize for **low, bounded memory
usage** and **fast processing** of the full 1 GB input. The input must be
processed as a stream — never load the whole file into memory.

## Input Specification

- A single CSV file (default name `ad_data.csv`), ~1 GB uncompressed.
- The file has a header row. Columns, in order:

  | Column         | Type    | Notes                                  |
  |----------------|---------|----------------------------------------|
  | `campaign_id`  | string  | Groups the data; non-empty.            |
  | `date`         | string  | Format `YYYY-MM-DD`.                   |
  | `impressions`  | integer | Count of ad impressions (>= 0).        |
  | `clicks`       | integer | Count of clicks (>= 0).                |
  | `spend`        | float   | Money spent (>= 0).                    |
  | `conversions`  | integer | Count of conversions (>= 0).           |

- Do not assume the file fits in RAM. Read it row by row.

## Functional Requirements

1. **Aggregate per `campaign_id`** across the whole file. For each campaign sum:
   `impressions`, `clicks`, `spend`, `conversions`.

2. **Compute two derived metrics per campaign:**
   - `CTR = clicks / impressions` (click-through rate).
   - `CPA = spend / conversions` (cost per acquisition).
   - If a campaign's total `conversions == 0`, its CPA is undefined — represent
     it as `null`/empty and **exclude it from the CPA ranking** (see below).
   - If a campaign's total `impressions == 0`, treat CTR as `0` (or `null`);
     pick one, state the choice, and be consistent.

3. **Produce two output files** in the output directory:
   - `top10_ctr.csv` — the 10 campaigns with the **highest CTR**.
   - `top10_cpa.csv` — the 10 campaigns with the **lowest CPA**, considering only
     campaigns with `conversions > 0`.
   - Each output row should include: `campaign_id`, the summed metrics
     (`impressions`, `clicks`, `spend`, `conversions`), and the relevant
     computed metric (`ctr` or `cpa`). Include a header row.

## Expected Architecture & Data Flow

Follow this pipeline:

```
[ad_data.csv] → csv.Reader (bufio-buffered)
                   │
                   ▼
             parse + validate row
                   │
                   ▼
         aggregate into map[campaignID]*agg
                   │     (sums: imp, clk, spend, conv)
                   │
     ─────── end of stream ────────
                   │
                   ▼
         compute CTR, CPA per campaign
                   │
             ┌─────┴─────┐
             ▼           ▼
       top10 CTR     top10 CPA (conv > 0)
             │           │
             ▼           ▼
      top10_ctr.csv  top10_cpa.csv
```

- Read with a `bufio`-buffered `encoding/csv` reader, one record at a time
  (`Reader.Read()`), so peak memory is bounded by the number of distinct
  campaigns, not by file size.
- Accumulate sums into a `map[string]*agg` keyed by `campaign_id`.
- Only after the stream is fully consumed, compute CTR/CPA and select the
  top-10 lists.

## CLI Interface

- Invoked as a single binary with flags:

  ```
  aggregator --input ad_data.csv --output results/
  ```

- `--input`: path to the input CSV (required).
- `--output`: directory for the two output CSV files (create it if missing).
- Exit with a non-zero status code and a clear stderr message on fatal errors.

## Non-Functional Requirements

- **Memory:** bounded and independent of file size. Do not read the whole file
  into memory or build a slice of every row. Memory should scale with the number
  of distinct campaigns only.
- **Performance:** process the full 1 GB file in a single pass. Measure and report
  wall-clock time and peak memory.
- **Determinism:** given the same input, output must be identical across runs,
  including tie ordering (define a stable tie-breaker, e.g. by `campaign_id`).

## Edge Cases & Error Handling

Handle these explicitly and describe the behavior in the README:

- Input file missing or unreadable → clear error, non-zero exit.
- Empty file or header-only file → produce empty (header-only) output files, do
  not crash.
- Malformed rows: wrong column count, non-numeric where a number is expected,
  negative values, or an unparseable date — skip the bad row, keep a count, and
  log a summary (e.g. "skipped N malformed rows") to stderr — do not abort the
  whole run.
- Division by zero: `impressions == 0` for CTR, `conversions == 0` for CPA — per
  the rules above.
- Fewer than 10 campaigns in the data → output however many exist (do not pad).
- Ties on CTR or CPA → apply the stable tie-breaker.

## Deliverables

1. **Source code**, cleanly organized (e.g. parsing, aggregation, ranking, and
   output as separable units with clear responsibilities).
2. **The two output files** generated from the provided dataset:
   `top10_ctr.csv` and `top10_cpa.csv`.
3. **Tests** that verify correctness of aggregation, metric computation, ranking,
   tie-breaking, and malformed-row handling — using small, in-repo fixtures (do
   not depend on the 1 GB file).
4. **A `Dockerfile`** that builds the binary and lets a reviewer run the tool
   without a local Go toolchain. Document the `docker build` / `docker run`
   commands (including how to mount the input file and read the outputs).
5. **A `README.md`** containing:
   - Install / build instructions.
   - How to run the CLI (and via Docker).
   - Libraries used and why.
   - Processing time measured on the 1 GB file.
   - Peak memory usage on the 1 GB file (if measurable).
   - Brief explanation of any non-obvious design decisions (tie-breaking,
     CTR-with-zero-impressions rule, malformed-row policy).
6. benchmark logs.

## Quality Bar

- The results must be correct.
- Code must be clean: no dead code, no commented-out blocks, no unused symbols.
- Robust error handling for missing files, malformed rows, and boundary cases.
- Memory-efficient by construction (streaming, bounded state).
- A test suite that actually verifies correctness, not just that code runs.
- Any non-obvious decision is briefly justified in the README.

## Open Decisions (your judgment)

You decide and justify the following — there is no single required answer:

- Internal package/module structure and function boundaries.
- Whether to keep a running top-10 heap vs. sorting at the end (either is fine;
  the campaign count is small relative to row count).
- Exact CSV library choice (standard `encoding/csv` is recommended and sufficient).
- The CTR-when-zero-impressions representation (`0` vs `null`) — pick one, apply
  it consistently, and note it.
- Output column ordering and number formatting (use sensible, consistent
  precision for `ctr`/`cpa`/`spend`).
