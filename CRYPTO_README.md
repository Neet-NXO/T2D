# T2D 加密说明

T2D 的加密是按方向独立配置的：
- 上行方向使用 `upstream_crypto`
- 下行方向使用 `downstream_crypto`

这两个方向可以使用不同算法、不同密码。

## 支持的加密类型

- `aes-256-gcm`
- `aes-128-gcm`
- `chacha20-poly1305`
- `xchacha20-poly1305`
- `none`
- `xor`

## 密码规则

- `none` / `xor`：密码可省略。
- 其余算法：必须提供 `password`。
- 同一方向上，客户端与服务端必须完全一致（类型+密码）。

## 方向匹配规则

- 客户端 `upstream_crypto` <-> 服务端 `upstream_crypto`
- 客户端 `downstream_crypto` <-> 服务端 `downstream_crypto`

只要同方向匹配即可，不要求上下行必须用同一个算法。

## 常见配置示例

### 示例 1：上下行都用 AES-256-GCM

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

### 示例 2：上下行使用不同算法

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

### 示例 3：测试环境关闭加密

```json
"upstream_crypto": {"type": "none"},
"downstream_crypto": {"type": "none"}
```

## 算法选择建议

- 高安全优先：`aes-256-gcm` / `xchacha20-poly1305`
- 性能与兼容平衡：`aes-128-gcm` / `chacha20-poly1305`
- 调试：`none`
- `xor` 仅是简单混淆，不是安全加密

## 说明（实现层面）

- AEAD 算法每个数据包都会带随机 nonce。
- 每个方向的加密/解密独立执行，不共享状态。
- 程序会在创建加密器时校验必填密码，缺失会直接报错并退出。

## 常见错误与处理

- 报错 `unsupported cipher type`：算法名拼写错误。
- 报错 `password required`：选择了需要密码的算法但未配置密码。
- 连上后大量解密失败：通常是同方向密码或算法不一致。
- 单向正常、反向失败：优先核对下行方向配置是否对齐。
