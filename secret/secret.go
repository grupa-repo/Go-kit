// Package secret handles shared secrets without ever printing one.
package secret

import (
	"crypto/sha256"
	"encoding/hex"
)

// Unset is what Fingerprint returns for the empty string.
const Unset = "unset"

// Fingerprint returns a short, non-reversible identifier for a secret, so two
// services can be checked for configuration drift without either of them
// printing the value. The same input always yields the same fingerprint.
//
// It is the first 8 hex characters of the SHA-256. The empty string yields
// Unset: otherwise an unset secret would print the hash of "", which reads
// like a configured value.
//
// The algorithm is a contract between everything that prints a fingerprint and
// everything that compares one. Changing it makes old and new output disagree
// for identical secrets, so it must not change.
func Fingerprint(value string) string {
	if value == "" {
		return Unset
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:8]
}
