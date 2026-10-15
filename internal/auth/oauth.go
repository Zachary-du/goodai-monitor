package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// GoogleAuthEndpoint is the standard Google OAuth 2.0 authorization URL.
	GoogleAuthEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"

	// GoogleTokenEndpoint is the Google OAuth 2.0 token exchange endpoint.
	GoogleTokenEndpoint = "https://oauth2.googleapis.com/token"

	// GoogleUserInfoEndpoint is the OpenID Connect userinfo endpoint.
	GoogleUserInfoEndpoint = "https://www.googleapis.com/oauth2/v3/userinfo"

	// OAuthStateCookieName is the cookie used to store the CSRF state token.
	OAuthStateCookieName = "goodai_oauth_state"

	// OAuthStateTTL is the maximum validity period for an OAuth state token.
	OAuthStateTTL = 10 * time.Minute

	// OAuthStateBytes defines the entropy of the random state token (256 bits).
	OAuthStateBytes = 32
)

var (
	// ErrOAuthStateMismatch indicates that the returned state does not match the cookie.
	ErrOAuthStateMismatch = errors.New("invalid or expired oauth state parameter")

	// ErrOAuthEmailNotVerified indicates that the Google account email is not verified.
	ErrOAuthEmailNotVerified = errors.New("google account email is not verified")

	// ErrOAuthCodeExchangeFailed indicates a token exchange error with Google.
	ErrOAuthCodeExchangeFailed = errors.New("failed to exchange authorization code with google")

	// ErrOAuthProfileFetchFailed indicates a failure retrieving the user profile.
	ErrOAuthProfileFetchFailed = errors.New("failed to retrieve google user profile")

	defaultOAuthClient = &http.Client{
		Timeout: 10 * time.Second,
	}
)

// GoogleTokenResponse represents the payload returned by Google's token endpoint.
type GoogleTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token,omitempty"`
}

// GoogleUserInfo represents the user profile data returned by Google's userinfo endpoint.
type GoogleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// GenerateOAuthState creates a cryptographically secure 256-bit random hex string.
func GenerateOAuthState() (string, error) {
	buf := make([]byte, OAuthStateBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate oauth state: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// NewOAuthStateCookie builds an HttpOnly, SameSite=Lax cookie containing the state token.
func NewOAuthStateCookie(state string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    state,
		Path:     "/",
		Expires:  time.Now().UTC().Add(OAuthStateTTL),
		MaxAge:   int(OAuthStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// ClearOAuthStateCookie creates an expired cookie to purge the state token from the browser.
func ClearOAuthStateCookie() *http.Cookie {
	return &http.Cookie{
		Name:     OAuthStateCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// BuildGoogleAuthURL constructs the authorization redirect URL with all required query parameters.
func BuildGoogleAuthURL(clientID, redirectURI, state string) string {
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "openid email profile")
	params.Set("state", state)
	params.Set("access_type", "online")
	params.Set("prompt", "select_account")

	return GoogleAuthEndpoint + "?" + params.Encode()
}

// ExchangeGoogleCode trades an authorization code for Google access and ID tokens.
func ExchangeGoogleCode(ctx context.Context, clientID, clientSecret, redirectURI, code string) (*GoogleTokenResponse, error) {
	return ExchangeGoogleCodeWithClient(ctx, defaultOAuthClient, GoogleTokenEndpoint, clientID, clientSecret, redirectURI, code)
}

// ExchangeGoogleCodeWithClient allows providing a custom http.Client and endpoint (for unit testing).
func ExchangeGoogleCodeWithClient(ctx context.Context, client *http.Client, endpoint, clientID, clientSecret, redirectURI, code string) (*GoogleTokenResponse, error) {
	if client == nil {
		client = defaultOAuthClient
	}

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthCodeExchangeFailed, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrOAuthCodeExchangeFailed, resp.StatusCode, string(bodyBytes))
	}

	var tokenResp GoogleTokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, errors.New("empty access token in google token response")
	}

	return &tokenResp, nil
}

// FetchGoogleUserInfo retrieves the verified profile for the given access token.
func FetchGoogleUserInfo(ctx context.Context, accessToken string) (*GoogleUserInfo, error) {
	return FetchGoogleUserInfoWithClient(ctx, defaultOAuthClient, GoogleUserInfoEndpoint, accessToken)
}

// FetchGoogleUserInfoWithClient allows providing a custom http.Client and endpoint (for unit testing).
func FetchGoogleUserInfoWithClient(ctx context.Context, client *http.Client, endpoint, accessToken string) (*GoogleUserInfo, error) {
	if client == nil {
		client = defaultOAuthClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthProfileFetchFailed, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("failed to read userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrOAuthProfileFetchFailed, resp.StatusCode, string(bodyBytes))
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(bodyBytes, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	if userInfo.Sub == "" || userInfo.Email == "" {
		return nil, errors.New("incomplete user profile: missing sub or email")
	}

	if !userInfo.EmailVerified {
		return nil, ErrOAuthEmailNotVerified
	}

	return &userInfo, nil
}
