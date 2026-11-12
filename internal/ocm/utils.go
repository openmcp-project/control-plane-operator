package ocm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RegistryCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CatalogResponse struct {
	Repositories []string `json:"repositories"`
}

/*
 * @Description: GetRepositoriesInOCIRegistry returns the repositories in the OCI registry
 * @param ociRegistry string - the OCI registry URL without paths
 * @param creds RegistryCredentials - the credentials to access the OCI registry
 * @param prefixFilter string - the prefix to filter the repositories
 * @param protocol string - the protocol to use to access the OCI registry, http or https
 * @return []string - the list of repositories in the OCI registry
 */
func GetRepositoriesInOCIRegistry(ociRegistry string, creds RegistryCredentials, prefixFilter string, protocol string) ([]string, error) {
	if protocol == "" {
		protocol = "https"
	}

	url := protocol + "://" + ociRegistry + "/v2/_catalog"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(creds.Username, creds.Password)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", url, err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Failed to close body: %v\n", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("catalog request to %s returned status %d: %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		return nil, err
	}

	var data CatalogResponse
	err = json.Unmarshal([]byte(buf.String()), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse catalog response from %s: %w", url, err)
	}

	repositories := data.Repositories

	var filteredRepositories []string
	for _, repository := range repositories {
		if prefixFilter == "" || strings.HasPrefix(repository, prefixFilter) {
			filteredRepositories = append(filteredRepositories, strings.TrimPrefix(repository, prefixFilter))
		}
	}

	return filteredRepositories, nil
}
