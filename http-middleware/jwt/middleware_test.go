package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
)

var (
	testKeyID = "test-key-id"
)

func TestJwtAuthorizer(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	jwks := &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:   &privateKey.PublicKey,
				Use:   "sig",
				KeyID: testKeyID,
			},
		},
	}

	config := &JwtConfig{
		QueryParam:   "token",
		SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
	}

	authorizer, err := NewJwtAuthorizer(config)
	assert.NoError(t, err)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		payload := r.Header.Get("X-JWT-Payload")
		assert.NotEmpty(t, payload)
		w.WriteHeader(http.StatusOK)
	})

	middleware := authorizer.Authorize(testHandler)

	t.Run("missing token", func(t *testing.T) {
		handlerCalled = false
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, handlerCalled)
	})

	t.Run("invalid token", func(t *testing.T) {
		handlerCalled = false
		req := httptest.NewRequest("GET", "/?token=invalid", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, handlerCalled)
	})

	t.Run("valid token", func(t *testing.T) {
		handlerCalled = false
		token := createToken(t, privateKey, testKeyID, jose.RS256)
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, handlerCalled)
	})

	t.Run("different query param", func(t *testing.T) {
		config.QueryParam = "auth"
		authorizer, err := NewJwtAuthorizer(config)
		assert.NoError(t, err)
		middleware := authorizer.Authorize(testHandler)

		handlerCalled = false
		token := createToken(t, privateKey, testKeyID, jose.RS256)
		req := httptest.NewRequest("GET", "/?auth="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, handlerCalled)
	})
}

func TestJwtAuthorizerWithDifferentAlgorithms(t *testing.T) {
	hmacKey := make([]byte, 32)
	rand.Read(hmacKey)
	jwks := &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Key:       hmacKey,
				Use:       "sig",
				Algorithm: string(jose.HS256),
				KeyID:     testKeyID,
			},
		},
	}

	config := &JwtConfig{
		QueryParam:   "token",
		SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
	}

	authorizer, err := NewJwtAuthorizer(config)
	assert.NoError(t, err)

	handlerCalled := false
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		payload := r.Header.Get("X-JWT-Payload")
		assert.NotEmpty(t, payload)
		w.WriteHeader(http.StatusOK)
	})

	middleware := authorizer.Authorize(testHandler)

	t.Run("HS256 signed token", func(t *testing.T) {
		handlerCalled = false
		token := createToken(t, hmacKey, testKeyID, jose.HS256)
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, handlerCalled)
	})
}

func createToken(t *testing.T, key interface{}, kid string, alg jose.SignatureAlgorithm) string {
	jwk := jose.JSONWebKey{
		Key:       key,
		Algorithm: string(alg),
		Use:       "sig",
		KeyID:     kid,
	}

	signerOptions := &(jose.SignerOptions{
		EmbedJWK: false,
	})

	signerOptions.WithType("JWT")
	signerOptions.WithHeader("kid", kid)

	signer, err := jose.NewSigner(jose.SigningKey{
		Algorithm: alg,
		Key:       jwk,
	}, signerOptions)
	assert.NoError(t, err)

	claims := jwt.Claims{
		Subject: "test-subject",
	}
	token, err := jwt.Signed(signer).Claims(claims).Serialize()
	assert.NoError(t, err)
	return token
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	assert.NoError(t, err)
	return data
}
