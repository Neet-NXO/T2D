# T2D-GNET Encryption Features

T2D-GNET supports multiple encryption algorithms to ensure secure data transmission. This document details the supported encryption types, configuration methods, and usage examples.

## Supported Encryption Algorithms

### 1. AES-256-GCM
- **Algorithm**: Advanced Encryption Standard with 256-bit key and Galois/Counter Mode
- **Security Level**: High
- **Performance**: Good
- **Key Length**: 32 bytes
- **Features**: Authenticated encryption, built-in integrity verification

### 2. AES-128-GCM
- **Algorithm**: Advanced Encryption Standard with 128-bit key and Galois/Counter Mode
- **Security Level**: High
- **Performance**: Better than AES-256
- **Key Length**: 16 bytes
- **Features**: Authenticated encryption, built-in integrity verification

### 3. ChaCha20-Poly1305
- **Algorithm**: ChaCha20 stream cipher with Poly1305 authenticator
- **Security Level**: High
- **Performance**: Excellent (especially on mobile devices)
- **Key Length**: 32 bytes
- **Features**: Modern encryption algorithm, resistant to timing attacks

### 4. XChaCha20-Poly1305
- **Algorithm**: Extended ChaCha20 with Poly1305 authenticator
- **Security Level**: High
- **Performance**: Excellent
- **Key Length**: 32 bytes
- **Features**: Extended nonce space, suitable for high-volume data transmission

### 5. No Encryption (none)
- **Algorithm**: Plain text transmission
- **Security Level**: None
- **Performance**: Best
- **Use Case**: Testing or trusted network environments

### 6. XOR Obfuscation (xor)
- **Algorithm**: Simple XOR operation
- **Security Level**: Very Low (obfuscation only)
- **Performance**: Excellent
- **Use Case**: Simple traffic obfuscation, not for security purposes

## Password Requirements

### General Requirements
- **Minimum Length**: 8 characters
- **Recommended Length**: 16+ characters
- **Character Set**: Support all UTF-8 characters
- **Encoding**: UTF-8

### Algorithm-Specific Requirements

| Algorithm | Recommended Password Length | Notes |
|-----------|----------------------------|-------|
| AES-256-GCM | 32+ characters | Longer passwords provide better entropy |
| AES-128-GCM | 16+ characters | Adequate security for most use cases |
| ChaCha20-Poly1305 | 32+ characters | Recommended for high-security scenarios |
| XChaCha20-Poly1305 | 32+ characters | Best for long-running connections |
| XOR | Any length | Security depends entirely on password secrecy |

## Configuration Principles

### Upstream/Downstream Encryption Matching

**Critical**: Client and server encryption configurations must match exactly:

```json
// Client configuration
{
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "client-to-server-key"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "server-to-client-key"
  }
}

// Server configuration (must be identical)
{
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "client-to-server-key"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "server-to-client-key"
  }
}
```

### Bidirectional Independent Encryption

T2D-GNET supports completely independent encryption for upstream and downstream:

- **Upstream**: Client → Server direction
- **Downstream**: Server → Client direction
- **Independence**: Different algorithms, different keys, different security levels

## Usage Examples

### Example 1: High Security Configuration

```json
{
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "AES256-upstream-secret-key-32chars!"
  },
  "downstream_crypto": {
    "type": "xchacha20-poly1305",
    "password": "XChaCha20-downstream-ultra-secure-key"
  }
}
```

**Use Case**: Financial data transmission, sensitive information protection

### Example 2: Performance Optimized Configuration

```json
{
  "upstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "fast-gaming-upstream-key-2024"
  },
  "downstream_crypto": {
    "type": "chacha20-poly1305",
    "password": "fast-gaming-downstream-key-2024"
  }
}
```

**Use Case**: Real-time gaming, video streaming, low-latency applications

### Example 3: Balanced Configuration

```json
{
  "upstream_crypto": {
    "type": "aes-128-gcm",
    "password": "balanced-upstream-key"
  },
  "downstream_crypto": {
    "type": "aes-128-gcm",
    "password": "balanced-downstream-key"
  }
}
```

**Use Case**: General web applications, file transfer, daily use

### Example 4: Development/Testing Configuration

```json
{
  "upstream_crypto": {
    "type": "none"
  },
  "downstream_crypto": {
    "type": "none"
  }
}
```

**Use Case**: Development environment, performance testing, debugging

### Example 5: Simple Obfuscation

```json
{
  "upstream_crypto": {
    "type": "xor",
    "password": "simple-obfuscation-key"
  },
  "downstream_crypto": {
    "type": "xor",
    "password": "simple-obfuscation-key"
  }
}
```

**Use Case**: Basic traffic obfuscation, bypassing simple DPI

## Security Best Practices

### 1. Password Management
- Use strong, unique passwords for each deployment
- Regularly rotate encryption keys
- Store passwords securely (environment variables, key management systems)
- Never hardcode passwords in configuration files for production

### 2. Algorithm Selection
- **High Security**: AES-256-GCM or XChaCha20-Poly1305
- **Balanced**: AES-128-GCM or ChaCha20-Poly1305
- **Performance Critical**: ChaCha20-Poly1305
- **Legacy Support**: AES-128-GCM

### 3. Configuration Security
- Use different passwords for upstream and downstream
- Implement proper file permissions for configuration files
- Consider using environment variables for sensitive data
- Enable logging for security auditing

### 4. Network Security
- Combine with TLS for additional security layers
- Implement proper firewall rules
- Monitor for unusual traffic patterns
- Regular security assessments

## Performance Comparison

| Algorithm | Encryption Speed | Decryption Speed | CPU Usage | Memory Usage |
|-----------|------------------|------------------|-----------|-------------|
| none | N/A | N/A | Minimal | Minimal |
| xor | Excellent | Excellent | Very Low | Very Low |
| ChaCha20-Poly1305 | Excellent | Excellent | Low | Low |
| XChaCha20-Poly1305 | Excellent | Excellent | Low | Low |
| AES-128-GCM | Good | Good | Medium | Medium |
| AES-256-GCM | Good | Good | Medium | Medium |

*Note: Performance may vary based on hardware capabilities (AES-NI support, etc.)*

## Troubleshooting

### Common Encryption Issues

1. **"Decryption failed" Error**
   - Check if client and server passwords match
   - Verify encryption algorithm consistency
   - Ensure configuration file encoding is UTF-8

2. **Performance Degradation**
   - Consider switching to ChaCha20-Poly1305 for better performance
   - Check if hardware supports AES acceleration
   - Monitor CPU usage during encryption operations

3. **Connection Establishment Failure**
   - Verify both upstream and downstream encryption configurations
   - Check for typos in algorithm names
   - Ensure password encoding is consistent

### Debug Configuration

Enable detailed encryption logging:

```json
{
  "log_level": "debug",
  "crypto_debug": true,
  "upstream_crypto": {
    "type": "aes-256-gcm",
    "password": "debug-test-password"
  },
  "downstream_crypto": {
    "type": "aes-256-gcm",
    "password": "debug-test-password"
  }
}
```

## Migration Guide

### Upgrading Encryption

1. **From no encryption to encrypted**:
   - Update both client and server configurations simultaneously
   - Test in development environment first
   - Plan for brief service interruption

2. **Changing encryption algorithms**:
   - Prepare new configuration files
   - Coordinate client and server updates
   - Consider gradual rollout for large deployments

3. **Password rotation**:
   - Generate new strong passwords
   - Update configurations in coordinated manner
   - Verify connectivity after changes

## Advanced Features

### Custom Key Derivation

T2D-GNET uses PBKDF2 for key derivation from passwords:

- **Hash Function**: SHA-256
- **Iterations**: 10,000
- **Salt**: Derived from algorithm type and connection parameters
- **Output Length**: Algorithm-specific key length

### Nonce Management

- **AES-GCM**: 12-byte random nonce per packet
- **ChaCha20-Poly1305**: 12-byte random nonce per packet
- **XChaCha20-Poly1305**: 24-byte random nonce per packet
- **Collision Prevention**: Cryptographically secure random number generator

### Integrity Verification

All authenticated encryption algorithms provide:

- **Data Integrity**: Automatic detection of data corruption
- **Authentication**: Verification of data source
- **Anti-Tampering**: Protection against malicious modifications
- **Replay Protection**: Sequence number verification

---

For more information about T2D-GNET configuration, see [CONFIG_GUIDE_EN.md](CONFIG_GUIDE_EN.md).