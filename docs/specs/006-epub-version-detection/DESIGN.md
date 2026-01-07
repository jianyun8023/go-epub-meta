# 006 - DESIGN：EPUB2/EPUB3 版本判定与混合风格处理

## 总览

本设计新增一个**轻量诊断层**，不改变现有“统一读写”主路径：

- **Reader.Open/parseOPF**：保持不变（继续宽松解析）
- **新增**：版本判定与 signals 检测逻辑（纯函数/小结构体）
- **CLI**：新增 `meta --diagnose` 输出诊断信息（默认输出不变）

该设计不引入新依赖，不触碰 zip “零重新压缩”拷贝策略。

---

## 数据结构

建议在 `epub` 包新增 `version.go`：

```go
package epub

type EPUBMajor int

const (
	EPUBUnknown EPUBMajor = iota
	EPUB2
	EPUB3
)

type VersionSignal struct {
	Code    string // e.g. "opf.package.version", "opf.package.prefix", "manifest.nav", "spine.ncx"
	Level   string // "info" | "warn" (可选)
	Details string // 人类可读细节
}

type VersionDiagnosis struct {
	DeclaredRaw   string      // 原始 package@version
	DeclaredMajor EPUBMajor
	Signals       []VersionSignal
	Mixed         bool
	Warnings      []string
}
```

并在 `*Reader` 或 `*Package` 上提供方法：

- `func (pkg *Package) DeclaredMajor() EPUBMajor`
- `func (r *Reader) DiagnoseVersion() VersionDiagnosis`

---

## 判定逻辑

### 1) Declared version

以 `Package.Version` 为准：

- `""` -> `unknown`
- `strings.HasPrefix(version, "3")` -> `EPUB3`
- `strings.HasPrefix(version, "2")` -> `EPUB2`
- 其它 -> `unknown`（保守）

### 2) Signals 检测

基于当前已解析结构（`Package`, `Metadata.Meta`, `Manifest.Items`, `Spine`）做检测：

- **S1**：`pkg.Prefix != ""` -> EPUB3 强信号
- **S2**：manifest item `properties` 包含 `nav`（空格分隔 token）-> EPUB3 强信号
- **S3**：metadata meta 中存在 `Property != ""` 或 `Refines != ""` -> EPUB3 强信号
- **S4**：`pkg.Spine.Toc != ""` -> EPUB2 强信号（NCX）
- **S5**：metadata meta 中存在 `Name == "cover"` -> EPUB2 弱信号
- **S6**：`pkg.Guide != nil` -> EPUB2 弱信号

### 3) Mixed 判定与 warning 规则

- declared=EPUB2 且出现 S1/S2/S3 任意 -> `Mixed=true` + warning（“declared 2.x but found EPUB3-style signals …”）
- declared=EPUB3 且出现 S4 -> `Mixed=true` + warning（“declared 3.x but spine@toc suggests NCX …”）
- declared=unknown -> 不标 mixed，只列 signals + 可选提示（“unable to determine declared version …”）

---

## CLI 设计

### `golibri meta --diagnose input.epub`

读模式下，在现有人类输出后（或前）追加一个区块：

- Declared version: `2.0` / `3.0` / `unknown`
- Detected major: EPUB2/EPUB3/unknown
- Mixed: true/false
- Signals:
  - `manifest.nav: found item properties contains "nav" (id=..., href=...)`
  - `meta.property: found <meta property="..."> ...`
  - ...
- Warnings:
  - `This EPUB declares 2.0 but contains EPUB3 signals. Writing may use legacy/meta-name style for some fields.`

### JSON 输出

保持 ebook-meta 兼容：

- 默认 **不新增字段**
- 如未来需要，可新增独立开关 `--json-extended` 输出 `epub_version`、`mixed`、`signals`（本需求不做）

---

## 测试策略

在 `epub/version_test.go` 中构造最小 OPF（通过 `Open()` 或直接构造 `Package`）覆盖：

- declared=2.0 + `meta property/refines` -> mixed=true
- declared=3.0 + `spine toc="ncx"` -> mixed=true
- declared=unknown + signals -> mixed=false 但 signals 非空
- nav detection：`properties="nav"` 与 `properties="something nav something"` token 化

测试不需要真实 zip 文件也可（优先对纯 `Package` 方法做单元测试），但可加 1-2 个端到端样例用最小 zip（类似现有 writer/reader 测试构造方式）。

---

## 与“统一读写”策略的关系

此设计只做诊断与解释，不改变写回机制。它的价值是：

- 让用户明确知道“当前 EPUB 的声明版本 vs 实际风格信号”
- 对可能存在语义缺口的字段（如 EPUB3 collection/series）提前提示

后续如果要提升 EPUB3 语义写入或 OPF 保真写回，应另开独立 spec（例如：**“EPUB3 collection/series 写入”**、**“OPF 最小补丁写回”**）。

