package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFullUpdate(t *testing.T) {
	// Skip this test in normal runs as it takes time
	if testing.Short() {
		t.Skip("Skipping full update test in short mode")
	}

	// Initialize global variables
	macs = make(MACAges)
	data = make(MACData)
	today = "2026-01-26" // Fixed date for testing
	now = "2026-01-26 00:00:00 +0000 UTC"

	var err error
	based, err = os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Load current dataset
	if err := loadCurrent(); err != nil {
		t.Fatalf("Failed to load current dataset: %v", err)
	}

	oldCount := len(data)

	// Load from local IEEE files instead of downloading
	if err := loadIEEEFromLocal(); err != nil {
		t.Fatalf("Failed to load IEEE data: %v", err)
	}

	newCount := len(data)

	t.Logf("Processed %d entries (%d -> %d)", len(data), oldCount, newCount)

	// Verify we have data
	if len(data) == 0 {
		t.Fatal("No data was loaded")
	}

	if len(macs) == 0 {
		t.Fatal("No MAC ages were tracked")
	}
}

func loadIEEEFromLocal() error {
	ieeeFiles := []struct {
		filename   string
		minRecords int
	}{
		{"oui.csv", 37585},
		{"cid.csv", 200},
		{"iab.csv", 4576},
		{"mam.csv", 5890},
		{"oui36.csv", 6560},
	}

	for _, fileInfo := range ieeeFiles {
		processed := make(map[string]bool)
		path := filepath.Join(based, "data", "ieee", fileInfo.filename)

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Parse CSV
		dataStr := strings.TrimRight(string(data), "\n\r")
		reader := csv.NewReader(strings.NewReader(dataStr))
		reader.LazyQuotes = true

		records, err := reader.ReadAll()
		if err != nil {
			return fmt.Errorf("failed to parse CSV %s: %w", path, err)
		}

		if len(records) < fileInfo.minRecords {
			return fmt.Errorf("file %s only has %d records (wanted >= %d)", fileInfo.filename, len(records), fileInfo.minRecords)
		}

		for _, info := range records {
			if len(info) < 4 {
				continue
			}

			// Skip header rows
			if strings.HasPrefix(info[0], "Registry") {
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
				continue
			}

			// Replace literal \n with actual newlines
			address := strings.ReplaceAll(info[3], "\\n", "\n")

			sourceName := "ieee-" + fileInfo.filename
			updateRegistration(addr, today, info[2], address, sourceName)
			updateAge(addr, today, sourceName)
			processed[addr] = true
		}
	}

	return nil
}

func TestJSONOutput(t *testing.T) {
	// Test that we can marshal the data structure
	testData := MACData{
		"000e02000000/24": []RegistrationEntry{
			{
				Date:    "2003-09-08",
				Type:    "add",
				Address: "657 Orly Ave.\nDorval Quebec H9P 1G1\n\n",
				Country: "CA",
				Org:     "Advantech AMT Inc.",
				Source:  "wireshark.org",
			},
		},
	}

	jsonData, err := json.Marshal(testData)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Verify we can unmarshal it back
	var decoded MACData
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(decoded) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(decoded))
	}
}
