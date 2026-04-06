package api

// --- Response type ---

type Settings struct {
	DefaultLockType    string `json:"defaultLockType"`
	DefaultExpiration  string `json:"defaultExpiration"`
	DefaultMaxViews    *int   `json:"defaultMaxViews"`
	DefaultCategory    *string `json:"defaultCategory"`
	PasswordTheme      string `json:"passwordTheme"`
}

// --- Update request ---
// All fields are optional — only included fields are updated.

type UpdateSettingsRequest struct {
	DefaultLockType   *string `json:"defaultLockType,omitempty"`
	DefaultExpiration *string `json:"defaultExpiration,omitempty"`
	DefaultMaxViews   *int    `json:"defaultMaxViews,omitempty"`
	DefaultCategory   *string `json:"defaultCategory,omitempty"`
	PasswordTheme     *string `json:"passwordTheme,omitempty"`
}

// --- Methods ---

// GetSettings fetches the user's preferences.
func (c *Client) GetSettings() (*Settings, error) {
	var resp Settings
	err := c.doJSON("GET", "/api/settings", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateSettings updates the user's preferences. Only non-nil fields are changed.
func (c *Client) UpdateSettings(req UpdateSettingsRequest) (*Settings, error) {
	var resp Settings
	err := c.doJSON("PATCH", "/api/settings", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
