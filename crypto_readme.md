## 5. 加密与解密

> **安全与最佳实践清单（Best Practices Checklist）**
>
> 1. **新系统首选方案**：对称加密优先使用 **AES-GCM**（带认证机制）；非对称加密优先使用 **RSA-OAEP**（加密）与 **RSA-PSS**
     （签名）。
> 2. **安全的默认值**：默认已禁用不安全的 **ECB 模式**、**CTR/CFB/OFB 非认证流模式** 以及 **未显式配置 IV 时退回 Key 派生
     IV** 的行为。
> 3. **兼容旧系统协议**：如需对接历史系统，可通过显式配置 `WithAllowUnsafeECB(true)`、`WithAllowUnsafeStreamMode(true)` 或
     `WithAllowUnsafeKeyIV(true)` 开启兼容。
> 4. **填充模式选择**：块加密（CBC/ECB）需搭配 `Pkcs7Padding` 或 `ZeroPadding`；旧协议流模式（CTR/CFB/OFB）和 GCM 无需补位，请搭配
     `NoPadding/NoUnPadding` 避免额外开销。
> 5. **密钥位数要求**：生产环境 RSA 密钥必须至少 **2048** 位，摘要算法推荐 SHA256/SHA384/SHA512。

---

### 5.1 摘要与哈希 (MD5 / SHA)

提供基础的单向散列函数，适用于签名或校验文件完整性，不建议用于存储密码（建议使用 bcrypt）。

```go
hashMD5 := utils.Md5("data")
hashSHA1 := utils.Sha1("data")
hashSHA256 := utils.Sha256("data")
hashSHA512 := utils.Sha512("data")
```

---

### 5.2 AES 与 DES 对称加密

支持 AES (16/24/32 字节密钥) 与 DES/3DES (8/24 字节密钥) 的统一加解密接口。

#### AES-GCM 认证加密（推荐）

```go
// 实例化 AES
a, err := utils.AES(key)

// GCM 加密，附加数据 additionalData 可为空
encryptStr, err := a.EncryptGCMString(data, base64.StdEncoding.EncodeToString, []byte("request-id=r-1"))

// GCM 解密，附加数据必须与加密时一致
got, err := a.DecryptGCMString(encryptStr, base64.StdEncoding.DecodeString, []byte("request-id=r-1"))
```

#### AES-CTR 流模式（旧系统兼容）

```go
// CTR 不提供完整性校验，仅在兼容旧协议时显式开启
a, err := utils.AES(key, utils.WithRandIV(true), utils.WithAllowUnsafeStreamMode(true))

encryptStr, err := a.Encrypt(data, utils.CTR, base64.StdEncoding.EncodeToString, utils.NoPadding)
got, err := a.Decrypt(encryptStr, utils.CTR, base64.StdEncoding.DecodeString, utils.NoUnPadding)
```

#### AES-CBC 块模式与历史系统兼容

```go
// CBC 模式搭配固定 IV，使用 Pkcs7Padding 填充
a, err := utils.AES(key, utils.WithIV(iv))

encryptStr, err := a.Encrypt(data, utils.CBC, base64.StdEncoding.EncodeToString, utils.Pkcs7Padding)
got, err := a.Decrypt(encryptStr, utils.CBC, base64.StdEncoding.DecodeString, utils.Pkcs7UnPadding)
```

#### 开启不安全模式（仅限旧系统兼容）

```go
// 若必须使用 ECB 模式或未指定 IV 时的退回机制
opts := []utils.CipherOption{
utils.WithAllowUnsafeECB(true),
utils.WithAllowUnsafeKeyIV(true),
}
a, err := utils.AES(key, opts...)
encryptStr, err := a.Encrypt(data, utils.ECB, base64.StdEncoding.EncodeToString, utils.Pkcs7Padding)
```

---

### 5.3 数据填充 (Padding)

用于将数据补齐至块加密（如 AES-CBC）所需的分组大小。

- `utils.Pkcs7Padding` / `utils.Pkcs7UnPadding`：标准 PKCS#7 填充方案（推荐块加密使用）。
- `utils.ZeroPadding` / `utils.ZeroUnPadding`：补 0 填充方案（兼容特定旧协议）。
- `utils.NoPadding` / `utils.NoUnPadding`：不进行任何填充，无额外内存拷贝（GCM 或旧协议流模式使用；CTR/CFB/OFB 不提供完整性校验）。

---

### 5.4 RSA 非对称加解密与签名

提供生产级的 RSA 密钥生成、加解密与签名验证支持。

#### 密钥生成与转换

```go
// 生成 2048 位以上的 RSA 密钥对，默认使用 PKCS8(公)/PKCS1(私) 格式
files, err := utils.GenerateKeyRSA("./keys", 2048)

// 支持移除或补充 PEM 标头，方便在 JSON/DB 中存储
pubKeyStr := utils.RemovePEMHeaders(rawPubKey)
```

#### 加解密 (OAEP / PKCS1v15)

```go
r, err := utils.NewRSA(publicKey, privateKey, utils.WithRSAFilePath(true))

// 推荐使用 OAEP 方案加密对称密钥或小段数据
cipherText, err := r.EncryptOAEP(data, base64.StdEncoding.EncodeToString, sha256.New())
plainText, err := r.DecryptOAEP(cipherText, base64.StdEncoding.DecodeString, sha256.New())
```

#### 签名与验签 (PSS / PKCS1v15)

```go
// 推荐使用 PSS 方案结合 SHA256 进行签名
sign, err := privRsa.SignPSS(data, crypto.SHA256, base64.StdEncoding.EncodeToString, nil)

// 验证 PSS 签名
err = pubRsa.VerifyPSS(data, sign, crypto.SHA256, base64.StdEncoding.DecodeString, nil)
```

---

### 5.5 加密模块性能验证命令

通过本地压测验证各项加密机制的吞吐与分配开销。

```bash
# 对称加密基准测试（对比 CBC、CTR 兼容模式与 GCM）
go test -run '^$' -bench 'BenchmarkCipherAES(CBCEncrypt|CBCDecrypt|CTRNoPaddingEncrypt|CTRNoPaddingDecrypt|GCMEncrypt|GCMDecrypt)$' -benchmem ./...

# RSA 基准测试（对比 OAEP 与 PSS）
go test -run '^$' -bench 'BenchmarkRSA(EncryptOAEP|DecryptOAEP|SignPSS|VerifyPSS)$' -benchmem ./...

# 全量竞态条件验证
go test -race ./...
```
