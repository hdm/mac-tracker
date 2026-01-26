package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// RegistrationEntry represents a single MAC address registration or change event
type RegistrationEntry struct {
	Date    string `json:"d"`
	Type    string `json:"t"`
	Source  string `json:"s"`
	Address string `json:"a"`
	Country string `json:"c"`
	Org     string `json:"o"`
}

// MACData stores the full registration history
type MACData map[string][]RegistrationEntry

// MACAges stores the earliest registration for each MAC
type MACAges map[string][2]string // [date, source]

const MaxRetries = 30

var (
	macs  MACAges
	data  MACData
	today string
	based string
	now   string
)

func main() {
	// Initialize global variables
	macs = make(MACAges)
	data = make(MACData)
	today = time.Now().Format("2006-01-02")
	now = time.Now().String()

	// Get the current working directory (equivalent to Ruby's behavior)
	var err error
	based, err = os.Getwd()
	if err != nil {
		logMsg(fmt.Sprintf("Failed to get working directory: %v", err))
		os.Exit(1)
	}

	logMsg(fmt.Sprintf("Starting update for %s", today))

	// Load current dataset
	logMsg("Loading current dataset")
	if err := loadCurrent(); err != nil {
		logMsg(fmt.Sprintf("Failed to load current dataset: %v", err))
		os.Exit(1)
	}
	logMsg("")

	// Load IEEE URLs
	logMsg("Loading the IEEE URLs")
	oldCount := len(data)
	if err := loadIEEEURLs(); err != nil {
		logMsg(fmt.Sprintf("Failed to load IEEE URLs: %v", err))
		os.Exit(1)
	}
	newCount := len(data)

	// Write results
	logMsg(fmt.Sprintf("Writing results for %d entries (%d -> %d)", len(data), oldCount, newCount))
	if err := writeResults(); err != nil {
		logMsg(fmt.Sprintf("Failed to write results: %v", err))
		os.Exit(1)
	}
}

func logMsg(msg string) {
	fmt.Printf("%s %s\n", time.Now().String(), msg)
}

var countryRegex = regexp.MustCompile(`\b([A-Z]{2})\b`)

func countryFromAddress(address string) string {
	if len(address) == 0 {
		return ""
	}
	// Split by whitespace and find the last 2-letter uppercase country code
	parts := regexp.MustCompile(`\s+`).Split(address, -1)
	var c string
	for i := len(parts) - 1; i >= 0; i-- {
		if matched := countryRegex.MatchString(parts[i]); matched {
			c = parts[i]
			break
		}
	}
	if c == "" || len(c) != 2 {
		return ""
	}
	return c
}

func mashEncoding(str string) string {
	// Force UTF-8 encoding and replace invalid bytes with hex representation
	result := strings.Builder{}
	for len(str) > 0 {
		r, size := utf8.DecodeRuneInString(str)
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 byte, encode as hex
			result.WriteString("<")
			result.WriteString(hex.EncodeToString([]byte{str[0]}))
			result.WriteString(">")
			str = str[1:]
		} else {
			result.WriteRune(r)
			str = str[size:]
		}
	}
	return strings.TrimSpace(result.String())
}

func squashCosmeticChanges(str string) string {
	return strings.ToLower(strings.TrimSpace(mashEncoding(str)))
}

func updateRegistration(addr, date, org, address, source string) {
	country := countryFromAddress(address)

	if _, exists := data[addr]; !exists {
		data[addr] = []RegistrationEntry{
			{
				Date:    date,
				Type:    "add",
				Source:  source,
				Address: mashEncoding(address),
				Country: country,
				Org:     mashEncoding(org),
			},
		}
		return
	}

	sNOrg := squashCosmeticChanges(org)
	sNAdd := squashCosmeticChanges(address)
	lastEntry := data[addr][len(data[addr])-1]
	sOOrg := squashCosmeticChanges(lastEntry.Org)
	sOAdd := squashCosmeticChanges(lastEntry.Address)

	if sNOrg != sOOrg || sNAdd != sOAdd {
		data[addr] = append(data[addr], RegistrationEntry{
			Date:    date,
			Type:    "change",
			Source:  source,
			Address: mashEncoding(address),
			Country: country,
			Org:     mashEncoding(org),
		})
	}
}

func updateAge(addr, date, source string) {
	if _, exists := macs[addr]; !exists {
		macs[addr] = [2]string{date, source}
		return
	}

	// Parse dates as integers (YYYYMMDD format)
	odate := parseDate(macs[addr][0])
	ndate := parseDate(date)

	// Overwrite if new record is older
	if ndate < odate {
		macs[addr] = [2]string{date, source}
	}
}

func parseDate(date string) int {
	// Convert YYYY-MM-DD to YYYYMMDD as integer
	dateStr := strings.ReplaceAll(date, "-", "")
	var result int
	fmt.Sscanf(dateStr, "%d", &result)
	return result
}

func loadCurrent() error {
	jsonPath := filepath.Join(based, "data", "macs.json")
	fileData, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(fileData, &data)
}

var matchRegistry = regexp.MustCompile(`^Registry$`)

func loadIEEEURLs() error {
	ieeeURLs := []struct {
		url        string
		minRecords int
	}{
		{"https://standards-oui.ieee.org/oui/oui.csv", 37585},
		{"https://standards-oui.ieee.org/cid/cid.csv", 200},
		{"https://standards-oui.ieee.org/iab/iab.csv", 4576},
		{"https://standards-oui.ieee.org/oui28/mam.csv", 5890},
		{"https://standards-oui.ieee.org/oui36/oui36.csv", 6560},
	}

	for _, urlInfo := range ieeeURLs {
		processed := make(map[string]bool)

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		records, err := downloadIEEECSV(ctx, urlInfo.url)
		cancel()

		if err != nil {
			return err
		}

		if len(records) < urlInfo.minRecords {
			return fmt.Errorf("URL %s only has %d records (wanted >= %d)", urlInfo.url, len(records), urlInfo.minRecords)
		}

		for _, info := range records {
			if len(info) < 4 {
				continue
			}

			// Skip header rows
			if matched := matchRegistry.MatchString(info[0]); matched {
				continue
			}

			addrBase := info[1]
			addrMask := int((float64(len(addrBase)) / 2.0) * 8)
			// Pad with zeros to 12 characters
			for len(addrBase) < 12 {
				addrBase += "0"
			}
			addr := fmt.Sprintf("%s/%d", strings.ToLower(addrBase), addrMask)

			// Skip duplicates
			if processed[addr] {
				logMsg(fmt.Sprintf("Skipping duplicate registration for %s from %s", addr, urlInfo.url))
				continue
			}

			// Replace literal \n with actual newlines
			address := strings.ReplaceAll(info[3], "\\n", "\n")

			sourceName := "ieee-" + filepath.Base(urlInfo.url)
			updateRegistration(addr, today, info[2], address, sourceName)
			updateAge(addr, today, sourceName)
			processed[addr] = true
		}
	}

	return nil
}

func downloadIEEECSV(ctx context.Context, url string) ([][]string, error) {
	name := path.Base(url)
	fpath := path.Join(based, "data", "ieee", name)

	var data []byte
	var err error

	// Retry logic: up to 6 attempts (initial + 5 retries)
	for retries := 0; retries <= 5; retries++ {
		if retries > 0 {
			time.Sleep(5 * time.Second)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			// Don't retry on timeout or context cancellation
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if retries == MaxRetries {
				return nil, err
			}
			logMsg(fmt.Sprintf("HTTP err %s from %s, retrying...", err, url))
			time.Sleep(time.Second)
			continue
		}

		data, err = io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			if retries == MaxRetries {
				return nil, err
			}
			logMsg(fmt.Sprintf("HTTP read error %s %s, retrying...", err, url))
			time.Sleep(time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			if retries == MaxRetries {
				return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
			}
			logMsg(fmt.Sprintf("HTTP %d from %s, retrying...", resp.StatusCode, url))
			time.Sleep(time.Second)
			continue
		}
		// Success
		break
	}

	data = bytes.TrimSpace(data)

	// Write to file
	if err := os.WriteFile(fpath, data, 0644); err != nil {
		return nil, err
	}

	// Parse CSV
	// Remove trailing newlines before parsing
	reader := csv.NewReader(bytes.NewReader(data))
	reader.LazyQuotes = true // liberal_parsing equivalent
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Trim whitespace from all fields
	for i := range records {
		for j := range records[i] {
			records[i][j] = strings.TrimSpace(records[i][j])
		}
	}

	return records, nil
}

func sortablePrefix(str string) string {
	parts := strings.Split(str, "/")
	if len(parts) != 2 {
		return str
	}
	prefix := parts[0]
	mask := parts[1]

	// Left-pad mask to 2 digits with zeros
	for len(mask) < 2 {
		mask = "0" + mask
	}

	return mask + prefix
}

func writeResults() error {
	// Write JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	jsonPath := filepath.Join(based, "data", "macs.json")
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return err
	}

	// Write MAC ages CSV
	csvPath := filepath.Join(based, "data", "mac-ages.csv")
	csvFile, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer csvFile.Close()

	// Sort keys by sortable prefix in descending order
	keys := make([]string, 0, len(macs))
	for k := range macs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return sortablePrefix(keys[j]) < sortablePrefix(keys[i])
	})

	for _, mac := range keys {
		fmt.Fprintf(csvFile, "%s,%s,%s\n", mac, macs[mac][0], macs[mac][1])
	}

	// Write updated timestamp
	updatedPath := filepath.Join(based, "data", "updated.txt")
	if err := os.WriteFile(updatedPath, []byte(now), 0644); err != nil {
		return err
	}

	return nil
}
