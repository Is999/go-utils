## 5. 加密与解密

> **默认行为与兼容选项**
>
> 1. **新系统首选方案**：对称加密优先使用 **AES-GCM**（带认证机制）；非对称加密优先使用 **RSA-OAEP**（加密）与 **RSA-PSS**
>    （签名）。
> 2. **安全的默认值**：默认已禁用不安全的 **ECB 模式**、**CTR/CFB/OFB 非认证流模式** 以及 **未显式配置 IV 时退回 Key 派生
>    IV** 的行为。
> 3. **兼容旧系统协议**：如需对接历史系统，可通过显式配置 `WithAllowUnsafeECB(true)`、`WithAllowUnsafeStreamMode(true)` 或
>    `WithAllowUnsafeKeyIV(true)` 开启兼容。
> 4. **填充模式选择**：块加密（CBC/ECB）通常搭配 `PKCS7Pad`；旧协议流模式（CTR/CFB/OFB）使用 `NoPad/NoUnpad`，GCM 接口无需填充参数。
> 5. **密钥位数要求**：RSA 生成和导入均要求至少 **2048** 位，摘要算法推荐 SHA256/SHA384/SHA512。

以下为函数内用法片段，省略重复 import；`utils` 为 `github.com/Is999/go-utils`。`key`、`iv`、`data` 由调用方提供，`publicKey`、`privateKey` 表示密钥文件路径。各示例在失败后立即返回。

---

### 5.1 摘要与哈希 (MD5 / SHA)

返回十六进制摘要；摘要本身不提供身份认证，也不适合直接存储密码。MD5/SHA1 仅用于历史协议兼容或非安全哈希标识。

```go
hashMD5 := utils.MD5("data")
hashSHA1 := utils.SHA1("data")
hashSHA256 := utils.SHA256("data")
hashSHA512 := utils.SHA512("data")
fmt.Println(hashMD5, hashSHA1, hashSHA256, hashSHA512)
```

---

### 5.2 AES 与 DES 对称加密

支持 AES (16/24/32 字节密钥) 与 DES/3DES (8/24 字节密钥) 的统一加解密接口。

#### AES-GCM 认证加密（推荐）

```go
a, err := utils.AES(key)
if err != nil {
	fmt.Println(err)
	return
}

// GCM 加密，附加数据 additionalData 可为空
encryptStr, err := a.EncryptGCMString(data, base64.StdEncoding.EncodeToString, []byte("request-id=r-1"))
if err != nil {
	fmt.Println(err)
	return
}

// GCM 解密，附加数据必须与加密时一致
got, err := a.DecryptGCMString(encryptStr, base64.StdEncoding.DecodeString, []byte("request-id=r-1"))
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println(got == data)
```

#### AES-CTR 流模式（旧系统兼容）

```go
// CTR 不提供完整性校验，仅在兼容旧协议时显式开启
a, err := utils.AES(key, utils.WithRandIV(true), utils.WithAllowUnsafeStreamMode(true))
if err != nil {
	fmt.Println(err)
	return
}

encryptStr, err := a.Encrypt(data, utils.CTR, base64.StdEncoding.EncodeToString, utils.NoPad)
if err != nil {
	fmt.Println(err)
	return
}
got, err := a.Decrypt(encryptStr, utils.CTR, base64.StdEncoding.DecodeString, utils.NoUnpad)
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println(got == data)
```

#### AES-CBC 块模式与历史系统兼容

```go
// IV 来自双方约定的协议，长度须为 AES 分组大小（16 字节）。
a, err := utils.AES(key, utils.WithIV(iv))
if err != nil {
	fmt.Println(err)
	return
}

encryptStr, err := a.Encrypt(data, utils.CBC, base64.StdEncoding.EncodeToString, utils.PKCS7Pad)
if err != nil {
	fmt.Println(err)
	return
}
got, err := a.Decrypt(encryptStr, utils.CBC, base64.StdEncoding.DecodeString, utils.PKCS7Unpad)
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println(got == data)
```

#### 开启不安全模式（仅限旧系统兼容）

```go
// 若必须使用 ECB 模式或未指定 IV 时的退回机制
opts := []utils.CipherOption{
	utils.WithAllowUnsafeECB(true),
	utils.WithAllowUnsafeKeyIV(true),
}
a, err := utils.AES(key, opts...)
if err != nil {
	fmt.Println(err)
	return
}
encryptStr, err := a.Encrypt(data, utils.ECB, base64.StdEncoding.EncodeToString, utils.PKCS7Pad)
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println("ciphertext length:", len(encryptStr))
```

---

### 5.3 数据填充 (Pad)

用于将数据补齐至块加密（如 AES-CBC）所需的分组大小。

- `utils.PKCS7Pad` / `utils.PKCS7Unpad`：标准 PKCS#7 填充方案（推荐块加密使用）。
- `utils.ZeroPad` / `utils.ZeroUnpad`：补 0 填充方案（兼容特定旧协议）。
- `utils.NoPad` / `utils.NoUnpad`：不改变内容，直接调用时返回独立副本；作为 `Cipher` 的回调时，部分路径会省去这次复制。

---

### 5.4 RSA 非对称加解密与签名

提供 RSA 密钥生成、加解密与签名验证接口；密钥输入格式见 [README](README.md#53-rsa-非对称加密与解密)。

#### 密钥生成与转换

```go
// 生成 2048 位 RSA 密钥对，默认公钥为 PKIX、私钥为 PKCS#1 格式
files, err := utils.GenerateKeyRSA("./keys", 2048)
if err != nil {
	fmt.Println(err)
	return
}

// 返回路径按公钥、私钥排列。
pubKey, err := os.ReadFile(files[0])
if err != nil {
	fmt.Println(err)
	return
}
pubKeyStr := utils.RemovePEMHeaders(string(pubKey))
fmt.Println("public key body length:", len(pubKeyStr))
```

#### 加解密 (OAEP / PKCS1v15)

```go
r, err := utils.NewRSA(publicKey, privateKey, utils.WithRSAFilePath(true))
if err != nil {
	fmt.Println(err)
	return
}

// 推荐使用 OAEP 方案加密对称密钥或小段数据
cipherText, err := r.EncryptOAEP(data, base64.StdEncoding.EncodeToString, sha256.New())
if err != nil {
	fmt.Println(err)
	return
}
plainText, err := r.DecryptOAEP(cipherText, base64.StdEncoding.DecodeString, sha256.New())
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println(plainText == data)
```

#### 签名与验签 (PSS / PKCS1v15)

```go
r, err := utils.NewRSA(publicKey, privateKey, utils.WithRSAFilePath(true))
if err != nil {
	fmt.Println(err)
	return
}

sign, err := r.SignPSS(data, crypto.SHA256, base64.StdEncoding.EncodeToString, nil)
if err != nil {
	fmt.Println(err)
	return
}

// 使用与签名时相同的正文、摘要算法和 PSS 选项验签。
err = r.VerifyPSS(data, sign, crypto.SHA256, base64.StdEncoding.DecodeString, nil)
if err != nil {
	fmt.Println(err)
	return
}
fmt.Println("signature verified")
```

---

### 5.5 加密模块性能验证命令

使用已有基准记录加解密耗时与分配；比较时保持 Go 版本、硬件和系统负载一致。

```bash
# 对称加密基准测试（对比 CBC、CTR 兼容模式与 GCM）
go test -run '^$' -bench 'BenchmarkCipherAES(CBCEncrypt|CBCDecrypt|CTRNoPadEncrypt|CTRNoPadDecrypt|GCMEncrypt|GCMDecrypt)$' -benchmem ./...

# RSA 基准测试（对比 OAEP 与 PSS）
go test -run '^$' -bench 'BenchmarkRSA(EncryptOAEP|DecryptOAEP|SignPSS|VerifyPSS)$' -benchmem ./...

# 全量竞态条件验证
go test -race ./...
```
