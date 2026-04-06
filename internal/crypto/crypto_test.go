package crypto

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// --- Base64 tests ---

func TestBase64RoundTrip(t *testing.T) {
	data := []byte("hello, mdpush!")
	encoded := Base64Encode(data)
	decoded, err := Base64Decode(encoded)
	if err != nil {
		t.Fatalf("Base64Decode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Fatalf("round-trip mismatch: got %q, want %q", decoded, data)
	}
}

func TestBase64URLRoundTrip(t *testing.T) {
	data := []byte{0xff, 0xfe, 0xfd, 0xfc, 0xfb} // will produce +/ in std base64
	encoded := Base64URLEncode(data)
	if strings.ContainsAny(encoded, "+/=") {
		t.Fatalf("base64url should not contain +/=, got %q", encoded)
	}
	decoded, err := Base64URLDecode(encoded)
	if err != nil {
		t.Fatalf("Base64URLDecode: %v", err)
	}
	if !bytes.Equal(data, decoded) {
		t.Fatalf("round-trip mismatch: got %x, want %x", decoded, data)
	}
}

func TestKeyToFragmentAndBack(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	fragment := KeyToFragment(key)
	recovered, err := ParseKeyFromFragment("#" + fragment)
	if err != nil {
		t.Fatalf("ParseKeyFromFragment: %v", err)
	}
	if !bytes.Equal(key, recovered) {
		t.Fatalf("key round-trip mismatch")
	}
}

func TestParseKeyFromFragmentNoHash(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i * 3)
	}
	fragment := KeyToFragment(key)
	recovered, err := ParseKeyFromFragment(fragment) // no # prefix
	if err != nil {
		t.Fatalf("ParseKeyFromFragment: %v", err)
	}
	if !bytes.Equal(key, recovered) {
		t.Fatalf("key round-trip mismatch without # prefix")
	}
}

// --- AES-256-GCM tests ---

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := "Hello, mdpush! This is a test."
	encrypted, err := Encrypt([]byte(plaintext), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := DecryptString(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("decrypted %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptWithIVDeterministic(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	iv := make([]byte, 12)
	for i := range iv {
		iv[i] = byte(i + 100)
	}

	plaintext := []byte("test")

	enc1, _ := EncryptWithIV(plaintext, key, iv)
	enc2, _ := EncryptWithIV(plaintext, key, iv)

	if enc1 != enc2 {
		t.Fatalf("same key+IV should produce same ciphertext")
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	key := make([]byte, 32)
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = 0xff
	}

	encrypted, _ := Encrypt([]byte("secret"), key)
	_, err := Decrypt(encrypted, wrongKey)
	if err == nil {
		t.Fatal("expected decryption to fail with wrong key")
	}
}

func TestEncryptInvalidKeySize(t *testing.T) {
	_, err := Encrypt([]byte("test"), make([]byte, 16))
	if err == nil {
		t.Fatal("expected error for 16-byte key")
	}
}

// --- Cross-platform test vectors ---

func TestEncryptWithIVCrossplatform(t *testing.T) {
	key := make([]byte, 32)
	iv := make([]byte, 12)
	plaintext := []byte("hello")

	encrypted, err := EncryptWithIV(plaintext, key, iv)
	if err != nil {
		t.Fatalf("EncryptWithIV: %v", err)
	}

	decrypted, err := DecryptString(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if decrypted != "hello" {
		t.Fatalf("got %q, want %q", decrypted, "hello")
	}

	raw, _ := Base64Decode(encrypted)
	for i := range 12 {
		if raw[i] != 0 {
			t.Fatalf("IV byte %d should be 0, got %d", i, raw[i])
		}
	}
	// Total length: 12 (IV) + 5 (plaintext "hello") + 16 (authTag) = 33
	if len(raw) != 33 {
		t.Fatalf("expected 33 bytes, got %d", len(raw))
	}
}

// --- Payload tests ---

func TestPayloadRoundTrip(t *testing.T) {
	key, err := GenerateDocKey()
	if err != nil {
		t.Fatalf("GenerateDocKey: %v", err)
	}

	doc := Document{
		Title:    "test-spec.md",
		Content:  "# Test\n\nThis is a test document.",
		Category: "debugging",
		Project:  "mdpush",
	}

	encrypted, err := EncryptPayload(doc, key)
	if err != nil {
		t.Fatalf("EncryptPayload: %v", err)
	}

	decrypted, err := DecryptPayload(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptPayload: %v", err)
	}

	if decrypted.Title != doc.Title {
		t.Fatalf("title: got %q, want %q", decrypted.Title, doc.Title)
	}
	if decrypted.Content != doc.Content {
		t.Fatalf("content mismatch")
	}
	if decrypted.Category != doc.Category {
		t.Fatalf("category: got %q, want %q", decrypted.Category, doc.Category)
	}
	if decrypted.Project != doc.Project {
		t.Fatalf("project: got %q, want %q", decrypted.Project, doc.Project)
	}
}

func TestPayloadOmitsEmptyFields(t *testing.T) {
	key, _ := GenerateDocKey()

	doc := Document{
		Title:   "just-a-title.md",
		Content: "content",
	}

	encrypted, _ := EncryptPayload(doc, key)
	plaintext, _ := Decrypt(encrypted, key)

	var raw map[string]any
	json.Unmarshal(plaintext, &raw)

	if _, ok := raw["category"]; ok {
		t.Fatal("empty category should be omitted from JSON")
	}
	if _, ok := raw["project"]; ok {
		t.Fatal("empty project should be omitted from JSON")
	}
}

func TestPayloadDefaultsToUntitled(t *testing.T) {
	key, _ := GenerateDocKey()

	jsonBytes, _ := json.Marshal(map[string]string{"content": "test"})
	encrypted, _ := Encrypt(jsonBytes, key)

	doc, err := DecryptPayload(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptPayload: %v", err)
	}
	if doc.Title != "Untitled" {
		t.Fatalf("expected 'Untitled', got %q", doc.Title)
	}
}

// --- SHA-256 tests ---

func TestSHA256Hex(t *testing.T) {
	hash := SHA256Hex("hello")
	expected := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if hash != expected {
		t.Fatalf("SHA256Hex('hello') = %s, want %s", hash, expected)
	}
}

func TestSHA256HexEmpty(t *testing.T) {
	hash := SHA256Hex("")
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expected {
		t.Fatalf("SHA256Hex('') = %s, want %s", hash, expected)
	}
}

func TestNormalizeLockCredential(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Gabriel", "gabriel"},
		{" GABRIEL ", "gabriel"},
		{"  gabriel.medeiros  ", "gabriel.medeiros"},
		{"user@EXAMPLE.com", "user@example.com"},
	}
	for _, tt := range tests {
		got := NormalizeLockCredential(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeLockCredential(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestHashLockCredential(t *testing.T) {
	hash := HashLockCredential(" Alice ")
	expected := SHA256Hex("alice")
	if hash != expected {
		t.Fatalf("got %s, want %s", hash, expected)
	}
}

func TestBuildLightLockHashes(t *testing.T) {
	hashes := BuildLightLockHashes("alice.smith@example.com")

	expectedVariants := map[string]bool{
		SHA256Hex("alice.smith@example.com"): false,
		SHA256Hex("alice.smith"):             false,
		SHA256Hex("alice smith"):             false,
		SHA256Hex("alicesmith"):              false,
	}

	for _, h := range hashes {
		if _, ok := expectedVariants[h]; ok {
			expectedVariants[h] = true
		}
	}

	for variant, found := range expectedVariants {
		if !found {
			t.Errorf("missing expected hash variant: %s", variant)
		}
	}
}

// --- Passphrase tests ---

func TestPassphraseSuggestionFormat(t *testing.T) {
	suggestion, err := GeneratePassphraseSuggestion()
	if err != nil {
		t.Fatalf("GeneratePassphraseSuggestion: %v", err)
	}

	parts := strings.Split(suggestion, "-")
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts (word-word-word-number), got %d: %s", len(parts), suggestion)
	}

	numPart := parts[3]
	if len(numPart) != 4 {
		t.Fatalf("number part should be 4 digits, got %q", numPart)
	}
	for _, c := range numPart {
		if c < '0' || c > '9' {
			t.Fatalf("number part contains non-digit: %q", c)
		}
	}

	for i := range 3 {
		found := false
		for _, w := range passphraseWords {
			if parts[i] == w {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("word %q not in word list", parts[i])
		}
	}
}

func TestPassphraseSuggestionUniqueness(t *testing.T) {
	s1, _ := GeneratePassphraseSuggestion()
	s2, _ := GeneratePassphraseSuggestion()
	if s1 == s2 {
		t.Fatal("two suggestions should not be identical (extremely unlikely)")
	}
}

// --- Key generation tests ---

func TestGenerateDocKeyLength(t *testing.T) {
	key, err := GenerateDocKey()
	if err != nil {
		t.Fatalf("GenerateDocKey: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(key))
	}
}

// --- Full end-to-end flow test ---

func TestFullShareFlow(t *testing.T) {
	// Simulate the complete mdpush share flow:
	// 1. Generate a doc key
	// 2. Encrypt the document payload
	// 3. Generate the URL fragment
	// 4. Verify the reader can decrypt via the fragment

	docKey, _ := GenerateDocKey()
	doc := Document{
		Title:    "quarterly-report.md",
		Content:  "# Q3 Report\n\nRevenue is up 15%.",
		Category: "new-feature",
		Project:  "mdpush",
	}

	encryptedPayload, err := EncryptPayload(doc, docKey)
	if err != nil {
		t.Fatalf("EncryptPayload: %v", err)
	}

	fragment := KeyToFragment(docKey)

	// Reader decrypts via URL fragment
	recoveredKey, err := ParseKeyFromFragment("#" + fragment)
	if err != nil {
		t.Fatalf("ParseKeyFromFragment: %v", err)
	}

	recoveredDoc, err := DecryptPayload(encryptedPayload, recoveredKey)
	if err != nil {
		t.Fatalf("DecryptPayload (reader): %v", err)
	}

	if recoveredDoc.Title != doc.Title {
		t.Fatalf("reader got title %q, want %q", recoveredDoc.Title, doc.Title)
	}
	if recoveredDoc.Content != doc.Content {
		t.Fatal("reader got wrong content")
	}
	if recoveredDoc.Category != doc.Category {
		t.Fatalf("reader got category %q, want %q", recoveredDoc.Category, doc.Category)
	}
	if recoveredDoc.Project != doc.Project {
		t.Fatalf("reader got project %q, want %q", recoveredDoc.Project, doc.Project)
	}
}
