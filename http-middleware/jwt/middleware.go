package jwt

import (
	"net/http"
)

type JwtConfig struct {
	Enabled    bool
	QueryParam string
	Secret     string
	Issuer     string
	Audience   string
}

type JwtAuthorizer struct {
	config *JwtConfig
}

func NewJwtAuthorizer(config *JwtConfig) *JwtAuthorizer {
	return &JwtAuthorizer{config: config}
}

func (a *JwtAuthorizer) Authorize(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get(a.config.QueryParam)
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// TODO validate jwt token
		// TODO requires jwt key to be loaded from somewhere - possible sources:
		// - raw key as param
		// - jwks file
		// - jwks url
		// - issuer with standard .well-known/openid-configuration url

		// TODO validate jwt token

		next.ServeHTTP(w, r)
	})
}
