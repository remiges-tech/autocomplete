package autocomplete

import "errors"

// Sentinel errors for common validation failures.

var (
	// ErrProviderNotFound is returned when a provider is not registered.
	// Usually means you forgot to import the provider package with an underscore.
	ErrProviderNotFound = errors.New("autocomplete provider not found")

	// ErrQueryTooShort is returned when the query is shorter than MinPrefixLength.
	ErrQueryTooShort = errors.New("query too short")

	// ErrLimitExceeded is returned when the requested limit exceeds MaxLimit.
	ErrLimitExceeded = errors.New("limit exceeded")

	// ErrEmptyID is returned when an empty ID is provided to Index or Delete.
	ErrEmptyID = errors.New("empty ID")

	// ErrEmptyText is returned when empty text is provided to Index.
	ErrEmptyText = errors.New("empty text")

	// ErrEmptyDisplay is returned when empty display text is provided to Index.
	ErrEmptyDisplay = errors.New("empty display")
)
