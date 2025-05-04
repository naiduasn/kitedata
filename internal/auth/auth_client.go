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
	fmt.Printf("Creating auth client with URL: %s and API key: %s\n", 
		authServiceURL, 
		apiKey)
	
	// Ensure URL is properly formatted and ends with /
	if !strings.HasSuffix(authServiceURL, "/") {
		authServiceURL = authServiceURL + "/"
		fmt.Printf("Added trailing slash to URL: %s\n", authServiceURL)
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
	// Fix any wrong URL formatting
	serviceURL := ac.authServiceURL
	if strings.HasSuffix(serviceURL, "/") {
		serviceURL = strings.TrimSuffix(serviceURL, "/")
	}
	
	// Construct request URL with service=true parameter to indicate service-to-service call
	url := fmt.Sprintf("%s/auth/%s/credentials?service=true", serviceURL, broker)
	
	fmt.Printf("Full request URL: %s\n", url)
	fmt.Println("Request headers:")
	fmt.Printf("  Content-Type: application/json\n")
	fmt.Printf("  X-API-Key: %s\n", ac.apiKey)

	// Create a new request to set headers
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("ERROR creating request: %v\n", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", ac.apiKey)

	// Make the request
	fmt.Println("Making HTTP request to auth service...")
	resp, err := ac.httpClient.Do(req)
	if err != nil {
		fmt.Printf("ERROR connecting to auth service: %v\n", err)
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	fmt.Printf("Response status code: %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("ERROR response body: %s\n", string(bodyBytes))
		return nil, fmt.Errorf("auth service returned error status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	fmt.Println("Received 200 OK response from auth service")

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth service response: %w", err)
	}
	
	// Debug output
	fmt.Printf("Auth service response: %s\n", string(bodyBytes))
	
	// Parse the response (create a new reader from the bytes since we already read the body)
	var credentials AuthCredentials
	if err := json.NewDecoder(io.NopCloser(bytes.NewReader(bodyBytes))).Decode(&credentials); err != nil {
		return nil, fmt.Errorf("failed to parse auth service response: %w", err)
	}

	// Debug output for credentials
	fmt.Printf("Parsed credentials - API Key: %s, Session Token: %s, Is Active: %v\n", 
		credentials.ApiKey, 
		credentials.SessionToken[:5]+"...", // Only show first 5 chars for security
		credentials.IsActive)

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