package awsping

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// Version describes application version
	Version   = "2.0.0"
	github    = "https://github.com/ekalinin/awsping"
	useragent = fmt.Sprintf("AwsPing/%s (+%s)", Version, github)
)

const (
	// ShowOnlyRegions describes a type of output when only region's name and code printed out
	ShowOnlyRegions = -1
)

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// Duration2ms converts time.Duration to ms (float64)
func Duration2ms(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1000 / 1000
}

// mkRandomString returns random string
func mkRandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// LatencyOutput prints data into console
type LatencyOutput struct {
	Level   int
	Repeats int
	w       io.Writer
}

// NewOutput creates a new LatencyOutput instance
func NewOutput(level, repeats int) *LatencyOutput {
	return &LatencyOutput{
		Level:   level,
		Repeats: repeats,
		w:       os.Stdout,
	}
}

func (lo *LatencyOutput) show(regions *AWSRegions) {
	for _, r := range *regions {
		fmt.Fprintf(lo.w, "%-15s %-s\n", r.Code, r.Name)
	}
}

func (lo *LatencyOutput) show0(regions *AWSRegions) {
	for _, r := range *regions {
		fmt.Fprintf(lo.w, "%-25s %20s\n", r.Name, r.GetLatencyStr())
	}
}

func (lo *LatencyOutput) show1(regions *AWSRegions) {
	outFmt := "%5v %-15s %-30s %20s\n"
	fmt.Fprintf(lo.w, outFmt, "", "Code", "Region", "Latency")
	for i, r := range *regions {
		fmt.Fprintf(lo.w, outFmt, i, r.Code, r.Name, r.GetLatencyStr())
	}
}

func (lo *LatencyOutput) show2(regions *AWSRegions) {
	// format
	outFmt := "%5v %-15s %-25s"
	outFmt += strings.Repeat(" %15s", lo.Repeats) + " %15s\n"
	// header
	outStr := []interface{}{"", "Code", "Region"}
	for i := 0; i < lo.Repeats; i++ {
		outStr = append(outStr, "Try #"+strconv.Itoa(i+1))
	}
	outStr = append(outStr, "Avg Latency")

	// show header
	fmt.Fprintf(lo.w, outFmt, outStr...)

	// each region stats
	for i, r := range *regions {
		outData := []interface{}{strconv.Itoa(i), r.Code, r.Name}
		for n := 0; n < lo.Repeats; n++ {
			outData = append(outData, fmt.Sprintf("%.2f ms",
				Duration2ms(r.Latencies[n])))
		}
		outData = append(outData, fmt.Sprintf("%.2f ms", r.GetLatency()))
		fmt.Fprintf(lo.w, outFmt, outData...)
	}
}

// Show print data
func (lo *LatencyOutput) Show(regions *AWSRegions) {
	switch lo.Level {
	case ShowOnlyRegions:
		lo.show(regions)
	case 0:
		lo.show0(regions)
	case 1:
		lo.show1(regions)
	case 2:
		lo.show2(regions)
	}
}

// GetRegions returns a list of regions
func GetRegions() AWSRegions {
	// URL of the region code to name mapping JSON file
	url := "https://raw.githubusercontent.com/burib/aws-region-table-parser/refs/heads/master/region_code_to_name_map.json"

	// Make HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		fmt.Errorf("failed to fetch regions JSON: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode JSON into a map
	var regionMap map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&regionMap); err != nil {
		fmt.Errorf("failed to decode JSON: %v", err)
	}

	// Convert map to slice of Region structs
	regions := make([]AWSRegion, 0, len(regionMap))
	for code, name := range regionMap {
		regions = append(regions, NewRegion(name, code))
	}

	// Sort by region code for consistent output
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].Code < regions[j].Code
	})

	return regions
}

// CalcLatency returns list of aws regions sorted by Latency
func CalcLatency(regions AWSRegions, repeats int, useHTTP bool, useHTTPS bool, service string) {
	regions.SetService(service)
	switch {
	case useHTTP:
		regions.SetCheckType(CheckTypeHTTP)
	case useHTTPS:
		regions.SetCheckType(CheckTypeHTTPS)
	default:
		regions.SetCheckType(CheckTypeTCP)
	}
	regions.SetDefaultTarget()

	var wg sync.WaitGroup
	for n := 1; n <= repeats; n++ {
		wg.Add(len(regions))
		for i := range regions {
			go regions[i].CheckLatency(&wg)
		}
		wg.Wait()
	}

	sort.Sort(regions)
}
