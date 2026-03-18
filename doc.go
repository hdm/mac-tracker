// Package mactracker provides IEEE OUI lookups for MAC addresses.
//
// The package ships with an embedded database of IEEE OUI/CID/IAB/OUI-28/OUI-36
// registrations, supplemented by tables for virtual-machine and other
// private prefixes.
//
// # Looking up a MAC address string
//
// The simplest entry point is [Lookup], which accepts any common MAC notation:
//
//	block := mactracker.Lookup("00:1b:c5:00:02:03")
//	if block != nil {
//		fmt.Println(block.Vendor) // "Converging Systems Inc."
//	}
//
// # Looking up raw bytes
//
// When you already have the address as a byte slice (e.g. from a packet
// capture), use [LookupBytes] to avoid an unnecessary string round-trip:
//
//	addr := []byte{0x00, 0x50, 0x56, 0x12, 0x34, 0x56}
//	block := mactracker.LookupBytes(addr)
//	if block != nil {
//		fmt.Println(block.Vendor) // "VMware"
//	}
//
// # Parsing a MAC address
//
// [ParseMAC] converts a string in any common format to an [OuiHardwareAddr]:
//
//	addr, err := mactracker.ParseMAC("0050.5612.3456")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(addr) // 00:50:56:12:34:56
//
// # Detecting virtual-machine MACs
//
// [Lookup] returns a [OUiBlock] with a [Virtual] flag for known virtual-machine prefixes,and you can also check against the virtual table directly with [LookupVirtual],
//
// [LookupVirtual] checks a MAC address against a table of
// hypervisor and cloud-provider prefixes:
//
//	platform := mactracker.LookupVirtual("00:50:56:12:34:56")
//	fmt.Println(platform) // "VMware"
//
// # Looking up against individual tables
//
// [LookupOUI] searches only the primary IEEE registration table:
//
//	block := mactracker.LookupOUI("00:1b:c5:00:02:03")
//	if block != nil {
//		fmt.Println(block.Vendor) // "Converging Systems Inc."
//	}
//
// [LookupOverride] searches only the override table for unofficial
// and private registrations:
//
//	block := mactracker.LookupOverride("d0:c9:07:aa:bb:cc")
//	if block != nil {
//		fmt.Println(block.Vendor) // "Govee"
//	}
//
// # Building a CIDR-style mask
//
// [MaskFromCIDR] creates a byte-level mask useful for custom prefix matching:
//
//	mask := mactracker.MaskFromCIDR(24, 48)
//	fmt.Printf("%x\n", mask) // ffffff000000
package mactracker
