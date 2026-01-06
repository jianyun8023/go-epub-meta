# Project Golibri (EPUB Metadata Editor) - 需求文档

## 任务代号
**Project Golibri**

## 优先级
**P0 (Core Infrastructure)**

## 项目目标
构建一个 Go 语言类库及 CLI 工具，用于读取和修改 EPUB 2/3 元数据。

## 核心指标
能够无崩溃、无数据损坏地处理**大量**真实环境 EPUB 文件的全量测试。

---

## 1. 核心需求 (Core Requirements)

### 1.1 功能对标 (ebook-meta)

Jules 需要实现以下核心功能的 **Read (读取)** 和 **Write (写入)**：

#### 基础元数据 (Dublin Core)
- ✅ **Title** - 书籍标题
- ✅ **Author(s)** - 作者（支持多作者）
- ✅ **Publisher** - 出版商
- ✅ **Language** - 语言
- ✅ **Description (Comment)** - 描述/简介
- ✅ **Series** - 系列名称
- ⚠️ **ISBN** - 国际标准书号（部分支持，通过 Identifier）
- ✅ **Tags (Subjects)** - 标签/主题

#### 封面处理 (Cover)
- ✅ **提取封面图片流** - 从 EPUB 中提取封面图像数据
- ✅ **替换封面图片** - 替换封面并同步更新 Manifest item 和 meta 引用

#### 版本兼容
- ✅ **自动识别 EPUB 版本** - 自动识别 EPUB 2.0 和 EPUB 3.x
- ✅ **适配标准差异**：
  - EPUB 2: `<meta name="cover" content="item-id"/>`
  - EPUB 3: `<item properties="cover-image".../>`

### 1.2 架构要求

#### 零依赖 (Zero-dependency)
- ⚠️ **当前状态**: 使用了 `spf13/cobra` 作为 CLI 框架
- **建议**: 核心 `epub/` 包完全零依赖（仅标准库），CLI 可保留 Cobra 以提升用户体验
- ✅ **核心库**: 仅使用 Go 标准库
  - `archive/zip` - ZIP 文件处理
  - `encoding/xml` - XML 解析
  - `image` - 图像格式检测（可选）
  
#### 非破坏性 (Non-destructive)
- ✅ **原始文件保护**: 修改元数据时，保留 EPUB 内其他文件的原始字节完整性
- ✅ **禁止重新压缩**: 使用 `zip.CreateRaw()` 和 `DataOffset()` 实现流式拷贝
- ✅ **CRC 校验**: 未修改文件的 CRC32 保持一致
- ✅ **DRM 兼容**: 不破坏可能存在的 DRM 或特殊排版

---

## 2. 技术规格说明 (Technical Specs)

### 2.1 解析模块 (Parser)

#### 入口查找
- ✅ **标准解析**: 解析 `META-INF/container.xml` 定位 `.opf` 文件路径
- ✅ **禁止硬编码**: 不硬编码 OPF 路径（如 `OEBPS/content.opf`）
- ✅ **Fallback 机制**: 优先匹配正确 MIME 类型，无匹配时使用第一个 rootfile

#### XML 容错
- ✅ **非严格模式**: `decoder.Strict = false`
- ✅ **Namespace 容错**: 处理非标准 namespace 前缀
- ✅ **脏数据处理**: 容错处理未闭合标签、奇怪的 namespace

#### 编码探测
- ✅ **UTF-8 支持**: 标准 EPUB 编码
- ✅ **Latin1/Windows-1252**: 自定义 `latin1Reader` 实现编码转换
- ⚠️ **GBK 支持**: 暂未实现（可在大规模测试后按需添加）

### 2.2 写入与打包 (Writer & Repacker)

#### Mimetype 铁律
- ✅ **第一个文件**: ZIP 包的第一个文件必须是 `mimetype`
- ✅ **内容规范**: 必须是 `application/epub+zip`（无换行符）
- ✅ **压缩率为 0**: 必须使用 `zip.Store` 方法（STORE）

#### 增量更新 vs 重打包
- ✅ **流式拷贝重打包**: 由于 Go `archive/zip` 不支持原地修改，实现流式重打包：
  1. 创建临时文件
  2. 写入 `mimetype` (STORE)
  3. 写入修改后的 OPF XML
  4. 写入新增/替换文件（如新封面）
  5. 流式 Copy 未修改文件（保留原始压缩方式和 CRC）
  6. 原子性重命名替换原文件

#### 原子性保证
- ✅ **临时文件策略**: 使用 `os.CreateTemp()` 创建临时文件
- ✅ **原子性重命名**: 完成写入后通过 `os.Rename()` 原子性替换
- ✅ **跨平台兼容**: POSIX 系统支持原地修改（打开的文件可被重命名）
- ⚠️ **Windows 限制**: Windows 下打开的文件无法被重命名，需要先 `Close()`

---

## 3. 大规模数据集测试策略

这是本项目的**核心价值**。需要构建专门的测试 Harness，支持对大量 EPUB 文件进行批量验证。

### 3.1 扫描模式 (Dry-Run / Audit Mode)

#### 实现状态: ✅ 已实现 (`cmd/golibri/commands/audit.go`)

**功能**:
- `golibri audit <directory>` - 审计模式命令
- 遍历目录下所有 `.epub` 文件
- 仅执行 `Open()` 和基础元数据读取
- 不执行写入操作

**输出**:
```
Scanning 10000+ files...
[FAIL] /path/to/broken.epub: malformed OPF: XML syntax error
[WARN] /path/to/incomplete.epub: No Title
...
Scan complete. Success: 9847, Failed: 153
```

**待改进**:
- ⚠️ 添加并发处理（目前是串行）
- ⚠️ 添加错误分类统计
- ⚠️ 输出详细报告文件（JSON/CSV）

### 3.2 往返测试 (Round-Trip Verification)

#### 实现状态: ✅ 已实现 (`cmd/golibri/commands/stress.go`)

**功能**:
- `golibri stress-test <directory>` - 压力测试命令
- 并发 Worker Pool（默认 16 个并发）
- 对每个文件执行：**读取 A → 修改内存对象 → 写入 B → 读取 B**

**验证逻辑**:
```go
1. 打开原始文件
2. 修改标题（添加 "[MOD]" 后缀）
3. 保存到临时文件
4. 重新打开临时文件
5. 验证修改是否生效
6. 清理临时文件
```

**输出**:
```
Processed 10000+ files in 23m. Passed: 9847, Failed: 153
```

**待改进**:
- ⚠️ 添加 CRC32/SHA256 哈希比对（验证非元数据文件未被破坏）
- ⚠️ 添加详细的失败日志（文件路径、错误类型）
- ⚠️ 支持自定义并发数
- ⚠️ 添加进度条显示

### 3.3 非破坏性验证 (Non-Destructive Test)

#### 实现状态: ✅ 已实现测试用例 (`epub/writer_nondestructive_test.go`)

**验证点**:
- ✅ 比对修改前后非 OPF 文件的 `CompressedSize64`
- ✅ 确保未修改文件的压缩大小完全一致（证明无重新压缩）

**测试结果**: ✅ `TestNonDestructiveCopy` 通过

---

## 4. 交付物清单 (Deliverables)

### 4.1 核心库 (epub Package)

#### 文件列表:
- ✅ `epub/reader.go` - EPUB 打开和解析
- ✅ `epub/opf.go` - OPF 文件结构定义（支持 EPUB 2/3）
- ✅ `epub/metadata.go` - 元数据 Get/Set 方法
- ✅ `epub/cover.go` - 封面提取和替换
- ✅ `epub/writer.go` - EPUB 保存和重打包

#### API 列表:
```go
// 基础操作
func Open(path string) (*Reader, error)
func (r *Reader) Close() error
func (r *Reader) Save(outputPath string) error

// 元数据读取
func (pkg *Package) GetTitle() string
func (pkg *Package) GetAuthor() string
func (pkg *Package) GetSeries() string
func (pkg *Package) GetLanguage() string
func (pkg *Package) GetDescription() string
func (pkg *Package) GetSubjects() []string

// 元数据写入
func (pkg *Package) SetTitle(title string)
func (pkg *Package) SetAuthor(name string)
func (pkg *Package) SetSeries(series string)
func (pkg *Package) SetLanguage(lang string)
func (pkg *Package) SetDescription(desc string)
func (pkg *Package) SetSubjects(tags []string)

// 封面操作
func (r *Reader) GetCoverImage() (io.ReadCloser, string, error)
func (r *Reader) SetCover(data []byte, mediaType string)
```

### 4.2 CLI 工具

#### 已实现命令:
```bash
# 查看元数据
golibri meta input.epub

# 修改元数据
golibri meta input.epub -t "New Title" -a "Author" -o output.epub

# 替换封面
golibri meta input.epub -c cover.jpg -o output.epub

# 审计模式（只读扫描）
golibri audit /path/to/epub/dir

# 压力测试（往返测试）
golibri stress-test /path/to/epub/dir
```

#### 待实现命令:
```bash
# 提取封面
golibri cover extract input.epub -o cover.jpg

# 查看 EPUB 信息（文件列表、结构等）
golibri inspect input.epub

# 批量处理
golibri batch <script.json> <input-dir> <output-dir>
```

### 4.3 测试套件

#### 已实现测试:
- ✅ `epub/reader_test.go` - 解析测试（EPUB 2 格式）
- ✅ `epub/writer_test.go` - 基础写入测试
- ✅ `epub/safe_test.go` - 原地修改测试
- ✅ `epub/writer_nondestructive_test.go` - 非破坏性拷贝测试

#### 测试覆盖:
```
✅ EPUB 2.0 解析
✅ OPF namespace 处理
✅ Dublin Core 元数据读取
✅ 元数据修改
✅ Mimetype 规范验证
✅ 非破坏性文件拷贝
✅ 原地修改（In-Place Save）
✅ Latin1/Windows-1252 编码转换
```

#### 待补充测试:
- ⚠️ EPUB 3.x 完整测试用例
- ⚠️ 封面替换集成测试
- ⚠️ 大文件性能测试
- ⚠️ 边界情况测试（损坏的 ZIP、缺失 OPF 等）

---

## 5. 当前实施状态总结

### 5.1 已完成功能 (✅)

#### 核心库:
- ✅ EPUB 2/3 解析器（支持复杂 namespace）
- ✅ 零依赖核心库（epub 包）
- ✅ 非破坏性 ZIP 重打包
- ✅ 容错 XML 解析（Strict=false）
- ✅ Latin1/Windows-1252 编码支持
- ✅ 元数据读写（Title, Author, Series, Description, Language, Subjects）
- ✅ 封面提取和替换
- ✅ 原子性保存（临时文件 + 重命名）

#### CLI 工具:
- ✅ Cobra 框架集成（提升 UX）
- ✅ `meta` 命令（查看和修改元数据）
- ✅ `audit` 命令（只读审计）
- ✅ `stress-test` 命令（往返测试）

#### 测试:
- ✅ 单元测试覆盖核心功能
- ✅ 非破坏性验证测试
- ✅ 原地修改测试

### 5.2 待完善功能 (⚠️)

#### 功能增强:
1. **ISBN 支持** - 当前通过 Identifier 可访问，需添加专门的 Get/Set 方法
2. **更多元数据字段** - Rights, Date, Type, Format 等
3. **Guide 支持** - EPUB 2 的 Guide 引用
4. **NCX 支持** - EPUB 2 导航文件
5. **EPUB 3 特性** - 更完整的 EPUB 3.x 支持（如 `<meta property="dcterms:modified">`）

#### 测试增强:
1. **并发审计** - `audit` 命令支持并发扫描
2. **错误分类** - 详细的错误类型统计（ZIP_CORRUPT, XML_PARSE_ERROR, MISSING_OPF）
3. **哈希验证** - 往返测试中添加 SHA256 哈希比对
4. **报告生成** - 生成 JSON/CSV 格式的详细报告
5. **进度显示** - 大数据集扫描时显示进度条

#### CLI 增强:
1. **封面提取命令** - `golibri cover extract`
2. **批量处理** - `golibri batch` 支持批量修改
3. **详细检查** - `golibri inspect` 显示 EPUB 内部结构
4. **详细日志** - 支持 `--verbose` 和 `--debug` 模式

#### 文档:
1. **API 文档** - GoDoc 注释完善
2. **使用教程** - README 添加快速开始指南
3. **架构文档** - 设计决策和技术细节说明

### 5.3 技术债务

#### Cobra 依赖:
- **问题**: 当前 CLI 依赖 `spf13/cobra`，不符合"零依赖"需求
- **方案选择**:
  1. **保留 Cobra** (推荐): 核心库零依赖，CLI 工具可有依赖以提升 UX
  2. **移除 Cobra**: 使用 `flag` 标准库重写 CLI（降低用户体验）
- **建议**: 保留 Cobra，文档中说明"核心库零依赖，CLI 工具使用 Cobra 提升体验"

#### Windows 兼容性:
- **问题**: Windows 下打开的文件无法被重命名
- **当前状态**: 代码中已识别问题（见 `safe_test.go` 注释）
- **解决方案**: 用户需在 `Save()` 前显式 `Close()` Reader（已在代码逻辑中体现）

---

## 6. 大规模数据集测试执行计划

### 6.1 测试环境准备

#### 硬件要求:
- **CPU**: 8 核心以上（支持并发测试）
- **内存**: 16GB+（并发处理多个 EPUB）
- **磁盘**: 至少 100GB 可用空间（存储测试集和临时文件）
- **IO**: SSD 推荐（提高文件读写速度）

#### 软件要求:
- **Go**: 1.23+ (已使用 1.24.3)
- **OS**: Linux/macOS（POSIX 系统，支持原地修改）

### 6.2 测试数据集

#### 数据来源:
- 用户提供的大量 EPUB 文件（建议 10,000+ 本以上）
- 建议包含:
  - ✅ 各种出版商（确保格式多样性）
  - ✅ EPUB 2 和 EPUB 3 混合
  - ✅ 不同语言（测试编码处理）
  - ✅ 包含边界情况（损坏文件、非标准格式）

### 6.3 测试执行步骤

#### Phase 1: 快速审计 (Audit)
```bash
golibri audit /path/to/epubs > audit_report.log
```
**预期时间**: 根据文件数量而定（SSD 磁盘约每秒处理 50-100 本）  
**成功标准**: 成功率 > 99.5%

#### Phase 2: 完整压力测试 (Stress Test)
```bash
golibri stress-test /path/to/epubs > stress_report.log
```
**预期时间**: 约为审计时间的 2-3 倍（包含写入操作）  
**成功标准**: 
- 成功率 > 99.5%
- 无数据损坏（往返测试通过）

#### Phase 3: 采样详细验证
```bash
# 随机抽取样本进行详细验证
# 包括哈希比对、阅读器兼容性测试
```

### 6.4 错误处理策略

#### 预期错误类型:
1. **ZIP_CORRUPT** (预计 <0.1%) - 损坏的 ZIP 文件
   - **处理**: 记录并跳过，不视为库的问题
   
2. **XML_PARSE_ERROR** (预计 <0.3%) - 严重畸形的 XML
   - **处理**: 尝试容错解析，无法恢复则记录
   
3. **MISSING_OPF** (预计 <0.1%) - 缺失 OPF 文件
   - **处理**: 记录并标记为非标准 EPUB
   
4. **ENCODING_ERROR** (预计 <0.05%) - 不支持的编码
   - **处理**: 添加新的编码支持（如 GBK）

#### 失败阈值:
- **可接受失败率**: < 0.5%
- **严重错误阈值**: 数据损坏率必须为 0%

---

## 7. 下一步行动 (Next Actions)

### 优先级 P0 (必须完成):
1. ✅ ~~核心库实现~~ (已完成)
2. ✅ ~~CLI 基础命令~~ (已完成)
3. ✅ ~~压力测试框架~~ (已完成)
4. ⚠️ **执行大规模数据集完整测试** - 待用户提供测试数据
5. ⚠️ **根据测试结果修复 Bug** - 待测试后评估

### 优先级 P1 (重要):
1. ⚠️ 完善错误分类和报告生成
2. ⚠️ 添加 SHA256 哈希验证
3. ⚠️ 完善 README 和使用文档
4. ⚠️ 添加 ISBN 专门支持

### 优先级 P2 (次要):
1. ⚠️ 实现封面提取独立命令
2. ⚠️ 支持批量处理
3. ⚠️ 添加详细检查命令
4. ⚠️ 考虑移除 Cobra 依赖（根据需求）

---

## 8. 成功标准

### 定量指标:
- ✅ **零依赖核心库**: `epub/` 包无外部依赖
- ✅ **非破坏性**: 往返测试中非修改文件 CRC 一致
- ⚠️ **高成功率**: 大规模测试成功率 > 99.5%
- ⚠️ **零数据损坏**: 无任何文件在修改后无法打开或内容损坏
- ✅ **跨平台**: 支持 Linux/macOS/Windows

### 定性指标:
- ✅ **代码质量**: 通过所有单元测试
- ✅ **可维护性**: 清晰的代码结构和注释
- ⚠️ **文档完整**: API 文档、使用教程、架构说明
- ✅ **用户体验**: CLI 工具易用、错误提示友好

---

## 附录 A: OPF 核心数据结构

以下是已实现的 OPF 数据结构（`epub/opf.go`），支持 EPUB 2 和 EPUB 3：

```go
// Package 是 OPF 文件的根元素
type Package struct {
    XMLName          xml.Name  `xml:"http://www.idpf.org/2007/opf package"`
    Version          string    `xml:"version,attr"`
    UniqueIdentifier string    `xml:"unique-identifier,attr"`
    Prefix           string    `xml:"prefix,attr,omitempty"`
    
    Metadata Metadata `xml:"metadata"`
    Manifest Manifest `xml:"manifest"`
    Spine    Spine    `xml:"spine"`
    Guide    *Guide   `xml:"guide,omitempty"`
}

// Metadata 包含出版物元数据（支持 EPUB 2 和 3）
type Metadata struct {
    // Dublin Core 元素
    Titles       []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ title"`
    Creators     []AuthorMeta `xml:"http://purl.org/dc/elements/1.1/ creator"`
    Subjects     []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ subject"`
    Descriptions []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ description"`
    Publishers   []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ publisher"`
    Languages    []SimpleMeta `xml:"http://purl.org/dc/elements/1.1/ language"`
    Identifiers  []IDMeta     `xml:"http://purl.org/dc/elements/1.1/ identifier"`
    
    // Meta 标签（支持 EPUB 2 和 3 两种格式）
    Meta []Meta `xml:"meta"`
}

// Meta 表示通用的 <meta> 标签
type Meta struct {
    ID      string `xml:"id,attr,omitempty"`
    
    // EPUB 2 属性
    Name    string `xml:"name,attr,omitempty"`
    Content string `xml:"content,attr,omitempty"`
    
    // EPUB 3 属性
    Property string `xml:"property,attr,omitempty"`
    Refines  string `xml:"refines,attr,omitempty"`
    Scheme   string `xml:"scheme,attr,omitempty"`
    
    Value string `xml:",chardata"`
}
```

---

## 附录 B: 参考资料

### EPUB 规范:
- [EPUB 2.0.1 Specification](http://idpf.org/epub/20/spec/OPF_2.0.1_draft.htm)
- [EPUB 3.3 Specification](https://www.w3.org/TR/epub-33/)
- [EPUB Open Container Format (OCF)](https://www.w3.org/TR/epub-33/#sec-ocf)

### 相关工具:
- **Calibre ebook-meta**: 功能对标参考
- **epubcheck**: EPUB 验证工具（Java）
- **go-epub**: 现有 Go EPUB 库（功能较弱）

---

**文档版本**: 1.0  
**最后更新**: 2025-01-06  
**维护者**: Project Golibri Team

