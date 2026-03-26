# Kanji Quiz

A user-made quiz for testing kanji knowledge.

### Required Environment Variables

| Variable | Description |
|---|---|
| `SESSION_AUTH_KEY` | Secret key used to authenticate session cookies via HMAC signing. Should be a random 32 or 64-byte string. |
| `SESSION_ENCRYPT_KEY` | Secret key used to encrypt session data. Must be 16, 24, or 32 bytes long to select AES-128, AES-192, or AES-256. |
| `ADMIN_PASS` | Password for the administrator page. |
| `SERVER_BASE_URL` | Base URL used to generate QR codes and join links. Example: `http://localhost:8080` |
| `DATABASE_URL` | Connection URL for the PostgreSQL database. Example: `postgres://postgres:postgres@db:5432/kanji-quiz?sslmode=disable` |
