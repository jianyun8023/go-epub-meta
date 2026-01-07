# 006 - EPUB2/EPUB3 版本判定与混合风格处理

## 背景

`golibri` 目标是作为 Calibre `ebook-meta` 的轻量 Go 原生替代品，并且坚持三条红线：

- **零重新压缩**：未修改的 zip 条目必须使用 `zip.CreateRaw` + `DataOffset` 原样拷贝。
- **零外部依赖**：仅允许 `etree`（XML）+ `cobra`（CLI）+ 标准库。
- **原子写入**：`Write Temp` -> `Validate` -> `os.Rename`。

目前实现采用“EPUB2/EPUB3 统一 OPF 结构”来读写元数据，实际 EPUB 文件中也常见“混合风格”（例如 EPUB3 中仍包含 EPUB2 的 `guide`，或 EPUB2 文件出现 EPUB3 风格 `meta@property`、`manifest@properties`）。

本需求希望让系统对 **EPUB2/EPUB3 的判定更清晰可解释**，并在存在“混合风格信号”时给出 **可控的告警/诊断信息**，同时明确“统一读写”策略的边界与非目标。

---

## 目标（Goals）

- **G1. 明确版本判定**：提供稳定的 EPUB 版本判定（EPUB2 vs EPUB3），优先以 OPF `package@version` 为准。
- **G2. 识别混合风格信号**：对常见“混合风格/非典型结构”进行检测并输出诊断信息（不强制报错）。
- **G3. 可解释输出**：在 CLI 层提供可选的诊断输出（例如 `meta --diagnose`），说明：
  - 声明版本（declared）
  - 信号来源（signals）
  - 是否混合（mixed）
  - 告警/建议（warnings）
- **G4. 不破坏既有行为**：默认输出与写入行为保持现状（不引入 breaking changes；不改变 `--json` 兼容性）。

---

## 非目标（Non-Goals）

- **NG1. 不实现完整 EPUB3 语义写入**：例如 `meta refines` 链（creator role/file-as 的 EPUB3 表达）、collection/series 的 EPUB3 标准表达等，除非作为后续需求单独推进。
- **NG2. 不做严格规范校验器**：不把 `golibri` 变成 EPUB lint 工具；诊断仅用于提示与排查。
- **NG3. 不重写/保真保存整个 OPF DOM**：本需求不要求实现“最小补丁写回”以保留未知节点顺序/注释等（可作为后续需求）。

---

## 用户故事（User Stories）

- **US1**：作为用户，我希望知道输入 EPUB 是 EPUB2 还是 EPUB3（至少基于 OPF `version` 给出明确结果）。
- **US2**：作为用户，我希望在 EPUB 结构“混合/不典型”时得到提示，避免误判读写结果（例如 series 写入在 EPUB3 中可能只写入 calibre 风格）。
- **US3**：作为开发者/QA，我希望能够在测试中稳定复现并断言“版本判定与混合信号”的行为。

---

## 需求（Requirements）

### R1. 版本判定规则（Declared Version）

- **R1.1**：读取 OPF 根元素 `package@version` 作为声明版本（declared version）。
- **R1.2**：若缺失/无法解析，声明版本为 `unknown`（不强行推断）。

### R2. 信号检测（Heuristics / Signals）

提供一组“信号检测”用于解释与辅助定位问题（不用于强制覆盖 declared version）：

- **S1**：`package@prefix` 非空（EPUB3 强信号）。
- **S2**：`manifest item@properties` 包含 `nav`（EPUB3 强信号）。
- **S3**：存在 `<meta property="...">` 或 `<meta refines="...">`（EPUB3 强信号）。
- **S4**：`spine@toc` 非空（EPUB2 强信号，指向 NCX）。
- **S5**：存在 `<meta name="cover" content="...">`（EPUB2 弱信号；EPUB3 也可能保留）。
- **S6**：存在 `<guide>`（EPUB2 弱信号；EPUB3 也常保留）。

### R3. 混合判定（Mixed）

- **R3.1**：若 declared version 为 `2.*`，但检测到 EPUB3 强信号（S1/S2/S3），标记 `mixed=true` 并给出 warning。
- **R3.2**：若 declared version 为 `3.*`，但出现 EPUB2 强信号（S4），标记 `mixed=true` 并给出 warning（S5/S6 仅作为 info）。
- **R3.3**：若 declared version 为 `unknown`，则不标记 mixed，但输出 signals 列表用于人工判断。

### R4. CLI 诊断输出

- **R4.1**：为 `cmd/golibri meta` 增加可选开关（建议：`--diagnose`），在读模式输出诊断信息。
- **R4.2**：默认输出保持现状；`--json` 输出保持 ebook-meta 兼容（不新增字段，或新增字段必须受独立 flag 控制）。

---

## 成功标准（Acceptance Criteria）

- **AC1**：单元测试覆盖 declared version 解析与 signals 检测（至少覆盖 R1、R2、R3 的主要路径）。
- **AC2**：对“declared=2 但包含 EPUB3 强信号”的样例，能稳定输出 `mixed=true` 与可读 warning。
- **AC3**：对“declared=3 但 spine@toc 指向 NCX”的样例，能稳定输出 `mixed=true` 与可读 warning。
- **AC4**：默认 CLI 输出与 `--json` 输出无破坏性变更。

---

## 风险与注意事项

- **解析宽松性**：现有 OPF 解析基于 `etree` 且对真实世界不规范 XML 做了预处理；signals 检测必须基于现有解析结构，不引入严格 XML 依赖。
- **写回保真**：当前写回使用结构体序列化，未知节点/属性可能丢失；诊断输出应告知用户这一点（作为提示，而非阻断）。

