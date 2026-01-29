// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: BSD-3-Clause

package cloudflare

import (
	"errors"

	cf "github.com/cloudflare/cloudflare-go/v4"
	"github.com/platform-engineering-labs/formae/pkg/plugin/resource"
)

// MapErrorCode maps a Cloudflare API error to a Formae operation error code.
func MapErrorCode(err error) resource.OperationErrorCode {
	if err == nil {
		return ""
	}

	var apiErr *cf.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 400:
			return resource.OperationErrorCodeInvalidRequest
		case 401:
			return resource.OperationErrorCodeInvalidCredentials
		case 403:
			return resource.OperationErrorCodeAccessDenied
		case 404:
			return resource.OperationErrorCodeNotFound
		case 429:
			return resource.OperationErrorCodeThrottling
		case 500, 502, 503, 504:
			return resource.OperationErrorCodeServiceInternalError
		}
	}

	return resource.OperationErrorCodeInternalFailure
}

// IsNotFound checks if an error is a 404 not found error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *cf.Error
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}

	return false
}

// IsRateLimited checks if an error is a rate limiting error.
func IsRateLimited(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *cf.Error
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 429
	}

	return false
}
