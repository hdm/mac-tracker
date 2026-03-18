package mactracker

/*

Copyright (c) 2017-2026 runZero, Inc.
Copyright (c) 2015-2016 HD Moore
Copyright (c) 2014 dutchcoders

Originally derived from https://github.com/jakewarren/go-ouitools/

	The MIT License (MIT)

	Copyright (c) 2014 dutchcoders

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE.

*/

import (
	"encoding/hex"
	"net"
	"strconv"
)

// OuiHardwareAddr extends net.HardwareAddr
type OuiHardwareAddr net.HardwareAddr

// HasLAA returns true if the locally administered bit is set in the OUI.
func (a OuiHardwareAddr) HasLAA() bool {
	return a[0]&2 == 2
}

// WithoutLAA returns a copy of the OUI with the locally administered bit cleared.
func (a OuiHardwareAddr) WithoutLAA() OuiHardwareAddr {
	if a.HasLAA() {
		a[0] &^= 2
	}
	return a
}

func (a OuiHardwareAddr) String() string {
	return net.HardwareAddr(a).String()
}

// OuiBlock defines an OUI prefix/mask for known hardware addresses
type OuiBlock struct {
	Oui     []byte
	Mask    int
	Vendor  string
	Added   string
	Country string
	Address string
}

// OuiDB provides an interface to a loaded OUI database
type OuiDB struct {
	Blocks map[string]*OuiBlock
}

// ouiTables is the list of sources to use for lookups, in order of priority.
// The first table to return a match will be used, so the order is important.
var ouiTables = map[string]OuiDB{
	"extra":   OUITableExtra,
	"default": OUITable,
	"virtual": OUITableVirtual,
}

// ouiMasks defines the CIDR mask sizes we use for lookups, in order of specificity. The /16 mask size isn't official but is used by QEMU.
var ouiMasks = []int{36, 28, 24, 16}

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

	return addr, nil
}

// CreateMACMaskFromCIDR returns a byte mask given a bit length
func CreateMACMaskFromCIDR(ones, bits int) []byte {
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
	return (m)
}

func MACGetLookupKeys(address OuiHardwareAddr) []string {
	res := make([]string, 0, len(ouiMasks))
	for _, m := range ouiMasks {
		mask := CreateMACMaskFromCIDR(m, len(address)*8)
		key := hex.EncodeToString(address.Mask(mask)) + "/" + strconv.Itoa(m)
		if _, skip := OUISkipPrefixes[key]; skip {
			continue
		}
		res = append(res, key)
	}
	return res
}

// MACLookup returns the first matching OUI match as a block
func MACLookup(s string) *OuiBlock {
	addr, err := ParseMAC(s)
	if err != nil {
		return nil
	}
	return MACLookupBytes(addr)
}

// MACLookupBytes returns the first matching OUI match as a block
func MACLookupBytes(addr []byte) *OuiBlock {
	hwa := OuiHardwareAddr(addr)
	for _, table := range ouiTables {
		block := table.Lookup(hwa)
		if block != nil {
			return block
		}
	}
	return nil
}

// Mask returns the result of masking the address with mask.
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

// Lookup finds the OUI for the specified address
func (m *OuiDB) Lookup(address OuiHardwareAddr) *OuiBlock {
	for _, k := range MACGetLookupKeys(address) {
		if _, skip := OUISkipPrefixes[k]; skip {
			continue
		}
		if f, ok := m.Blocks[k]; ok {
			return f
		}
	}
	return nil
}
