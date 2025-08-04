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
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
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
		payload := r.Context().Value(JwtClaimsKey{}).(map[string]interface{})
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

	// Test issuer validation
	t.Run("valid issuer claim", func(t *testing.T) {
		issuerConfig := &JwtConfig{
			QueryParam:   "token",
			SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
			Issuer:       "test-issuer",
		}
		authorizer, err := NewJwtAuthorizer(issuerConfig)
		assert.NoError(t, err)
		middleware := authorizer.Authorize(testHandler)

		handlerCalled = false
		token := createTokenWithClaims(t, privateKey, testKeyID, jose.RS256, map[string]interface{}{
			"iss": "test-issuer",
			"sub": "test-subject",
		})
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, handlerCalled)
	})

	t.Run("invalid issuer claim", func(t *testing.T) {
		issuerConfig := &JwtConfig{
			QueryParam:   "token",
			SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
			Issuer:       "expected-issuer",
		}
		authorizer, err := NewJwtAuthorizer(issuerConfig)
		assert.NoError(t, err)
		middleware := authorizer.Authorize(testHandler)

		handlerCalled = false
		token := createTokenWithClaims(t, privateKey, testKeyID, jose.RS256, map[string]interface{}{
			"iss": "wrong-issuer",
			"sub": "test-subject",
		})
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, handlerCalled)
	})

	t.Run("missing issuer when required", func(t *testing.T) {
		issuerConfig := &JwtConfig{
			QueryParam:   "token",
			SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
			Issuer:       "required-issuer",
		}
		authorizer, err := NewJwtAuthorizer(issuerConfig)
		assert.NoError(t, err)
		middleware := authorizer.Authorize(testHandler)

		handlerCalled = false
		token := createTokenWithClaims(t, privateKey, testKeyID, jose.RS256, map[string]interface{}{
			"sub": "test-subject",
		})
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, handlerCalled)
	})

	// Test audience validation - string format
	t.Run("valid audience claim as string", func(t *testing.T) {
		audienceConfig := &JwtConfig{
			QueryParam:   "token",
			SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
			Audience:     "test-audience",
		}
		authorizer, err := NewJwtAuthorizer(audienceConfig)
		assert.NoError(t, err)
		middleware := authorizer.Authorize(testHandler)

		handlerCalled = false
		token := createTokenWithClaims(t, privateKey, testKeyID, jose.RS256, map[string]interface{}{
			"aud": "test-audience",
			"sub": "test-subject",
		})
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, handlerCalled)
	})

	t.Run("valid audience claim as array", func(t *testing.T) {
		audienceConfig := &JwtConfig{
			QueryParam:   "token",
			SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
			Audience:     "target-audience",
		}
		authorizer, err := NewJwtAuthorizer(audienceConfig)
		assert.NoError(t, err)
		middleware := authorizer.Authorize(testHandler)

		handlerCalled = false
		token := createTokenWithClaims(t, privateKey, testKeyID, jose.RS256, map[string]interface{}{
			"aud": []string{"other-audience", "target-audience", "another-audience"},
			"sub": "test-subject",
		})
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, handlerCalled)
	})

	t.Run("invalid audience claim as string", func(t *testing.T) {
		audienceConfig := &JwtConfig{
			QueryParam:   "token",
			SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
			Audience:     "expected-audience",
		}
		authorizer, err := NewJwtAuthorizer(audienceConfig)
		assert.NoError(t, err)
		middleware := authorizer.Authorize(testHandler)

		handlerCalled = false
		token := createTokenWithClaims(t, privateKey, testKeyID, jose.RS256, map[string]interface{}{
			"aud": "wrong-audience",
			"sub": "test-subject",
		})
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, handlerCalled)
	})

	t.Run("invalid audience claim as array", func(t *testing.T) {
		audienceConfig := &JwtConfig{
			QueryParam:   "token",
			SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
			Audience:     "expected-audience",
		}
		authorizer, err := NewJwtAuthorizer(audienceConfig)
		assert.NoError(t, err)
		middleware := authorizer.Authorize(testHandler)

		handlerCalled = false
		token := createTokenWithClaims(t, privateKey, testKeyID, jose.RS256, map[string]interface{}{
			"aud": []string{"wrong-audience", "another-wrong-audience"},
			"sub": "test-subject",
		})
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, handlerCalled)
	})

	t.Run("missing audience when required", func(t *testing.T) {
		audienceConfig := &JwtConfig{
			QueryParam:   "token",
			SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
			Audience:     "required-audience",
		}
		authorizer, err := NewJwtAuthorizer(audienceConfig)
		assert.NoError(t, err)
		middleware := authorizer.Authorize(testHandler)

		handlerCalled = false
		token := createTokenWithClaims(t, privateKey, testKeyID, jose.RS256, map[string]interface{}{
			"sub": "test-subject",
		})
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, handlerCalled)
	})

	// Test combined issuer and audience validation
	t.Run("valid issuer and audience claims", func(t *testing.T) {
		combinedConfig := &JwtConfig{
			QueryParam:   "token",
			SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
			Issuer:       "valid-issuer",
			Audience:     "valid-audience",
		}
		authorizer, err := NewJwtAuthorizer(combinedConfig)
		assert.NoError(t, err)
		middleware := authorizer.Authorize(testHandler)

		handlerCalled = false
		token := createTokenWithClaims(t, privateKey, testKeyID, jose.RS256, map[string]interface{}{
			"iss": "valid-issuer",
			"aud": "valid-audience",
			"sub": "test-subject",
		})
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, handlerCalled)
	})

	t.Run("valid issuer but invalid audience", func(t *testing.T) {
		combinedConfig := &JwtConfig{
			QueryParam:   "token",
			SecretSource: &RawJWKSProvider{Content: mustMarshal(t, jwks)},
			Issuer:       "valid-issuer",
			Audience:     "expected-audience",
		}
		authorizer, err := NewJwtAuthorizer(combinedConfig)
		assert.NoError(t, err)
		middleware := authorizer.Authorize(testHandler)

		handlerCalled = false
		token := createTokenWithClaims(t, privateKey, testKeyID, jose.RS256, map[string]interface{}{
			"iss": "valid-issuer",
			"aud": "wrong-audience",
			"sub": "test-subject",
		})
		req := httptest.NewRequest("GET", "/?token="+token, nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.False(t, handlerCalled)
	})
}

func TestJwtAuthorizerWithDifferentAlgorithms(t *testing.T) {
	hmacKey := make([]byte, 32)
	_, err := rand.Read(hmacKey)
	assert.NoError(t, err)
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
		payload := r.Context().Value(JwtClaimsKey{}).(map[string]interface{})
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

func createTokenWithClaims(t *testing.T, key interface{}, kid string, alg jose.SignatureAlgorithm, customClaims map[string]interface{}) string {
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

	token, err := jwt.Signed(signer).Claims(customClaims).Serialize()
	assert.NoError(t, err)
	return token
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	assert.NoError(t, err)
	return data
}
