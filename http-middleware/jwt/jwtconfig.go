package jwt

type JwtConfig struct {
	Enabled      bool
	QueryParam   string
	SecretSource KeyProvider
	Issuer       string
	Audience     string
}
