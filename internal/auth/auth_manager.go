package auth

import (
	"fmt"

	"github.com/sabarim/kitedata/internal/config"
	kiteconnect "github.com/zerodha/gokiteconnect/v4"
)

// AuthManager handles authentication with the broker API
type AuthManager struct {
	config     *config.Config
	kite       *kiteconnect.Client
	authClient *AuthClient
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config *config.Config) *AuthManager {
	// Create the auth client if auth service URL is provided
	var authClient *AuthClient
	if config.Auth.AuthServiceURL != "" {
		authClient = NewAuthClient(config.Auth.AuthServiceURL, config.Auth.AuthServiceAPIKey)
	}

	// Initialize the KiteConnect client with empty API key for now
	// We'll set it after getting credentials
	kite := kiteconnect.New("")

	return &AuthManager{
		config:     config,
		kite:       kite,
		authClient: authClient,
	}
}

// Login generates an access token for the broker API
func (am *AuthManager) Login() (AuthCredentialsResult, error) {
	var creds AuthCredentialsResult
	
	// First try to get credentials from auth service if configured
	if am.authClient != nil && am.config.Auth.BrokerName != "" {
		// Try to get credentials from the auth service
		fmt.Println("==== AUTH SERVICE DEBUG ====")
		fmt.Printf("AuthServiceURL: %s\n", am.config.Auth.AuthServiceURL)
		fmt.Printf("AuthServiceAPIKey: %s\n", am.config.Auth.AuthServiceAPIKey)
		fmt.Printf("BrokerName: %s\n", am.config.Auth.BrokerName)
		
		// Make the request to the auth service with detailed logging
		fmt.Println("Calling auth service...")
		credentials, err := am.authClient.GetBrokerCredentials(am.config.Auth.BrokerName)
		
		if err != nil {
			// Auth service returned an error
			fmt.Printf("AUTH SERVICE ERROR: %v\n", err)
		} else {
			// Check if credentials are valid
			fmt.Printf("Received credentials - API Key: %s\n", credentials.ApiKey)
			fmt.Printf("Session Token present: %v\n", credentials.SessionToken != "")
			fmt.Printf("Is Active: %v\n", credentials.IsActive)
			
			if credentials.ApiKey != "" && credentials.SessionToken != "" {
				// Set up the KiteConnect client with auth service credentials
				fmt.Printf("Setting up KiteConnect client with API key: %s\n", credentials.ApiKey)
				am.kite = kiteconnect.New(credentials.ApiKey)
				
				// Set the access token
				fmt.Println("Setting the access token...")
				am.kite.SetAccessToken(credentials.SessionToken)
				
				// Return the credentials
				fmt.Println("AUTH SERVICE AUTHENTICATION SUCCESSFUL")
				fmt.Println("==== END AUTH SERVICE DEBUG ====")
				
				creds = AuthCredentialsResult{
					ApiKey:       credentials.ApiKey,
					SessionToken: credentials.SessionToken,
				}
				return creds, nil
			} else {
				fmt.Println("AUTH SERVICE RETURNED INCOMPLETE CREDENTIALS")
			}
		}
		fmt.Println("==== END AUTH SERVICE DEBUG ====")
	}

	// If auth service failed or wasn't configured, try direct credentials
	if am.config.Auth.ApiKey != "" && am.config.Auth.SessionToken != "" {
		fmt.Println("Falling back to direct credentials...")
		
		// Set up the KiteConnect client with direct credentials
		am.kite = kiteconnect.New(am.config.Auth.ApiKey)
		am.kite.SetAccessToken(am.config.Auth.SessionToken)
		
		creds = AuthCredentialsResult{
			ApiKey:       am.config.Auth.ApiKey,
			SessionToken: am.config.Auth.SessionToken,
		}
		return creds, nil
	}

	// If no credentials were set, return an error
	return AuthCredentialsResult{}, fmt.Errorf("no valid credentials available; please set API key and session token in config or ensure auth_service is working")
}

// GetClient returns the authenticated KiteConnect client
func (am *AuthManager) GetClient() (*kiteconnect.Client, error) {
	// ALWAYS force reauthentication for debugging
	fmt.Println("Forcing authentication...")
	_, err := am.Login()
	if err != nil {
		return nil, fmt.Errorf("failed to login before getting client: %w", err)
	}
	
	// Double-check that we have a valid client
	if am.kite == nil {
		return nil, fmt.Errorf("KiteConnect client was not properly initialized")
	}
	
	return am.kite, nil
}