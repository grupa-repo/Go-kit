package secret_test

import (
	"testing"

	"github.com/grupa-repo/go-kit/secret"
)

// These values are the contract. If one of them changes, fingerprints printed
// by one release stop matching those printed by another.
func TestFingerprintIsStable(t *testing.T) {
	cases := map[string]string{
		"":         "unset",
		"a":        "ca978112",
		"password": "5e884898",
	}
	for in, want := range cases {
		if got := secret.Fingerprint(in); got != want {
			t.Errorf("Fingerprint(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFingerprintDistinguishesSecrets(t *testing.T) {
	if secret.Fingerprint("one") == secret.Fingerprint("two") {
		t.Error("different secrets produced the same fingerprint")
	}
}
