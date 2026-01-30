// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"github.com/platform-engineering-labs/formae/pkg/plugin"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// ErrNotImplemented is returned by stub methods that need implementation.
var ErrNotImplemented = errors.New("not implemented")


// =============================================================================
// Configuration Types
// =============================================================================

// TargetConfig holds the credentials and configuration for Cloudflare API access.
type TargetConfig struct {
	APIToken string `json:"api_token"`
	ZoneID   string `json:"zone_id"`
}

// DNSRecordProperties represents the properties of a DNS record resource.
type DNSRecordProperties struct {
	RecordType string  `json:"record_type"`
	Name       string  `json:"name"`
	Content    string  `json:"content"`
	TTL        int     `json:"ttl"`
	Proxied    bool    `json:"proxied"`
	Priority   *int    `json:"priority,omitempty"`
	Comment    *string `json:"comment,omitempty"`
}

// Supported record types
var supportedRecordTypes = map[string]bool{
	"A":     true,
	"AAAA":  true,
	"CNAME": true,
	"MX":    true,
	"TXT":   true,
	"NS":    true,
	"CAA":   true,
	"SRV":   true,
}

// Record types that can be proxied through Cloudflare
var proxyableRecordTypes = map[string]bool{
	"A":     true,
	"AAAA":  true,
	"CNAME": true,
}

// Record types that require priority
var priorityRequiredTypes = map[string]bool{
	"MX":  true,
	"SRV": true,
}

// =============================================================================
// Helper Functions
// =============================================================================

// parseTargetConfig parses and validates the target configuration JSON.
func parseTargetConfig(configJSON json.RawMessage) (*TargetConfig, error) {
	var config TargetConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to parse target config: %w", err)
	}

	if config.APIToken == "" {
		return nil, fmt.Errorf("api_token is required in target config")
	}
	if config.ZoneID == "" {
		return nil, fmt.Errorf("zone_id is required in target config")
	}

	return &config, nil
}

// parseProperties parses and validates the DNS record properties JSON.
func parseProperties(propsJSON json.RawMessage) (*DNSRecordProperties, error) {
	// Set defaults
	props := &DNSRecordProperties{
		TTL:     1,     // Cloudflare automatic TTL
		Proxied: false, // Not proxied by default
	}

	if err := json.Unmarshal(propsJSON, props); err != nil {
		return nil, fmt.Errorf("failed to parse properties: %w", err)
	}

	// Validate required fields
	if props.RecordType == "" {
		return nil, fmt.Errorf("record_type is required")
	}
	if props.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if props.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	return props, nil
}

// validateProperties validates DNS record properties based on record type.
func validateProperties(props *DNSRecordProperties) error {
	// Validate record type
	if !supportedRecordTypes[props.RecordType] {
		return fmt.Errorf("unsupported record type: %s", props.RecordType)
	}

	// Validate priority for MX and SRV records
	if priorityRequiredTypes[props.RecordType] && props.Priority == nil {
		return fmt.Errorf("priority is required for %s records", props.RecordType)
	}

	// Validate proxied is only set for proxyable types
	if props.Proxied && !proxyableRecordTypes[props.RecordType] {
		return fmt.Errorf("proxied can only be set for A, AAAA, and CNAME records")
	}

	return nil
}

// createCloudflareClient creates a Cloudflare API client from the target config.
func createCloudflareClient(config *TargetConfig) (*cloudflare.API, error) {
	return cloudflare.NewWithAPIToken(config.APIToken)
}

// propsToCreateParams converts DNSRecordProperties to Cloudflare CreateDNSRecordParams.
func propsToCreateParams(props *DNSRecordProperties) cloudflare.CreateDNSRecordParams {
	params := cloudflare.CreateDNSRecordParams{
		Type:    props.RecordType,
		Name:    props.Name,
		Content: props.Content,
		TTL:     props.TTL,
		Proxied: &props.Proxied,
	}

	if props.Priority != nil {
		priority := uint16(*props.Priority)
		params.Priority = &priority
	}

	if props.Comment != nil {
		params.Comment = *props.Comment
	}

	return params
}

// propsToUpdateParams converts DNSRecordProperties to Cloudflare UpdateDNSRecordParams.
func propsToUpdateParams(props *DNSRecordProperties, recordID string) cloudflare.UpdateDNSRecordParams {
	params := cloudflare.UpdateDNSRecordParams{
		ID:      recordID,
		Type:    props.RecordType,
		Name:    props.Name,
		Content: props.Content,
		TTL:     props.TTL,
		Proxied: &props.Proxied,
		Comment: props.Comment,
	}

	if props.Priority != nil {
		priority := uint16(*props.Priority)
		params.Priority = &priority
	}

	return params
}

// recordToProperties converts a Cloudflare DNSRecord to DNSRecordProperties.
// zoneName is used to strip the zone suffix from the FQDN returned by Cloudflare.
func recordToProperties(record cloudflare.DNSRecord, zoneName string) *DNSRecordProperties {
	// Cloudflare returns the FQDN (e.g., "www.example.com"), but we store the short name ("www")
	name := record.Name
	if zoneName != "" {
		suffix := "." + zoneName
		name = strings.TrimSuffix(name, suffix)
		// Handle zone apex - Cloudflare returns the zone name itself
		if name == zoneName {
			name = "@"
		}
	}

	props := &DNSRecordProperties{
		RecordType: record.Type,
		Name:       name,
		Content:    record.Content,
		TTL:        record.TTL,
	}

	if record.Proxied != nil {
		props.Proxied = *record.Proxied
	}

	if record.Priority != nil {
		priority := int(*record.Priority)
		props.Priority = &priority
	}

	if record.Comment != "" {
		props.Comment = &record.Comment
	}

	return props
}

// getZoneName fetches the zone name from Cloudflare using the zone ID.
func getZoneName(ctx context.Context, client *cloudflare.API, zoneID string) (string, error) {
	zone, err := client.ZoneDetails(ctx, zoneID)
	if err != nil {
		return "", fmt.Errorf("failed to get zone details: %w", err)
	}
	return zone.Name, nil
}

// propertiesToJSON converts DNSRecordProperties to a JSON string.
func propertiesToJSON(props *DNSRecordProperties) (string, error) {
	bytes, err := json.Marshal(props)
	if err != nil {
		return "", fmt.Errorf("failed to marshal properties: %w", err)
	}
	return string(bytes), nil
}

// isNotFoundError checks if the error is a "not found" error from Cloudflare.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "does not exist") ||
		strings.Contains(errStr, "404")
}

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
// Cloudflare API limit: 1200 requests per 5 minutes = 4 requests per second.
func (p *Plugin) RateLimit() plugin.RateLimitConfig {
	return plugin.RateLimitConfig{
		Scope:                            plugin.RateLimitScopeNamespace,
		MaxRequestsPerSecondForNamespace: 4,
	}
}

// DiscoveryFilters returns filters to exclude certain resources from discovery.
// Resources matching ALL conditions in a filter are excluded.
// Return nil if you want to discover all resources.
func (p *Plugin) DiscoveryFilters() []plugin.MatchFilter {
	// Example: exclude resources with a specific tag
	// return []plugin.MatchFilter{
	//     {
	//         ResourceTypes: []string{"CLOUDFLARE::Service::Resource"},
	//         Conditions: []plugin.FilterCondition{
	//             {PropertyPath: "$.Tags[?(@.Key=='skip-discovery')].Value", PropertyValue: "true"},
	//         },
	//     },
	// }
	return nil
}

// LabelConfig returns the configuration for extracting human-readable labels
// from discovered resources.
func (p *Plugin) LabelConfig() plugin.LabelConfig {
	return plugin.LabelConfig{
		// Use the DNS record name as the label
		DefaultQuery: "$.name",

		// No overrides needed
		ResourceOverrides: map[string]string{},
	}
}

// =============================================================================
// CRUD Operations
// =============================================================================

// Create provisions a new resource.
func (p *Plugin) Create(ctx context.Context, req *resource.CreateRequest) (*resource.CreateResult, error) {
	// Parse target config
	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   fmt.Sprintf("Invalid target config: %v", err),
			},
		}, nil
	}

	// Parse properties
	props, err := parseProperties(req.Properties)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   fmt.Sprintf("Invalid properties: %v", err),
			},
		}, nil
	}

	// Validate properties
	if err := validateProperties(props); err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   fmt.Sprintf("Invalid properties: %v", err),
			},
		}, nil
	}

	// Create Cloudflare client
	client, err := createCloudflareClient(config)
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   fmt.Sprintf("Failed to create Cloudflare client: %v", err),
			},
		}, nil
	}

	// Create the DNS record
	rc := cloudflare.ZoneIdentifier(config.ZoneID)
	record, err := client.CreateDNSRecord(ctx, rc, propsToCreateParams(props))
	if err != nil {
		return &resource.CreateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationCreate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   fmt.Sprintf("Failed to create DNS record: %v", err),
			},
		}, nil
	}

	return &resource.CreateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCreate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        record.ID,
		},
	}, nil
}

// Read retrieves the current state of a resource.
func (p *Plugin) Read(ctx context.Context, req *resource.ReadRequest) (*resource.ReadResult, error) {
	// Parse target config
	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		return &resource.ReadResult{
			ResourceType: req.ResourceType,
			ErrorCode:    resource.OperationErrorCodeInvalidRequest,
		}, nil
	}

	// Create Cloudflare client
	client, err := createCloudflareClient(config)
	if err != nil {
		return &resource.ReadResult{
			ResourceType: req.ResourceType,
			ErrorCode:    resource.OperationErrorCodeInternalFailure,
		}, nil
	}

	// Get the zone name for stripping from FQDN
	zoneName, err := getZoneName(ctx, client, config.ZoneID)
	if err != nil {
		return &resource.ReadResult{
			ResourceType: req.ResourceType,
			ErrorCode:    resource.OperationErrorCodeInternalFailure,
		}, nil
	}

	// Get the DNS record
	rc := cloudflare.ZoneIdentifier(config.ZoneID)
	record, err := client.GetDNSRecord(ctx, rc, req.NativeID)
	if err != nil {
		// Check if record not found
		if isNotFoundError(err) {
			return &resource.ReadResult{
				ResourceType: req.ResourceType,
				ErrorCode:    resource.OperationErrorCodeNotFound,
			}, nil
		}
		return &resource.ReadResult{
			ResourceType: req.ResourceType,
			ErrorCode:    resource.OperationErrorCodeInternalFailure,
		}, nil
	}

	// Convert to properties
	props := recordToProperties(record, zoneName)
	propsJSON, err := propertiesToJSON(props)
	if err != nil {
		return &resource.ReadResult{
			ResourceType: req.ResourceType,
			ErrorCode:    resource.OperationErrorCodeInternalFailure,
		}, nil
	}

	return &resource.ReadResult{
		ResourceType: req.ResourceType,
		Properties:   propsJSON,
	}, nil
}

// Update modifies an existing resource.
func (p *Plugin) Update(ctx context.Context, req *resource.UpdateRequest) (*resource.UpdateResult, error) {
	// Parse target config
	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   fmt.Sprintf("Invalid target config: %v", err),
			},
		}, nil
	}

	// Parse desired properties
	props, err := parseProperties(req.DesiredProperties)
	if err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   fmt.Sprintf("Invalid properties: %v", err),
			},
		}, nil
	}

	// Validate properties
	if err := validateProperties(props); err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   fmt.Sprintf("Invalid properties: %v", err),
			},
		}, nil
	}

	// Create Cloudflare client
	client, err := createCloudflareClient(config)
	if err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   fmt.Sprintf("Failed to create Cloudflare client: %v", err),
			},
		}, nil
	}

	// Update the DNS record
	rc := cloudflare.ZoneIdentifier(config.ZoneID)
	_, err = client.UpdateDNSRecord(ctx, rc, propsToUpdateParams(props, req.NativeID))
	if err != nil {
		return &resource.UpdateResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationUpdate,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   fmt.Sprintf("Failed to update DNS record: %v", err),
			},
		}, nil
	}

	return &resource.UpdateResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationUpdate,
			OperationStatus: resource.OperationStatusSuccess,
			NativeID:        req.NativeID,
		},
	}, nil
}

// Delete removes a resource.
func (p *Plugin) Delete(ctx context.Context, req *resource.DeleteRequest) (*resource.DeleteResult, error) {
	// Parse target config
	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInvalidRequest,
				StatusMessage:   fmt.Sprintf("Invalid target config: %v", err),
			},
		}, nil
	}

	// Create Cloudflare client
	client, err := createCloudflareClient(config)
	if err != nil {
		return &resource.DeleteResult{
			ProgressResult: &resource.ProgressResult{
				Operation:       resource.OperationDelete,
				OperationStatus: resource.OperationStatusFailure,
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   fmt.Sprintf("Failed to create Cloudflare client: %v", err),
			},
		}, nil
	}

	// Delete the DNS record
	rc := cloudflare.ZoneIdentifier(config.ZoneID)
	err = client.DeleteDNSRecord(ctx, rc, req.NativeID)
	if err != nil {
		// Check if record not found - consider it already deleted
		if isNotFoundError(err) {
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
				ErrorCode:       resource.OperationErrorCodeInternalFailure,
				StatusMessage:   fmt.Sprintf("Failed to delete DNS record: %v", err),
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
// Called when Create/Update/Delete return InProgress status.
// Cloudflare DNS operations are synchronous, so this always returns success.
func (p *Plugin) Status(ctx context.Context, req *resource.StatusRequest) (*resource.StatusResult, error) {
	// Cloudflare DNS operations are synchronous - they complete immediately
	return &resource.StatusResult{
		ProgressResult: &resource.ProgressResult{
			Operation:       resource.OperationCheckStatus,
			OperationStatus: resource.OperationStatusSuccess,
		},
	}, nil
}

// List returns all resource identifiers of a given type.
// Called during discovery to find unmanaged resources.
func (p *Plugin) List(ctx context.Context, req *resource.ListRequest) (*resource.ListResult, error) {
	// Parse target config
	config, err := parseTargetConfig(req.TargetConfig)
	if err != nil {
		return &resource.ListResult{
			NativeIDs:     []string{},
			NextPageToken: nil,
		}, nil
	}

	// Create Cloudflare client
	client, err := createCloudflareClient(config)
	if err != nil {
		return &resource.ListResult{
			NativeIDs:     []string{},
			NextPageToken: nil,
		}, nil
	}

	// Set up pagination
	pageSize := 100 // Default page size
	if req.PageSize > 0 {
		pageSize = int(req.PageSize)
	}

	page := 1
	if req.PageToken != nil && *req.PageToken != "" {
		// Parse page number from token (ignore errors, default to page 1)
		_, _ = fmt.Sscanf(*req.PageToken, "%d", &page)
	}

	// List DNS records
	rc := cloudflare.ZoneIdentifier(config.ZoneID)
	records, resultInfo, err := client.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{
		ResultInfo: cloudflare.ResultInfo{
			Page:    page,
			PerPage: pageSize,
		},
	})
	if err != nil {
		return &resource.ListResult{
			NativeIDs:     []string{},
			NextPageToken: nil,
		}, nil
	}

	// Extract record IDs
	nativeIDs := make([]string, 0, len(records))
	for _, record := range records {
		nativeIDs = append(nativeIDs, record.ID)
	}

	// Determine if there are more pages
	var nextPageToken *string
	if resultInfo != nil && resultInfo.Page < resultInfo.TotalPages {
		token := fmt.Sprintf("%d", resultInfo.Page+1)
		nextPageToken = &token
	}

	return &resource.ListResult{
		NativeIDs:     nativeIDs,
		NextPageToken: nextPageToken,
	}, nil
}
