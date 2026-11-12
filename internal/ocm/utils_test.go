package ocm

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Non-200 responses must surface the status code and body, not a cryptic JSON
// unmarshal error. Regression guard for the masked "can't get repositories in
// OCI registry" failure.
func TestGetRepositoriesInOCIRegistry_NonOKStatusReturnsDetailedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("authentication required"))
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	_, err := GetRepositoriesInOCIRegistry(host, RegistryCredentials{Username: "u", Password: "p"}, "", "http")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
	assert.Contains(t, err.Error(), "authentication required")
}

// Happy path: valid catalog JSON is parsed and filtered by prefix.
func TestGetRepositoriesInOCIRegistry_OKParsesAndFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"repositories":["foo/a","bar/b","foo/c"]}`))
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	repos, err := GetRepositoriesInOCIRegistry(host, RegistryCredentials{Username: "u", Password: "p"}, "foo/", "http")

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"a", "c"}, repos)
}
