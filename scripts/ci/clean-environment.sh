#!/bin/bash
# © 2025 Platform Engineering Labs Inc.
# SPDX-License-Identifier: BSD-3-Clause
#
# Clean Environment Hook
# ======================
# This script is called before AND after conformance tests to clean up
# test resources in your cloud environment.
#
# Purpose:
# - Before tests: Remove orphaned resources from previous failed runs
# - After tests: Clean up resources created during the test run
#
# The script should be idempotent - safe to run multiple times.
# It should delete all resources matching the test resource prefix.
#
# Required environment variables:
# - CLOUDFLARE_API_TOKEN: API token with DNS edit permissions
# - CLOUDFLARE_ZONE_ID: Zone ID for the DNS zone

set -euo pipefail

# Prefix used for test resources - should match what conformance tests create
TEST_PREFIX="${TEST_PREFIX:-formae-}"

# Check required environment variables
if [[ -z "${CLOUDFLARE_API_TOKEN:-}" ]]; then
    echo "clean-environment.sh: CLOUDFLARE_API_TOKEN not set, skipping cleanup"
    exit 0
fi

if [[ -z "${CLOUDFLARE_ZONE_ID:-}" ]]; then
    echo "clean-environment.sh: CLOUDFLARE_ZONE_ID not set, skipping cleanup"
    exit 0
fi

echo "clean-environment.sh: Cleaning DNS records with prefix '${TEST_PREFIX}'"

# Cloudflare API base URL
API_BASE="https://api.cloudflare.com/client/v4"

# List all DNS records in the zone
echo "Fetching DNS records..."
RECORDS=$(curl -s -X GET "${API_BASE}/zones/${CLOUDFLARE_ZONE_ID}/dns_records" \
    -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
    -H "Content-Type: application/json")

# Check for API errors
if ! echo "${RECORDS}" | jq -e '.success' > /dev/null 2>&1; then
    echo "clean-environment.sh: Failed to list DNS records"
    echo "${RECORDS}" | jq -r '.errors[]?.message // "Unknown error"'
    exit 1
fi

# Find records matching the test prefix
RECORD_IDS=$(echo "${RECORDS}" | jq -r ".result[] | select(.name | startswith(\"${TEST_PREFIX}\")) | .id")

# Delete matching records
if [[ -z "${RECORD_IDS}" ]]; then
    echo "clean-environment.sh: No DNS records found with prefix '${TEST_PREFIX}'"
else
    echo "clean-environment.sh: Found records to delete:"
    echo "${RECORDS}" | jq -r ".result[] | select(.name | startswith(\"${TEST_PREFIX}\")) | \"  - \(.name) (\(.type)): \(.id)\""

    for RECORD_ID in ${RECORD_IDS}; do
        echo "Deleting record ${RECORD_ID}..."
        DELETE_RESULT=$(curl -s -X DELETE "${API_BASE}/zones/${CLOUDFLARE_ZONE_ID}/dns_records/${RECORD_ID}" \
            -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
            -H "Content-Type: application/json")

        if echo "${DELETE_RESULT}" | jq -e '.success' > /dev/null 2>&1; then
            echo "  Deleted successfully"
        else
            # Don't fail on delete errors (resource may already be gone)
            echo "  Warning: Delete may have failed"
            echo "${DELETE_RESULT}" | jq -r '.errors[]?.message // "Unknown error"' || true
        fi
    done
fi

echo "clean-environment.sh: Cleanup complete"
