package models

type OAuthApp struct {
	ID               string `orm:"primary-key"`
	ClientID         string `orm:"unique"`
	ClientSecretHash string
	Name             string
	Description      string `orm:"data-only"`
	TenantID         string
	AllowedScopes    []string
	CreatedAt        int64
	UpdatedAt        int64
}

func (OAuthApp) TableName() string {
	return "oauth_apps"
}

type OAuthToken struct {
	ID        string `orm:"primary-key"`
	TokenHash string `orm:"unique-compound:0"`
	ClientID  string `orm:"unique-compound:0"`
	Scopes    []string
	TTL       int64 `orm:"ttl"`
}

func (OAuthToken) TableName() string {
	return "oauth_tokens"
}

type CreateOAuthAppRequest struct {
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	AllowedScopes []string `json:"allowedScopes" binding:"required"`
}

type OAuthAppCredentialsResponse struct {
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	Name         string   `json:"name"`
	Scopes       []string `json:"scopes"`
}

type OAuthTokenRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type" binding:"required"` // must be "client_credentials"
	ClientID     string `json:"client_id" form:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" form:"client_secret" binding:"required"`
	Scope        string `json:"scope" form:"scope"`
}

type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"` // usually "Bearer"
	ExpiresIn   int64  `json:"expires_in"` // expiration in seconds
	Scope       string `json:"scope"`
}
