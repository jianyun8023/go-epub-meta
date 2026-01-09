# AGENTS.md - Golibri AI Context
# 核心原则
- **定位**: Calibre `ebook-meta` 的轻量级 Go 原生替代品（<10MB vs >300MB）。
- **红线**:
    1. **零重新压缩**: 必须使用 `zip.CreateRaw` + `DataOffset`。严禁解码再编码未修改的数据流。
    2. **零外部依赖**: 仅允许 `etree` (XML) + `cobra` (CLI) + 标准库。严禁 CGO/Python/Qt。
    3. **原子写入**: `Write Temp` -> `Validate` -> `os.Rename`。

## 元数据写入策略 (EPUB 3 vs EPUB 2)

本库在写入元数据时，遵循 **“严格符合标准，同时有条件地维护兼容性”** 的原则，具体策略为 **“有则维护，无则不加”**。

### EPUB 3 模式下的行为
当检测到 EPUB 版本为 3.0 或更高时，写入操作（如设置系列、评分、封面等）会优先使用 EPUB 3 标准的 `property` 属性或专门的 XML 结构（如 `belongs-to-collection`）。

对于非标准的旧式兼容标签（主要是 Calibre 引入的 `calibre:series`, `calibre:rating` 或 EPUB 2 风格的 `<meta name="cover">`）：
- **如果原文件中已存在这些标签**：本库会**同步更新**它们的值，以保证在旧设备上的兼容性不退化。
- **如果原文件中不存在这些标签**：本库**不会主动添加**它们。这保持了 EPUB 3 文件的“纯净度”，避免引入不必要的非标准元数据。

### EPUB 2 模式下的行为
当检测到 EPUB 版本为 2.0 时，本库会自动回退到使用 `name` / `content` 属性的旧式元数据写法（如 `calibre:series`），以确保最大兼容性。

### 开发者提示
你不需要手动处理这些差异，只需调用统一的 API（如 `SetSeries`, `SetCover`），库内部会自动根据文件版本和现有内容应用上述策略。

---

## 👥 智能体团队 (Agent Team)

### 🧹 Maintainer (维护者)
- **职责**: 架构治理、代码清理、文档分层。
- **当前任务 (Phase 5 - Release Prep)**:
    1. **清理**: 删除临时文件（CSV/txt/二进制），创建 `.gitignore`。
    2. **发布**: 准备 v0.4.0 发布说明，标记版本。
    3. **文档**: 整合散落的 Markdown 文档，保留核心文档。
- **Backstory**: 极简主义架构师，已完成代码库瘦身（删除 audit/stress/old-tools）。

### ✨ Feature Dev (开发专家)
- **职责**: 核心功能开发 (TDD 驱动)。
- **当前任务 (Phase 5 - Stabilization)**:
    1. ✅ **完善 JSON**: 已处理特殊字符转义和多作者字段。
    2. **封面提取**: 考虑添加 `meta --cover-extract` 功能（可选）。
    3. **批量处理**: 评估是否需要支持目录批量元数据修改。
- **Backstory**: 资深 Go 开发者，已实现 `meta --json` 并对齐 ebook-meta 格式。

### 🧪 QA Engineer (测试专家)
- **职责**: 质量保证、独立测试工具构建。
- **当前任务 (Phase 5 - CI/CD)**:
    1. **自动化**: 将 `test-suite` 集成到 GitHub Actions。
    2. **错误路径**: 在 `test-suite functional` 中添加错误场景覆盖。
    3. **EPUB3 写入**: 补齐 collection/series EPUB3 表达的端到端测试。
- **已完成**:
    - ✅ 单元测试补齐（series/isbn/asin/identifier/cover）
    - ✅ EPUB3/多作者解析测试
    - ✅ metadata API 68 个测试用例
- **Backstory**: 自动化测试工程师，已构建统一的 `test-suite` 工具链。

---

## 🛠 技术栈与规范

### 核心技术
- **Language**: Go 1.24+
- **CLI**: Cobra
- **XML**: etree (容错解析)
- **Format**: EPUB 2.0 & 3.x (自动适配)

### 决策日志 (ADR)
| ID | 决策 | 原因 |
|----|------|------|
| 001 | 删除 `audit/stress` 命令 | 保持 `golibri` 专注单一职责；测试功能移至 `test-suite`。 |
| 002 | `README` vs `AGENTS` 分离 | `README` 服务用户（What/How），`AGENTS` 服务开发者/AI（Why/Rules）。 |
| 003 | 兼容 `ebook-meta` JSON | 降低用户迁移成本，利用现有生态。 |
| 004 | 统一测试工具链 | 合并分散的 `compare-meta` 和 `analyze-results` 到 `test-suite`。 |
| 005 | 添加 `.gitignore` | 排除生成文件（二进制/CSV/cache），保持仓库整洁。 |

---

## 📅 完成路线图 (Roadmap)

### ✅ Phase 1: Cleanup & Infra
- [x] Maintainer: 删除 `audit.go`, `stress.go`。
- [x] Maintainer: 重写 `README.md` (移除开发细节，保留用户指南)。

### ✅ Phase 2: Core Alignment
- [x] Feature Dev: 编写 `TestJSONOutput`。
- [x] Feature Dev: 实现 `meta --json`。

### ✅ Phase 3: QA System
- [x] QA Engineer: 创建 `cmd/test-suite`。
- [x] QA Engineer: 迁移 `compare` 和 `report` 功能。
- [x] Maintainer: 删除旧工具目录。

### ✅ Phase 4: Test Coverage Enhancement (2025-01-07)
- [x] QA Engineer: 补齐 `meta` 写入路径单元测试（series/isbn/asin/identifier/cover）。
- [x] QA Engineer: 添加 EPUB3/多作者解析测试。
- [x] QA Engineer: 添加 metadata API 单元测试（68 个测试用例）。
- [x] QA Engineer: 为 test-suite 添加基础单元测试。

### 🚧 Phase 5: Release Prep (2026-01-07 ~)
- [ ] Maintainer: 创建 `.gitignore`，清理临时文件（CSV/txt/二进制）。
- [ ] Maintainer: 整合散落文档，精简项目根目录。
- [ ] QA Engineer: 添加 GitHub Actions CI 配置。
- [ ] Maintainer: 发布 v0.4.0 版本。

---

## 🚀 需求的开发流程 (Development Workflow)

遵循结构化的开发流程，确保代码质量与设计的一致性。

### 1. 规范化目录 (Specs Structure)
在 `docs/specs` 下为每个独立需求创建子目录，命名格式为 `ID-short-name` (例如 `005-cover-extraction`)。
目录中应包含：
- `REQ.md`: 需求说明与背景。
- `DESIGN.md`: 技术方案设计。
- `PROGRESS.md`: 任务拆解与进度追踪。

### 2. 标准流程 (The Loop)

#### Step 1: Research (调研) 🕵️
- **Owner**: Feature Dev / Maintainer
- **Action**:
  - 分析用户需求或现有痛点。
  - 调研竞品（如 Calibre）的行为模式。
  - **产出**: `REQ.md` (明确 Scope 和 Non-Goals)。

#### Step 2: Plan (设计) 📐
- **Owner**: Feature Dev (Reviewer: Maintainer)
- **Action**:
  - 编写技术设计文档。
  - 定义关键接口、数据结构和错误处理。
  - 制定测试策略（涵盖边缘情况）。
  - **产出**: `DESIGN.md`。

#### Step 3: Implementation (实施) 🔨
- **Owner**: Feature Dev
- **Action**:
  - **TDD First**: 先写失败的测试用例。
  - 编码实现，遵循 "零重新压缩" 和 "原子写入" 原则。
  - 提交原子化的 Git Commits。

#### Step 4: Verify (验证) 🧪
- **Owner**: QA Engineer
- **Action**:
  - 运行单元测试 (`go test ./...`)。
  - 使用 `test-suite` 进行回归测试。
  - 如果涉及兼容性，进行跨版本/跨工具验证。
  - 如果遇到 Go 构建缓存权限问题，可指定仓库内缓存目录（例如设置 `GOCACHE=.gocache` 后再运行 `go test`）。

#### Step 5: Finalize (交付) 📦
- **Owner**: Maintainer
- **Action**:
  - 更新项目级文档 (`README.md`, `AGENTS.md`)。
  - 清理临时代码或调试日志。
  - 合并代码并标记完成。

---

## 🧪 测试覆盖现状：元数据读写（`cmd/golibri meta`）

目标：确保 `cmd/golibri` 的 `meta` 命令在真实 EPUB 上 **可读、可写、可输出 JSON**，并持续符合三条红线（零重新压缩 / 原子写入 / 零外部依赖）。

### 📊 当前覆盖率（2026-01-07 验证）
| 包 | 覆盖率 | 状态 |
|----|--------|------|
| `golibri/epub` | **68.4%** | ✅ 良好 |
| `golibri/cmd/golibri/commands` | **52.8%** | ✅ 达标 |
| `golibri/cmd/test-suite/commands` | **7.1%** | ⚠️ 基础 |

### ✅ 已覆盖（足以保证"可发布门槛"）

#### 核心读写链路
- **EPUB 读取/解析**：能打开 EPUB、解析 `container.xml` 与 OPF（`epub/reader_test.go`）。
- **写入与保存**：
  - 修改 OPF 元数据后 `Save()` 会写回（`epub/writer_test.go`）。
  - `mimetype` 仍为第一个条目且 `Store`（`epub/writer_test.go`）。
  - **零重新压缩**：未修改文件条目压缩大小保持不变（`epub/writer_nondestructive_test.go`）。
  - **原子 in-place**：同路径保存后可重新打开读取到新元数据（`epub/safe_test.go`）。

#### 元数据 API（`epub/metadata_test.go` - 新增 68 个测试）
- **Title**: GetTitle/SetTitle
- **Author**: GetAuthor/SetAuthor/GetAuthorSort
- **Description**: GetDescription/SetDescription
- **Language**: GetLanguage/SetLanguage
- **Series**: GetSeries/SetSeries/GetSeriesIndex/SetSeriesIndex
- **Subjects**: GetSubjects/SetSubjects
- **Identifiers**: GetIdentifiers/SetIdentifier/GetISBN/SetISBN/GetASIN/SetASIN
- **Publisher**: GetPublisher/SetPublisher
- **Publish Date**: GetPublishDate/SetPublishDate
- **Producer**: GetProducer
- **辅助函数**: parseIdentifier/normalizeScheme/isISBN（含边界测试）

#### CLI 写入路径（`cmd/golibri/commands/meta_test.go` - 大幅扩展）
- ✅ `--title/-t`：修改标题
- ✅ `--author/-a`：修改作者
- ✅ `--series/-s`：修改系列名
- ✅ `--isbn`：设置 ISBN 标识符
- ✅ `--asin`：设置 ASIN 标识符
- ✅ `--identifier/-i`：设置自定义标识符（douban/goodreads 等）
- ✅ `--cover/-c`：设置封面图片
- ✅ 多字段同时修改
- ✅ 特殊字符处理（中文、emoji、XML 实体）

#### EPUB3 / 复杂 OPF
- ✅ 多作者解析（EPUB3 `dc:creator` + `meta refines`）
- ✅ EPUB3 `meta property` 解析

#### test-suite 工具链（`cmd/test-suite/commands/compare_test.go` - 新增）
- ✅ normalize/normalizeLanguage/normalizeDate/normalizeIdentifiers
- ✅ findEpubFiles（文件发现）
- ✅ ComparisonResult 数据结构

### ⚠️ 待完善缺口
- **CLI 错误路径集成测试**：坏 container、找不到 OPF、损坏 zip 等场景需在 `test-suite functional` 中系统化验证（CLI 使用 `os.Exit(1)` 处理错误，单元测试无法捕获）。
- **EPUB3 写入路径**：collection/series 的 EPUB3 表达写入尚未端到端测试。
- **test-suite 命令覆盖率**：CLI 入口点（runCompare/runFunctional）依赖外部二进制，覆盖率较低但不影响核心功能。

### ✅ 推荐的回归门槛（最低保障）
```bash
# 1. 单元测试全绿
GOCACHE=.gocache go test ./...

# 2. 构建二进制后运行集成测试
go build -o golibri ./cmd/golibri/
./test-suite functional <epub-dir> --mode all

# 3. 兼容性敏感改动需对比验证
./test-suite compare <epub-dir> --format json
```
