package mactracker

const (
	VirtTypeAWS          = "AWS"
	VirtTypeAzure        = "Azure"
	VirtTypeGCP          = "GCP"
	VirtTypeHyperV       = "Hyper-V"
	VirtTypeMicrosoft    = "Microsoft"
	VirtTypeVMware       = "VMware"
	VirtTypeProxmox      = "Proxmox"
	VirtTypeVirtualBox   = "VirtualBox"
	VirtTypeQEMU         = "QEMU"
	VirtTypeKVM          = "KVM"
	VirtTypeXen          = "Xen"
	VirtTypeParallels    = "Parallels"
	VirtTypeOracle       = "Oracle"
	VirtTypeNutanix      = "Nutanix"
	VirtTypeDigitalOcean = "DigitalOcean"
)

// OUITableVirtual maps well-known virtual-machine MAC prefixes to their platform names.
var OUITableVirtual = OuiDB{Blocks: map[string]*OuiBlock{
	// Mask: 24
	"505400000000/24": {Oui: []byte{0x50, 0x54, 0x00, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "QEMU", Added: "2003-03-18", Virtual: VirtTypeQEMU},
	"545200000000/24": {Oui: []byte{0x54, 0x52, 0x00, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Linux KVM", Added: "2008-09-01", Virtual: VirtTypeKVM},
	"001c42000000/24": {Oui: []byte{0x00, 0x1c, 0x42, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Parallels, Inc.", Added: "2007-05-13", Virtual: VirtTypeParallels},
	"2cc260000000/24": {Oui: []byte{0x2c, 0xc2, 0x60, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Oracle Corporation", Added: "2012-01-19", Virtual: VirtTypeOracle},
	"080027000000/24": {Oui: []byte{0x08, 0x00, 0x27, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Oracle Corporation", Added: "2004-01-23", Virtual: VirtTypeVirtualBox}, // VirtualBox reused a registered OUI, but it's widely recognized as a virtual prefix
	"0021f6000000/24": {Oui: []byte{0x00, 0x21, 0xf6, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Oracle Corporation", Added: "2008-06-18", Virtual: VirtTypeVirtualBox},
	"000f4b000000/24": {Oui: []byte{0x00, 0x0f, 0x4b, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Oracle Corporation", Added: "2004-01-23", Virtual: VirtTypeVirtualBox},
	"005056000000/24": {Oui: []byte{0x00, 0x50, 0x56, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "VMware, Inc.", Added: "2000-01-04", Virtual: VirtTypeVMware},
	"000c29000000/24": {Oui: []byte{0x00, 0x0c, 0x29, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "VMware, Inc.", Added: "2003-01-21", Virtual: VirtTypeVMware},
	"000569000000/24": {Oui: []byte{0x00, 0x05, 0x69, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "VMware, Inc.", Added: "2001-04-17", Virtual: VirtTypeVMware},
	"001c14000000/24": {Oui: []byte{0x00, 0x1c, 0x14, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "VMware, Inc.", Added: "2007-04-08", Virtual: VirtTypeVMware},
	"00163e000000/24": {Oui: []byte{0x00, 0x16, 0x3e, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Xensource, Inc.", Added: "2005-10-29", Virtual: VirtTypeXen},
	"00cafe000000/24": {Oui: []byte{0x00, 0xca, 0xfe, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Xensource, Inc.", Added: "2005-10-29", Virtual: VirtTypeXen},
	"00155d000000/24": {Oui: []byte{0x00, 0x15, 0x5d, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Microsoft Corporation", Added: "2005-08-04", Virtual: VirtTypeHyperV},
	"0003ff000000/24": {Oui: []byte{0x00, 0x03, 0xff, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Microsoft Corporation", Added: "2000-11-09", Virtual: VirtTypeHyperV},
	"001dd8000000/24": {Oui: []byte{0x00, 0x1d, 0xd8, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Microsoft Corporation", Added: "2007-09-25", Virtual: VirtTypeHyperV},
	"bc2411000000/24": {Oui: []byte{0xbc, 0x24, 0x11, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Proxmox Server Solutions GmbH", Added: "2023-06-16", Virtual: VirtTypeProxmox},
	"42010a000000/24": {Oui: []byte{0x42, 0x01, 0x0a, 0x00, 0x02, 0x01}, Mask: 24, Vendor: "Google LLC", Added: "2023-06-16", Virtual: VirtTypeGCP},
}}

// LookupVirtual returns the virtualization platform name (e.g. "VMware", "QEMU")
// for a MAC address string, or an empty string if the address is not a known virtual OUI.
func LookupVirtual(mac string) string {
	addr, err := ParseMAC(mac)
	if err != nil {
		return ""
	}

	block := OUITableVirtual.Lookup(addr)
	if block != nil {
		return block.Virtual
	}
	return ""
}
