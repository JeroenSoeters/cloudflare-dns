// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: BSD-3-Clause

// Package cloudflare provides a client wrapper for the Cloudflare API.
package cloudflare

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"

	cf "github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
)

// Client wraps the Cloudflare SDK client for DNS operations.
type Client struct {
	api *cf.Client
}

// NewClient creates a new Cloudflare client using the CLOUDFLARE_API_TOKEN
// environment variable for authentication.
func NewClient() (*Client, error) {
	token := os.Getenv("CLOUDFLARE_API_TOKEN")
	if token == "" {
		return nil, errors.New("CLOUDFLARE_API_TOKEN environment variable not set")
	}

	client := cf.NewClient(option.WithAPIToken(token))
	return &Client{api: client}, nil
}

// DNSRecord represents a DNS record with the fields we care about.
type DNSRecord struct {
	ID         string
	ZoneID     string
	Name       string
	RecordType string
	Content    string
	TTL        int64
	Proxied    bool
	Comment    string
	Priority   *int64
}

// CreateRecord creates a new DNS record in the specified zone.
func (c *Client) CreateRecord(ctx context.Context, record *DNSRecord) (*DNSRecord, error) {
	params := dns.RecordNewParams{
		ZoneID: cf.F(record.ZoneID),
	}

	// Build the appropriate record type based on RecordType
	switch record.RecordType {
	case "A":
		params.Body = dns.ARecordParam{
			Name:    cf.F(record.Name),
			Type:    cf.F(dns.ARecordTypeA),
			Content: cf.F(record.Content),
			TTL:     cf.F(dns.TTL(record.TTL)),
			Proxied: cf.F(record.Proxied),
			Comment: cf.F(record.Comment),
		}
	case "AAAA":
		params.Body = dns.AAAARecordParam{
			Name:    cf.F(record.Name),
			Type:    cf.F(dns.AAAARecordTypeAAAA),
			Content: cf.F(record.Content),
			TTL:     cf.F(dns.TTL(record.TTL)),
			Proxied: cf.F(record.Proxied),
			Comment: cf.F(record.Comment),
		}
	case "CNAME":
		params.Body = dns.CNAMERecordParam{
			Name:    cf.F(record.Name),
			Type:    cf.F(dns.CNAMERecordTypeCNAME),
			Content: cf.F(record.Content),
			TTL:     cf.F(dns.TTL(record.TTL)),
			Proxied: cf.F(record.Proxied),
			Comment: cf.F(record.Comment),
		}
	case "TXT":
		params.Body = dns.TXTRecordParam{
			Name:    cf.F(record.Name),
			Type:    cf.F(dns.TXTRecordTypeTXT),
			Content: cf.F(record.Content),
			TTL:     cf.F(dns.TTL(record.TTL)),
			Comment: cf.F(record.Comment),
		}
	case "MX":
		priority := float64(10)
		if record.Priority != nil {
			priority = float64(*record.Priority)
		}
		params.Body = dns.MXRecordParam{
			Name:     cf.F(record.Name),
			Type:     cf.F(dns.MXRecordTypeMX),
			Content:  cf.F(record.Content),
			TTL:      cf.F(dns.TTL(record.TTL)),
			Priority: cf.F(priority),
			Comment:  cf.F(record.Comment),
		}
	case "NS":
		params.Body = dns.NSRecordParam{
			Name:    cf.F(record.Name),
			Type:    cf.F(dns.NSRecordTypeNS),
			Content: cf.F(record.Content),
			TTL:     cf.F(dns.TTL(record.TTL)),
			Comment: cf.F(record.Comment),
		}
	default:
		return nil, errors.New("unsupported record type: " + record.RecordType)
	}

	resp, err := c.api.DNS.Records.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return recordFromResponse(resp, record.ZoneID), nil
}

// GetRecord retrieves a DNS record by ID.
func (c *Client) GetRecord(ctx context.Context, zoneID, recordID string) (*DNSRecord, error) {
	resp, err := c.api.DNS.Records.Get(ctx, recordID, dns.RecordGetParams{
		ZoneID: cf.F(zoneID),
	})
	if err != nil {
		return nil, err
	}

	return recordFromResponse(resp, zoneID), nil
}

// UpdateRecord updates an existing DNS record.
func (c *Client) UpdateRecord(ctx context.Context, record *DNSRecord) (*DNSRecord, error) {
	params := dns.RecordUpdateParams{
		ZoneID: cf.F(record.ZoneID),
	}

	// Build the appropriate record type based on RecordType
	switch record.RecordType {
	case "A":
		params.Body = dns.ARecordParam{
			Name:    cf.F(record.Name),
			Type:    cf.F(dns.ARecordTypeA),
			Content: cf.F(record.Content),
			TTL:     cf.F(dns.TTL(record.TTL)),
			Proxied: cf.F(record.Proxied),
			Comment: cf.F(record.Comment),
		}
	case "AAAA":
		params.Body = dns.AAAARecordParam{
			Name:    cf.F(record.Name),
			Type:    cf.F(dns.AAAARecordTypeAAAA),
			Content: cf.F(record.Content),
			TTL:     cf.F(dns.TTL(record.TTL)),
			Proxied: cf.F(record.Proxied),
			Comment: cf.F(record.Comment),
		}
	case "CNAME":
		params.Body = dns.CNAMERecordParam{
			Name:    cf.F(record.Name),
			Type:    cf.F(dns.CNAMERecordTypeCNAME),
			Content: cf.F(record.Content),
			TTL:     cf.F(dns.TTL(record.TTL)),
			Proxied: cf.F(record.Proxied),
			Comment: cf.F(record.Comment),
		}
	case "TXT":
		params.Body = dns.TXTRecordParam{
			Name:    cf.F(record.Name),
			Type:    cf.F(dns.TXTRecordTypeTXT),
			Content: cf.F(record.Content),
			TTL:     cf.F(dns.TTL(record.TTL)),
			Comment: cf.F(record.Comment),
		}
	case "MX":
		priority := float64(10)
		if record.Priority != nil {
			priority = float64(*record.Priority)
		}
		params.Body = dns.MXRecordParam{
			Name:     cf.F(record.Name),
			Type:     cf.F(dns.MXRecordTypeMX),
			Content:  cf.F(record.Content),
			TTL:      cf.F(dns.TTL(record.TTL)),
			Priority: cf.F(priority),
			Comment:  cf.F(record.Comment),
		}
	case "NS":
		params.Body = dns.NSRecordParam{
			Name:    cf.F(record.Name),
			Type:    cf.F(dns.NSRecordTypeNS),
			Content: cf.F(record.Content),
			TTL:     cf.F(dns.TTL(record.TTL)),
			Comment: cf.F(record.Comment),
		}
	default:
		return nil, errors.New("unsupported record type: " + record.RecordType)
	}

	resp, err := c.api.DNS.Records.Update(ctx, record.ID, params)
	if err != nil {
		return nil, err
	}

	return recordFromResponse(resp, record.ZoneID), nil
}

// DeleteRecord deletes a DNS record.
func (c *Client) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	_, err := c.api.DNS.Records.Delete(ctx, recordID, dns.RecordDeleteParams{
		ZoneID: cf.F(zoneID),
	})
	return err
}

// ListRecords lists all DNS records in a zone.
func (c *Client) ListRecords(ctx context.Context, zoneID string, pageToken *string, pageSize int) ([]*DNSRecord, *string, error) {
	params := dns.RecordListParams{
		ZoneID: cf.F(zoneID),
	}

	if pageSize > 0 {
		params.PerPage = cf.F(float64(pageSize))
	} else {
		// Default page size
		pageSize = 100
		params.PerPage = cf.F(float64(pageSize))
	}

	if pageToken != nil && *pageToken != "" {
		page, err := strconv.ParseFloat(*pageToken, 64)
		if err == nil && page >= 1 {
			params.Page = cf.F(page)
		}
	}

	resp, err := c.api.DNS.Records.List(ctx, params)
	if err != nil {
		return nil, nil, err
	}

	records := make([]*DNSRecord, 0, len(resp.Result))
	for i := range resp.Result {
		records = append(records, recordFromListItem(&resp.Result[i], zoneID))
	}

	// Determine next page token - if we got a full page, there might be more
	var nextToken *string
	if len(resp.Result) == pageSize {
		nextPage := strconv.FormatInt(resp.ResultInfo.Page+1, 10)
		nextToken = &nextPage
	}

	return records, nextToken, nil
}

// recordFromResponse converts a DNS record response to our DNSRecord type.
func recordFromResponse(resp *dns.RecordResponse, zoneID string) *DNSRecord {
	record := &DNSRecord{
		ID:         resp.ID,
		ZoneID:     zoneID,
		Name:       resp.Name,
		RecordType: string(resp.Type),
		Content:    resp.Content,
		Proxied:    resp.Proxied,
		Comment:    resp.Comment,
		TTL:        int64(resp.TTL),
	}

	// Handle priority for MX records
	if resp.Priority != 0 {
		priority := int64(resp.Priority)
		record.Priority = &priority
	}

	return record
}

// recordFromListItem converts a list result item to our DNSRecord type.
func recordFromListItem(r *dns.RecordResponse, zoneID string) *DNSRecord {
	record := &DNSRecord{
		ID:         r.ID,
		ZoneID:     zoneID,
		Name:       r.Name,
		RecordType: string(r.Type),
		Content:    r.Content,
		Proxied:    r.Proxied,
		Comment:    r.Comment,
		TTL:        int64(r.TTL),
	}

	// Handle priority for MX records
	if r.Priority != 0 {
		priority := int64(r.Priority)
		record.Priority = &priority
	}

	return record
}

// GetZoneDomain retrieves the domain name for a zone by its ID.
func (c *Client) GetZoneDomain(ctx context.Context, zoneID string) (string, error) {
	resp, err := c.api.Zones.Get(ctx, zones.ZoneGetParams{
		ZoneID: cf.F(zoneID),
	})
	if err != nil {
		return "", err
	}
	return resp.Name, nil
}

// NormalizeName strips the zone domain suffix from an FQDN to get the short name.
// If the name equals the zone domain (apex record), it returns "@".
// If the name doesn't end with the zone domain, it returns the name unchanged.
func NormalizeName(fqdn, zoneDomain string) string {
	// Handle apex records (name equals zone domain)
	if fqdn == zoneDomain {
		return "@"
	}

	// Check if the FQDN ends with the zone domain
	suffix := "." + zoneDomain
	if strings.HasSuffix(fqdn, suffix) {
		return strings.TrimSuffix(fqdn, suffix)
	}

	// Return unchanged if it doesn't match the expected pattern
	return fqdn
}
