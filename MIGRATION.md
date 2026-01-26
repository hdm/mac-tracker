# Update Script Migration from Ruby to Go

## Overview
This document describes the migration of the daily update script from Ruby to Go, completed in 2026.

## Changes Made

### New Files
- `update.go` - Main Go implementation of the update script
- `update_test.go` - Unit tests for helper functions
- `integration_test.go` - Integration tests for full update process
- `go.mod` - Go module definition

### Modified Files
- `.github/workflows/update.yml` - Updated to build and use Go binary
- `.gitignore` - Added Go build artifacts
- `README.md` - Updated with Go build instructions

### Renamed Files
- `update` → `update.rb` - Original Ruby implementation kept for reference

## Key Features Preserved

### 1. UTF-8 Encoding Handling
The `mashEncoding` function preserves the Ruby behavior of converting invalid UTF-8 bytes to hex representation:
```go
// Invalid byte 0xFF becomes "<ff>"
mashEncoding("Test\xFFstring") // Returns "Test<ff>string"
```

### 2. Country Code Extraction
Extracts the last 2-letter uppercase country code from addresses:
```go
countryFromAddress("657 Orly Ave. Dorval Quebec CA H9P 1G1") // Returns "CA"
```

### 3. MAC Address Sorting
Uses the sortable prefix logic to sort by mask size (descending), then prefix:
```go
sortablePrefix("000e02000000/24") // Returns "24000e02000000"
sortablePrefix("70b3d5c3c000/36") // Returns "3670b3d5c3c000"
```

### 4. Change Detection
Only creates change records when normalized organization or address differs:
- Normalization: lowercase, trim whitespace, convert invalid UTF-8
- Compares against last entry in history

## New Features

### 1. Automatic Retry Logic
The Go implementation adds automatic retry for IEEE website downloads:
- Up to 6 attempts (initial + 5 retries)
- 5-second delay between retries
- Skips retry for timeout/cancellation errors

### 2. Context-Based Timeout
Uses Go's context package for clean 300-second timeout handling per URL.

### 3. Modern Go Idioms
- Proper error handling with wrapped errors
- Structured types for data
- Efficient string building
- CSV parsing with lazy quotes

## Testing

### Unit Tests
```bash
go test -v
```
Tests cover:
- UTF-8 encoding handling
- Country extraction
- Sortable prefix generation

### Integration Tests
```bash
go test -v -run TestFullUpdate
```
Tests full update process using local IEEE CSV files.

## Building

### Development
```bash
go build -o update update.go
```

### Production (optimized)
```bash
go build -ldflags="-s -w" -o update update.go
```

The `-ldflags="-s -w"` flags reduce binary size by:
- `-s` - Omit symbol table
- `-w` - Omit DWARF debug info

## Compatibility

The Go implementation produces byte-for-byte identical JSON output to the Ruby version:
- Same field names and order in JSON
- Same encoding for invalid UTF-8
- Same sorting algorithm for CSV
- Same change detection logic

## Performance

The Go implementation is expected to:
- Start faster (compiled vs interpreted)
- Use less memory (no Ruby runtime)
- Complete updates in similar time (network-bound)

## Migration Notes

1. The workflow now requires Go 1.24+ instead of Ruby 3.4
2. The binary is built on each workflow run (not committed to repo)
3. The original Ruby script is preserved as `update.rb`
4. All existing data formats and quirks are maintained
