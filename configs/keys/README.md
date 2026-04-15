# JWT Keys

Generate RSA key pair for RS256 signing:

```bash
openssl genpkey -algorithm RSA -out jwt_private.pem -pkeyopt rsa_keygen_bits:2048
openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem
```

Private key stays on backend only. Public key is shared with APISIX for JWT verification.
