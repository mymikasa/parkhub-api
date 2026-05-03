package domain

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type RS256Signer struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
	issuer     string
}

func NewRS256Signer(privateKeyPEM, publicKeyPEM []byte, kid, issuer string) (*RS256Signer, error) {
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return &RS256Signer{
		privateKey: privKey,
		publicKey:  pubKey,
		kid:        kid,
		issuer:     issuer,
	}, nil
}

func LoadRS256Signer(privateKeyPath, publicKeyPath, kid, issuer string) (*RS256Signer, error) {
	privPEM, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	pubPEM, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	return NewRS256Signer(privPEM, pubPEM, kid, issuer)
}

func (s *RS256Signer) Sign(claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.kid
	return token.SignedString(s.privateKey)
}

func (s *RS256Signer) Verify(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	}, jwt.WithIssuer(s.issuer))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

func (s *RS256Signer) JWKS() ([]byte, error) {
	n := s.publicKey.N.Bytes()
	e := make([]byte, 4)
	e[0] = byte(s.publicKey.E >> 24)
	e[1] = byte(s.publicKey.E >> 16)
	e[2] = byte(s.publicKey.E >> 8)
	e[3] = byte(s.publicKey.E)
	i := 0
	for i < 3 {
		if e[i] != 0 {
			break
		}
		i++
	}
	e = e[i:]

	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": s.kid,
				"n":   encodeBase64URL(n),
				"e":   encodeBase64URL(e),
			},
		},
	}
	return json.Marshal(jwks)
}

func encodeBase64URL(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}
