package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AuthClient is a client for interacting with the auth_service
type AuthClient struct {
	authServiceURL string
	apiKey         string
	httpClient     *http.Client
}

// NewAuthClient creates a new auth client
func NewAuthClient(authServiceURL string, apiKey string) *AuthClient {
	// Ensure URL is properly formatted and ends with /
	if !strings.HasSuffix(authServiceURL, "/") {
		authServiceURL = authServiceURL + "/"
	}
	
	return &AuthClient{
		authServiceURL: authServiceURL,
		apiKey:         apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetBrokerCredentials fetches broker credentials from the auth_service
func (ac *AuthClient) GetBrokerCredentials(broker string) (*AuthCredentials, error) {
	// Construct request URL with service=true parameter
	url := fmt.Sprintf("%sauth/%s/credentials?service=true", ac.authServiceURL, broker)

	// Create a new request to set headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", ac.apiKey)

	// Make the request
	resp, err := ac.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("auth service returned error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth service response: %w", err)
	}
	
	// Parse the response (create a new reader from the bytes since we already read the body)
	var credentials AuthCredentials
	if err := json.NewDecoder(io.NopCloser(bytes.NewReader(bodyBytes))).Decode(&credentials); err != nil {
		return nil, fmt.Errorf("failed to parse auth service response: %w", err)
	}

	// Check if credentials are valid
	if credentials.ApiKey == "" || credentials.ApiSecret == "" {
		return nil, fmt.Errorf("received incomplete credentials from auth service")
	}

	// Check if the session token exists
	if credentials.SessionToken == "" {
		return nil, fmt.Errorf("received credentials without session token from auth service")
	}

	// Check if the credentials are active
	if !credentials.IsActive {
		return nil, fmt.Errorf("received inactive credentials from auth service")
	}

	return &credentials, nil
}