package jwt

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-jose/go-jose/v4"
)

// JwtClaimsKey is the context key for storing JWT claims
type JwtClaimsKey struct{}

type JwtAuthorizer struct {
	queryParam string
	issuer     string
	audience   string
	keys       *jose.JSONWebKeySet
}

func NewJwtAuthorizer(config *JwtConfig) (*JwtAuthorizer, error) {
	keys, err := config.SecretSource.GetKeys()
	if err != nil {
		slog.Error("Failed to get keys", "error", err)
		return nil, err
	}
	return &JwtAuthorizer{
		queryParam: config.QueryParam,
		issuer:     config.Issuer,
		audience:   config.Audience,
		keys:       keys,
	}, nil
}

func (a *JwtAuthorizer) Authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get(a.queryParam)
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		signature, err := jose.ParseSigned(token, []jose.SignatureAlgorithm{
			jose.EdDSA,
			jose.HS256,
			jose.HS384,
			jose.HS512,
			jose.RS256,
			jose.RS384,
			jose.RS512,
			jose.ES256,
			jose.ES384,
			jose.ES512,
			jose.PS256,
			jose.PS384,
			jose.PS512,
		})

		if err != nil {
			slog.Debug("Failed to parse signed token", "error", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		t, err := signature.Verify(a.keys)
		if err != nil {
			slog.Debug("Failed to verify signed token", "error", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims := make(map[string]interface{})
		if err := json.Unmarshal(t, &claims); err != nil {
			slog.Debug("Failed to unmarshal claims", "error", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate issuer if configured
		if a.issuer != "" {
			if iss, ok := claims["iss"].(string); !ok || iss != a.issuer {
				slog.Error("Invalid issuer", "issuer", iss, "expected", a.issuer)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Validate audience if configured
		if a.audience != "" {
			if aud, ok := claims["aud"]; ok {
				// Handle both string and []string audience formats
				switch v := aud.(type) {
				case string:
					if v != a.audience {
						slog.Debug("Invalid audience", "audience", v, "expected", a.audience)
						http.Error(w, "Unauthorized", http.StatusUnauthorized)
						return
					}
				case []interface{}:
					found := false
					for _, aud := range v {
						if str, ok := aud.(string); ok && str == a.audience {
							found = true
							break
						}
					}
					if !found {
						slog.Debug("Invalid audience", "audience", v, "expected", a.audience)
						http.Error(w, "Unauthorized", http.StatusUnauthorized)
						return
					}
				default:
					slog.Debug("Invalid audience", "audience", v, "expected", a.audience)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			} else {
				slog.Debug("Missing audience", "audience", a.audience)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		ctx := context.WithValue(r.Context(), JwtClaimsKey{}, claims)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
