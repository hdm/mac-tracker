package main

import (
	"testing"
)

func TestMashEncoding(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "Hello World"},
		{"Test  String", "Test  String"},
		{"  Trimmed  ", "Trimmed"},
		{"Test\xFFstring", "Test<ff>string"},
		{"\xC0\xC1\xF5", "<c0><c1><f5>"},
	}

	for _, test := range tests {
		result := mashEncoding(test.input)
		if result != test.expected {
			t.Errorf("mashEncoding(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

func TestCountryFromAddress(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"657 Orly Ave. Dorval Quebec CA H9P 1G1", "CA"},
		{"No.388 Ning Qiao Road,Jin Qiao Pudong Shanghai Shanghai   CN 201206 ", "CN"},
		{"2121 RDU Center Drive  Morrisville NC US 27560", "US"},
		{"No country code here", ""},
		{"", ""},
	}

	for _, test := range tests {
		result := countryFromAddress(test.input)
		if result != test.expected {
			t.Errorf("countryFromAddress(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

func TestSortablePrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"000e02000000/24", "24000e02000000"},
		{"70b3d5c3c000/36", "3670b3d5c3c000"},
		{"8c1f64ffc000/36", "368c1f64ffc000"},
	}

	for _, test := range tests {
		result := sortablePrefix(test.input)
		if result != test.expected {
			t.Errorf("sortablePrefix(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}
