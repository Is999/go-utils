# go-utils

#### 介绍

golang 帮助函数

#### 安装教程

1. github安装 go get -u github.com/Is999/go-utils

2. gitee安装 go get -u gitee.com/Is999/go-utils

### 使用说明

1. utils包中代码仅供参考，造成损失概不负责。
2. 版本要求golang 1.22

### 历史变更

1. 版本要求golang 1.18变更为1.22
2. 移除了1.21 版本前的Max、Min 两个函数，推荐使用golang 内置函数 max、min
3. 移除了1.21 版本前的Logger 文件，使用标准库中 log/slog
4. Curl 和 Response 记录日志方式使用了标准库 log/slog记录日志
5. 根据1.21版本 log/slog 增加了errors文件，实现了LogValuer 接口，对error的日志追踪
6. utils中返回的error 统一使用了WrapError, 支持error记录追踪
7. RSA加密解密增加了对长文本的支持，增加了对PEM key 去除头尾标记和还原头尾标记方法
8. mathd/rand 改1.22版本 mathd/rand/v2, 部分函数形参 rand.Source 改为*rand.Rand

# Go常用标准库方法及utils包帮助函数

>
开发中使用频率较高的Go标准库中的方法及utils包中的帮助方法。ustils包中的方法都可以在单元测试中找到使用方法示例。版本要求 >=
1.22版本。

> 注意：utils包中代码仅供参考，不建议用于商业生产，造成损失概不负责。

## 1. 字符串

​        **strings** 和 **bytes** 两个包对字符串的操作基本相同拥有**相同的方法名称和参数**，只是参数类型的不同。

------

### 1.1 截取字符串

> **推荐**：包含中文（宽字符）时，使用 string转rune切片截取字符串。

------

#### type	string

```go
string[start: end]
```

| 参数       | 描述                                                   |
|----------|------------------------------------------------------|
| *string* | 原字符串。                                                |
| *start*  | 表示要截取的第一个字符所在的索引（截取时包含该字符）。如果不指定，默认为 0，也就是从字符串的开头截取。 |
| *end*    | 表示要截取的最后一个字符所在的索引（截取时不包含该字符）。如果不指定，默认为字符串的长度。        |

------

#### func    [utils.Substr](https://github.com/Is999/go-utils/blob/master/string.go#L48)

```go
func Substr(str string, start, length int) string
```

| 参数       | 描述                                                                                                                                                                                                                                    |
|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| *str*    | 原字符串。                                                                                                                                                                                                                                 |
| *start*  | 截取的起始位置，即截取的第一个字符所在的索引：<br />start小于0时，start = len(str) + start                                                                                                                                                                       |
| *length* | 截取的截止位置，即截取的最后一个字符所在的索引：<br />length大于0时，length表示为截取子字符串的**长度**，截取的最后一个字符所在的索引，值为：**start + length** 。<br />length小于0时，length表示为截取的最后一个字符所在的**索引**，值为：**len(str) + length + 1** 。例如：等于 **-1** 时，表示截取到最后一个字符；等于 **-2** 时，表示截取到倒数第二个字符。 |

备注：Substr 内部实现string转rune切片。

```go
// string转rune切片
runes := []rune(str)
string(runes[start: end])
```

------

### 1.2 拼接字符串

> **推荐**：strings.Builder+预设大小的方式拼接字符串。


------

#### struct	strings.Builder

```go
var b strings.Builder
b.Grow(lenth)     // 预设大小
b.WriteString(s1) // 写入字符串s1
b.WriteString(s2) // 写入字符串s2
s3 := b.String() // 拼接后的字符串
```

------

#### func	fmt.Sprintf

```go
var str = fmt.Sprintf("%s%d%s", s1, i, s2)
```

------

#### func	strings.Join

```go
var str []string = []string{s1, s2}
s := strings.Join(str, "")
```

------

#### struct	bytes.Buffer

```go
var bt bytes.Buffer
bt.WriteString(s1)
bt.WriteString(s2)
s3 := bt.String()
```

------

### 1.3 获取字符串长度

> **推荐**：包含宽字符（宽字符算一个长度）时，使用utf8.RuneCount、 utf8.RuneCountInString获取长度。

------

#### func	len

```go
len(str)
```

------

#### func	utf8.RuneCount

```go
func RuneCount(p []byte) int
```

------

#### func	utf8.RuneCountInString

```go
func RuneCountInString(s string) (n int)
```

------

#### func	bytes.Count

```go
func Count(s, sep []byte) int
```

| 参数    | 描述           |
|-------|--------------|
| *str* | 要获取长度的字符串。   |
| *sep* | 分隔符，一般传 nil。 |

备注：利用 sep 长度等于0的逻辑获取字符串长度；sep长度等于0内部调用 utf8.RuneCount(s) + 1，所以返回的**长度需要减 1**。

```go
lenth := bytes.Count([]byte(str), nil)) - 1
```

------

#### func	strings.Count

```go
func Count(s, substr string) int
```

| 参数       | 描述         |
|----------|------------|
| *str*    | 要获取长度的字符串。 |
| *substr* | 子串，传入空即可。  |

备注：利用 substr 长度等于0的逻辑获取字符串长度；substr长度等于0内部调用 utf8.RuneCountInString(s) + 1，所以返回的**长度需要减
1**。

```go
lenth := strings.Count(str, "") - 1
```

------

### 1.4 分割字符串

分割字符串我们可以分为几种情况，分别为：按空格分割、按子字符串分割和按字符分割。

------

#### func	strings.Fields

```go
func Fields(s string) []string
```

备注：按**空格分割**字符串

------

#### func	strings.FieldsFunc

```go
func FieldsFunc(s string, f func (rune) bool) []string 
```

备注：按**字符分割**字符串

------

#### func	strings.Split

```go
func Split(s, sep string) []string
```

| 参数    | 描述       |
|-------|----------|
| *s*   | 要分割的字符串。 |
| *sep* | 字符串的分割符。 |

备注：按**子字符串分割**字符串

```go
// 按空格(空字符串)分割
strings.Split(s, " ")
// 按xxx字符串分割
strings.Split(s, "xxx")
```

------

### 1.5 统计子串在字符串中出现的次数

------

#### func	strings.Count

```go
func Count(s, substr string) int
```

| 参数       | 描述       |
|----------|----------|
| *s*      | 原字符串。    |
| *substr* | 要检索的字符串。 |

------

### 1.6 查找子串在字符串中出现位置

------

#### func	strings.Index

```go
func Index(s, substr string) int
```

| 参数       | 描述       |
|----------|----------|
| *s*      | 原字符串。    |
| *substr* | 要检索的字符串。 |

备注：返回的是**字符串第一次出现的位置**。查找不存在的字符串返回 **-1**。

------

#### func	stringsLastIndex

```go
func LastIndex(s, substr string) int
```

| 参数       | 描述       |
|----------|----------|
| *s*      | 原字符串。    |
| *substr* | 要检索的字符串。 |

备注：返回的是**字符串最后一次出现的位置**。查找不存在的字符串返回 **-1**。

------

#### func	strings.IndexAny

```go
func IndexAny(s, chars string) int
```

| 参数      | 描述        |
|---------|-----------|
| *s*     | 原字符串。     |
| *chars* | 要检索的字符序列。 |

备注：返回**第一次出现字符序列的索引**；反之，则返回 **-1**。

------

#### func	strings.LastIndexAny

```go
func LastIndexAny(s, chars string) int
```

| 参数      | 描述        |
|---------|-----------|
| *s*     | 原字符串。     |
| *chars* | 要检索的字符序列。 |

备注：返回**最后一次出现字符序列的索引**；反之，则返回 **-1**。

------

#### func	strings.IndexByte

```go
func IndexByte(s string, c byte) int
```

| 参数  | 描述        |
|-----|-----------|
| *s* | 原字符串。     |
| *c* | 表示要检索的字符。 |

备注：返回**第一次出现字符的索引**；反之，则返回 **-1**。

------

#### func	strings.LastIndexByte

```go
func LastIndexByte(s string, c byte) int
```

| 参数  | 描述        |
|-----|-----------|
| *s* | 原字符串。     |
| *r* | 表示要检索的字符。 |

备注：返回**最后一次出现字符的索引**；反之，则返回 **-1**。

------

#### func	strings.IndexRune

```go
func IndexRune(s string, r rune) int
```

| 参数  | 描述        |
|-----|-----------|
| *s* | 原字符串。     |
| *r* | 表示要检索的字符。 |

备注：返回**第一次出现字符的索引**；反之，则返回 **-1**。

------

#### func	strings.LastIndexRune

```go
func LastIndexRune(s string, r rune) int
```

| 参数  | 描述        |
|-----|-----------|
| *s* | 原字符串。     |
| *c* | 表示要检索的字符。 |

备注：返回**最后一次出现字符的索引**；反之，则返回 **-1**。

------

#### func	strings.IndexFunc

```go
func IndexFunc(s string, f func (rune) bool) int
```

| 参数  | 描述               |
|-----|------------------|
| *s* | 原字符串。            |
| *f* | 表示要检索的字符的条件判断函数。 |

备注：返回**第一次出现字符的索引**；反之，则返回 **-1**。

------

#### func	strings.LastIndexFunc

```go
func LastIndexFunc(s string, f func (rune) bool) int
```

| 参数  | 描述               |
|-----|------------------|
| *s* | 原字符串。            |
| *f* | 表示要检索的字符的条件判断函数。 |

备注：返回**最后一次出现字符的索引**；反之，则返回 **-1**。

------

### 1.7 判断字符串是否包含子串

------

#### func	strings.HasPrefix

```go
func HasPrefix(s, prefix string) bool
```

| 参数       | 描述      |
|----------|---------|
| *s*      | 原字符串。   |
| *prefix* | 要检索的子串。 |

备注：检索**字符串s**是否以指定**字符串prefix开头**，如果是返回 **True**；反之返回 **False**。

------

#### func	strings.HasSuffix

```go
func HasSuffix(s, suffix string) bool
```

| 参数       | 描述      |
|----------|---------|
| *s*      | 原字符串。   |
| *suffix* | 要检索的子串。 |

备注：检索**字符串s**是否以指定**字符串suffix结尾**，如果是返回 **True**；反之返回 **False**。

------

#### func	strings.Contains

```go
func Contains(s, substr string) bool
```

| 参数       | 描述      |
|----------|---------|
| *s*      | 原字符串。   |
| *substr* | 要检索的子串。 |

备注：检索**字符串s**是否包含**字符串substr**。函数内部实现 **strings.Index >= 0** 。

------

#### func	strings.ContainsRune

```go
func ContainsRune(s string, r rune) bool
```

| 参数  | 描述        |
|-----|-----------|
| *s* | 原字符串。     |
| *r* | 表示要检索的字符。 |

备注：检索**字符串s**是否包含**字符r**。函数内部实现 **strings.IndexRune >= 0** 。

------

#### func	strings.ContainsAny

```go
func ContainsAny(s, chars string) bool
```

| 参数      | 描述         |
|---------|------------|
| *s*     | 原字符串。      |
| *chars* | 表示要检索的字符串。 |

备注：检索**字符串s**是否包含**字符串chars**。函数内部实现 **strings.IndexAny >= 0** 。

------

#### func	strings.Cut

```go
func Cut(s, sep string) (before, after string, found bool)
```

| 参数    | 描述      |
|-------|---------|
| *s*   | 原字符串。   |
| *sep* | 查找的字符串。 |

备注：在**字符串s**中找到**字符串sep**, 并返回**字符串sep前后部分**以及**是否找到字符串sep**。

------

### 1.8 字符串大小写转换

------

#### func	strings.ToTitle

```go
func ToTitle(s string) string 
```

备注：将字符串**首字母转成大写**。

------

#### func	strings.ToLower

```go
func ToLower(s string) string
```

备注：将**字符串转成小写**。

------

#### func	strings.ToUpper

```go
func ToUpper(s string) string
```

备注：将**字符串转成大写**。

------

### 1.9 去除字符串指定字符

------

#### func	strings.TrimSpace

```go
func TrimSpace(s string) string
```

备注：将**字符串左右两边的空格去除**。

------

#### func	strings.Trim

```go
func Trim(s string, cutset string) string
```

| 参数       | 描述        |
|----------|-----------|
| *s*      | 原字符串。     |
| *cutset* | 需要去除的字符串。 |

备注：将**字符串左右两边的指定字符串 cutset 去除**。

------

#### func	strings.TrimLeft

```go
func TrimLeft(s, cutset string) string
```

| 参数       | 描述        |
|----------|-----------|
| *s*      | 原字符串。     |
| *cutset* | 需要去除的字符串。 |

备注：将**字符串左边的指定字符串 cutset 去除**。

------

#### func	strings.TrimRight

```go
func TrimRight(s, cutset string) string
```

| 参数       | 描述        |
|----------|-----------|
| *s*      | 原字符串。     |
| *cutset* | 需要去除的字符串。 |

备注：将**字符串右边的指定字符串 cutset 去除**。

------

#### func	strings.TrimPrefix

```go
TrimPrefix(s, prefix string) string
```

| 参数       | 描述          |
|----------|-------------|
| *s*      | 原字符串。       |
| *prefix* | 需要去除的前缀字符串。 |

备注：**去除字符串的前缀prefix**。

------

#### func	strings.TrimSuffix

```go
func TrimSuffix(s, suffix string) string
```

| 参数       | 描述          |
|----------|-------------|
| *s*      | 原字符串。       |
| *suffix* | 需要去除的后缀字符串。 |

备注：**去除字符串的后缀suffix**。

------

#### func	strings.TrimFunc

```go
func TrimFunc(s string, f func (rune) bool) string 
```

| 参数  | 描述             |
|-----|----------------|
| *s* | 原字符串。          |
| *f* | 需要去除的字符串的规则函数。 |

备注：将**字符串左右两边符合函数 f 规则字符串去除**。函数 f，接受一个 **rune** 类型的参数，返回一个 **bool** 类型的变量，如果函数
f 返回 **true**，那说明符合规则，**字符将被移除**。

------

#### func	strings.TrimLeftFunc

```go
func TrimLeftFunc(s string, f func (rune) bool) string
```

| 参数  | 描述             |
|-----|----------------|
| *s* | 原字符串。          |
| *f* | 需要去除的字符串的规则函数。 |

备注：将**字符串左边符合函数 f 规则字符串去除**。函数 f，接受一个 **rune** 类型的参数，返回一个 **bool** 类型的变量，如果函数
f 返回 **true**，那说明符合规则，**字符将被移除**。

------

#### func	strings.TrimRightFunc

```go
func TrimRightFunc(s string, f func (rune) bool) string
```

| 参数  | 描述             |
|-----|----------------|
| *s* | 原字符串。          |
| *f* | 需要去除的字符串的规则函数。 |

备注：将**字符串右边符合函数 f 规则字符串去除**。函数 f，接受一个 **rune** 类型的参数，返回一个 **bool** 类型的变量，如果函数
f 返回 **true**，那说明符合规则，**字符将被移除**。

------

### 1.10 字符串遍历处理

------

#### func	strings.Map

```go
func Map(mapping func (rune) rune, s string) string
```

| 参数        | 描述               |
|-----------|------------------|
| *mapping* | 对字符串中每一个字符的处理函数。 |
| *s*       | 原字符串。            |

备注：对字符串 s 中的每一个字符都做 mapping 处理。

------

### 1.11 字符串比较

#### func	strings.Compare

```go
func Compare(a, b string) int
```

备注：比较字符串 a 和字符串 b 是否相等，如果 a > b，返回一个大于 0 的数，如果 a == b，返回 0，否则，返回负数。

------

#### func	strings.EqualFold

```go
func EqualFold(s, t string) bool
```

备注：比较字符串 s 和字符串 t 是否相等，如果相等，返回 true，否则，返回 false。该函数比较字符串大小是忽略大小写的。

------

### 1.12 字符串重复指定次数

------

#### func	strings.Repeat

```go
func Repeat(s string, count int) string
```

| 参数      | 描述      |
|---------|---------|
| *s*     | 原字符串。   |
| *count* | 要重复的次数。 |

备注：将字符串s重复count次。

------

### 1.13 字符串替换

------

#### func	strings.Replace

```go
func Replace(s, old, new string, n int) string
```

| 参数    | 描述                                      |
|-------|-----------------------------------------|
| *s*   | 原字符串。                                   |
| *old* | 要替换的字符串。                                |
| *new* | 替换成什么字符串。                               |
| *n*   | 要替换的次数，-1，那么就会将字符串 s 中的所有的 old 替换成 new。 |

备注：将字符串 s 中的 old 字符串替换成 new 字符串，替换 n 次，返回替换后的字符串。如果 n 是 -1，那么就会将字符串 s 中的所有的
old 替换成 new。

------

#### func	strings.ReplaceAll

```go
func ReplaceAll(s, old, new string) string
```

| 参数    | 描述        |
|-------|-----------|
| *s*   | 原字符串。     |
| *old* | 要替换的字符串。  |
| *new* | 替换成什么字符串。 |

备注：将字符串 s 中的 old 字符串全部替换成 new 字符串，返回替换后的字符串。

------

#### struct	strings.Replacer

```go
strings.NewReplacer(oldnew...).Replace(s)
```

备注：字符串替换

------

#### func    [utils.Replace](https://github.com/Is999/go-utils/blob/master/string.go#L27)

```go
func Replace(s string, oldnew map[string]string) string 
```

| 参数       | 描述                                        |
|----------|-------------------------------------------|
| *s*      | 原字符串。                                     |
| *oldnew* | 替换规则，map类型， map的键为要替换的字符串，map的值为替换成什么字符串。 |

备注：内部实现 strings.NewReplacer(oldnew...).Replace(s)， 将map 类型 oldnew 转换成了切片， 使用map类型更加直观。

------

### 1.14 字符串克隆

------

#### func	strings.Clone

```go
func Clone(s string) string
```

备注：返回 s 的新副本。

------

### 1.15 字符串反转

------

#### func    [utils.StrRev](https://github.com/Is999/go-utils/blob/master/string.go#L88)

```go
func StrRev(str string) string
```

备注：将支付串str反转

------

### 1.16 生成随机字符串

------

#### func    [utils.UniqId](https://github.com/Is999/go-utils/blob/master/string.go#L141)

```go
func UniqId(l uint8, r ...*rand.Rand) string
```

| 参数  | 描述                                          |
|-----|---------------------------------------------|
| *l* | 生成字符串的长度。                                   |
| *r* | 随机种子 utils.RandPool()：批量生成时传入r参数可提升生成随机数效率。 |

备注：生成一个长度范围16-32位的唯一ID字符串(可排序的字符串)，UniqId函数生成字符串并不保证唯一性。

------

#### func    [utils.RandStr](https://github.com/Is999/go-utils/blob/master/string.go#L96)

```go
func RandStr(n int, r ...*rand.Rand) string
```

| 参数  | 描述                                          |
|-----|---------------------------------------------|
| *n* | 生成字符串的长度。                                   |
| *r* | 随机种子 utils.RandPool()：批量生成时传入r参数可提升生成随机数效率。 |

备注：随机生成字符串 LETTERS。LETTERS 值为：a-zA-z

------

#### func    [utils.RandStr2](https://github.com/Is999/go-utils/blob/master/string.go#L104)

```go
func RandStr2(n int, r ...*rand.Rand) string
```

| 参数  | 描述                                          |
|-----|---------------------------------------------|
| *n* | 生成字符串的长度。                                   |
| *r* | 随机种子 utils.RandPool()：批量生成时传入r参数可提升生成随机数效率。 |

备注：随机生成字符串 ALPHANUM。ALPHANUM 值为：0-9a-zA-z

------

#### func    [utils.RandStr3](https://github.com/Is999/go-utils/blob/master/string.go#L120)

```go
func RandStr3(n int, alpha string, r ...*rand.Rand) string
```

| 参数      | 描述                                          |
|---------|---------------------------------------------|
| *n*     | 生成字符串的长度。                                   |
| *alpha* | 生成随机字符串的种子。                                 |
| *r*     | 随机种子 utils.RandPool()：批量生成时传入r参数可提升生成随机数效率。 |

备注：随机生成字符串。alpha 指定生成随机字符串的种子。

------

### 1.17 字符串 Read

------

#### struct	Reader

```go
NewReader(s string).Read((b []byte)
```

备注：实现了read接口。

------

### 1.18 字符转义

------

#### func	html.EscapeString

```go
func EscapeString(s string) string
```

备注：将html文本中的字符转换为实体字符。

------

#### func	html.UnescapeString

```go
func UnescapeString(s string) string
```

备注：将实体字符转换为可编译的html字符。

------

#### func	url.QueryEscape

```go
func QueryEscape(s string) string
```

备注：将URL中的字符进行转义。

------

#### func	url.QueryUnescape

```go
func QueryUnescape(s string) (string, error)
```

备注：将URL中的转义字符转换为对应的字符。

------

## 2. 编码与解码

------

### 2.1 JSON编码解码

------

#### func	json.Marshal

```go
func Marshal(v any) ([]byte, error)
```

备注：对数据进行 JSON 编码

------

#### func	json.Unmarshal

```go
func Unmarshal(data []byte, v any) error
```

备注：对数据进行 JSON 解码

------

### 2.2 BASE64编码解码

------

#### 文本数据进行 base64 编码

```go
base64.StdEncoding.EncodeToString(src)
```

#### 文本数据进行 base64 解码

```go
base64.StdEncoding.DecodeString(s)
```

#### URL或文件名进行 base64 编码

```go
base64.URLEncoding.EncodeToString(src)
```

#### URL或文件名进行 base64 解码

```go
base64.URLEncoding.DecodeString(s)
```

------

## 3. MATH 函数

------

### 3.1 math函数

------

#### func	math.Pow

```go
func Pow(x, y float64) float64
```

备注：返回x的y次方。

------

#### func	math.Abs

```go
func Abs(x float64) float64
```

备注：返回x的绝对值。

------

#### func	math.Round

```go
func Round(x float64) float64
```

备注：返回最接近的整数，从零四舍五入。

------

#### func	math.Ceil

```go
func Ceil(x float64) float64
```

备注：返回x的上舍入值。

------

#### func	math.Floor

```go
func Floor(x float64) float64
```

备注：返回x的下舍入值。

------

#### func	math.Mod

```go
func Mod(x, y float64) float64
```

备注：返回x/y的余数。

------

#### func	math.Max

```go
func Max(x, y float64) float64
```

备注：返回x和y中最大值。

------

#### func	math.Min

```go
func Min(x, y float64) float64
```

备注：返回x和y中最小值。

------

### 3.2 utils函数

------

#### func    [utils.Rand](https://github.com/Is999/go-utils/blob/master/math.go#L14)

```go
func Rand(min, max int64, r ...*rand.Rand) int64 
```

| 参数    | 描述                                          |
|-------|---------------------------------------------|
| *min* | 最小值。                                        |
| *max* | 最大值。                                        |
| *r*   | 随机种子 utils.RandPool()：批量生成时传入r参数可提升生成随机数效率。 |

备注：返回min~max之间的随机数，值可能包含min和max。

------

#### func    [utils.Round](https://github.com/Is999/go-utils/blob/master/math.go#L30)

```go
func Round(value float64, precision int) float64
```

| 参数          | 描述     |
|-------------|--------|
| *num*       | 原数值。   |
| *precision* | 保留小数位。 |

备注：对num进行四舍五入，并保留指定小数位。

------

## 4. 文件

------

### 4.1 获取文件信息

------

#### func	os.Stat

```go
func Stat(name string) (FileInfo, error)
```

备注：获取名为name的文件或目录信息。

------

### 4.2 创建与移除

------

#### func	os.Mkdir

```go
func Mkdir(name string, perm FileMode) error
```

备注：创建目录。

------

#### func	os.Remove

```go
func Remove(name string) error
```

备注：删除名为name的文件。

------

### 4.3 打开与创建

------

#### func	os.OpenFile

```go
func OpenFile(name string, flag int, perm FileMode) (*File, error)
```

备注：打开名为name的文件。

------

#### func	os.Create

```go
func Create(name string) (*File, error)
```

备注：创建名为name的文件。

------

### 4.4 读取与写入

------

#### func	os.ReadFile

```go
func ReadFile(name string) ([]byte, error)
```

备注：读取名为name的文件内容。

------

#### func	os.WriteFile

```go
func WriteFile(name string, data []byte, perm FileMode) error
```

备注：将data数据写入name文件。

------

### 4.5 重命名与移动

#### func	os.Rename

```go
func Rename(oldpath, newpath string) error
```

备注：重命名文件(夹)oldpath为newpath，或移动文件。

------

### 4.6 获取目录路径

------

#### func	os.Getwd

```go
func Getwd() (dir string, err error)
```

备注：取得当前工作目录的根路径。

------

#### func	filepath.Abs

```go
func Abs(path string) (string, error)
```

备注：获取path的绝对路径。

------

#### func	filepath.IsAbs

```go
func IsAbs(path string) bool
```

备注：判断path路径是否是绝对路径。

------

#### func	filepath.Rel

```go
func Rel(basepath, targpath string) (string, error)
```

备注：返回一个相对路径。

------

#### func	filepath.Clean

```go
func Clean(path string) string
```

备注：返回相path的最短路径名。

------

### 4.7 权限

------

#### func	os.Chmod

```go
 func Chmod(name string, mode FileMode) error
```

备注：改变文件(夹)name的权限。

------

#### func	os.Chown

```go
  func Chown(name string, uid, gid int) error
```

备注： 改变文件的所有者。

------

### 4.8 解析路径名

------

#### func	filepath.Ext

```go
  func Ext(path string) string
```

备注：返回path文件扩展名。

------

#### func	filepath.Base

```go
  func Base(path string) string
```

备注： (path为一个文件路径)可获取path中的文件名。

------

#### func	filepath.Dir

```go
  func Dir(path string) string
```

备注：获取path路径的目录。

------

### 4.9 路径的切分和拼接

------

#### func	filepath.Split

```go
  func Split(path string) (dir, file string)
```

备注：将path路径分成dir目录和file文件。

------

#### func	filepath.Join

```go
  func Join(elem ...string) string
```

备注：Join函数可以将任意数量的路径元素放入一个单一路径里。

------

### 4.10  路径判断

------

#### func    [utils.IsDir](https://github.com/Is999/go-utils/blob/master/file.go#L22)

```go
func IsDir(path string) bool
```

备注：判断给定路径是否是一个目录。

------

#### func    [utils.IsFile](https://github.com/Is999/go-utils/blob/master/file.go#L31)

```go
func IsFile(filepath string) bool
```

备注：判断给定的文件路径名是否是一个文件。

------

#### func    [utils.IsExist](https://github.com/Is999/go-utils/blob/master/file.go#L36)

```go
func IsExist(path string) bool
```

备注：判断一个文件（夹）是否存在。

------

### 4.11 获取文件大小

------

#### func    [utils.Size](https://github.com/Is999/go-utils/blob/master/file.go#L42)

```go
func Size(filepath string) (int64, error) 
```

备注：取得文件大小。

------

#### func    [utils.SizeFormat](https://github.com/Is999/go-utils/blob/master/file.go#L401)

```go
func SizeFormat(size int64, decimals uint) string 
```

| 参数         | 描述            |
|------------|---------------|
| *size*     | 文件实际大小(Byte)。 |
| *decimals* | 保留几位小数。       |

备注：文件大小格式化已可读式显示文件大小。

------

### 4.12 复制文件

------

#### func    [utils.Copy](https://github.com/Is999/go-utils/blob/master/file.go#L54)

```go
func Copy(src, dst string) error
```

| 参数    | 描述      |
|-------|---------|
| *src* | 拷贝的原文件。 |
| *dst* | 拷贝后的文件。 |

备注：拷贝文件。

------

### 4.13 获取目录文件

------

#### func    [utils.FindFiles](https://github.com/Is999/go-utils/blob/master/file.go#L101)

```go
func FindFiles(path string, depth bool, match ...string) (files []FileInfo, err error)
```

| 参数      | 描述                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
|---------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| *path*  | 目录路径。                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| *depth* | 深度查找: true 采用filepath.WalkDir遍历; false 只在当前目录查找。                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| *match* | 匹配规则:<br />   - `无参` : 匹配所有文件名 FindFiles(path, depth)<br />   - `*`   : 匹配所有文件名 FindFiles(path, depth, `*`) <br />  - `文件完整名`      : 精准匹配文件名 FindFiles(path, depth, fullFileName) <br />  - `e`, `文件完整名` : 精准匹配文件名 FindFiles(path, depth, `e`, fullFileName) <br />  - `p`, `文件前缀名` : 匹配前缀文件名 FindFiles(path, depth, `p`, fileNamePrefix)<br />  - `s`, `文件后缀名` : 匹配后缀文件名 FindFiles(path, depth, `s`, fileNameSuffix) <br />  - `r`, `正则表达式` : 正则匹配文件名 FindFiles(path, depth, `r`, fileNameReg) |

备注：获取目录下所有匹配文件。

------

### 4.14 内容读取

------

#### func    [utils.Scan](https://github.com/Is999/go-utils/blob/master/file.go#L230)

```go
func Scan(r io.Reader, handle ReadScan, size ...int) error
```

| 参数       | 描述                                                                                                                                                                    |
|----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| *r*      | 实现io.Reader接口。                                                                                                                                                        |
| *handle* | func(num int, line []byte, err error) error 函数。<br /> num 行号: 当前扫描到第几行<br /> line 行数据: 当前扫描的行数据<br /> err 扫描错误信息<br /> error 处理错误信息: 返回的 error == DONE 代表正确处理完数据并终止扫描 |
| *size*   | 设置Scanner.maxTokenSize 的大小(默认值: 64*1024): 单行内容大于该值则无法读取                                                                                                               |

备注：使用scan扫描文件每一行数据。

------

#### func    [utils.Line](https://github.com/Is999/go-utils/blob/master/file.go#L256)

```go
func Line(r io.Reader, handle ReadLine) error
```

| 参数       | 描述                                                                                                                                                                                                                                 |
|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| *r*      | 实现io.Reader接口。                                                                                                                                                                                                                     |
| *handle* | func(num int, line []byte, lineDone bool) error 函数。<br /> num 行号: 当前扫描到第几行<br /> line 行数据: 当前扫描的行数据<br /> lineDone 当前行(num)数据是否读取完毕: true 当前行(num)数据读取完毕; false 当前行(num)数据未读完<br /> error 处理错误信息: 返回的 error == DONE 代表正确处理完数据并终止扫描 |

备注：使用scan扫描文件每一行数据。

------

#### func    [utils.Read](https://github.com/Is999/go-utils/blob/master/file.go#L286)

```go
func Read(r io.Reader, handle ReadBlock) error
```

| 参数       | 描述                                                                                                                                 |
|----------|------------------------------------------------------------------------------------------------------------------------------------|
| *r*      | 实现io.Reader接口。                                                                                                                     |
| *handle* | func(size int, block []byte) error 函数。<br /> size 读取的数据块大小<br /> block 读取的数据块<br /> error 处理错误信息: 返回的 error == DONE 代表正确处理完数据并终止扫描 |

备注：使用scan扫描文件每一行数据。

------

### 4.15 写入内容到文件

------

#### func    [utils.WriteFile](https://github.com/Is999/go-utils/blob/master/file.go#L316)

```go
func NewWrite(fileName string, isAppend bool, perm ...os.FileMode) (*WriteFile, error)
```

| 参数         | 描述                                                 |
|------------|----------------------------------------------------|
| *fileName* | 文件路径名。                                             |
| *isAppend* | 是否追加文件数据: true 每次写入数据在文件末尾追加数据; false 打开文件时会先清除数据。 |
| *perm*     | 文件权限: 默认权限 文件夹0744, 文件0644                         |

备注：返回一个WriteFile实例。

```go
// 实例化一个WriteFile
w, err := NewWrite(fileName, isAppend, perm)
if err != nil {
fmt.Errorf("Write() error = %v", err)
return
}

// 关闭文件
defer func () {
if err := w.Close(); err != nil {
fmt.Errorf("Close() err %v", err)
}
}()

// 写入byte数据
n, err := w.Write([]byte{})

// 写入string
n, err := w.WriteString(string)

// 使用bufio写入数据
_, err := w.WriteBuf(func (write *bufio.Writer) (int, error) {
for j := 0; j < 10000; j++ {
_, err := write.WriteString(fmt.Sprintf("WriteBuf %d Name %v; 红酥肯放琼苞碎。探著南枝开遍未。不知酝藉几多香，但见包藏无限意。道人憔悴春窗底。闷损阑干愁不倚。要来小酌便来休，未必明朝风不起。\n", j, tt.name))
if err != nil {
return 0, err
}
}
return 0, nil
})
```

备注：WriteFile实例化，文件关闭，数据写入。

------

### 4.16 获取文件类型

------

#### func    [utils.FileType](https://github.com/Is999/go-utils/blob/master/file.go#L435)

```go
func FileType(f *os.File) (string, error)
```

备注：获取文件类型

------

## 5. 加密与解密

------

### 5.1 MD5 加密

------

#### func    [utils.Md5](https://github.com/Is999/go-utils/blob/master/md5.go#L9)

```go
func Md5(str string) string
```

备注：md5加密

------

### 5.2 SHA 加密

------

#### func    [utils.Sha1](https://github.com/Is999/go-utils/blob/master/sha.go#L11)

```go
func Sha1(str string) string
```

备注：sha1加密

------

#### func    [utils.Sha256](https://github.com/Is999/go-utils/blob/master/sha.go#L17)

```go
func Sha256(str string) string
```

备注：sha256加密

------

#### func    [utils.Sha512](https://github.com/Is999/go-utils/blob/master/sha.go#L23)

```go
func Sha512(str string) string
```

备注：sha512加密

------

### 5.3 RSA 非对称加密与解密

------

#### func    [utils.GenerateKeyRSA](https://github.com/Is999/go-utils/blob/master/rsa.go#L381)

```go
func GenerateKeyRSA(path string, bits int, pkcs ...bool) ([]string, error)
```

| 参数     | 描述                                                                                                                                      |
|--------|-----------------------------------------------------------------------------------------------------------------------------------------|
| *path* | 文件名路径。                                                                                                                                  |
| *bits* | 生成秘钥位大小: 512、1024、2048、4096。                                                                                                            |
| *pkcs* | 秘钥格式, 默认格式(公钥PKCS8格式 私钥PKCS1格式):<br />   - pkcs[0] isPubPKCS8 公钥是否是PKCS8格式: 默认 true <br />   - pkcs[1] isPriPKCS1 私钥是否是PKCS1格式: 默认 true |

备注：生成秘钥，默认格式(公钥PKCS8格式 私钥PKCS1格式)。返回两个文件名, 第一个公钥文件名, 第二个私钥文件名。

------

#### RSA [加密与解密](https://github.com/Is999/go-utils/blob/master/rsa.go#L166)

```go
// 实例化RSA，并设置key
r, err := NewRSA(publicKey, privateKey, isFilePath)
if err != nil {
fmt.Errorf("NewRSA() err = %v", err)
return
}

// 源数据
marshal, err := json.Marshal(map[string]interface{}{
"Title":   tt.name,
"Content": strings.Repeat("测试内容8282@334&-", 1024) + tt.name,
})

// 公钥加密 PKCS1v15
encodeString, err := r.Encrypt(string(marshal), base64.StdEncoding.EncodeToString)
if err != nil {
fmt.Errorf("Encrypt() err = %v", err)
return
}

// 私钥解密 PKCS1v15
decryptString, err := r.Decrypt(encodeString, base64.StdEncoding.DecodeString)
if err != nil {
fmt.Errorf("Decrypt() err = %v", err)
return
}

// 公钥加密 OAEP
encodeString, err = r.EncryptOAEP(string(marshal), base64.StdEncoding.EncodeToString, sha256.New())
if err != nil {
fmt.Errorf("Encrypt() err = %v", err)
return
}

// 私钥解密 OAEP
decryptString, err = r.DecryptOAEP(encodeString, base64.StdEncoding.DecodeString, sha256.New())
if err != nil {
fmt.Errorf("Decrypt() err = %v", err)
return
}
```

备注：先实例化RSA 设置公钥私钥，使用公钥加密数据， 私钥解密数据。

------

#### RSA [签名与验签](https://github.com/Is999/go-utils/blob/master/rsa.go#L229)

```go
// 实例化RSA，并设置key
r, err := NewRSA(publicKey, privateKey, isFilePath)
if err != nil {
fmt.Errorf("NewRSA() err = %v", err)
return
}

// 源数据
marshal, err := json.Marshal(map[string]interface{}{
"Title":   tt.name,
"Content": strings.Repeat("测试内容8282@334&-", 1024) + tt.name,
})

// 私钥签名 PKCS1v15
sign, err := r.Sign(string(marshal), crypto.SHA256, base64.StdEncoding.EncodeToString)
if err != nil {
fmt.Errorf("Sign() err = %v", err)
return
}

// 公钥验签 PKCS1v15
if err := r.Verify(string(marshal), sign, crypto.SHA256, base64.StdEncoding.DecodeString); err != nil {
fmt.Errorf("Verify() err = %v", err)
return
} else {
fmt.Log("Verify() = 验证成功")
}

// 私钥签名 PSS
sign, err = privRsa.SignPSS(string(marshal), crypto.SHA256, base64.StdEncoding.EncodeToString, nil)
if err != nil {
fmt.Errorf("Sign() err = %v", err)
return
}

// 公钥验签 PSS
if err := pubRsa.VerifyPSS(string(marshal), sign, crypto.SHA256, base64.StdEncoding.DecodeString, nil); err != nil {
fmt.Errorf("Verify() err = %v", err)
return
} else {
fmt.Log("Verify() = 验证成功")
}
```

备注：先实例化RSA 设置公钥私钥，使用私钥签名，公钥验签。

------

#### RSA [秘钥格式转换](https://github.com/Is999/go-utils/blob/master/rsa.go#L478)

```go
// 读取公钥文件内容
pub, err := os.ReadFile(pubFile)
if err != nil {
t.Errorf("ReadFile() WrapError = %v", err)
}

//fmt.Println("公钥 %s", string(pub))
rPub := utils.RemovePEMHeaders(string(pub))
//fmt.Println("remove 公钥 %s", rPub)
aPub := utils.AddPEMHeaders(rPub, "public")
//fmt.Println("add 公钥 %s %v", aPub, strings.EqualFold(aPub, strings.TrimSpace(string(pub))))
if !strings.EqualFold(aPub, strings.TrimSpace(string(pub))) {
fmt.Errorf("转换后的公钥与原始公钥不相等")
}

// 读取私钥文件内容
pri, err := os.ReadFile(priFile)
if err != nil {
t.Errorf("ReadFile() WrapError = %v", err)
}
//fmt.Println("私钥 %s", string(pri))
rPri := utils.RemovePEMHeaders(string(pri))
//fmt.Println("remove 私钥 %s", rPri)
aPri := utils.AddPEMHeaders(rPri, "private")
//fmt.Println("add 私钥 %s %v", aPri, strings.EqualFold(aPri, strings.TrimSpace(string(pri))))
if !strings.EqualFold(aPri, strings.TrimSpace(string(pri))) {
fmt.Errorf("转换后的私钥与原始私钥不相等")
}
```

------

### 5.4 AES 加密与解密

------

#### AES [加密与解密](https://github.com/Is999/go-utils/blob/master/aes.go#L12)

```go
// 实例化AES，并设置key
a, err := AES(key, false)
if err != nil {
fmt.Errorf("NewAES() error = %v", err)
return
}

// 设置iv
if err := a.SetIv(iv); err != nil {
fmt.Errorf("SetIv() error = %v", err)
return
}

// 加密数据
encryptStr, err := a.Encrypt(data, CBC, base64.StdEncoding.EncodeToString, Pkcs7Padding)
if err != nil {
fmt.Errorf("Encrypt() mode = %v error = %v", CBC, err)
return
}

// 解密数据
got, err := a.Decrypt(encryptStr, CBC, base64.StdEncoding.DecodeString, Pkcs7UnPadding)
if err != nil {
fmt.Errorf("Decrypt() mode = %v error = %v", CBC, err)
return
}
```

备注：先实例化AES 设置key，更新需要设置IV，之后便可加密或解密数据。

------

### 5.5 DES 加密与解密

------

#### DES [加密与解密](https://github.com/Is999/go-utils/blob/master/des.go#L12)

```go
// 实例化DES，并设置key
a, err := DES(key, false)
if err != nil {
fmt.Errorf("NewAES() error = %v", err)
return
}

// 设置iv
if err := a.SetIv(iv); err != nil {
fmt.Errorf("SetIv() error = %v", err)
return
}

// 加密数据
encryptStr, err := a.Encrypt(data, MCRYPT_MODE_CBC, base64.StdEncoding.EncodeToString, Pkcs7Padding)
if err != nil {
fmt.Errorf("Encrypt() mode = %v error = %v", CBC, err)
return
}

// 解密数据
got, err := a.Decrypt(encryptStr, MCRYPT_MODE_CBC, base64.StdEncoding.DecodeString, Pkcs7UnPadding)
if err != nil {
fmt.Errorf("Decrypt() mode = %v error = %v", CBC, err)
return
}
```

备注：先实例化DES 设置key，更新需要设置IV，之后便可加密或解密数据。

------

### 5.6 pkcs7 填充与反填充

------

#### func    [utils.Pkcs7Padding](https://github.com/Is999/go-utils/blob/master/pkcs7.go#L9)

```go
func Pkcs7Padding(data []byte, blockSize int) []byte
```

备注：数据填充。

------

#### func    [utils.Pkcs7UnPadding](https://github.com/Is999/go-utils/blob/master/pkcs7.go#L18)

```go
func Pkcs7UnPadding(data []byte) ([]byte, error) 
```

备注：数据反填充。

------

### 5.7 0 填充与反填充

------

#### func    [utils.ZeroPadding](https://github.com/Is999/go-utils/blob/master/zero.go#L9)

```go
func ZeroPadding(data []byte, blockSize int) []byte 
```

备注：数据填充。

------

#### func    [utils.ZeroUnPadding](https://github.com/Is999/go-utils/blob/master/zero.go#L18)

```go
func ZeroUnPadding(data []byte) ([]byte, error)
```

备注：数据反填充。

------

## 6. 整形、字符串转换

------

### 6.1 字符串转整形

------

#### func	strconv.ParseInt

```go
func ParseInt(s string, base int, bitSize int) (i int64, err error)
```

备注：string 转 int64。

------

#### func	strconv.Atoi

```go
func Atoi(s string) (int, error)
```

备注：string 转 int。

------

#### func    [utils.Str2Int64](https://github.com/Is999/go-utils/blob/master/strconv.go#L12)

```go
func Str2Int64(s string) (i int64)
```

备注：string 转 int64，转换失败返回零值。

------

#### func    [utils.Str2Int](https://github.com/Is999/go-utils/blob/master/strconv.go#L6)

```go
func Str2Int(s string) (i int) 
```

备注：string 转 int，转换失败返回零值。

------

### 6.2 整形转字符串

------

#### func	strconv.FormatInt

```go
func FormatInt(i int64, base int) string
```

备注：int64 转 string。

------

#### func	strconv.Itoa

```go
func Itoa(i int) string
```

备注：int 转 string。

------

### 6.3 字符串转浮点数

------

#### func	strconv.ParseFloat

```go
func ParseFloat(s string, bitSize int) (float64, error) 
```

备注：string 转 float64。

------

#### func    [utils.Str2Float](https://github.com/Is999/go-utils/blob/master/strconv.go#L18)

```go
func Str2Float(s string) (i float64)
```

备注：string 转 float64，失败返回零值。

------

### 6.4 浮点数转字符串

------

#### func	strconv.FormatFloat

```go
func FormatFloat(f float64, fmt byte, prec, bitSize int) string
```

备注：float64 转 string。

------

### 6.5 以千位分隔符方式格式化一个数字

------

#### func    [utils.NumberFormat](https://github.com/Is999/go-utils/blob/master/misce.go#L26)

```go
func NumberFormat(number float64, decimals uint, decPoint, thousandsSep string) string 
```

| 参数           | 描述        |
|--------------|-----------|
| *number*     | 需要格式化的数字。 |
| *decimals*   | 保留几位小数。   |
| *decPoint*   | 小数点[.]    |
| thousandsSep | 千位分隔符[,]  |

备注：以千位分隔符方式格式化一个数字。

------

### 6.6 进制转换

------

#### func    [utils.BinOct](https://github.com/Is999/go-utils/blob/master/strconv.go#L27)

```go
func BinOct(str string) (string, error)
```

备注：二进制转换为八进制。

------

#### func    [utils.BinDec](https://github.com/Is999/go-utils/blob/master/strconv.go#L36)

```go
func BinDec(str string) (int64, error)
```

备注：二进制转换为十进制。

------

#### func    [utils.BinHex](https://github.com/Is999/go-utils/blob/master/strconv.go#L41)

```go
func BinHex(str string) (string, error)
```

备注：二进制转换为十六进制。

------

#### func    [utils.OctBin](https://github.com/Is999/go-utils/blob/master/strconv.go#L50)

```go
func OctBin(data string) (string, error)
```

备注：八进制转换为二进制。

------

#### func    [utils.OctDec](https://github.com/Is999/go-utils/blob/master/strconv.go#L59)

```go
func OctDec(str string) (int64, error)
```

备注：八进制转换为十进制。

------

#### func    [utils.OctHex](https://github.com/Is999/go-utils/blob/master/strconv.go#L64)

```go
func OctHex(data string) (string, error)
```

备注：八进制转换为十六进制。

------

#### func    [utils.DecBin](https://github.com/Is999/go-utils/blob/master/strconv.go#L73)

```go
func DecBin(number int64) string
```

备注：十进制转换为二进制。

------

#### func    [utils.DecOct](https://github.com/Is999/go-utils/blob/master/strconv.go#L78)

```go
func DecOct(number int64) string
```

备注：十进制转换为八进制。

------

#### func    [utils.DecHex](https://github.com/Is999/go-utils/blob/master/strconv.go#L83)

```go
func DecHex(number int64) string
```

备注：十进制转换为十六进制。

------

#### func    [utils.HexBin](https://github.com/Is999/go-utils/blob/master/strconv.go#L88)

```go
func HexBin(data string) (string, error) 
```

备注：十六进制转换为二进制。

------

#### func    [utils.HexOct](https://github.com/Is999/go-utils/blob/master/strconv.go#L97)

```go
func HexOct(str string) (string, error)
```

备注：十六进制转换为八进制。

------

#### func    [utils.HexDec](https://github.com/Is999/go-utils/blob/master/strconv.go#L106)

```go
func HexDec(str string) (int64, error)
```

备注：十六进制转换为十进制。

------

## 7. 数组/切片/链表

------

### 7.1 检查数组中是否存在某个值

------

#### func    [utils.IsHas](https://github.com/Is999/go-utils/blob/master/slices.go#L4)

```go
func IsHas[T Ordered](v T, s []T) bool
```

备注：检查s中是否存在v。1.21版本以上推荐使用标准库 slices.Contains(s,v)

------

### 7.2 统计某个值在数组中出现次数

------

#### func    [utils.HasCount](https://github.com/Is999/go-utils/blob/master/slices.go#L14)

```go
func HasCount[T Ordered](v T, s []T) (count int) 
```

备注：统计v在s中出现次数。

------

### 7.3 反转数组

------

#### func    [utils.Reverse](https://github.com/Is999/go-utils/blob/master/slices.go#L24)

```go
func Reverse[T Ordered](s []T) []T
```

备注：反转s。1.21版本以上推荐使用标准库 slices.Reverse(s)

------

### 7.4 去除数组中重复的值

------

#### func    [utils.Unique](https://github.com/Is999/go-utils/blob/master/slices.go#L32)

```go
func Unique[T Ordered](s []T) []T
```

备注：去除s中重复的值。

------

### 7.5 计算两个数组的差集

------

#### func    [utils.Diff](https://github.com/Is999/go-utils/blob/master/slices.go#L49)

```go
func Diff[T Ordered](s1, s2 []T) []T
```

备注：计算s1与s2的差集。

------

### 7.6 计算两个数组的交集

------

#### func    [utils.Intersect](https://github.com/Is999/go-utils/blob/master/slices.go#L64)

```go
func Intersect[T Ordered](s1, s2 []T) []T
```

备注：计算s1与s2的交集。

------

### 7.7 列表 status container/list.List

------

#### 创建列表

```go
// 通过 list.New 创建列表
l := list.New()

// 添加元素
l.PushBack("5")
l.PushBack("6")

// 列表遍历
for i := l.Front(); i != nil; i = i.Next() {
fmt.Println("Element =", i.Value)
}
```

输出：

```tex
Element = 5
Element = 6
```

------

#### 在列表头部插入元素

```go
// 在列表头部插入
ele4 := l.PushFront("4")
```

输出：

```tex
Element = 4
Element = 5
Element = 6
```

------

#### 在列表尾部插元素

```go
// 在列表尾部插
ele8 := l.PushBack("8")
```

输出：

```tex
Element = 4
Element = 5
Element = 6
Element = 8
```

------

#### 在列表指定元素前插入

```go
// 在指定元素ele4前插入元素"3"
ele3 := l.InsertBefore("3", ele4)
```

输出：

```tex
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
```

------

#### 在列表指定元素后插入

```go
// 在指定元素ele8后插入元素"9"
ele9 := l.InsertAfter("9", ele8)
```

输出：

```tex
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
```

------

#### 获取列表头结点

```go
// 获取列表头结点
front := l.Front()
fmt.Println("front =", front.Value)
```

输出：

```tex
front = 3
```

------

#### 获取列表尾结点

```go
// 获取列表尾结点
back := l.Back()
fmt.Println("back =", back.Value)
```

输出：

```tex
back = 9
```

------

#### 获取上一个结点

```go
// 获取ele4上一个结点
prev := ele4.Prev()
fmt.Println("ele4 prev =", prev.Value)
```

输出：

```tex
ele4 prev = 3
```

------

#### 获取下一个结点

```go
// 获取ele4下一个结点
next := ele4.Next()
fmt.Println("ele4 next =", next.Value)
```

输出：

```tex
ele4 next = 5
```

------

#### 移动到某元素的前面

```go
// 将ele4元素移动到back元素的前面
l.MoveBefore(ele4, back)
```

输出：

```tex
移动前：
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
--------------
移动后：
Element = 3
Element = 5
Element = 6
Element = 8
Element = 4
Element = 9
```

------

#### 移动到某元素的后面

```go
// 将ele4元素移动到back元素的后面
l.MoveAfter(ele4, back)
```

输出：

```tex
移动前：
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
--------------
移动后：
Element = 3
Element = 5
Element = 6
Element = 8
Element = 9
Element = 4
```

------

#### 移动到列表的最前面

```go
// 将ele9元素移动到列表的最前面
l.MoveToFront(ele9)
```

输出：

```tex
移动前：
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
--------------
移动后：
Element = 9
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
```

------

#### 移动到列表的最面

```go
// 将ele3元素移动到列表的最后面
l.MoveToFront(ele3)
```

输出：

```tex
移动前：
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
--------------
移动后：
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
```

------

#### 列表正序遍历

```go
// 列表正序遍历
for i := l.Front(); i != nil; i = i.Next() {
fmt.Println("Element =", i.Value)
}
```

输出：

```tex
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
```

------

#### 列表倒叙遍历

```go
// 列表倒叙遍历
for i := l.Back(); i != nil; i = i.Prev() {
fmt.Println("Element =", i.Value)
}
```

输出：

```tex
Element = 9
Element = 8
Element = 6
Element = 5
Element = 4
Element = 3
```

------

#### 在列表中删除元素

```go
// 在列表中删除ele4元素
l.Remove(ele4)
// 在列表中删除ele8元素
l.Remove(ele8)
```

输出：

```tex
删除前：
Element = 3
Element = 4
Element = 5
Element = 6
Element = 8
Element = 9
--------------
删除后：
Element = 3
Element = 5
Element = 6
Element = 9
```

------

#### 在列表头部插入一个列表

```go
// 创建一个列表（头部列表）
frontList := list.New()
frontList.PushBack("f1")
frontList.PushBack("f2")

// 在列表头部插入一个列表
l.PushFrontList(frontList)
```

输出：

```tex
添加前：
Element = 3
Element = 4
--------------
添加后：
Element = f1
Element = f2
Element = 3
Element = 4
```

------

#### 在列表尾部插入一个列表

```go
// 创建一个列表（尾部列表）
backList := list.New()
backList.PushBack("b1")
backList.PushBack("b2")

// 在列表尾部插入一个列表
l.PushBackList(backList)
```

输出：

```tex
添加前：
Element = 3
Element = 4
--------------
添加后：
Element = 3
Element = 4
Element = b1
Element = b2
```

------

#### 获取列表长度

```go
l.Len()
```

------

#### 初始化或清除列表

```go
l.Init()
```

------

## 8. map/syncMap

------

### 8.1 获取map的所有key

------

#### func    [utils.MapKeys](https://github.com/Is999/go-utils/blob/master/map.go#L8)

```go
func MapKeys[K Ordered, V any](m map[K]V) []K 
```

备注：获取map所有的key

------

### 8.2 有序获取map的所有value

------

#### func    [utils.MapValues](https://github.com/Is999/go-utils/blob/master/map.go#L19)

```go
func MapValues[K Ordered, V any](m map[K]V, isReverse ...bool) []V 
```

| 参数          | 描述                       |
|-------------|--------------------------|
| *m*         | map。                     |
| *isReverse* | 是否降序排列：true 降序，false 升序。 |

备注：对map的key排序并按排序后的key返回其value

------

### 8.3 有序遍历map元素

------

#### func    [utils.MapRange](https://github.com/Is999/go-utils/blob/master/map.go#L40)

```go
func MapRange[K Ordered, V any](m map[K]V, f func (key K, value V) bool, isReverse ...bool)
```

| 参数          | 描述                                          |
|-------------|---------------------------------------------|
| *m*         | map。                                        |
| *f*         | f 函数接收key与value，返回一个bool值，如果f函数返回false则终止遍历 |
| *isReverse* | 是否降序排列：true 降序，false 升序。                    |

备注：对map的key排序并按排序后的key遍历map的元素，如果f 函数返回 false则终止遍历。

------

### 8.4 过滤map的元素

------

#### func    [utils.MapFilter](https://github.com/Is999/go-utils/blob/master/map.go#L60)

```go
func MapFilter[K Ordered, V any](m map[K]V, f func (key K, val V) bool) map[K]V 
```

| 参数  | 描述                                                   |
|-----|------------------------------------------------------|
| *m* | map。                                                 |
| *f* | f 函数接收key与value，返回一个bool值，如果f函数返回false则过滤掉该元素（删除该元素） |

备注：使用回调函数过滤map的元素，如果f 函数返回 false则过滤掉该元素（删除该元素）。

------

### 8.5 计算两个map的差集

------

#### func    [utils.MapDiff](https://github.com/Is999/go-utils/blob/master/map.go#L70)

```go
func MapDiff[K, V Ordered](m1, m2 map[K]V) []V
```

备注：计算m1与m2的值差集。

------

#### func    [utils.MapDiffKey](https://github.com/Is999/go-utils/blob/master/map.go#L96)

```go
func MapDiffKey[K Ordered, V any](m1, m2 map[K]V) []K 
```

备注：计算m1与m2的键差集。

------

### 8.6 计算两个map的交集

------

#### func    [utils.MapIntersect](https://github.com/Is999/go-utils/blob/master/map.go#L83)

```go
func MapIntersect[K, V Ordered](m1, m2 map[K]V) []V
```

备注：计算m1与m2的值交集。

------

#### func    [utils.MapIntersectKey](https://github.com/Is999/go-utils/blob/master/map.go#L107)

```go
func MapIntersectKey[K Ordered, V any](m1, m2 map[K]V) []K
```

备注：计算m1与m2的键交集。

------

### 8.7 sync.Map

------

#### 创建sync.Map

```go
//sync.Map 声明完之后，可以立即使用
var m sync.Map
m.Store("Server", "Golang")
m.Store("JavaScript", "Vue")
```

备注：创建map。

------

#### 添加元素 Store

```go
func (m *Map) Store(key, value interface{})
```

备注：向 map 中存入键为 key，值为 value 的键值对，这里的 key 和 value 都是 *
*[interface](https://haicoder.net/golang/golang-interface.html)** 类型的，因此 key 和 value 可以存入任意的类型。

------

#### 获取元素 Load

```go
func (m *Map) Load(key interface{}) (value interface{}, ok bool)
```

备注：返回的 value 是 interface 类型的，因此 value 我们不可以直接使用，而必须要转换之后才可以使用，返回的ok是 bool
值，表明获取是否成功。

------

#### 获取或添加 LoadOrStore

```go
func (m *Map) LoadOrStore(key, value interface{}) (actual interface{}, loaded bool) 
```

备注：获取的 key 存在，返回 key 对应的元素，如果获取的 key 不存在，就返回设置的值，并且将设置的值，存入 map。

------

#### 删除元素 Delete

```go
func (m *Map) Delete(key interface{})
```

备注：删除元素，使用 sync.Map Delete 删除不存在的元素，不会报错。

------

#### 获取并删除 LoadAndDelete

```go
func (m *Map) LoadAndDelete(key any) (value any, loaded bool)
```

备注：如果key存在返回key的值并删除该key。

------

#### 遍历元素 Range

```go
func (m *Map) Range(f func (key, value interface{}) bool)
```

备注：遍历元素，如果f 函数返回 false则终止遍历。

------

## 9. 时间

------

###       

### 9.1 时区

------

#### func    [utils.Local](https://github.com/Is999/go-utils/blob/master/time.go#L11)

```go
func Local() *time.Location
```

备注：系统运行时区。

------

#### func    [utils.CST](https://github.com/Is999/go-utils/blob/master/time.go#L16)

```go
func CST() *time.Location 
```

备注：东八时区。

------

#### func    [utils.UTC](https://github.com/Is999/go-utils/blob/master/time.go#L21)

```go
func UTC() *time.Location 
```

备注：UTC时区。

------

### 9.2  验证日期：年、月、日

------

#### func    [utils.CheckDate](https://github.com/Is999/go-utils/blob/master/time.go#L84)

```go
func CheckDate(year, month, day int) bool
```

| 参数      | 描述  |
|---------|-----|
| *year*  | 年份。 |
| *month* | 月份。 |
| day     | 日期。 |

备注：验证日期：年、月、日。

------

### 9.3  获取指定月份有多少天

------

#### func    [utils.MonthDay](https://github.com/Is999/go-utils/blob/master/time.go#L66)

```go
func MonthDay(year int, month int) (days int)
```

| 参数      | 描述  |
|---------|-----|
| *year*  | 年份。 |
| *month* | 月份。 |

备注：获取指定月份有多少天。

------

### 9.4  增加时间

------

#### func    [utils.AddTime](https://github.com/Is999/go-utils/blob/master/time.go#L111)

```go
func AddTime(t time.Time, addTimes ...string) (time.Time, error)
```

| 参数         | 描述                                   |
|------------|--------------------------------------|
| *addTimes* | 增加时间（Y年，M月，D日，H时，I分，S秒，L毫秒，C微妙，N纳秒)。 |

备注：增加时间。

------

### 9.5  获取日期信息

------

#### func    [utils.DateInfo](https://github.com/Is999/go-utils/blob/master/time.go#L155)

```go
func DateInfo(t time.Time) map[string]interface{}
```

备注：获取日期信息。

```
//	返回：year int - 年，
//		month int - 月，monthEn string - 英文月，
//		day int - 日，yearDay int - 一年中第几日， weekDay int - 一周中第几日，
//		hour int - 时，hour int - 分，second int - 秒，
//		millisecond int - 毫秒，microsecond int - 微妙，nanosecond int - 纳秒，
//		unix int64 - 时间戳-秒，unixNano int64 - 时间戳-纳秒，
//		weekDay int - 星期几，weekDayEn string - 星期几英文， yearWeek int - 一年中第几周，
//		date string - 格式化日期，dateNs string - 格式化日期（纳秒)
```

------

### 9.6  时间格式化为时间字符串

------

#### func    [utils.TimeFormat](https://github.com/Is999/go-utils/blob/master/time.go#L204)

```go
func TimeFormat(timeZone *time.Location, layout string, timestamp ...int64) string
```

| 参数          | 描述                  |
|-------------|---------------------|
| *timeZone*  | 时区。                 |
| layout      | 格式化。                |
| *timestamp* | Unix 时间sec秒和nsec纳秒。 |

备注：时间格式化为时间字符串。

------

### 9.7  解析时间字符串为time.Time

------

#### func    [utils.TimeParse](https://github.com/Is999/go-utils/blob/master/time.go#L227)

```go
func TimeParse(timeZone *time.Location, layout, timeStr string) (time.Time, error)
```

| 参数         | 描述     |
|------------|--------|
| *timeZone* | 时区。    |
| layout     | 格式化。   |
| *timeStr*  | 时间字符串。 |

备注：解析时间字符串为time.Time。

------

### 9.8  两个时间字符串判断

------

#### func    [utils.Before](https://github.com/Is999/go-utils/blob/master/time.go#L327)

```go
func Before(layout string, t1, t2 string) (bool, error) 
```

| 参数     | 描述     |
|--------|--------|
| layout | 格式化。   |
| *t1*   | 时间字符串。 |
| t2     | 时间字符串。 |

备注：返回true: t1在t2之前（t1小于t2），返回false: t1大于等于t2。

------

#### func    [utils.After](https://github.com/Is999/go-utils/blob/master/time.go#L340)

```go
func After(layout string, t1, t2 string) (bool, error) 
```

| 参数     | 描述     |
|--------|--------|
| layout | 格式化。   |
| *t1*   | 时间字符串。 |
| t2     | 时间字符串。 |

备注：返回true: t1在t2之后（t1大于t2），返回false: t1小于等于t2。

------

#### func    [utils.Equal](https://github.com/Is999/go-utils/blob/master/time.go#L353)

```go
func Equal(layout string, t1, t2 string) (bool, error)
```

| 参数     | 描述     |
|--------|--------|
| layout | 格式化。   |
| *t1*   | 时间字符串。 |
| t2     | 时间字符串。 |

备注：判断t1是否与t2相等。

------

### 9.9  求两个时间字符串的时间差

------

#### func    [utils.Sub](https://github.com/Is999/go-utils/blob/master/time.go#L366)

```go
func Sub(layout string, t1, t2 string) (int, error)
```

| 参数     | 描述     |
|--------|--------|
| layout | 格式化。   |
| *t1*   | 时间字符串。 |
| t2     | 时间字符串。 |

备注：t1与t2的时间差，t1>t2 结果大于0，否则结果小于等于0。

------

#### func	time.Since

```go
func Since(t Time) Duration
```

示例，程序运行时间：

```go
// 开始时间
start := time.Now()

// 业务逻辑
time.Sleep(time.Second)

// 求时间差
diff := time.Since(start)
fmt.Printf("运行时间：%v", diff) // 运行时间：1.001284412s
```

备注：求某个时间与现在的时间差。

------

## 10. 函数验证、正则验证

------

#### func    [utils.Empty](https://github.com/Is999/go-utils/blob/master/regexp.go#L12)

```go
func Empty(value string) bool 
```

备注：空字符串验证。

------

#### func    [utils.QQ](https://github.com/Is999/go-utils/blob/master/regexp.go#L20)

```go
func QQ(value string) bool
```

备注：QQ号验证。

------

#### func    [utils.Email](https://github.com/Is999/go-utils/blob/master/regexp.go#L26)

```go
func Email(value string) bool
```

备注：电子邮件验证。

------

#### func    [utils.Mobile](https://github.com/Is999/go-utils/blob/master/regexp.go#L32)

```go
func Mobile(value string) bool
```

备注：中国大陆手机号码验证。

------

#### func    [utils.Phone](https://github.com/Is999/go-utils/blob/master/regexp.go#L38)

```go
func Phone(value string) bool
```

备注：中国大陆电话号码验证。

------

#### func    [utils.Numeric](https://github.com/Is999/go-utils/blob/master/regexp.go#L44)

```go
func Numeric(value string) bool
```

备注：有符号数字验证。

------

#### func    [utils.UnNumeric](https://github.com/Is999/go-utils/blob/master/regexp.go#L51)

```go
func UnNumeric(value string) bool
```

备注：无符号数字验证。

------

#### func    [utils.UnInteger](https://github.com/Is999/go-utils/blob/master/regexp.go#L57)

```go
func UnInteger(value string) bool
```

备注：无符号整数(正整数)验证。

------

#### func    [utils.UnIntZero](https://github.com/Is999/go-utils/blob/master/regexp.go#L63)

```go
func UnIntZero(value string) bool
```

备注：无符号整数(正整数+0)验证。

------

#### func    [utils.Amount](https://github.com/Is999/go-utils/blob/master/regexp.go#L73)

```go
func Amount(amount string, decimal uint8, signed ...bool) bool
```

| 参数        | 描述             |
|-----------|----------------|
| amount    | 金额字符串。         |
| *decimal* | 保留小数位长度。       |
| signed    | 带符号的金额: 默认无符号。 |

备注：金额验证。

------

#### func    [utils.Alpha](https://github.com/Is999/go-utils/blob/master/regexp.go#L94)

```go
func Alpha(value string) bool
```

备注：英文字母验证。

------

#### func    [utils.Zh](https://github.com/Is999/go-utils/blob/master/regexp.go#L100)

```go
func Zh(value string) bool
```

备注：中文字符验证。

------

#### func    [utils.MixStr](https://github.com/Is999/go-utils/blob/master/regexp.go#L106)

```go
func MixStr(value string) bool 
```

备注：英文、数字、特殊字符(不包含换行符)。

------

#### func    [utils.Alnum](https://github.com/Is999/go-utils/blob/master/regexp.go#L112)

```go
func Alnum(value string) bool
```

备注：英文字母+数字验证。

------

#### func    [utils.Domain](https://github.com/Is999/go-utils/blob/master/regexp.go#L118)

```go
func Domain(value string) bool
```

备注：域名(64位内正确的域名，可包含中文、字母、数字和.-)。

------

#### func    [utils.TimeMonth](https://github.com/Is999/go-utils/blob/master/regexp.go#L124)

```go
func TimeMonth(value string) bool
```

备注：时间格式验证 yyyy-MM yyyy/MM。

------

#### func    [utils.TimeDay](https://github.com/Is999/go-utils/blob/master/regexp.go#L130)

```go
func TimeDay(value string) bool
```

备注：时间格式验证 yyyy-MM-dd。

------

#### func    [utils.Timestamp](https://github.com/Is999/go-utils/blob/master/regexp.go#L152)

```go
func Timestamp(value string) bool
```

备注：Timestamp 时间格式验证 yyyy-MM-dd hh:mm:ss。

------

#### func    [utils.Account](https://github.com/Is999/go-utils/blob/master/regexp.go#L173)

```go
func Account(value string, min, max uint8) error
```

备注：帐号验证(字母开头，允许字母数字下划线，长度在min-max之间)。

------

#### func    [utils.PassWord](https://github.com/Is999/go-utils/blob/master/regexp.go#L197)

```go
func PassWord(value string, min, max uint8) error
```

备注：密码(字母开头，允许字母数字下划线，长度在 min - max之间)。

------

#### func    [utils.PassWord2](https://github.com/Is999/go-utils/blob/master/regexp.go#L215)

```go
func PassWord2(value string, min, max uint8) error
```

备注：强密码(必须包含大小写字母和数字的组合，不能使用特殊字符，长度在min-max之间)。

------

#### func    [utils.PassWord3](https://github.com/Is999/go-utils/blob/master/regexp.go#L255)

```go
func PassWord3(value string, min, max uint8) error
```

备注：强密码(必须包含大小写字母和数字的组合，可以使用特殊字符，长度在min-max之间)。

------

#### func    [utils.HasSymbols](https://github.com/Is999/go-utils/blob/master/regexp.go#L295)

```go
func HasSymbols(value string) bool
```

备注：是否包含符号。

------

#### [两个时间字符串判断](https://github.com/Is999/go-utils/blob/master/time.go#L365)

备注：参考9.7 两个时间字符串判断。

------

## 11. http/curl

------

### 11.1 模拟curl请求

> 请求方式：GET、POST（form，file）、HEAD、PUT、PATCH、DELETE、OPTIONS

------

#### [GET 请求方式](https://github.com/Is999/go-utils/blob/master/curl.go#L783)

```go
func (c *Curl) Get(url string) (err error)
```

备注：参考测试用例：[TestGet](https://github.com/Is999/go-utils/blob/master/curl_test.go#L38)

------

#### [POST 请求方式](https://github.com/Is999/go-utils/blob/master/curl.go#L792)

```go
func (c *Curl) Post(url string) (err error) 
```

备注：参考测试用例：[TestPost](https://github.com/Is999/go-utils/blob/master/curl_test.go#L222)

------

#### [POST FORM 请求方式](https://github.com/Is999/go-utils/blob/master/curl.go#L801)

```go
func (c *Curl) PostForm(url string) error 
```

备注：参考测试用例：[TestPostForm](https://github.com/Is999/go-utils/blob/master/curl_test.go#L461)

------

#### [POST FILE 请求方式](https://github.com/Is999/go-utils/blob/master/curl.go#L792)

```go
func (c *Curl) Post(url string) (err error) 
```

备注：参考测试用例：[TestPostFile](https://github.com/Is999/go-utils/blob/master/curl_test.go#L546)

------

## 12. http/response

------

### 12.1 重定向

------

#### func    [utils.Redirect](https://github.com/Is999/go-utils/blob/master/response.go#L329)

```go
func Redirect(w http.ResponseWriter, url string, statusCode ...int)
```

| 参数         | 描述             |
|------------|----------------|
| url        | 重定向地址          |
| statusCode | 响应状态码：默认响应 302 |

```go
http.HandleFunc("/response/redirect", func(w http.ResponseWriter, r *http.Request) {
		// 重定向
		utils.Redirect(w, "/response/json")
	})
```

备注：重定向。

------

### 12.2 响应JSON

------

#### func    [utils.JsonResp](https://github.com/Is999/go-utils/blob/master/response.go#L297)

```go
// JsonResp 响应Json数据
func JsonResp[T any](w http.ResponseWriter, statusCode ...int) *Response[T]

// Success 成功响应返回Json数据
func (r *Response[T]) Success(code int, data T, message ...string)

// Fail 失败响应返回Json数据
func (r *Response[T]) Fail(code int, message string, data ...T)
```

示例：

```go
// 响应json数据
http.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {

  // 获取URL查询字符串参数
  queryParam := r.URL.Query().Get("v")

  // 响应的数据
  user := User{
    Name:      "张三",
    Age:       22,
    Sex:       "男",
    IsMarried: false,
    Address:   "北京市",
    phone:     "131188889999",
  }

  if queryParam == "fail" {
    // 错误响应
    utils.JsonResp[User](w, http.StatusNotAcceptable).Fail(2000, "fail", user)
    return
  }
  // 成功响应
  utils.JsonResp[User](w).Success(1000, user)
})
```

备注：响应JSON数据，响应成功：JsonResp().Success()，响应失败：JsonResp().Fail()。

------

### 12.3 响应HTML

------

```go
// 响应html
http.HandleFunc("/response/html", func(w http.ResponseWriter, r *http.Request) {

  // 响应html数据
  utils.View(w).Html("<p>这是一个<b style=\"color: red\">段落!</b></p>")
})
```

备注：响应HTML文本 View().Html()。

------

### 12.4 响应XML

------

```go
// 响应xml
http.HandleFunc("/response/xml", func(w http.ResponseWriter, r *http.Request) {

  // 响应的数据
  user := User{
    Name:      "张三",
    Age:       22,
    Sex:       "男",
    IsMarried: false,
    Address:   "北京市",
    phone:     "131188889999",
  }

  // 响应xml数据
  utils.View(w).Xml(user)
})
```

备注：响应XML文本 View().Xml()。

------

### 12.5 响应TEXT

------

```go
// 响应text
http.HandleFunc("/response/text", func(w http.ResponseWriter, r *http.Request) {
  // 响应text数据
  utils.View(w).Text("<p>这是一个<b style=\"color: red\">段落!</b></p>")
})
```

备注：响应TEXT文本 View().Text()。

------

### 12.6 显示图片

------

```go
// 响应image
http.HandleFunc("/response/show", func(w http.ResponseWriter, r *http.Request) {
  // 获取URL查询字符串参数
  file := r.URL.Query().Get("file")
  if utils.IsExist(file) {
    // 显示文件内容

    utils.View(w).Show(file)
    return
  }
  // 处理错误
  utils.View(w, http.StatusNotFound).Text("不存在的文件：" + file)
})
```

备注：显示文件内容 View().Show()。

------

### 12.7 下载文件

------

```go
// 下载文件
http.HandleFunc("/response/download", func(w http.ResponseWriter, r *http.Request) {
		// 获取URL查询字符串参数
		file := r.URL.Query().Get("file")
		if utils.IsExist(file) {
			// 下载文件数据
			utils.View(w).Download(file)
			return
		}
		// 处理错误
		utils.View(w, http.StatusNotFound).Text("不存在的文件：" + file)
})
```

备注：下载文件 View().Download()。

------

## 13. 打包压缩

------

### 13.1 zip

------

#### func    [utils.Zip](https://github.com/Is999/go-utils/blob/master/zip.go#L16)

```go
func Zip(zipFile string, files []string) error
```

| 参数      | 描述         |
|---------|------------|
| zipFile | 打包压缩后文件    |
| files   | 待打包压缩文件【夹】 |

备注：使用zip打包并压缩。

------

#### func    [utils.UnZip](https://github.com/Is999/go-utils/blob/master/zip.go#L144)

```go
func UnZip(zipFile, destDir string) error
```

| 参数      | 描述     |
|---------|--------|
| zipFile | 代解压的文件 |
| destDir | 解压文件目录 |

备注：解压zip文件。

------

### 13.2 tar

------

#### func    [utils.Tar](https://github.com/Is999/go-utils/blob/master/tar.go#L17)

```go
func Tar(tarFile string, files []string) error 
```

| 参数      | 描述       |
|---------|----------|
| tarFile | 打包后文件    |
| files   | 待打包文件【夹】 |

备注：使用tar打包。

------

#### func    [utils.TarGz](https://github.com/Is999/go-utils/blob/master/tar.go#L49)

```go
func TarGz(tarGzFile string, files []string) error
```

| 参数        | 描述         |
|-----------|------------|
| tarGzFile | 打包压缩后文件    |
| files     | 待打包压缩文件【夹】 |

备注：使用tar打包。

------

#### func    [utils.UnTar](https://github.com/Is999/go-utils/blob/master/tar.go#L180)

```go
func UnTar(tarFile, destDir string) error 
```

| 参数      | 描述     |
|---------|--------|
| tarFile | 代解压的文件 |
| destDir | 解压文件目录 |

备注：解压zip文件。

------

## 14. 日志

------

### 14.1 默认日志（使用标准库的 `log` 包来记录日志）

------

#### 设置日志等级和输出格式

```go
// 日志等级
levelVar := &slog.LevelVar{}
levelVar.Set(slog.LevelDebug)

opts := &slog.HandlerOptions{
  AddSource: true,     // 输出日志的文件和行号
  Level:     levelVar, // 日志等级
}

//日志输出文件
file, err := os.OpenFile("sys.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
if err != nil {
  fmt.Printf("Faild to open error logger file: %v\n", err)
  return
}

// 日志输出格式
//handler := slog.NewTextHandler(os.Stdout, opts)
handler := slog.NewJSONHandler(io.MultiWriter(file, os.Stderr), opts)

// 修改默认的日志输出方式
slog.SetDefault(slog.New(handler))
```

备注：设置日志级别，小于该级别的日志不会输出。禁用日志 设置 *DISABLE* 级别

------

####  

## 15. 杂项

------

### 15.1 环境变量

------

#### func	os.Getenv

```go
func Getenv(key string) strin
```

备注：获取环境变量。

------

#### func    [utils.GetEnv](https://github.com/Is999/go-utils/blob/master/env.go#L12)

```go
func GetEnv(key string, defaultVal ...string) string 
```

| 参数           | 描述           |
|--------------|--------------|
| key          | 变量名。         |
| *defaultVal* | 未获取到时默认返回的值。 |

备注：获取环境变量。

------

#### func	os.Setenv

```go
func Setenv(key, value string) error
```

备注：设置环境变量。

------

#### func	os.Unsetenv

```go
func Unsetenv(key string) error
```

备注：删除环境变量。

------

####       

### 15.2 IP

------

#### func    [utils.ServerIP](https://github.com/Is999/go-utils/blob/master/ip.go#L12)

```go
func ServerIP() string
```

备注：服务器对外IP。

------

#### func    [utils.LocalIP](https://github.com/Is999/go-utils/blob/master/ip.go#L27)

```go
func LocalIP() string
```

备注：服务器本地IP。

------

#### func    [utils.ClientIP](https://github.com/Is999/go-utils/blob/master/ip.go#L55)

```go
func ClientIP(r *http.Request) string
```

备注：获取客户端IP。

------

### 15.3 三目运算

------

#### func    [utils.Ternary](https://github.com/Is999/go-utils/blob/master/misce.go#L13)

```go
func Ternary[T any](expr bool, trueVal, falseVal T) T 
```

| 参数        | 描述              |
|-----------|-----------------|
| expr      | bool表达式。        |
| *trueVal* | expr为true时返回值。  |
| falseVal  | expr为false时返回值。 |

备注：类似于三目运算的函数。

### 15.4 获取当前行号、方法名、文件地址

------

#### func    [utils.RuntimeInfo](https://github.com/Is999/go-utils/blob/master/runtime.go#L14)

```go
func RuntimeInfo(skip int) *Frame
```

备注：获取当前行号、方法名、文件地址。
