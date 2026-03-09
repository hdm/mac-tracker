package mactracker

import "testing"

func TestOuiLookup(t *testing.T) {
	// Test known OUI
	block := MACLookupBytes(OuiHardwareAddr{0x00, 0x1b, 0xc5, 0x00, 0x00, 0x00})
	if block == nil || block.Vendor != "Converging Systems Inc." {
		t.Errorf("Expected to find Converging Systems Inc., got %v", block)
	}

	// The zero OUI is registered to Xerox but should not return a valid registration
	block = MACLookupBytes(OuiHardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if block != nil {
		t.Errorf("Expected to find no block for unknown OUI, got %v", block)
	}

	// Test private OUI with override
	block = MACLookupBytes(OuiHardwareAddr{0x50, 0x14, 0x00, 0x00, 0x00, 0x00})
	if block == nil || block.Vendor != "QEMU" {
		t.Errorf("Expected to find QEMU for private OUI, got %v", block)
	}
}
