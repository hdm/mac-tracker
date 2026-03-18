package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type registrationEntry struct {
	Date    string `json:"d"`
	Type    string `json:"t"`
	Source  string `json:"s"`
	Address string `json:"a"`
	Country string `json:"c"`
	Org     string `json:"o"`
}

type ouiEntry struct {
	oui     [6]byte
	mask    int
	vendor  string
	added   string
	country string
	address string
}

var ouiMagic = [4]byte{'O', 'U', 'I', 0x01}

func main() {
	dir := findBaseDir()

	jsonPath := filepath.Join(dir, "data", "macs.json")
	log.Printf("Reading %s", jsonPath)
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		log.Fatalf("read macs.json: %v", err)
	}

	var macData map[string][]registrationEntry
	if err := json.Unmarshal(jsonData, &macData); err != nil {
		log.Fatalf("parse macs.json: %v", err)
	}

	log.Printf("Processing %d MAC prefixes", len(macData))

	entries := make([]ouiEntry, 0, len(macData))
	for prefix, regs := range macData {
		parts := strings.SplitN(prefix, "/", 2)
		maskStr := "24"
		if len(parts) == 2 {
			maskStr = parts[1]
		}
		mask, err := strconv.Atoi(maskStr)
		if err != nil {
			log.Fatalf("bad mask in %q: %v", prefix, err)
		}

		addrHex := parts[0]
		for len(addrHex) < 12 {
			addrHex += "0"
		}
		addrBytes, err := hex.DecodeString(addrHex)
		if err != nil {
			log.Fatalf("bad hex in %q: %v", prefix, err)
		}
		var oui [6]byte
		copy(oui[:], addrBytes)

		firstAdded := ""
		lastOrg := ""
		lastCountry := ""
		lastAddress := ""
		for _, entry := range regs {
			if firstAdded == "" && entry.Type == "add" {
				firstAdded = strings.TrimSpace(entry.Date)
			}
			lastOrg = strings.TrimSpace(entry.Org)
			lastAddress = strings.TrimSpace(entry.Address)
			lastCountry = strings.TrimSpace(entry.Country)
		}

		entries = append(entries, ouiEntry{
			oui:     oui,
			mask:    mask,
			vendor:  sanitize(lastOrg),
			added:   firstAdded,
			country: sanitize(strings.ToUpper(lastCountry)),
			address: sanitize(lastAddress),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		ki := hex.EncodeToString(entries[i].oui[:]) + "/" + strconv.Itoa(entries[i].mask)
		kj := hex.EncodeToString(entries[j].oui[:]) + "/" + strconv.Itoa(entries[j].mask)
		return ki < kj
	})

	log.Printf("Encoding %d entries", len(entries))
	data, err := encode(entries)
	if err != nil {
		log.Fatalf("encode: %v", err)
	}

	outPath := filepath.Join(dir, "oui_table.bin.gz")
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		log.Fatalf("write: %v", err)
	}

	log.Printf("Wrote %s (%d bytes, %d entries)", outPath, len(data), len(entries))
}

func encode(entries []ouiEntry) ([]byte, error) {
	var raw bytes.Buffer

	raw.Write(ouiMagic[:])
	if err := binary.Write(&raw, binary.LittleEndian, uint32(len(entries))); err != nil {
		return nil, err
	}

	for _, e := range entries {
		raw.Write(e.oui[:])
		raw.WriteByte(byte(e.mask))
		for _, s := range []string{e.vendor, e.added, e.country, e.address} {
			if err := writeStr(&raw, s); err != nil {
				return nil, err
			}
		}
	}

	var compressed bytes.Buffer
	gz, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := gz.Write(raw.Bytes()); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

func writeStr(w *bytes.Buffer, s string) error {
	if len(s) > 65535 {
		return fmt.Errorf("string too long: %d", len(s))
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(len(s))); err != nil {
		return err
	}
	w.WriteString(s)
	return nil
}

func sanitize(s string) string {
	s = strings.ToValidUTF8(s, "")
	return strings.ReplaceAll(s, "\x00", "")
}

func findBaseDir() string {
	wd, _ := os.Getwd()
	for _, rel := range []string{".", "..", "../.."} {
		p := filepath.Join(wd, rel, "data", "macs.json")
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(filepath.Join(wd, rel))
			return abs
		}
	}
	log.Fatal("cannot find data/macs.json from working directory")
	return ""
}
