package api

import "fmt"

// --- Request types ---

type CreateDocRequest struct {
	EncryptedPayload     string   `json:"encryptedPayload"`
	LockType             string   `json:"lockType"`
	LockCredentialHashes []string `json:"lockCredentialHashes"`
	RecipientEmails      []string `json:"recipientEmails,omitempty"`
	ReadingTheme         string   `json:"readingTheme,omitempty"`
	ExpiresIn            *int     `json:"expiresIn,omitempty"`
	MaxViews             *int     `json:"maxViews,omitempty"`
}

type UpdateDocRequest struct {
	EncryptedPayload string `json:"encryptedPayload"`
}

type ExtendDocRequest struct {
	AddSeconds *int `json:"addSeconds,omitempty"`
	AddViews   *int `json:"addViews,omitempty"`
}

// --- Response types ---

type CreateDocResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type DocResponse struct {
	ID               string  `json:"id"`
	EncryptedPayload string  `json:"encryptedPayload"`
	LockType         string  `json:"lockType"`
	ReadingTheme     string  `json:"readingTheme"`
	ExpiresAt        *string `json:"expiresAt"`
	MaxViews         *int    `json:"maxViews"`
	CurrentViews     int     `json:"currentViews"`
	CreatedAt        string  `json:"createdAt"`
}

type EncryptedDoc struct {
	ID               string  `json:"id"`
	EncryptedPayload string  `json:"encryptedPayload"`
	LockType         string  `json:"lockType"`
	ReadingTheme     string  `json:"readingTheme"`
	ExpiresAt        *string `json:"expiresAt"`
	MaxViews         *int    `json:"maxViews"`
	CurrentViews     int     `json:"currentViews"`
	Revoked          bool    `json:"revoked"`
	CreatedAt        string  `json:"createdAt"`
}

type ListDocsResponse struct {
	Sent []EncryptedDoc `json:"sent"`
}

type UpdateDocResponse struct {
	ID        string `json:"id"`
	Updated   bool   `json:"updated"`
	UpdatedAt string `json:"updatedAt"`
}

type DeleteDocResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

type RevokeDocResponse struct {
	ID      string `json:"id"`
	Revoked bool   `json:"revoked"`
}

type ExtendDocResponse struct {
	ID        string  `json:"id"`
	ExpiresAt *string `json:"expiresAt"`
	MaxViews  *int    `json:"maxViews"`
}

// RawDocHeaders holds the metadata headers from the /raw endpoint.
type RawDocHeaders struct {
	Encrypted bool
	Algorithm string
}

// --- Methods ---

// CreateDoc uploads an encrypted document.
func (c *Client) CreateDoc(req CreateDocRequest) (*CreateDocResponse, error) {
	var resp CreateDocResponse
	err := c.doJSON("POST", "/api/docs", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListDocs returns the authenticated user's sent and received documents.
func (c *Client) ListDocs() (*ListDocsResponse, error) {
	var resp ListDocsResponse
	err := c.doJSON("GET", "/api/docs", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetDoc retrieves a document after lock verification.
// The credential is the light lock answer or strong lock password.
func (c *Client) GetDoc(id, credential string) (*DocResponse, error) {
	headers := map[string]string{}
	if credential != "" {
		headers["X-MDPush-Auth"] = credential
	}

	var resp DocResponse
	// Use doRaw to set the custom header, then parse JSON
	body, _, err := c.doRaw("GET", "/api/docs/"+id, headers)
	if err != nil {
		return nil, err
	}

	if err := jsonUnmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decoding doc response: %w", err)
	}
	return &resp, nil
}

// GetDocRaw returns the raw encrypted blob and metadata headers for agent consumption.
func (c *Client) GetDocRaw(id, credential string) ([]byte, *RawDocHeaders, error) {
	headers := map[string]string{}
	if credential != "" {
		headers["X-MDPush-Auth"] = credential
	}

	body, respHeaders, err := c.doRaw("GET", "/api/docs/"+id+"/raw", headers)
	if err != nil {
		return nil, nil, err
	}

	rawHeaders := &RawDocHeaders{
		Encrypted: respHeaders.Get("X-MDPush-Encrypted") == "true",
		Algorithm: respHeaders.Get("X-MDPush-Algorithm"),
	}

	return body, rawHeaders, nil
}

// UpdateDoc updates a document's encrypted payload (owner only).
func (c *Client) UpdateDoc(id string, encryptedPayload string) (*UpdateDocResponse, error) {
	var resp UpdateDocResponse
	err := c.doJSON("PATCH", "/api/docs/"+id, UpdateDocRequest{EncryptedPayload: encryptedPayload}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteDoc permanently deletes a document (owner only).
func (c *Client) DeleteDoc(id string) (*DeleteDocResponse, error) {
	var resp DeleteDocResponse
	err := c.doJSON("DELETE", "/api/docs/"+id, nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// RevokeDoc revokes a document so readers can no longer access it (owner only).
func (c *Client) RevokeDoc(id string) (*RevokeDocResponse, error) {
	var resp RevokeDocResponse
	err := c.doJSON("POST", "/api/docs/"+id+"/revoke", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// RestoreDoc un-revokes a previously revoked document (owner only).
func (c *Client) RestoreDoc(id string) (*RevokeDocResponse, error) {
	var resp RevokeDocResponse
	err := c.doJSON("DELETE", "/api/docs/"+id+"/revoke", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ExtendDoc extends a document's expiration time and/or view limit (owner only).
func (c *Client) ExtendDoc(id string, req ExtendDocRequest) (*ExtendDocResponse, error) {
	var resp ExtendDocResponse
	err := c.doJSON("POST", "/api/docs/"+id+"/extend", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

