#!/bin/bash
# © 2025 Platform Engineering Labs Inc.
# SPDX-License-Identifier: BSD-3-Clause
#
# Clean Environment Hook
# ======================
# This script is called before AND after conformance tests to clean up
# test resources in your Cloudflare zone.
#
# Purpose:
# - Before tests: Remove orphaned resources from previous failed runs
# - After tests: Clean up resources created during the test run
#
# The script should be idempotent - safe to run multiple times.

set -euo pipefail

# Prefix used for test resources - should match what conformance tests create
TEST_PREFIX="${TEST_PREFIX:-formae-test-}"

# Check for required environment variables
if [[ -z "${CLOUDFLARE_API_TOKEN:-}" ]]; then
    echo "clean-environment.sh: CLOUDFLARE_API_TOKEN not set, skipping cleanup"
    exit 0
fi

if [[ -z "${CLOUDFLARE_ZONE_ID:-}" ]]; then
    echo "clean-environment.sh: CLOUDFLARE_ZONE_ID not set, skipping cleanup"
    exit 0
fi

echo "clean-environment.sh: Cleaning DNS records with prefix '${TEST_PREFIX}' in zone ${CLOUDFLARE_ZONE_ID}"

# List all DNS records matching the test prefix and delete them
# Using the Cloudflare API directly
API_BASE="https://api.cloudflare.com/client/v4"

# Get list of DNS records
response=$(curl -s -X GET \
    "${API_BASE}/zones/${CLOUDFLARE_ZONE_ID}/dns_records?name=contains:${TEST_PREFIX}" \
    -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
    -H "Content-Type: application/json")

# Check if the request was successful
success=$(echo "$response" | jq -r '.success // false')
if [[ "$success" != "true" ]]; then
    echo "clean-environment.sh: Failed to list DNS records"
    echo "$response" | jq .
    exit 0  # Don't fail the build for cleanup issues
fi

# Extract record IDs
record_ids=$(echo "$response" | jq -r '.result[].id // empty')

if [[ -z "$record_ids" ]]; then
    echo "clean-environment.sh: No test DNS records found to clean up"
    exit 0
fi

# Delete each record
deleted_count=0
for record_id in $record_ids; do
    echo "clean-environment.sh: Deleting DNS record ${record_id}"
    delete_response=$(curl -s -X DELETE \
        "${API_BASE}/zones/${CLOUDFLARE_ZONE_ID}/dns_records/${record_id}" \
        -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
        -H "Content-Type: application/json")

    delete_success=$(echo "$delete_response" | jq -r '.success // false')
    if [[ "$delete_success" == "true" ]]; then
        ((deleted_count++))
    else
        echo "clean-environment.sh: Warning - failed to delete record ${record_id}"
    fi
done

echo "clean-environment.sh: Cleanup complete - deleted ${deleted_count} record(s)"
