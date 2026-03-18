package mactracker

import "testing"

func TestOuiLookup(t *testing.T) {
	// Test known OUI
	block := LookupBytes(OuiHardwareAddr{0x00, 0x1b, 0xc5, 0x00, 0x00, 0x00})
	if block == nil || block.Vendor != "Converging Systems Inc." {
		t.Errorf("Expected to find Converging Systems Inc., got %v", block)
	}

	// The zero OUI is registered to Xerox but should not return a valid registration
	block = LookupBytes(OuiHardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if block != nil {
		t.Errorf("Expected to find no block for unknown OUI, got %v", block)
	}

	// Test private OUI with override
	block = LookupBytes(OuiHardwareAddr{0x50, 0x14, 0x00, 0x00, 0x00, 0x00})
	if block == nil || block.Vendor != "QEMU" {
		t.Errorf("Expected to find QEMU for private OUI, got %v", block)
	}
}

func TestLookupVirtual(t *testing.T) {
	tests := []struct {
		mac  string
		want string
	}{
		{mac: "50:54:00:12:34:56", want: VirtTypeQEMU},
		{mac: "54:52:00:12:34:56", want: VirtTypeKVM},
		{mac: "00:1c:42:12:34:56", want: VirtTypeParallels},
		{mac: "2c:c2:60:12:34:56", want: VirtTypeOracle},
		{mac: "08:00:27:12:34:56", want: VirtTypeVirtualBox},
		{mac: "00:50:56:12:34:56", want: VirtTypeVMware},
		{mac: "00:16:3E:12:34:56", want: VirtTypeXen},
		{mac: "00:15:5d:12:34:56", want: VirtTypeHyperV},
		{mac: "bc:24:11:12:34:56", want: VirtTypeProxmox},
		{mac: "00:00:00:00:00:00", want: ""},
		{mac: "invalid", want: ""},
	}

	for _, test := range tests {
		if got := LookupVirtual(test.mac); got != test.want {
			t.Errorf("LookupVirtual(%q) = %q, want %q", test.mac, got, test.want)
		}
	}
}
