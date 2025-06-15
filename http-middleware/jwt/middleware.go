package jwt

import (
	"fmt"

	"net/http"
)

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

		secret, err := a.config.SecretSource.GetKeys()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// TODO: remove this
		fmt.Println("secret", secret)

		// TODO: Use secret to validate JWT token

		next.ServeHTTP(w, r)
	})
}
