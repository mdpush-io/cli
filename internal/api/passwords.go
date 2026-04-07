package api

// --- Request/Response types ---

type GeneratePasswordRequest struct {
	Theme string `json:"theme"` // "books", "animals", "numbers"
}

type GeneratePasswordResponse struct {
	Password string `json:"password"`
	Theme    string `json:"theme"`
}

// --- Methods ---

// GeneratePassword generates a themed password for strong lock documents.
func (c *Client) GeneratePassword(theme string) (*GeneratePasswordResponse, error) {
	var resp GeneratePasswordResponse
	err := c.doJSON("POST", "/api/generate-password", GeneratePasswordRequest{Theme: theme}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
