package mactracker

// OUISkipPrefixes is a set of prefixes to skip to avoid bad lookups.
var OUISkipPrefixes = map[string]struct{}{
	// Mask: 24
	"000000000000/24": {}, // The zero OUI is registered to Xerox but is typically invalid input
}

// OUITableExtra is a set of overrides for unofficial and private registrations.
var OUITableExtra = OuiDB{Blocks: map[string]*OuiBlock{
	// Mask: 24
	"d0c907000000/24": {Oui: []byte{0xd0, 0xc9, 0x07, 0x00, 0x00, 0x00}, Mask: 24, Vendor: "Govee", Added: "2023-12-14", Private: true}, // Private: Smart light bulbs
}}
