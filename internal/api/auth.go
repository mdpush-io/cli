package api

import "fmt"

// --- Request types ---

type SendCodeRequest struct {
	Email string `json:"email"`
}

type VerifyCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type RegisterRequest struct {
	VerificationToken string `json:"verificationToken"`
	DeviceLabel       string `json:"deviceLabel,omitempty"`
}

type LoginRequest struct {
	VerificationToken string `json:"verificationToken"`
	DeviceLabel       string `json:"deviceLabel,omitempty"`
}

type RevokeSessionRequest struct {
	SessionID string `json:"sessionId"`
}

type DeleteAccountRequest struct {
	Email string `json:"email"`
}

// --- Response types ---

type SendCodeResponse struct {
	Message string `json:"message"`
}

type VerifyCodeResponse struct {
	VerificationToken string `json:"verificationToken"`
	IsNewUser         bool   `json:"isNewUser"`
}

type RegisterResponse struct {
	UserID       string `json:"userId"`
	SessionToken string `json:"sessionToken"`
	ExpiresAt    string `json:"expiresAt"`
}

type LoginResponse struct {
	UserID       string `json:"userId"`
	SessionToken string `json:"sessionToken"`
	ExpiresAt    string `json:"expiresAt"`
}

type MeResponse struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

type Session struct {
	ID          string `json:"id"`
	DeviceLabel string `json:"deviceLabel"`
	ExpiresAt   string `json:"expiresAt"`
	CreatedAt   string `json:"createdAt"`
	IsCurrent   bool   `json:"isCurrent"`
}

type ListSessionsResponse struct {
	CurrentSessionID string    `json:"currentSessionId"`
	Sessions         []Session `json:"sessions"`
}

type RevokeSessionResponse struct {
	ID      string `json:"id"`
	Revoked bool   `json:"revoked"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}

type DeleteAccountResponse struct {
	Deleted bool `json:"deleted"`
}

// --- Methods ---

// SendCode sends a 6-digit login code to the given email.
func (c *Client) SendCode(email string) (*SendCodeResponse, error) {
	var resp SendCodeResponse
	err := c.doJSON("POST", "/api/auth/send-code", SendCodeRequest{Email: email}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// VerifyCode verifies the login code and returns a verification token.
func (c *Client) VerifyCode(email, code string) (*VerifyCodeResponse, error) {
	var resp VerifyCodeResponse
	err := c.doJSON("POST", "/api/auth/verify-code", VerifyCodeRequest{Email: email, Code: code}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Register creates a new account after email verification.
func (c *Client) Register(req RegisterRequest) (*RegisterResponse, error) {
	var resp RegisterResponse
	err := c.doJSON("POST", "/api/auth/register", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Login authenticates an existing user after email verification.
func (c *Client) Login(req LoginRequest) (*LoginResponse, error) {
	var resp LoginResponse
	err := c.doJSON("POST", "/api/auth/login", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Logout destroys the current session.
func (c *Client) Logout() (*LogoutResponse, error) {
	var resp LogoutResponse
	err := c.doJSON("POST", "/api/auth/logout", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetMe returns the current user's profile.
func (c *Client) GetMe() (*MeResponse, error) {
	var resp MeResponse
	err := c.doJSON("GET", "/api/auth/me", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListSessions returns all active sessions for the current user.
func (c *Client) ListSessions() (*ListSessionsResponse, error) {
	var resp ListSessionsResponse
	err := c.doJSON("GET", "/api/auth/sessions", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// RevokeSession revokes a specific session by ID.
func (c *Client) RevokeSession(sessionID string) (*RevokeSessionResponse, error) {
	var resp RevokeSessionResponse
	err := c.doJSON("DELETE", "/api/auth/sessions", RevokeSessionRequest{SessionID: sessionID}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteAccount permanently deletes the user's account. Requires email confirmation.
func (c *Client) DeleteAccount(email string) (*DeleteAccountResponse, error) {
	var resp DeleteAccountResponse
	err := c.doJSON("DELETE", "/api/auth/account", DeleteAccountRequest{Email: email}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// Export downloads all user data as JSON.
func (c *Client) Export() ([]byte, error) {
	body, _, err := c.doRaw("GET", "/api/auth/export", nil)
	if err != nil {
		return nil, fmt.Errorf("exporting data: %w", err)
	}
	return body, nil
}
