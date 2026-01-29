// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"encoding/json"

	"github.com/platform-engineering-labs/formae/pkg/plugin"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"

	cfclient "github.com/platform-engineering-labs/formae-plugin-cloudflare-dns/pkg/cloudflare"
)

// Plugin implements the Formae ResourcePlugin interface.
// The SDK automatically provides identity methods (Name, Version, Namespace)
// by reading formae-plugin.pkl at startup.
type Plugin struct{}

// Compile-time check: Plugin must satisfy ResourcePlugin interface.
var _ plugin.ResourcePlugin = &Plugin{}

// =============================================================================
// Configuration Methods
// =============================================================================

// RateLimit returns the rate limiting configuration for this plugin.
// Cloudflare allows ~1200 requests per 5 minutes, so we limit to 4/sec.
func (p *Plugin) RateLimit() plugin.RateLimitConfig {
	return plugin.RateLimitConfig{
		Scope:                            plugin.RateLimitScopeNamespace,
		MaxRequestsPerSecondForNamespace: 4,
	}
}

// DiscoveryFilters returns filters to exclude certain resources from discovery.
// Return nil to discover all resources.
func (p *Plugin) DiscoveryFilters() []plugin.MatchFilter {
	return nil
}

// LabelConfig returns the configuration for extracting human-readable labels
// from discovered resources.
func (p *Plugin) LabelConfig() plugin.LabelConfig {
	return plugin.LabelConfig{
		DefaultQuery: "$.name",
	}
}

// =============================================================================
// Target Config
// =============================================================================

// targetConfig represents the target configuration for Cloudflare DNS.
type targetConfig struct {
	ZoneID string `json:"zone_id"`
}

// parseTargetConfig parses the target config from json.RawMessage.
func parseTargetConfig(raw json.RawMessage) (*targetConfig, error) {
	var cfg targetConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// =============================================================================
// CRUD Operations
// =============================================================================

// dnsRecordProperties represents the JSON structure of a DNS record's properties.
type dnsRecordProperties struct {
	ID         string `json:"id,omitempty"`
	ZoneID     string `json:"zone_id"`
	Name       string `json:"name"`
	RecordType string `json:"record_type"`
	Content    string `json:"content"`
	TTL        int64  `json:"ttl"`
	Proxied    bool   `json:"proxied"`
	Comment    string `json:"comment,omitempty"`
	Priority   *int64 `json:"priority,omitempty"`
}

// Create provisions a new DNS record.
func (p *Plugin) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	// Parse request properties
	var props dnsRecordProperties
	if err := json.Unmarshal(req.Properties, &props); err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   "Failed to parse properties: " + err.Error(),
			},
		}, nil
	}

	// Create Cloudflare client
	client, err := cfclient.NewClient()
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidCredentials,
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	// Create the DNS record
	record := &cfclient.DNSRecord{
		ZoneID:     props.ZoneID,
		Name:       props.Name,
		RecordType: props.RecordType,
		Content:    props.Content,
		TTL:        props.TTL,
		Proxied:    props.Proxied,
		Comment:    props.Comment,
		Priority:   props.Priority,
	}

	created, err := client.CreateRecord(ctx, record)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       cfclient.MapErrorCode(err),
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	// Marshal the created resource properties
	// Use the original user-provided name, not Cloudflare's FQDN response
	// Cloudflare returns the FQDN (e.g., "test.example.com") but users specify
	// the short name (e.g., "test"), and we need to match for idempotency
	createdProps := dnsRecordProperties{
		ID:         created.ID,
		ZoneID:     created.ZoneID,
		Name:       props.Name, // Use original name, not FQDN from Cloudflare
		RecordType: created.RecordType,
		Content:    created.Content,
		TTL:        created.TTL,
		Proxied:    created.Proxied,
		Comment:    created.Comment,
		Priority:   created.Priority,
	}

	propsJSON, err := json.Marshal(createdProps)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   "Failed to marshal properties: " + err.Error(),
			},
		}, nil
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationCreate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           created.ID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// Read retrieves the current state of a DNS record.
func (p *Plugin) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	// Parse target config to get zone_id
	cfg, err := parseTargetConfig(req.TargetConfig)
	if err != nil || cfg.ZoneID == "" {
		return &resource.ReadResult{
			ResourceType: req.ResourceType,
			ErrorCode:    resource.OperationErrorCodeInvalidRequest,
		}, nil
	}

	// Create Cloudflare client
	client, err := cfclient.NewClient()
	if err != nil {
		return &resource.ReadResult{
			ResourceType: req.ResourceType,
			ErrorCode:    resource.OperationErrorCodeInvalidCredentials,
		}, nil
	}

	// Get the DNS record
	record, err := client.GetRecord(ctx, cfg.ZoneID, req.NativeID)
	if err != nil {
		if cfclient.IsNotFound(err) {
			return &resource.ReadResult{
				ResourceType: req.ResourceType,
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return &resource.ReadResult{
			ResourceType: req.ResourceType,
			ErrorCode:    cfclient.MapErrorCode(err),
		}, nil
	}

	// Get zone domain to normalize the record name
	// Cloudflare returns FQDNs but users specify short names
	zoneDomain, err := client.GetZoneDomain(ctx, cfg.ZoneID)
	if err != nil {
		return &resource.ReadResult{
			ResourceType: req.ResourceType,
			ErrorCode:    cfclient.MapErrorCode(err),
		}, nil
	}

	// Normalize the name by stripping the zone domain suffix
	normalizedName := cfclient.NormalizeName(record.Name, zoneDomain)

	// Marshal the record properties
	props := dnsRecordProperties{
		ID:         record.ID,
		ZoneID:     record.ZoneID,
		Name:       normalizedName,
		RecordType: record.RecordType,
		Content:    record.Content,
		TTL:        record.TTL,
		Proxied:    record.Proxied,
		Comment:    record.Comment,
		Priority:   record.Priority,
	}

	propsJSON, err := json.Marshal(props)
	if err != nil {
		return &resource.ReadResult{
			ResourceType: req.ResourceType,
			ErrorCode:    resource.OperationErrorCodeInternalFailure,
		}, nil
	}

	return &resource.ReadResult{
		ResourceType: req.ResourceType,
		Properties:   string(propsJSON),
	}, nil
}

// Update modifies an existing DNS record.
func (p *Plugin) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	// Parse desired properties
	var props dnsRecordProperties
	if err := json.Unmarshal(req.DesiredProperties, &props); err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   "Failed to parse properties: " + err.Error(),
			},
		}, nil
	}

	// Create Cloudflare client
	client, err := cfclient.NewClient()
	if err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidCredentials,
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	// Update the DNS record
	record := &cfclient.DNSRecord{
		ID:         req.NativeID,
		ZoneID:     props.ZoneID,
		Name:       props.Name,
		RecordType: props.RecordType,
		Content:    props.Content,
		TTL:        props.TTL,
		Proxied:    props.Proxied,
		Comment:    props.Comment,
		Priority:   props.Priority,
	}

	updated, err := client.UpdateRecord(ctx, record)
	if err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       cfclient.MapErrorCode(err),
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	// Marshal the updated resource properties
	// Use the original user-provided name, not Cloudflare's FQDN response
	updatedProps := dnsRecordProperties{
		ID:         updated.ID,
		ZoneID:     updated.ZoneID,
		Name:       props.Name, // Use original name, not FQDN from Cloudflare
		RecordType: updated.RecordType,
		Content:    updated.Content,
		TTL:        updated.TTL,
		Proxied:    updated.Proxied,
		Comment:    updated.Comment,
		Priority:   updated.Priority,
	}

	propsJSON, err := json.Marshal(updatedProps)
	if err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   "Failed to marshal properties: " + err.Error(),
			},
		}, nil
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:          resource.OperationUpdate,
			OperationStatus:    resource.OperationStatusSuccess,
			NativeID:           updated.ID,
			ResourceProperties: propsJSON,
		},
	}, nil
}

// Delete removes a DNS record.
func (p *Plugin) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	// Parse target config to get zone_id
	cfg, err := parseTargetConfig(req.TargetConfig)
	if err != nil || cfg.ZoneID == "" {
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   "zone_id not found in target config",
			},
		}, nil
	}

	// Create Cloudflare client
	client, err := cfclient.NewClient()
	if err != nil {
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidCredentials,
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	// Delete the DNS record
	err = client.DeleteRecord(ctx, cfg.ZoneID, req.NativeID)
	if err != nil {
		// Treat 404 as success (idempotent delete)
		if cfclient.IsNotFound(err) {
			return &resource.DeleteResult{
				ProgressResult: &resource.ProgressResult{
					Operation:       resource.OperationDelete,
					OperationStatus: resource.OperationStatusSuccess,
				},
			}, nil
		}
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       cfclient.MapErrorCode(err),
				StatusMessage:   err.Error(),
			},
		}, nil
	}

	return &resource.DeleteResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationDelete,
			OperationStatus: resource.OperationStatusSuccess,
		},
	}, nil
}

// Status checks the progress of an async operation.
// DNS operations are synchronous, so we return success immediately.
func (p *Plugin) Status(ctx context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
		},
	}, nil
}

// List returns all DNS record identifiers in the zone.
func (p *Plugin) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	// Parse target config to get zone_id
	cfg, err := parseTargetConfig(req.TargetConfig)
	if err != nil || cfg.ZoneID == "" {
		return &resource.ListResult{
			NativeIDs:     []string{},
			NextPageToken: nil,
		}, nil
	}

	// Create Cloudflare client
	client, err := cfclient.NewClient()
	if err != nil {
		return &resource.ListResult{
			NativeIDs:     []string{},
			NextPageToken: nil,
		}, nil
	}

	// List DNS records
	records, nextToken, err := client.ListRecords(ctx, cfg.ZoneID, req.PageToken, int(req.PageSize))
	if err != nil {
		return &resource.ListResult{
			NativeIDs:     []string{},
			NextPageToken: nil,
		}, nil
	}

	// Extract native IDs
	nativeIDs := make([]string, 0, len(records))
	for _, record := range records {
		nativeIDs = append(nativeIDs, record.ID)
	}

	return &resource.ListResult{
		NativeIDs:     nativeIDs,
		NextPageToken: nextToken,
	}, nil
}
