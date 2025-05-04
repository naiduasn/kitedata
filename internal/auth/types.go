package auth

// AuthCredentials represents the credentials received from auth_service
type AuthCredentials struct {
	ID           int    `json:"id"`
	Broker       string `json:"broker"`
	ApiKey       string `json:"api_key"`
	ApiSecret    string `json:"api_secret"`
	SessionToken string `json:"session_token"`
	IsActive     bool   `json:"is_active"`
	AccountID    string `json:"account_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// AuthCredentialsResult holds the result of a login attempt
type AuthCredentialsResult struct {
	ApiKey       string
	SessionToken string
}