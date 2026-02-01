package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

type MACUpdate struct {
	ages  MACAges
	data  MACData
	today string
	dir   string
	now   string
}

func main() {
	info := &MACUpdate{
		ages:  make(MACAges),
		data:  make(MACData),
		today: time.Now().Format("2006-01-02"),
		now:   time.Now().String(),
		dir:   getBaseDirectory(),
	}

	log.Printf("Starting update for %s in %s", info.today, info.dir)

	// Load current dataset
	log.Printf("Loading current dataset")
	if err := loadCurrent(info); err != nil {
		log.Printf("Failed to load current dataset: %v", err)
		os.Exit(1)
	}

	// Calculate MAC ages based on the full registration data
	log.Printf("Calculating MAC ages from current dataset")
	info.ages = make(MACAges)
	for addr, entries := range info.data {
		if len(entries) == 0 {
			continue
		}
		earliest := entries[0]
		for _, entry := range entries {
			if parseDate(entry.Date) < parseDate(earliest.Date) {
				earliest = entry
			}
		}
		info.ages[addr] = [2]string{earliest.Date, earliest.Source}
	}

	// Load IEEE URLs
	log.Printf("Loading the IEEE URLs")
	oldCount := len(info.data)
	if err := loadIEEEURLs(info); err != nil {
		log.Printf("Failed to load IEEE URLs: %v", err)
		os.Exit(1)
	}
	newCount := len(info.data)

	// Write results
	log.Printf("Writing results for %d entries (%d -> %d)", len(info.data), oldCount, newCount)
	if err := writeResults(info); err != nil {
		log.Printf("Failed to write results: %v", err)
		os.Exit(1)
	}
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
		} else if r == '\r' {
			// Replace \r with \n
			result.WriteRune('\n')
			str = str[size:]
		} else if r == '\x00' {
			// Replace null byte with <00>
			result.WriteString("<00>")
			str = str[size:]
		} else {
			// Valid rune, write as is
			result.WriteRune(r)
			str = str[size:]
		}
	}
	// Normalize multiple newlines to a single newline
	normalized := strings.ReplaceAll(result.String(), "\n\n", "\n")

	return strings.TrimSpace(normalized)
}

func squashCosmeticChanges(str string) string {
	return strings.ToLower(strings.TrimSpace(mashEncoding(str)))
}

func updateRegistration(info *MACUpdate, addr, date, org, address, source string) {
	country := countryFromAddress(address)

	if _, exists := info.data[addr]; !exists {
		info.data[addr] = []RegistrationEntry{
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
	lastEntry := info.data[addr][len(info.data[addr])-1]
	sOOrg := squashCosmeticChanges(lastEntry.Org)
	sOAdd := squashCosmeticChanges(lastEntry.Address)

	if sNOrg != sOOrg || sNAdd != sOAdd {
		info.data[addr] = append(info.data[addr], RegistrationEntry{
			Date:    date,
			Type:    "change",
			Source:  source,
			Address: mashEncoding(address),
			Country: country,
			Org:     mashEncoding(org),
		})
	}
}

func updateAge(info *MACUpdate, addr, date, source string) {
	if _, exists := info.ages[addr]; !exists {
		info.ages[addr] = [2]string{date, source}
		return
	}

	// Parse dates into integers (YYYYMMDD format)
	odate := parseDate(info.ages[addr][0])
	ndate := parseDate(date)

	// Overwrite if new record is older
	if ndate < odate {
		info.ages[addr] = [2]string{date, source}
	}
}

func parseDate(date string) int {
	// Convert YYYY-MM-DD to YYYYMMDD as integer
	dateStr := strings.ReplaceAll(date, "-", "")
	result, _ := strconv.ParseUint(dateStr, 10, 32)
	return int(result)
}

func loadCurrent(info *MACUpdate) error {
	jsonPath := filepath.Join(info.dir, "data", "macs.json")
	fileData, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(fileData, &info.data)
}

func loadCurrentMACAges(info *MACUpdate) error {
	csvPath := filepath.Join(info.dir, "data", "mac-ages.csv")
	csvFile, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	for _, rec := range records {
		if len(rec) < 3 {
			continue
		}
		info.ages[rec[0]] = [2]string{rec[1], rec[2]}
	}
	return nil
}

var matchRegistry = regexp.MustCompile(`^Registry$`)

func loadIEEEURLs(info *MACUpdate) error {
	ieeeURLs := []struct {
		url        string
		minRecords int
	}{
		{"https://standards-oui.ieee.org/oui/oui.csv", 38831},
		{"https://standards-oui.ieee.org/cid/cid.csv", 210},
		{"https://standards-oui.ieee.org/iab/iab.csv", 4575},
		{"https://standards-oui.ieee.org/oui28/mam.csv", 6235},
		{"https://standards-oui.ieee.org/oui36/oui36.csv", 6873},
	}

	for _, urlInfo := range ieeeURLs {
		processed := make(map[string]bool)

		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
		records, err := downloadIEEECSV(info, ctx, urlInfo.url)
		cancel()

		if err != nil {
			return err
		}

		if len(records) < urlInfo.minRecords {
			return fmt.Errorf("URL %s only has %d records (wanted >= %d)", urlInfo.url, len(records), urlInfo.minRecords)
		}

		for _, rec := range records {
			if len(rec) < 4 {
				continue
			}

			// Skip header rows
			if matched := matchRegistry.MatchString(rec[0]); matched {
				continue
			}

			addrBase := rec[1]
			addrMask := int((float64(len(addrBase)) / 2.0) * 8)

			// Pad with zeros to 12 characters
			for len(addrBase) < 12 {
				addrBase += "0"
			}
			addr := fmt.Sprintf("%s/%d", strings.ToLower(addrBase), addrMask)

			// Skip duplicates
			if processed[addr] {
				log.Printf("Skipping duplicate registration for %s from %s [%+v] addr=%s", addr, urlInfo.url, rec, addr)
				continue
			}

			// Replace literal \n with actual newlines
			address := strings.ReplaceAll(rec[3], "\\n", "\n")

			// Remove any \r characters
			address = strings.ReplaceAll(address, "\r", "")

			sourceName := "ieee-" + filepath.Base(urlInfo.url)
			updateRegistration(info, addr, info.today, rec[2], address, sourceName)
			updateAge(info, addr, info.today, sourceName)
			processed[addr] = true
		}
	}

	return nil
}

func downloadIEEECSV(info *MACUpdate, ctx context.Context, url string) ([][]string, error) {
	name := path.Base(url)
	fpath := path.Join(info.dir, "data", "ieee", name)

	var rdata []byte
	var err error

	// Retry logic: up to 26 attempts (initial + 25 retries)
	for retries := 0; retries <= MaxRetries; retries++ {
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
			log.Printf("HTTP error %s from %s, retrying...", err, url)
			time.Sleep(time.Second)
			continue
		}

		rdata, err = io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			if retries == MaxRetries {
				return nil, err
			}
			log.Printf("HTTP read error %s %s, retrying...", err, url)
			time.Sleep(time.Second)
			continue
		}

		if resp.StatusCode != 200 {
			if retries == MaxRetries {
				return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
			}
			log.Printf("HTTP %d from %s, retrying...", resp.StatusCode, url)
			time.Sleep(time.Second)
			continue
		}
		// Success
		break
	}

	st, err := os.Stat(fpath)
	if err == nil && int64(len(rdata)) < (st.Size()-512) {
		return nil, fmt.Errorf("Downloaded file %s is substantially smaller than existing file for %s: cur:%d, existing:%d", url, fpath, len(rdata), st.Size())
	}

	// Write the registry files exactly as provided from IEEE (weird line endings/quotes/etcs)
	if err := os.WriteFile(fpath, rdata, 0644); err != nil {
		return nil, err
	}

	// Parse CSV
	// Remove trailing newlines before parsing
	reader := csv.NewReader(bytes.NewReader(rdata))
	reader.LazyQuotes = true // liberal_parsing equivalent
	reader.TrimLeadingSpace = true
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

func writeResults(info *MACUpdate) error {
	// Write JSON
	jsonData, err := json.Marshal(info.data)
	if err != nil {
		return err
	}

	jsonPath := filepath.Join(info.dir, "data", "macs.json")
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return err
	}

	// Write MAC ages CSV
	csvPath := filepath.Join(info.dir, "data", "mac-ages.csv")
	csvFile, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer csvFile.Close()

	// Sort keys by sortable prefix in descending order
	keys := make([]string, 0, len(info.ages))
	for k := range info.ages {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return sortablePrefix(keys[j]) < sortablePrefix(keys[i])
	})

	cw := csv.NewWriter(csvFile)
	cw.UseCRLF = false

	for _, mac := range keys {
		err := cw.Write([]string{mac, info.ages[mac][0], info.ages[mac][1]})
		if err != nil {
			return err
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return err
	}

	// Write updated timestamp
	updatedPath := filepath.Join(info.dir, "data", "updated.txt")
	if err := os.WriteFile(updatedPath, []byte(info.now), 0644); err != nil {
		return err
	}

	return nil
}

func getBaseDirectory() string {
	based, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get executable path: %v", err)
		os.Exit(1)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(based), "data", "ieee")); err == nil {
		// Prefer the executable directory
		return filepath.Dir(based)
	}
	// Fallback to working directory for test environments
	based, _ = os.Getwd()
	return based
}
