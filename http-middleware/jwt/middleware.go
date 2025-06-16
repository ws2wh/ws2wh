package jwt

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-jose/go-jose/v4"
)

// JwtPayloadKey is the context key for storing JWT payload
type JwtPayloadKey struct{}

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
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		t, err := signature.Verify(a.keys)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), JwtPayloadKey{}, string(t))
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
