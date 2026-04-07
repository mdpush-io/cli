package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Helper: mock server ---

func mockServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	server := httptest.NewServer(handler)
	client := &Client{
		BaseURL:    server.URL,
		HTTPClient: http.DefaultClient,
	}
	return server, client
}

// --- APIError tests ---

func TestAPIErrorMessage(t *testing.T) {
	err := &APIError{
		StatusCode: 401,
		ErrorCode:  "unauthorized",
		Message:    "Authentication required.",
	}
	want := "unauthorized: Authentication required. (HTTP 401)"
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

func TestAPIErrorNoMessage(t *testing.T) {
	err := &APIError{
		StatusCode: 401,
		ErrorCode:  "unauthorized",
	}
	want := "unauthorized (HTTP 401)"
	if err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
	}
}

func TestAPIErrorHelpers(t *testing.T) {
	tests := []struct {
		status       int
		isUnauth     bool
		isNotFound   bool
		isRateLimit  bool
	}{
		{401, true, false, false},
		{404, false, true, false},
		{429, false, false, true},
		{500, false, false, false},
	}
	for _, tt := range tests {
		err := &APIError{StatusCode: tt.status}
		if err.IsUnauthorized() != tt.isUnauth {
			t.Errorf("status %d: IsUnauthorized() = %v", tt.status, err.IsUnauthorized())
		}
		if err.IsNotFound() != tt.isNotFound {
			t.Errorf("status %d: IsNotFound() = %v", tt.status, err.IsNotFound())
		}
		if err.IsRateLimited() != tt.isRateLimit {
			t.Errorf("status %d: IsRateLimited() = %v", tt.status, err.IsRateLimited())
		}
	}
}

// --- Auth endpoint tests (mocked) ---

func TestSendCode(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/auth/send-code" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body SendCodeRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Email != "test@example.com" {
			t.Fatalf("expected email test@example.com, got %s", body.Email)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(SendCodeResponse{Message: "If this email is valid, a code has been sent."})
	})
	defer server.Close()

	resp, err := client.SendCode("test@example.com")
	if err != nil {
		t.Fatalf("SendCode: %v", err)
	}
	if resp.Message == "" {
		t.Fatal("expected non-empty message")
	}
}

func TestVerifyCodeInvalid(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "invalid_code",
			"message": "Invalid or expired code.",
		})
	})
	defer server.Close()

	_, err := client.VerifyCode("test@example.com", "000000")
	if err == nil {
		t.Fatal("expected error for invalid code")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", apiErr.StatusCode)
	}
	if apiErr.ErrorCode != "invalid_code" {
		t.Fatalf("expected invalid_code, got %s", apiErr.ErrorCode)
	}
}

func TestVerifyCodeSuccess(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(VerifyCodeResponse{
			VerificationToken: "eyJ...",
			IsNewUser:         true,
		})
	})
	defer server.Close()

	resp, err := client.VerifyCode("test@example.com", "123456")
	if err != nil {
		t.Fatalf("VerifyCode: %v", err)
	}
	if resp.VerificationToken != "eyJ..." {
		t.Fatalf("expected token eyJ..., got %s", resp.VerificationToken)
	}
	if !resp.IsNewUser {
		t.Fatal("expected isNewUser=true")
	}
}

func TestRegister(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/auth/register" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body RegisterRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.VerificationToken == "" {
			t.Fatal("expected verificationToken")
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(RegisterResponse{
			UserID:       "uuid-123",
			SessionToken: "tok-abc",
			ExpiresAt:    "2026-05-04T00:00:00Z",
		})
	})
	defer server.Close()

	resp, err := client.Register(RegisterRequest{
		VerificationToken: "eyJ...",
		DeviceLabel:       "CLI test",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if resp.UserID != "uuid-123" {
		t.Fatalf("expected uuid-123, got %s", resp.UserID)
	}
	if resp.SessionToken != "tok-abc" {
		t.Fatalf("expected tok-abc, got %s", resp.SessionToken)
	}
}

func TestLogin(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(LoginResponse{
			UserID:       "uuid-456",
			SessionToken: "tok-def",
			ExpiresAt:    "2026-05-04T00:00:00Z",
		})
	})
	defer server.Close()

	resp, err := client.Login(LoginRequest{
		VerificationToken: "eyJ...",
		DeviceLabel:       "CLI test",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if resp.UserID != "uuid-456" {
		t.Fatalf("expected uuid-456, got %s", resp.UserID)
	}
}

func TestBearerTokenSent(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-secret-token" {
			t.Fatalf("expected Bearer my-secret-token, got %q", auth)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(MeResponse{UserID: "id", Email: "a@b.com", Name: "a"})
	})
	defer server.Close()

	authed := client.WithToken("my-secret-token")
	_, err := authed.GetMe()
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
}

// --- Doc endpoint tests (mocked) ---

func TestCreateDoc(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/docs" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(CreateDocResponse{ID: "XCU01ijFFV", URL: "https://mdpush.io/d/XCU01ijFFV"})
	})
	defer server.Close()

	authed := client.WithToken("tok")
	resp, err := authed.CreateDoc(CreateDocRequest{
		EncryptedPayload:     "encrypted...",
		LockType:             "light",
		LockCredentialHashes: []string{"hash1", "hash2"},
	})
	if err != nil {
		t.Fatalf("CreateDoc: %v", err)
	}
	if resp.ID != "XCU01ijFFV" {
		t.Fatalf("expected XCU01ijFFV, got %s", resp.ID)
	}
}

func TestGetDocWithCredential(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		cred := r.Header.Get("X-MDPush-Auth")
		if cred == "" {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(map[string]any{
				"error":    "auth_required",
				"lockType": "light",
				"message":  "Who sent you this?",
			})
			return
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(DocResponse{
			ID:               "abc123",
			EncryptedPayload: "encrypted...",
			LockType:         "light",
			ReadingTheme:     "clean",
			CurrentViews:     1,
			CreatedAt:        "2026-04-04T00:00:00Z",
		})
	})
	defer server.Close()

	// Without credential — should fail
	_, err := client.GetDoc("abc123", "")
	if err == nil {
		t.Fatal("expected error without credential")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.LockType != "light" {
		t.Fatalf("expected lockType=light, got %s", apiErr.LockType)
	}

	// With credential — should succeed
	resp, err := client.GetDoc("abc123", "gabriel")
	if err != nil {
		t.Fatalf("GetDoc: %v", err)
	}
	if resp.ID != "abc123" {
		t.Fatalf("expected abc123, got %s", resp.ID)
	}
}

func TestGetDocRaw(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/docs/abc123/raw" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("X-MDPush-Encrypted", "true")
		w.Header().Set("X-MDPush-Algorithm", "aes-256-gcm")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(200)
		w.Write([]byte("encrypted-blob-bytes"))
	})
	defer server.Close()

	body, headers, err := client.GetDocRaw("abc123", "gabriel")
	if err != nil {
		t.Fatalf("GetDocRaw: %v", err)
	}
	if string(body) != "encrypted-blob-bytes" {
		t.Fatalf("unexpected body: %s", body)
	}
	if !headers.Encrypted {
		t.Fatal("expected Encrypted=true")
	}
	if headers.Algorithm != "aes-256-gcm" {
		t.Fatalf("expected aes-256-gcm, got %s", headers.Algorithm)
	}
}

func TestListDocs(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(ListDocsResponse{
			Sent: []EncryptedDoc{
				{ID: "doc1", LockType: "light", CreatedAt: "2026-04-04"},
			},
		})
	})
	defer server.Close()

	resp, err := client.WithToken("tok").ListDocs()
	if err != nil {
		t.Fatalf("ListDocs: %v", err)
	}
	if len(resp.Sent) != 1 || resp.Sent[0].ID != "doc1" {
		t.Fatal("unexpected sent docs")
	}
}

func TestRevokeAndRestore(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			json.NewEncoder(w).Encode(RevokeDocResponse{ID: "doc1", Revoked: true})
		} else if r.Method == "DELETE" {
			json.NewEncoder(w).Encode(RevokeDocResponse{ID: "doc1", Revoked: false})
		}
	})
	defer server.Close()

	authed := client.WithToken("tok")

	revResp, err := authed.RevokeDoc("doc1")
	if err != nil {
		t.Fatalf("RevokeDoc: %v", err)
	}
	if !revResp.Revoked {
		t.Fatal("expected revoked=true")
	}

	restResp, err := authed.RestoreDoc("doc1")
	if err != nil {
		t.Fatalf("RestoreDoc: %v", err)
	}
	if restResp.Revoked {
		t.Fatal("expected revoked=false")
	}
}

// --- Settings tests (mocked) ---

func TestGetSettings(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Settings{
			DefaultLockType:   "light",
			DefaultExpiration: "7d",
			PasswordTheme:     "books",
		})
	})
	defer server.Close()

	resp, err := client.WithToken("tok").GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if resp.DefaultLockType != "light" {
		t.Fatalf("expected light, got %s", resp.DefaultLockType)
	}
}

func TestGeneratePassword(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		var body GeneratePasswordRequest
		json.NewDecoder(r.Body).Decode(&body)
		if body.Theme != "books" {
			t.Fatalf("expected theme books, got %s", body.Theme)
		}
		json.NewEncoder(w).Encode(GeneratePasswordResponse{
			Password: "brave-new-world",
			Theme:    "books",
		})
	})
	defer server.Close()

	resp, err := client.WithToken("tok").GeneratePassword("books")
	if err != nil {
		t.Fatalf("GeneratePassword: %v", err)
	}
	if resp.Password != "brave-new-world" {
		t.Fatalf("expected brave-new-world, got %s", resp.Password)
	}
}

// --- Non-JSON error response ---

func TestNonJSONError(t *testing.T) {
	server, client := mockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
	})
	defer server.Close()

	_, err := client.SendCode("test@example.com")
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 500 {
		t.Fatalf("expected 500, got %d", apiErr.StatusCode)
	}
	// Should contain the raw text as message
	if apiErr.Message != "Internal Server Error" {
		t.Fatalf("expected raw message, got %q", apiErr.Message)
	}
}

// --- Production endpoint test ---

func TestProductionSendCodeShape(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping production test in short mode")
	}

	client := NewClient()
	resp, err := client.SendCode("cli-test-noreply@example.com")
	if err != nil {
		t.Fatalf("SendCode against production: %v", err)
	}
	if resp.Message == "" {
		t.Fatal("expected non-empty message from production")
	}
}

func TestProductionUnauthorized(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping production test in short mode")
	}

	client := NewClient()
	_, err := client.ListDocs()
	if err == nil {
		t.Fatal("expected 401 from production")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", apiErr.StatusCode)
	}
	if apiErr.ErrorCode != "unauthorized" {
		t.Fatalf("expected 'unauthorized', got %q", apiErr.ErrorCode)
	}
}

func TestProductionInvalidCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping production test in short mode")
	}

	client := NewClient()
	_, err := client.VerifyCode("test@test.com", "000000")
	if err == nil {
		t.Fatal("expected error for invalid code")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.ErrorCode != "invalid_code" {
		t.Fatalf("expected 'invalid_code', got %q", apiErr.ErrorCode)
	}
}

func TestProductionDocNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping production test in short mode")
	}

	client := NewClient()
	_, err := client.GetDoc("nonexistent99", "test")
	if err == nil {
		t.Fatal("expected 404")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Fatalf("expected 404, got %d", apiErr.StatusCode)
	}
}
