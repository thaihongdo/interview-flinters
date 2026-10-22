package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

type campaign struct {
	ID          string
	Impressions int64
	Clicks      int64
	Spend       float64
	Conversions int64
}

func (c *campaign) CTR() float64 {
	if c.Impressions == 0 {
		return 0
	}
	return float64(c.Clicks) / float64(c.Impressions)
}

func (c *campaign) CPA() (float64, bool) {
	if c.Conversions == 0 {
		return 0, false
	}
	return c.Spend / float64(c.Conversions), true
}

func run(inputPath, outputDir string) error {
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("opening input: %w", err)
	}
	defer f.Close()

	campaigns, skipped, err := aggregate(f)
	if err != nil {
		return err
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "skipped %d malformed rows\n", skipped)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	ctrTop := topByCTR(campaigns, 10)
	cpaTop := topByCPA(campaigns, 10)

	if err := writeCTRReport(filepath.Join(outputDir, "top10_ctr.csv"), ctrTop); err != nil {
		return fmt.Errorf("writing CTR report: %w", err)
	}
	if err := writeCPAReport(filepath.Join(outputDir, "top10_cpa.csv"), cpaTop); err != nil {
		return fmt.Errorf("writing CPA report: %w", err)
	}

	return nil
}

func aggregate(r io.Reader) (map[string]*campaign, int, error) {
	cr := csv.NewReader(bufio.NewReader(r))
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true

	header, err := cr.Read()
	if err == io.EOF {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("reading header: %w", err)
	}
	if len(header) != 6 {
		return nil, 0, fmt.Errorf("expected 6 header columns, got %d", len(header))
	}

	campaigns := make(map[string]*campaign)
	skipped := 0

	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			var pe *csv.ParseError
			if errors.As(err, &pe) {
				skipped++
				continue
			}
			return nil, skipped, fmt.Errorf("reading CSV: %w", err)
		}
		if len(record) != 6 {
			skipped++
			continue
		}

		id := record[0]
		if id == "" {
			skipped++
			continue
		}

		if _, err := time.Parse("2006-01-02", record[1]); err != nil {
			skipped++
			continue
		}

		impressions, err := strconv.ParseInt(record[2], 10, 64)
		if err != nil || impressions < 0 {
			skipped++
			continue
		}

		clicks, err := strconv.ParseInt(record[3], 10, 64)
		if err != nil || clicks < 0 {
			skipped++
			continue
		}

		spend, err := strconv.ParseFloat(record[4], 64)
		if err != nil || spend < 0 {
			skipped++
			continue
		}

		conversions, err := strconv.ParseInt(record[5], 10, 64)
		if err != nil || conversions < 0 {
			skipped++
			continue
		}

		c, ok := campaigns[id]
		if !ok {
			c = &campaign{ID: id}
			campaigns[id] = c
		}
		c.Impressions += impressions
		c.Clicks += clicks
		c.Spend += spend
		c.Conversions += conversions
	}

	return campaigns, skipped, nil
}

func topByCTR(campaigns map[string]*campaign, n int) []*campaign {
	list := make([]*campaign, 0, len(campaigns))
	for _, c := range campaigns {
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool {
		ci, cj := list[i].CTR(), list[j].CTR()
		if ci != cj {
			return ci > cj
		}
		return list[i].ID < list[j].ID
	})
	if len(list) > n {
		list = list[:n]
	}
	return list
}

func topByCPA(campaigns map[string]*campaign, n int) []*campaign {
	list := make([]*campaign, 0)
	for _, c := range campaigns {
		if c.Conversions > 0 {
			list = append(list, c)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		cpai, _ := list[i].CPA()
		cpaj, _ := list[j].CPA()
		if cpai != cpaj {
			return cpai < cpaj
		}
		return list[i].ID < list[j].ID
	})
	if len(list) > n {
		list = list[:n]
	}
	return list
}

func writeCTRReport(path string, campaigns []*campaign) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	w.Write([]string{"campaign_id", "impressions", "clicks", "spend", "conversions", "ctr"})
	for _, c := range campaigns {
		w.Write([]string{
			c.ID,
			strconv.FormatInt(c.Impressions, 10),
			strconv.FormatInt(c.Clicks, 10),
			strconv.FormatFloat(c.Spend, 'f', 2, 64),
			strconv.FormatInt(c.Conversions, 10),
			strconv.FormatFloat(c.CTR(), 'f', 6, 64),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func writeCPAReport(path string, campaigns []*campaign) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	w.Write([]string{"campaign_id", "impressions", "clicks", "spend", "conversions", "cpa"})
	for _, c := range campaigns {
		cpa, _ := c.CPA()
		w.Write([]string{
			c.ID,
			strconv.FormatInt(c.Impressions, 10),
			strconv.FormatInt(c.Clicks, 10),
			strconv.FormatFloat(c.Spend, 'f', 2, 64),
			strconv.FormatInt(c.Conversions, 10),
			strconv.FormatFloat(cpa, 'f', 6, 64),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
