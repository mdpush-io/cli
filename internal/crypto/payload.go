package crypto

import (
	"encoding/json"
	"fmt"
)

// Document represents the decrypted payload structure.
// Matches the web app's DecryptedDocument interface in client.ts.
type Document struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Category string `json:"category,omitempty"`
	Project  string `json:"project,omitempty"`
}

// EncryptPayload encrypts a Document into a single AES-256-GCM blob.
// The JSON structure { title, content, category?, project? } is serialized
// and then encrypted with the doc key.
// Matches the web app's encryptPayload().
func EncryptPayload(doc Document, docKey []byte) (string, error) {
	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshaling document: %w", err)
	}
	return Encrypt(jsonBytes, docKey)
}

// DecryptPayload decrypts an AES-256-GCM blob back into a Document.
// Matches the web app's decryptPayload().
func DecryptPayload(encryptedBase64 string, docKey []byte) (Document, error) {
	plaintext, err := Decrypt(encryptedBase64, docKey)
	if err != nil {
		return Document{}, fmt.Errorf("decrypting payload: %w", err)
	}

	var doc Document
	if err := json.Unmarshal(plaintext, &doc); err != nil {
		return Document{}, fmt.Errorf("parsing decrypted payload: %w", err)
	}

	if doc.Title == "" {
		doc.Title = "Untitled"
	}

	return doc, nil
}
