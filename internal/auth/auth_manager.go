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
	var authMethodUsed string
	
	fmt.Println("==== Authentication Debug ====")
	fmt.Printf("Config values - Auth Service URL: %s, API Key: %s, Session Token: %v\n", 
		am.config.Auth.AuthServiceURL, 
		am.config.Auth.ApiKey, 
		am.config.Auth.SessionToken != "")
	
	// First try to get credentials from auth service if configured
	if am.authClient != nil && am.config.Auth.BrokerName != "" {
		fmt.Println("Attempting to get credentials from auth service...")
		credentials, err := am.authClient.GetBrokerCredentials(am.config.Auth.BrokerName)
		if err != nil {
			fmt.Printf("Failed to get credentials from auth service: %v. Will try direct credentials.\n", err)
		} else {
			// Validate credentials from auth service
			if credentials.ApiKey != "" && credentials.SessionToken != "" {
				fmt.Println("Using credentials from auth service")
				authMethodUsed = "auth_service"
				
				// Set up the KiteConnect client with auth service credentials
				fmt.Printf("Setting up KiteConnect client with API key: %s\n", credentials.ApiKey)
				am.kite = kiteconnect.New(credentials.ApiKey)
				fmt.Printf("Setting access token: %s...\n", credentials.SessionToken[:5])
				am.kite.SetAccessToken(credentials.SessionToken)
				
				creds = AuthCredentialsResult{
					ApiKey:       credentials.ApiKey,
					SessionToken: credentials.SessionToken,
				}
			} else {
				fmt.Println("Auth service returned incomplete credentials (missing API key or session token)")
			}
		}
	}

	// If auth service failed or wasn't configured, try direct credentials
	if authMethodUsed == "" && am.config.Auth.ApiKey != "" && am.config.Auth.SessionToken != "" {
		fmt.Println("Using direct credentials from config")
		authMethodUsed = "direct_config"
		
		// Set up the KiteConnect client with direct credentials
		fmt.Printf("Setting up KiteConnect client with API key: %s\n", am.config.Auth.ApiKey)
		am.kite = kiteconnect.New(am.config.Auth.ApiKey)
		
		fmt.Printf("Setting access token from config: %s...\n", am.config.Auth.SessionToken[:5])
		am.kite.SetAccessToken(am.config.Auth.SessionToken)
		
		creds = AuthCredentialsResult{
			ApiKey:       am.config.Auth.ApiKey,
			SessionToken: am.config.Auth.SessionToken,
		}
	}

	// If no credentials were set, return an error
	if authMethodUsed == "" {
		fmt.Println("No valid credentials available!")
		return AuthCredentialsResult{}, fmt.Errorf("no valid credentials available; please set API key and session token in config or ensure auth_service is working")
	}
	
	// Verify the client was properly initialized
	if am.kite == nil {
		fmt.Println("Failed to initialize KiteConnect client!")
		return AuthCredentialsResult{}, fmt.Errorf("failed to initialize KiteConnect client")
	}
	
	fmt.Printf("Successfully authenticated using %s\n", authMethodUsed)
	fmt.Println("==== End Authentication Debug ====")
	return creds, nil
}

// GetClient returns the authenticated KiteConnect client
func (am *AuthManager) GetClient() (*kiteconnect.Client, error) {
	fmt.Println("GetClient called - forcing authentication...")
	
	// Force re-login every time for debugging
	creds, err := am.Login()
	if err != nil {
		fmt.Printf("Login failed with error: %v\n", err)
		return nil, fmt.Errorf("failed to login before getting client: %w", err)
	}
	
	// Double-check that we have a valid client
	if am.kite == nil {
		fmt.Println("KiteConnect client is nil after Login()")
		return nil, fmt.Errorf("KiteConnect client was not properly initialized")
	}
	
	// Debug logging
	credSource := "auth service"
	if creds.ApiKey == am.config.Auth.ApiKey {
		credSource = "config"
	}
	fmt.Printf("Authentication successful with credentials from %s\n", credSource)
	
	return am.kite, nil
}