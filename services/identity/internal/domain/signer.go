package domain

type TokenSigner interface {
	Sign(claims Claims) (string, error)
	Verify(token string) (*Claims, error)
	JWKS() ([]byte, error)
}
