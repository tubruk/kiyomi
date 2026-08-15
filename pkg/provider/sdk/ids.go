package sdk

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// Deprecated: Providers should use clean, URL-safe resource IDs directly.
// EncodeID generates a clean, URL-safe internal resource ID from providerID and relative path.
func EncodeID(providerID, path string) string {
	raw := providerID + "|" + path
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// Deprecated: Providers should use clean, URL-safe resource IDs directly.
// DecodeID decodes a clean internal resource ID back into providerID and relative path.
func DecodeID(encodedID string) (providerID, path string, err error) {
	data, err := base64.RawURLEncoding.DecodeString(encodedID)
	if err != nil {
		return "", "", fmt.Errorf("invalid resource id encoding: %w", err)
	}
	parts := strings.SplitN(string(data), "|", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid resource id format")
	}
	return parts[0], parts[1], nil
}
