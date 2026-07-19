// Package idempotency provides short-lived Redis-backed HTTP duplicate protection.
//
// It stores a request fingerprint and first response for a bounded retry window.
package idempotency
