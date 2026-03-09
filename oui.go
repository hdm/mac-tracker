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
	"strings"
)

// OuiHardwareAddr extends net.HardwareAddr
type OuiHardwareAddr net.HardwareAddr

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

// OUISkipPrefixes is a set of prefixes to skip to avoid bad lookups.
var OUISkipPrefixes = map[string]struct{}{
	// Mask: 24
	"000000000000/24": {}, // The zero OUI is registered to Xerox but is typically invalid input
}

// OUITableExtra is an override for unofficial and private registrations.
var OUITableExtra = OuiDB{Blocks: map[string]*OuiBlock{
	// Mask: 24
	"00cafe000000/24": {Oui: []byte{0x00, 0xca, 0xfe, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Xensource, Inc.", Added: "2005-10-29"},
	"d0c907000000/24": {Oui: []byte{0xd0, 0xc9, 0x07, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Govee", Added: "2023-12-14"}, // Private: Smart light bulbs
	// Mask: 16
	"525400000000/16": {Oui: []byte{0x52, 0x54, 0x00, 0x00, 0x00, 0x00}, Mask: 16, Vendor: "QEMU", Added: "2009-03-04"}, // With LAA bit
	"501400000000/16": {Oui: []byte{0x50, 0x14, 0x00, 0x00, 0x00, 0x00}, Mask: 16, Vendor: "QEMU", Added: "2009-03-04"}, // Without LAA bit
}}

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

	// Decode the remaining string as hex
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
	block := OUITable.Lookup(OuiHardwareAddr(addr))
	if block != nil {
		// If the registration is private, check for an override in our Extra table
		if strings.EqualFold(block.Vendor, "Private") {
			privateBlock := OUITableExtra.Lookup(OuiHardwareAddr(addr))
			if privateBlock != nil {
				return privateBlock
			}
		}
		return block
	}
	return OUITableExtra.Lookup(OuiHardwareAddr(addr))
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
		if f, ok := m.Blocks[k]; ok {
			return f
		}
	}
	return nil
}
