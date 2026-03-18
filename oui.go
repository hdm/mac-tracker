package mactracker

import (
	"encoding/hex"
	"net"
	"strconv"
)

// ouiTables is the list of sources to use for lookups, in order of priority.
var ouiTables = []*OuiDB{
	// Specific overrides for unofficial and private registrations
	&OUITableExtra,
	// Virtual machine prefixes (some of which conflict with official registrations)
	&OUITableVirtual,
	// The official IEEE OUI registrations
	&OUITable,
}

// ouiMasks defines the CIDR mask sizes we use for lookups, in order of specificity. The /16 mask size isn't official but is used by QEMU.
var ouiMasks = []int{36, 28, 24, 16}

// OuiHardwareAddr is a 6-byte (or 8-byte) hardware address derived from net.HardwareAddr.
type OuiHardwareAddr net.HardwareAddr

// HasLAA reports whether the locally-administered address (LAA) bit is set.
func (a OuiHardwareAddr) HasLAA() bool {
	return a[0]&2 == 2
}

// WithoutLAA returns a copy of the address with the locally-administered bit cleared.
func (a OuiHardwareAddr) WithoutLAA() OuiHardwareAddr {
	if a.HasLAA() {
		a[0] &^= 2
	}
	return a
}

// String returns the colon-separated hex representation of the address.
func (a OuiHardwareAddr) String() string {
	return net.HardwareAddr(a).String()
}

// OuiBlock represents a single OUI registration entry with its prefix, mask, and metadata.
type OuiBlock struct {
	Oui     []byte
	Mask    int
	Vendor  string
	Added   string
	Country string
	Address string
	Virtual string
	Private bool
}

// OuiDB is a collection of OUI blocks indexed by masked-prefix keys.
type OuiDB struct {
	Blocks map[string]*OuiBlock
}

// ParseMAC parses s as an IEEE 802 MAC-48, EUI-48, or EUI-64 using one of the
// following formats:
//
//	01:23:45:67:89:ab
//	01:23:45:67:89:ab:cd:ef
//	01-23-45-67-89-ab
//	01-23-45-67-89-ab-cd-ef
//	0123.4567.89ab
//	0123.4567.89ab.cdef
//	0123 4567 89ab cdEF
func ParseMAC(s string) (OuiHardwareAddr, error) {
	// Remove non-hex characters from the string
	// As primitive as this loop is, it's extremely fast
	cleanStr := make([]byte, 0, len(s))
	for _, c := range s {
		if c == ':' || c == '-' || c == '.' || c == ' ' || c == '_' || c == '\t' || c == '[' || c == ']' {
			continue
		}
		cleanStr = append(cleanStr, byte(c))
	}

	// Decode the remaining string as hex.
	// This is optimized to avoid converting the []byte to a string
	// only to have hex.DecodeString() convert it back to a []byte
	// which is about 2x faster than just letting DecodeString() do it.
	addr := make([]byte, hex.DecodedLen(len(cleanStr)))
	_, err := hex.Decode(addr, cleanStr)
	if err != nil {
		return addr, err
	}

	return OuiHardwareAddr(addr), nil
}

// MaskFromCIDR builds a byte-slice mask of length bits/8 with the leading ones bits set.
func MaskFromCIDR(ones, bits int) []byte {
	if ones < 0 || bits < 0 {
		return nil
	}
	l := bits / 8
	m := make([]byte, l)
	n := uint(ones)
	for i := range l {
		if n >= 8 {
			m[i] = 0xff
			n -= 8
			continue
		}
		m[i] = ^byte(0xff >> n)
		n = 0
	}
	return m
}

func lookupKeys(address OuiHardwareAddr) []string {
	res := make([]string, 0, len(ouiMasks))
	for _, m := range ouiMasks {
		mask := MaskFromCIDR(m, len(address)*8)
		key := hex.EncodeToString(address.Mask(mask)) + "/" + strconv.Itoa(m)
		if _, skip := OUISkipPrefixes[key]; skip {
			continue
		}
		res = append(res, key)
	}
	return res
}

// Lookup resolves a MAC address string to the best-matching OUI block.
// It accepts any common MAC notation (colon, dash, dot, or bare hex).
// Returns nil when the address is unparseable or has no matching registration.
func Lookup(s string) *OuiBlock {
	addr, err := ParseMAC(s)
	if err != nil {
		return nil
	}
	return LookupBytes(addr)
}

// LookupBytes is like Lookup but accepts a raw byte-slice address (6 or 8 bytes).
func LookupBytes(addr []byte) *OuiBlock {
	hwa := OuiHardwareAddr(addr)
	for _, table := range ouiTables {
		block := table.Lookup(hwa)
		if block != nil {
			return block
		}
	}
	return nil
}

// LookupOUI searches only the primary IEEE OUI registration table for a MAC address string.
// Returns nil when the address is unparseable or has no matching IEEE registration.
func LookupOUI(s string) *OuiBlock {
	addr, err := ParseMAC(s)
	if err != nil {
		return nil
	}
	return OUITable.Lookup(OuiHardwareAddr(addr))
}

// LookupOverride searches only the curated override table for unofficial and
// private MAC registrations. Returns nil when the address is unparseable or
// has no matching override entry.
func LookupOverride(s string) *OuiBlock {
	addr, err := ParseMAC(s)
	if err != nil {
		return nil
	}
	return OUITableExtra.Lookup(OuiHardwareAddr(addr))
}

// Mask applies a byte-level AND between the address and mask, returning the masked result.
func (address OuiHardwareAddr) Mask(mask []byte) []byte {
	addrLen := len(address)
	maskLen := len(mask)
	masked := make([]byte, addrLen)
	for i := range addrLen {
		maskByte := byte(0)
		if i < maskLen {
			maskByte = mask[i]
		}
		masked[i] = address[i] & maskByte
	}
	return masked
}

// Lookup searches the database for the most-specific OUI block matching address.
func (m *OuiDB) Lookup(address OuiHardwareAddr) *OuiBlock {
	for _, k := range lookupKeys(address) {
		if _, skip := OUISkipPrefixes[k]; skip {
			continue
		}
		if f, ok := m.Blocks[k]; ok {
			return f
		}
	}
	return nil
}
