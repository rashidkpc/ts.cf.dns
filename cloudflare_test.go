package main

import (
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

func TestFilterManagedRecords_OwnershipComment(t *testing.T) {
	raw := []cloudflare.DNSRecord{
		{ID: "managed", Name: "foo.ts.example.com", Comment: managedComment},
		{ID: "no-comment", Name: "bar.ts.example.com", Comment: ""},
		{ID: "wrong-comment", Name: "baz.ts.example.com", Comment: "someone-else"},
	}
	got := filterManagedRecords(raw, "ts.example.com")
	if len(got) != 1 || got[0].ID != "managed" {
		t.Errorf("expected only the managed record, got %+v", got)
	}
}

func TestFilterManagedRecords_WrongSuffix(t *testing.T) {
	raw := []cloudflare.DNSRecord{
		{ID: "in-base", Name: "foo.ts.example.com", Comment: managedComment},
		{ID: "wrong-base", Name: "foo.other.example.com", Comment: managedComment},
		{ID: "root-only", Name: "ts.example.com", Comment: managedComment}, // equals base, no prefix dot
	}
	got := filterManagedRecords(raw, "ts.example.com")
	if len(got) != 1 || got[0].ID != "in-base" {
		t.Errorf("expected only record under base suffix, got %+v", got)
	}
}

func TestFilterManagedRecords_Empty(t *testing.T) {
	got := filterManagedRecords(nil, "ts.example.com")
	if len(got) != 0 {
		t.Errorf("expected empty result for nil input, got %+v", got)
	}
}
