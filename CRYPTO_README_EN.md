# T2D Encryption Guide

T2D encryption is configured independently per direction:
- `upstream_crypto` for upstream traffic
- `downstream_crypto` for downstream traffic

You can use different algorithms and different passwords for each direction.

## Supported Cipher Types

- `aes-256-gcm`
- `aes-128-gcm`
- `chacha20-poly1305`
- `xchacha20-poly1305`
- `none`
- `xor`

## Password Rules

- `none` / `xor`: password is optional.
- Other cipher types: `password` is required.
- Client and server must match exactly per direction (type + password).

## Directional Matching

- Client `upstream_crypto` <-> Server `upstream_crypto`
- Client `downstream_crypto` <-> Server `downstream_crypto`

Matching is directional. Upstream and downstream do not need to use the same cipher.

## Common Config Examples

### Example 1: AES-256-GCM on both directions

```json
"upstream_crypto": {
  "type": "aes-256-gcm",
  "password": "upstream-strong-pass"
},
"downstream_crypto": {
  "type": "aes-256-gcm",
  "password": "downstream-strong-pass"
}
```

### Example 2: Different ciphers per direction

```json
"upstream_crypto": {
  "type": "aes-128-gcm",
  "password": "upstream-pass"
},
"downstream_crypto": {
  "type": "xchacha20-poly1305",
  "password": "downstream-pass"
}
```

### Example 3: No encryption for testing

```json
"upstream_crypto": {"type": "none"},
"downstream_crypto": {"type": "none"}
```

## Cipher Selection Guidance

- Security-first: `aes-256-gcm` / `xchacha20-poly1305`
- Balanced: `aes-128-gcm` / `chacha20-poly1305`
- Debugging: `none`
- `xor` is obfuscation only, not strong security

## Implementation Notes

- AEAD modes use a random nonce per packet.
- Encryption/decryption state is independent per direction.
- Missing required password fails fast at cipher initialization.

## Common Errors

- `unsupported cipher type`: invalid cipher name.
- `password required`: missing password for a required cipher.
- frequent decrypt failures after connect: direction-mismatched cipher or password.
- one-way traffic only: verify downstream crypto alignment first.
