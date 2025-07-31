package jwt

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
)

func TestRawJWKSProvider(t *testing.T) {
	// Create a sample JWKS
	testKey := []byte("test-key")
	jwks := &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:   testKey,
				KeyID: "test-key-id",
			},
		},
	}

	// Marshal the JWKS to JSON
	jwksJSON, err := json.Marshal(jwks)
	assert.NoError(t, err)

	// Create provider with the JSON content
	provider := &RawJWKSProvider{
		Content: jwksJSON,
	}

	// Test GetKeys
	result, err := provider.GetKeys()
	assert.NoError(t, err)
	assert.Equal(t, len(jwks.Keys), len(result.Keys))
	assert.Equal(t, jwks.Keys[0].KeyID, result.Keys[0].KeyID)
	assert.Equal(t, testKey, result.Keys[0].Key)
}

func TestJWKSFileProvider(t *testing.T) {
	// Create a temporary file with JWKS content
	testKey := []byte("test-key")
	jwks := &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:   testKey,
				KeyID: "test-key-id",
			},
		},
	}

	jwksJSON, err := json.Marshal(jwks)
	assert.NoError(t, err)

	tmpFile, err := os.CreateTemp("", "jwks-*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(jwksJSON)
	assert.NoError(t, err)
	tmpFile.Close()

	// Create provider with the file path
	provider := &JWKSFileProvider{
		FilePath: tmpFile.Name(),
	}

	// Test GetKeys
	result, err := provider.GetKeys()
	assert.NoError(t, err)
	assert.Equal(t, len(jwks.Keys), len(result.Keys))
	assert.Equal(t, jwks.Keys[0].KeyID, result.Keys[0].KeyID)
	assert.Equal(t, testKey, result.Keys[0].Key)

	// Test error case with non-existent file
	provider.FilePath = "non-existent-file.json"
	_, err = provider.GetKeys()
	assert.Error(t, err)
}

func TestJWKSURLProvider(t *testing.T) {
	// Create a test server
	testKey := []byte("test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwks := &jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{
				{
					Key:   testKey,
					KeyID: "test-key-id",
				},
			},
		}
		json.NewEncoder(w).Encode(jwks)
	}))
	defer server.Close()

	// Create provider with the test server URL
	provider := &JWKSURLProvider{
		URL: server.URL,
	}

	// Test GetKeys
	result, err := provider.GetKeys()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Keys))
	assert.Equal(t, "test-key-id", result.Keys[0].KeyID)
	assert.Equal(t, testKey, result.Keys[0].Key)

	// Test error case with invalid URL
	provider.URL = "http://invalid-url"
	_, err = provider.GetKeys()
	assert.Error(t, err)
}

func TestOpenIDConfigProvider(t *testing.T) {
	// Create a test server for OpenID configuration
	testKey := []byte("test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/openid-configuration" {
			config := struct {
				JWKSUri string `json:"jwks_uri"`
			}{
				JWKSUri: "http://" + r.Host + "/jwks",
			}
			json.NewEncoder(w).Encode(config)
			return
		}

		if r.URL.Path == "/jwks" {
			jwks := &jose.JSONWebKeySet{
				Keys: []jose.JSONWebKey{
					{
						Key:   testKey,
						KeyID: "test-key-id",
					},
				},
			}
			json.NewEncoder(w).Encode(jwks)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create provider with the test server URL
	provider := &OpenIDConfigProvider{
		Issuer: server.URL,
	}

	// Test GetKeys
	result, err := provider.GetKeys()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(result.Keys))
	assert.Equal(t, "test-key-id", result.Keys[0].KeyID)
	assert.Equal(t, testKey, result.Keys[0].Key)

	// Test error case with invalid issuer
	provider.Issuer = "http://invalid-issuer"
	_, err = provider.GetKeys()
	assert.Error(t, err)
}
