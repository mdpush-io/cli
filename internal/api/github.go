package api

// --- GitHub Device Flow types ---

type DeviceCodeResponse struct {
	DeviceCode      string `json:"deviceCode"`
	UserCode        string `json:"userCode"`
	VerificationURI string `json:"verificationUri"`
	ExpiresIn       int    `json:"expiresIn"`
	Interval        int    `json:"interval"`
}

type DeviceTokenRequest struct {
	DeviceCode  string `json:"deviceCode"`
	DeviceLabel string `json:"deviceLabel,omitempty"`
}

// DeviceTokenResponse covers all status values returned by the polling
// endpoint: pending, slow_down, expired, denied, authorized, error.
// When Status == "authorized", SessionToken/UserID/ExpiresAt/Email are set.
type DeviceTokenResponse struct {
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
	Interval     int    `json:"interval,omitempty"`
	UserID       string `json:"userId,omitempty"`
	SessionToken string `json:"sessionToken,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
	Email        string `json:"email,omitempty"`
}

// --- Methods ---

// RequestGitHubDeviceCode initiates the GitHub Device Flow.
func (c *Client) RequestGitHubDeviceCode() (*DeviceCodeResponse, error) {
	var resp DeviceCodeResponse
	err := c.doJSON("POST", "/api/auth/github/device", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// PollGitHubDeviceToken polls for completion of an in-progress device flow.
// Caller is responsible for honoring DeviceCodeResponse.Interval between calls.
func (c *Client) PollGitHubDeviceToken(req DeviceTokenRequest) (*DeviceTokenResponse, error) {
	var resp DeviceTokenResponse
	err := c.doJSON("POST", "/api/auth/github/device/token", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
