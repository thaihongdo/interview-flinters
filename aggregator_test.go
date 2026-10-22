package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAggregate(t *testing.T) {
	input := `campaign_id,date,impressions,clicks,spend,conversions
camp_a,2024-01-01,1000,100,50.00,10
camp_a,2024-01-02,2000,200,100.00,20
camp_b,2024-01-01,500,50,25.00,5
`
	camps, skipped, err := aggregate(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0", skipped)
	}
	if len(camps) != 2 {
		t.Fatalf("got %d campaigns, want 2", len(camps))
	}

	a := camps["camp_a"]
	if a.Impressions != 3000 || a.Clicks != 300 || a.Conversions != 30 {
		t.Errorf("camp_a sums wrong: imp=%d clk=%d conv=%d", a.Impressions, a.Clicks, a.Conversions)
	}
	if a.Spend != 150.0 {
		t.Errorf("camp_a spend = %f, want 150.0", a.Spend)
	}

	b := camps["camp_b"]
	if b.Impressions != 500 || b.Clicks != 50 || b.Spend != 25.0 || b.Conversions != 5 {
		t.Errorf("camp_b wrong: %+v", b)
	}
}

func TestCTR(t *testing.T) {
	tests := []struct {
		imp, clk int64
		want     float64
	}{
		{1000, 100, 0.1},
		{0, 0, 0},
		{500, 0, 0},
	}
	for _, tt := range tests {
		c := &campaign{Impressions: tt.imp, Clicks: tt.clk}
		if got := c.CTR(); got != tt.want {
			t.Errorf("CTR(%d,%d) = %f, want %f", tt.imp, tt.clk, got, tt.want)
		}
	}
}

func TestCPA(t *testing.T) {
	c := &campaign{Spend: 100, Conversions: 50}
	cpa, ok := c.CPA()
	if !ok || cpa != 2.0 {
		t.Errorf("CPA = %f, ok=%v; want 2.0, true", cpa, ok)
	}

	c2 := &campaign{Spend: 100, Conversions: 0}
	_, ok = c2.CPA()
	if ok {
		t.Error("CPA should be undefined when conversions=0")
	}
}

func TestTopByCTR(t *testing.T) {
	camps := map[string]*campaign{
		"c":  {ID: "c", Impressions: 1000, Clicks: 100},  // 0.1
		"a":  {ID: "a", Impressions: 1000, Clicks: 100},  // 0.1 (tie)
		"b":  {ID: "b", Impressions: 1000, Clicks: 500},  // 0.5
		"d":  {ID: "d", Impressions: 0, Clicks: 0},        // 0
		"e":  {ID: "e", Impressions: 1000, Clicks: 200},  // 0.2
	}
	top := topByCTR(camps, 3)
	if len(top) != 3 {
		t.Fatalf("got %d, want 3", len(top))
	}
	wantIDs := []string{"b", "e", "a"}
	for i, c := range top {
		if c.ID != wantIDs[i] {
			t.Errorf("top[%d] = %s, want %s", i, c.ID, wantIDs[i])
		}
	}
}

func TestTopByCPA(t *testing.T) {
	camps := map[string]*campaign{
		"a": {ID: "a", Spend: 100, Conversions: 10}, // CPA=10
		"b": {ID: "b", Spend: 50, Conversions: 10},  // CPA=5
		"c": {ID: "c", Spend: 50, Conversions: 10},  // CPA=5 (tie)
		"d": {ID: "d", Spend: 200, Conversions: 0},  // excluded
	}
	top := topByCPA(camps, 10)
	if len(top) != 3 {
		t.Fatalf("got %d, want 3 (d excluded)", len(top))
	}
	wantIDs := []string{"b", "c", "a"}
	for i, c := range top {
		if c.ID != wantIDs[i] {
			t.Errorf("top[%d] = %s, want %s", i, c.ID, wantIDs[i])
		}
	}
}

func TestMalformedRows(t *testing.T) {
	input := `campaign_id,date,impressions,clicks,spend,conversions
good,2024-01-01,100,10,5.00,1
bad_date,not-a-date,100,10,5.00,1
bad_imp,2024-01-01,xxx,10,5.00,1
short,2024-01-01,100,10,5.00
neg_imp,2024-01-01,-1,10,5.00,1
,2024-01-01,100,10,5.00,1
neg_spend,2024-01-01,100,10,-5.00,1
`
	camps, skipped, err := aggregate(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 6 {
		t.Errorf("skipped = %d, want 6", skipped)
	}
	if len(camps) != 1 {
		t.Errorf("got %d campaigns, want 1", len(camps))
	}
	if _, ok := camps["good"]; !ok {
		t.Error("expected 'good' campaign to survive")
	}
}

func TestHeaderOnly(t *testing.T) {
	input := "campaign_id,date,impressions,clicks,spend,conversions\n"
	camps, skipped, err := aggregate(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 {
		t.Errorf("skipped = %d, want 0", skipped)
	}
	if len(camps) != 0 {
		t.Errorf("got %d campaigns, want 0", len(camps))
	}
}

func TestEmptyFile(t *testing.T) {
	camps, skipped, err := aggregate(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if skipped != 0 || camps != nil {
		t.Errorf("expected nil map and 0 skipped for empty file")
	}
}

func TestRunIntegration(t *testing.T) {
	outDir := t.TempDir()
	if err := run("testdata/basic.csv", outDir); err != nil {
		t.Fatal(err)
	}

	ctrData, err := os.ReadFile(filepath.Join(outDir, "top10_ctr.csv"))
	if err != nil {
		t.Fatal(err)
	}
	ctrLines := strings.Split(strings.TrimSpace(string(ctrData)), "\n")
	if len(ctrLines) != 5 {
		t.Errorf("top10_ctr.csv has %d lines, want 5 (header + 4 campaigns)", len(ctrLines))
	}
	if ctrLines[0] != "campaign_id,impressions,clicks,spend,conversions,ctr" {
		t.Errorf("unexpected header: %s", ctrLines[0])
	}

	cpaData, err := os.ReadFile(filepath.Join(outDir, "top10_cpa.csv"))
	if err != nil {
		t.Fatal(err)
	}
	cpaLines := strings.Split(strings.TrimSpace(string(cpaData)), "\n")
	if len(cpaLines) != 3 {
		t.Errorf("top10_cpa.csv has %d lines, want 3 (header + 2 campaigns with conv>0)", len(cpaLines))
	}
}

func TestRunHeaderOnly(t *testing.T) {
	outDir := t.TempDir()
	if err := run("testdata/header_only.csv", outDir); err != nil {
		t.Fatal(err)
	}

	ctrData, err := os.ReadFile(filepath.Join(outDir, "top10_ctr.csv"))
	if err != nil {
		t.Fatal(err)
	}
	ctrLines := strings.Split(strings.TrimSpace(string(ctrData)), "\n")
	if len(ctrLines) != 1 {
		t.Errorf("top10_ctr.csv has %d lines, want 1 (header only)", len(ctrLines))
	}
}

func TestRunMalformed(t *testing.T) {
	outDir := t.TempDir()
	if err := run("testdata/malformed.csv", outDir); err != nil {
		t.Fatal(err)
	}

	ctrData, err := os.ReadFile(filepath.Join(outDir, "top10_ctr.csv"))
	if err != nil {
		t.Fatal(err)
	}
	ctrLines := strings.Split(strings.TrimSpace(string(ctrData)), "\n")
	if len(ctrLines) != 3 {
		t.Errorf("top10_ctr.csv has %d lines, want 3 (header + camp_a + camp_g)", len(ctrLines))
	}
}

func TestRunMissingFile(t *testing.T) {
	err := run("testdata/nonexistent.csv", t.TempDir())
	if err == nil {
		t.Error("expected error for missing file")
	}
}
