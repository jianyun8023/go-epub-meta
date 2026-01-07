# 006 - PROGRESS：EPUB2/EPUB3 版本判定与混合风格处理

## 状态

- 当前：已建立需求与设计文档（REQ/DESIGN）。
- 下一步：进入实现与测试（若决定落地）。

---

## 任务拆解

### Phase A：实现（epub 包）

- [ ] 新增 `epub/version.go`
  - [ ] `EPUBMajor` 枚举与字符串化
  - [ ] `VersionDiagnosis` / `VersionSignal` 结构体
  - [ ] `(*Package) DeclaredMajor()`
  - [ ] `(*Reader) DiagnoseVersion()`（或 `(*Package) DiagnoseVersion()`）
- [ ] 在实现中加入 token 化工具（解析 `item.Properties` 的空格分隔 tokens）

### Phase B：CLI（meta 命令）

- [ ] 为 `cmd/golibri meta` 增加 `--diagnose` flag
- [ ] 读模式下追加诊断输出（保持默认输出不变）
- [ ] 明确 `--json` 不变（不引入字段；必要时设计 `--json-extended` 作为未来扩展点，但本需求不实现）

### Phase C：测试

- [ ] `epub/version_test.go`（纯 `Package` 单测为主）
  - [ ] declared=2.0 + EPUB3 强信号 -> mixed=true
  - [ ] declared=3.0 + spine@toc -> mixed=true
  - [ ] declared=unknown -> mixed=false + signals 列表
  - [ ] nav token 解析覆盖
- [ ] 可选：端到端最小 zip（复用现有 writer/reader 测试构造方式）

---

## 验收清单（对照 REQ）

- [ ] AC1：单元测试覆盖 R1/R2/R3 主路径
- [ ] AC2：2.x + EPUB3 强信号 => mixed=true + warning
- [ ] AC3：3.x + spine toc => mixed=true + warning
- [ ] AC4：默认 CLI 输出与 `--json` 输出无破坏性变更

