//go:build unit

// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// TargetConfig Tests
// =============================================================================

func TestParseTargetConfig_Valid(t *testing.T) {
	configJSON := `{"api_token": "test-token-123", "zone_id": "zone-abc-456"}`

	config, err := parseTargetConfig(json.RawMessage(configJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if config.APIToken != "test-token-123" {
		t.Errorf("expected APIToken 'test-token-123', got '%s'", config.APIToken)
	}
	if config.ZoneID != "zone-abc-456" {
		t.Errorf("expected ZoneID 'zone-abc-456', got '%s'", config.ZoneID)
	}
}

func TestParseTargetConfig_MissingAPIToken(t *testing.T) {
	configJSON := `{"zone_id": "zone-abc-456"}`

	_, err := parseTargetConfig(json.RawMessage(configJSON))
	if err == nil {
		t.Fatal("expected error for missing api_token, got nil")
	}
}

func TestParseTargetConfig_MissingZoneID(t *testing.T) {
	configJSON := `{"api_token": "test-token-123"}`

	_, err := parseTargetConfig(json.RawMessage(configJSON))
	if err == nil {
		t.Fatal("expected error for missing zone_id, got nil")
	}
}

func TestParseTargetConfig_InvalidJSON(t *testing.T) {
	configJSON := `{invalid json}`

	_, err := parseTargetConfig(json.RawMessage(configJSON))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// =============================================================================
// DNSRecordProperties Tests
// =============================================================================

func TestParseProperties_ARecord(t *testing.T) {
	propsJSON := `{
		"record_type": "A",
		"name": "test.example.com",
		"content": "192.0.2.1",
		"ttl": 300,
		"proxied": true
	}`

	props, err := parseProperties(json.RawMessage(propsJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if props.RecordType != "A" {
		t.Errorf("expected RecordType 'A', got '%s'", props.RecordType)
	}
	if props.Name != "test.example.com" {
		t.Errorf("expected Name 'test.example.com', got '%s'", props.Name)
	}
	if props.Content != "192.0.2.1" {
		t.Errorf("expected Content '192.0.2.1', got '%s'", props.Content)
	}
	if props.TTL != 300 {
		t.Errorf("expected TTL 300, got %d", props.TTL)
	}
	if !props.Proxied {
		t.Error("expected Proxied true, got false")
	}
}

func TestParseProperties_MXRecord(t *testing.T) {
	propsJSON := `{
		"record_type": "MX",
		"name": "example.com",
		"content": "mail.example.com",
		"priority": 10,
		"ttl": 1
	}`

	props, err := parseProperties(json.RawMessage(propsJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if props.RecordType != "MX" {
		t.Errorf("expected RecordType 'MX', got '%s'", props.RecordType)
	}
	if props.Priority == nil || *props.Priority != 10 {
		t.Error("expected Priority 10")
	}
}

func TestParseProperties_WithComment(t *testing.T) {
	propsJSON := `{
		"record_type": "TXT",
		"name": "example.com",
		"content": "v=spf1 -all",
		"comment": "SPF record"
	}`

	props, err := parseProperties(json.RawMessage(propsJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if props.Comment == nil || *props.Comment != "SPF record" {
		t.Error("expected Comment 'SPF record'")
	}
}

func TestParseProperties_DefaultValues(t *testing.T) {
	propsJSON := `{
		"record_type": "A",
		"name": "test.example.com",
		"content": "192.0.2.1"
	}`

	props, err := parseProperties(json.RawMessage(propsJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// TTL defaults to 1 (automatic)
	if props.TTL != 1 {
		t.Errorf("expected default TTL 1, got %d", props.TTL)
	}
	// Proxied defaults to false
	if props.Proxied {
		t.Error("expected default Proxied false, got true")
	}
}

func TestParseProperties_MissingRecordType(t *testing.T) {
	propsJSON := `{
		"name": "test.example.com",
		"content": "192.0.2.1"
	}`

	_, err := parseProperties(json.RawMessage(propsJSON))
	if err == nil {
		t.Fatal("expected error for missing record_type, got nil")
	}
}

func TestParseProperties_MissingName(t *testing.T) {
	propsJSON := `{
		"record_type": "A",
		"content": "192.0.2.1"
	}`

	_, err := parseProperties(json.RawMessage(propsJSON))
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestParseProperties_MissingContent(t *testing.T) {
	propsJSON := `{
		"record_type": "A",
		"name": "test.example.com"
	}`

	_, err := parseProperties(json.RawMessage(propsJSON))
	if err == nil {
		t.Fatal("expected error for missing content, got nil")
	}
}

// =============================================================================
// Validation Tests
// =============================================================================

func TestValidateProperties_ValidARecord(t *testing.T) {
	props := &DNSRecordProperties{
		RecordType: "A",
		Name:       "test.example.com",
		Content:    "192.0.2.1",
		TTL:        300,
		Proxied:    true,
	}

	err := validateProperties(props)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProperties_ValidMXRecord(t *testing.T) {
	priority := 10
	props := &DNSRecordProperties{
		RecordType: "MX",
		Name:       "example.com",
		Content:    "mail.example.com",
		TTL:        300,
		Priority:   &priority,
	}

	err := validateProperties(props)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProperties_MXRecordMissingPriority(t *testing.T) {
	props := &DNSRecordProperties{
		RecordType: "MX",
		Name:       "example.com",
		Content:    "mail.example.com",
		TTL:        300,
	}

	err := validateProperties(props)
	if err == nil {
		t.Fatal("expected error for MX record without priority, got nil")
	}
}

func TestValidateProperties_SRVRecordMissingPriority(t *testing.T) {
	props := &DNSRecordProperties{
		RecordType: "SRV",
		Name:       "_sip._tcp.example.com",
		Content:    "5 5060 sipserver.example.com",
		TTL:        300,
	}

	err := validateProperties(props)
	if err == nil {
		t.Fatal("expected error for SRV record without priority, got nil")
	}
}

func TestValidateProperties_ValidSRVRecord(t *testing.T) {
	priority := 10
	props := &DNSRecordProperties{
		RecordType: "SRV",
		Name:       "_sip._tcp.example.com",
		Content:    "5 5060 sipserver.example.com",
		TTL:        300,
		Priority:   &priority,
	}

	err := validateProperties(props)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProperties_InvalidRecordType(t *testing.T) {
	props := &DNSRecordProperties{
		RecordType: "INVALID",
		Name:       "test.example.com",
		Content:    "192.0.2.1",
		TTL:        300,
	}

	err := validateProperties(props)
	if err == nil {
		t.Fatal("expected error for invalid record type, got nil")
	}
}

func TestValidateProperties_ProxiedOnNonProxyableType(t *testing.T) {
	props := &DNSRecordProperties{
		RecordType: "TXT",
		Name:       "example.com",
		Content:    "some text",
		TTL:        300,
		Proxied:    true,
	}

	err := validateProperties(props)
	if err == nil {
		t.Fatal("expected error for proxied TXT record, got nil")
	}
}

func TestValidateProperties_AllSupportedTypes(t *testing.T) {
	tests := []struct {
		recordType string
		priority   *int
	}{
		{"A", nil},
		{"AAAA", nil},
		{"CNAME", nil},
		{"MX", intPtr(10)},
		{"TXT", nil},
		{"NS", nil},
		{"CAA", nil},
		{"SRV", intPtr(10)},
	}

	for _, tt := range tests {
		t.Run(tt.recordType, func(t *testing.T) {
			props := &DNSRecordProperties{
				RecordType: tt.recordType,
				Name:       "test.example.com",
				Content:    "test-content",
				TTL:        300,
				Priority:   tt.priority,
			}

			err := validateProperties(props)
			if err != nil {
				t.Fatalf("unexpected error for %s record: %v", tt.recordType, err)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
