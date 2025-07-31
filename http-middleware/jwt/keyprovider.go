package jwt

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-jose/go-jose/v4"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// KeyProvider defines an interface for different ways to provide JWT secret
type KeyProvider interface {
	GetKeys() (*jose.JSONWebKeySet, error)
}

// RawJWKSProvider provides secret from a raw JWKS
type RawJWKSProvider struct {
	Content []byte
}

func (p *RawJWKSProvider) GetKeys() (*jose.JSONWebKeySet, error) {
	return decodeJWKS(p.Content)
}

func decodeJWKS(content []byte) (*jose.JSONWebKeySet, error) {
	var jwks jose.JSONWebKeySet
	err := json.Unmarshal(content, &jwks)
	return &jwks, err
}

// JWKSFileProvider provides secret from a JWKS file
type JWKSFileProvider struct {
	FilePath string
	RawJWKSProvider
}

func (p *JWKSFileProvider) GetKeys() (*jose.JSONWebKeySet, error) {
	content, err := os.ReadFile(p.FilePath)
	if err != nil {
		return nil, err
	}

	return decodeJWKS(content)
}

// JWKSURLProvider provides secret from a JWKS URL
type JWKSURLProvider struct {
	URL string
	RawJWKSProvider
}

func (p *JWKSURLProvider) GetKeys() (*jose.JSONWebKeySet, error) {
	return p.fetchJWKS()
}

func (p *JWKSURLProvider) fetchJWKS() (*jose.JSONWebKeySet, error) {
	resp, err := httpClient.Get(p.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch JWKS: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWKS: %w", err)
	}

	return decodeJWKS(body)
}

// OpenIDConfigProvider provides secret using OpenID Connect discovery
type OpenIDConfigProvider struct {
	Issuer string
	JWKSURLProvider
}

func (p *OpenIDConfigProvider) GetKeys() (*jose.JSONWebKeySet, error) {
	resp, err := httpClient.Get(p.Issuer + "/.well-known/openid-configuration")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch OpenID configuration: %s", resp.Status)
	}

	openIDConfig := struct {
		JWKSUri string `json:"jwks_uri"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&openIDConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to decode OpenID configuration: %w", err)
	}

	p.URL = openIDConfig.JWKSUri

	return p.fetchJWKS()
}
