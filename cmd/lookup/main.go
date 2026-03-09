package main

import (
	"fmt"
	"os"

	mactracker "github.com/hdm/mac-tracker"
)

func main() {
	for _, v := range os.Args[1:] {
		block := mactracker.MACLookup(v)
		if block == nil {
			fmt.Printf("%s: No match found\n", v)
			continue
		}
		fmt.Printf("%s: [%s] %s - %s\n", v, block.Added, block.Vendor, block.Address)
	}
}
