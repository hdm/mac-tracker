package mactracker

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
)

// Binary format for OUI database:
//
//	Header:
//	  4 bytes  magic   "OUI\x01"
//	  4 bytes  count   uint32 LE, number of entries
//	Per entry:
//	  6 bytes  oui     raw prefix bytes
//	  1 byte   mask    CIDR mask width
//	  2 bytes  vLen    uint16 LE vendor string length
//	  vLen     vendor  UTF-8
//	  2 bytes  aLen    uint16 LE added-date string length
//	  aLen     added   UTF-8
//	  2 bytes  cLen    uint16 LE country string length
//	  cLen     country UTF-8
//	  2 bytes  dLen    uint16 LE address string length
//	  dLen     address UTF-8

var ouiMagic = [4]byte{'O', 'U', 'I', 0x01}

// EncodeOUIDB serializes an OuiDB into a gzip-compressed binary blob.
func EncodeOUIDB(db *OuiDB) ([]byte, error) {
	var raw bytes.Buffer

	// Magic
	raw.Write(ouiMagic[:])

	// Entry count
	count := uint32(len(db.Blocks))
	if err := binary.Write(&raw, binary.LittleEndian, count); err != nil {
		return nil, err
	}

	for _, block := range db.Blocks {
		if err := encodeBlock(&raw, block); err != nil {
			return nil, err
		}
	}

	// Gzip compress
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

func encodeBlock(w *bytes.Buffer, b *OuiBlock) error {
	// OUI prefix: always 6 bytes (pad if shorter)
	var oui [6]byte
	copy(oui[:], b.Oui)
	w.Write(oui[:])

	// Mask
	w.WriteByte(byte(b.Mask))

	// Strings: vendor, added, country, address
	for _, s := range []string{b.Vendor, b.Added, b.Country, b.Address} {
		if err := writeString16(w, s); err != nil {
			return err
		}
	}
	return nil
}

func writeString16(w *bytes.Buffer, s string) error {
	if len(s) > 65535 {
		return fmt.Errorf("string too long: %d bytes", len(s))
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(len(s))); err != nil {
		return err
	}
	w.WriteString(s)
	return nil
}

// DecodeOUIDB deserializes a gzip-compressed binary blob into an OuiDB Blocks map.
func DecodeOUIDB(data []byte) (map[string]*OuiBlock, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip open: %w", err)
	}
	defer gz.Close()

	raw, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("gzip read: %w", err)
	}

	r := bytes.NewReader(raw)

	// Magic
	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if magic != ouiMagic {
		return nil, fmt.Errorf("bad magic: %x", magic)
	}

	// Count
	var count uint32
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("read count: %w", err)
	}

	blocks := make(map[string]*OuiBlock, count)
	for range count {
		block, key, err := decodeBlock(r)
		if err != nil {
			return nil, err
		}
		blocks[key] = block
	}
	return blocks, nil
}

func decodeBlock(r *bytes.Reader) (*OuiBlock, string, error) {
	// OUI prefix
	var oui [6]byte
	if _, err := io.ReadFull(r, oui[:]); err != nil {
		return nil, "", fmt.Errorf("read oui: %w", err)
	}

	// Mask
	maskByte, err := r.ReadByte()
	if err != nil {
		return nil, "", fmt.Errorf("read mask: %w", err)
	}
	mask := int(maskByte)

	// Strings
	vendor, err := readString16(r)
	if err != nil {
		return nil, "", fmt.Errorf("read vendor: %w", err)
	}
	added, err := readString16(r)
	if err != nil {
		return nil, "", fmt.Errorf("read added: %w", err)
	}
	country, err := readString16(r)
	if err != nil {
		return nil, "", fmt.Errorf("read country: %w", err)
	}
	address, err := readString16(r)
	if err != nil {
		return nil, "", fmt.Errorf("read address: %w", err)
	}

	key := hex.EncodeToString(oui[:]) + "/" + strconv.Itoa(mask)

	block := &OuiBlock{
		Oui:     oui[:],
		Mask:    mask,
		Vendor:  vendor,
		Added:   added,
		Country: country,
		Address: address,
	}
	return block, key, nil
}

func readString16(r *bytes.Reader) (string, error) {
	var length uint16
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return "", err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}
