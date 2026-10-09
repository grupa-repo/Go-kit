package appenv_test

import (
	"testing"

	"github.com/grupa-repo/go-kit/appenv"
)

func TestIsDeployed(t *testing.T) {
	cases := []struct {
		name string
		want bool
		why  string
	}{
		{"", false, "unset is a local run"},
		{"local", false, ""},
		{"test", false, ""},
		{"integrationtest", false, ""},
		{"development", true, "a deployed app can run under this name"},
		{"qa", true, ""},
		{"staging", true, ""},
		{"production", true, ""},
		{"prod", true, ""},
		{"developmnet", true, "a typo on a server must not read as local"},
		{"Local", true, "names are case-sensitive; an unexpected spelling fails loudly"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := appenv.IsDeployed(c.name); got != c.want {
				t.Errorf("IsDeployed(%q) = %v, want %v (%s)", c.name, got, c.want, c.why)
			}
		})
	}
}

func TestDocsEnabled(t *testing.T) {
	cases := []struct {
		name string
		want bool
		why  string
	}{
		{"", true, "unset is a local run"},
		{"local", true, ""},
		{"test", true, ""},
		{"integrationtest", true, ""},
		{"development", true, ""},
		{"qa", true, ""},
		{"staging", false, ""},
		{"production", false, ""},
		{"prod", false, ""},
		{"developmnet", false, "an unrecognised name serves nothing"},
		{"QA", false, "names are case-sensitive; an unexpected spelling serves nothing"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := appenv.DocsEnabled(c.name); got != c.want {
				t.Errorf("DocsEnabled(%q) = %v, want %v (%s)", c.name, got, c.want, c.why)
			}
		})
	}
}
