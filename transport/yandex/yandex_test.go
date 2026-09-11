package yandex

import (
	"reflect"
	"testing"
)

func TestParseDocURLs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single URL",
			input:    "https://disk.yandex.ru/i/single_doc_123",
			expected: []string{"https://disk.yandex.ru/i/single_doc_123"},
		},
		{
			name:  "Comma-separated URLs",
			input: "https://disk.yandex.ru/i/doc1, https://disk.yandex.ru/i/doc2,https://disk.yandex.ru/i/doc3",
			expected: []string{
				"https://disk.yandex.ru/i/doc1",
				"https://disk.yandex.ru/i/doc2",
				"https://disk.yandex.ru/i/doc3",
			},
		},
		{
			name:  "Multiline URLs with whitespace and semicolons",
			input: "https://disk.yandex.ru/i/doc1\nhttps://disk.yandex.ru/i/doc2;\r\n  https://disk.yandex.ru/i/doc3  ",
			expected: []string{
				"https://disk.yandex.ru/i/doc1",
				"https://disk.yandex.ru/i/doc2",
				"https://disk.yandex.ru/i/doc3",
			},
		},
		{
			name:     "Non-http fallback",
			input:    "custom_placeholder",
			expected: []string{"custom_placeholder"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseDocURLs(tc.input)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}
