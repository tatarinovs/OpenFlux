package yandex

import (
	"testing"

	"openflux/transport"
)

func TestNewYandexDocsTransport(t *testing.T) {
	docURL := "https://disk.yandex.ru/i/single_doc_123"
	trans := NewYandexDocsTransport(docURL, transport.DefaultConfig())
	if trans.docURL != docURL {
		t.Fatalf("expected docURL %q, got %q", docURL, trans.docURL)
	}
}
