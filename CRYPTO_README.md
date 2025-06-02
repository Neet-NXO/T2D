# T2D 加密功能说明

## 概述

T2D 现在支持对TCP连接的上行和下行数据进行加密，提供多种加密算法选择。

## 支持的加密算法

### 1. AES-256-GCM
- **类型**: `aes-256-gcm`
- **密钥长度**: 256位
- **需要密码**: 是
- **特点**: 高安全性，广泛支持

### 2. AES-128-GCM
- **类型**: `aes-128-gcm`
- **密钥长度**: 128位
- **需要密码**: 是
- **特点**: 较快的加密速度

### 3. ChaCha20-Poly1305
- **类型**: `chacha20-poly1305`
- **密钥长度**: 256位
- **需要密码**: 是
- **特点**: 现代流密码，在移动设备上性能优异

### 4. XChaCha20-Poly1305
- **类型**: `xchacha20-poly1305`
- **密钥长度**: 256位
- **需要密码**: 是
- **特点**: 扩展nonce版本，更高安全性

### 5. 无加密
- **类型**: `none`
- **需要密码**: 否
- **特点**: 不进行加密，直接传输

### 6. XOR混淆
- **类型**: `xor`
- **需要密码**: 否
- **特点**: 简单的XOR混淆，仅用于基本的数据混淆

## 重要说明

### 1. 密码要求
- 当选择 `none` 或 `xor` 时，无需提供密码
- 其他所有加密算法都需要提供密码
- 客户端和服务器端必须使用相同的密码

### 2. 上行和下行加密
- **上行加密**: 客户端加密 → 服务器端解密
- **下行加密**: 服务器端加密 → 客户端解密
- 上行和下行可以使用不同的加密算法和密码

### 3. 配置匹配
- 客户端的 `upstream_crypto` 必须与服务器端的 `upstream_crypto` 匹配
- 客户端的 `downstream_crypto` 必须与服务器端的 `downstream_crypto` 匹配

## 使用示例

### 示例1: 使用AES-256-GCM加密

客户端配置:
```json
"upstream_crypto": {
  "type": "aes-256-gcm",
  "password": "my-super-secret-key-2024"
},
"downstream_crypto": {
  "type": "aes-256-gcm",
  "password": "my-super-secret-key-2024"
}
```

服务器端配置:
```json
"upstream_crypto": {
  "type": "aes-256-gcm",
  "password": "my-super-secret-key-2024"
},
"downstream_crypto": {
  "type": "aes-256-gcm",
  "password": "my-super-secret-key-2024"
}
```

### 示例2: 使用ChaCha20-Poly1305加密

```json
"upstream_crypto": {
  "type": "chacha20-poly1305",
  "password": "chacha20-secret-2024"
},
"downstream_crypto": {
  "type": "xchacha20-poly1305",
  "password": "xchacha20-secret-2024"
}
```

### 示例3: 混合配置

```json
"upstream_crypto": {
  "type": "aes-256-gcm",
  "password": "upstream-password"
},
"downstream_crypto": {
  "type": "chacha20-poly1305",
  "password": "downstream-password"
}
```

### 示例4: 无加密

```json
"upstream_crypto": {
  "type": "none"
},
"downstream_crypto": {
  "type": "none"
}
```

## 安全建议

1. **使用强密码**: 建议使用至少32个字符的随机密码
2. **定期更换密码**: 建议定期更换加密密码
3. **选择合适的算法**: 
   - 对于高安全性要求: 推荐 `aes-256-gcm` 或 `xchacha20-poly1305`
   - 对于性能要求: 推荐 `aes-128-gcm` 或 `chacha20-poly1305`
4. **避免使用弱加密**: 生产环境中避免使用 `none` 或 `xor`

## 故障排除

### 常见错误

1. **密码不匹配**: 确保客户端和服务器端使用相同的密码
2. **算法不匹配**: 确保客户端和服务器端使用相同的加密算法
3. **缺少密码**: 使用需要密码的算法时必须提供密码

### 调试建议

1. 检查日志中的加密/解密错误信息
2. 验证配置文件中的加密配置是否正确
3. 确认客户端和服务器端的配置是否匹配