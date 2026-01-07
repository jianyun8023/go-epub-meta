# Golibri 快速开始指南

## 🎉 全部完成！

基于 TDD 模式，所有功能已实现并测试通过。

## 📦 编译

```bash
# 编译 golibri 主工具
go build -o golibri ./cmd/golibri/

# 编译测试套件
go build -o test-suite ./cmd/test-suite/
```

## 🚀 Golibri 使用

### 1. 查看元数据（文本格式）

```bash
./golibri meta book.epub
```

输出：
```
--- Metadata ---
Title:       测试书籍
Author:      张三
Publisher:   测试出版社
Published:   2023-12-31T16:00:00+00:00
Language:    zh
Series:      测试系列
Identifiers: isbn:9787544798501
Cover:       Not Found
```

### 2. 查看元数据（JSON 格式）✨ 新功能

```bash
./golibri meta --json book.epub
```

输出：
```json
{
  "title": "测试书籍",
  "authors": ["张三"],
  "publisher": "测试出版社",
  "published": "2023-12-31T16:00:00+00:00",
  "language": "zh",
  "series": "测试系列",
  "identifiers": {
    "isbn": "9787544798501"
  },
  "cover": false
}
```

### 3. 修改元数据

```bash
./golibri meta book.epub \
  -t "新标题" \
  -a "新作者" \
  --isbn "9781234567890" \
  -o output.epub
```

## 🧪 测试套件使用

### 功能测试

```bash
# 测试所有功能（推荐 - 包含读取+写入的完整测试）
./test-suite functional /path/to/epubs --mode all

# 仅测试基本读取
./test-suite functional /path/to/epubs --mode read

# 仅测试 JSON 输出
./test-suite functional /path/to/epubs --mode json

# 仅测试往返（读取→保存→再读取，无修改）
./test-suite functional /path/to/epubs --mode roundtrip

# 仅测试元数据修改（写入）
./test-suite functional /path/to/epubs --mode write

# 使用内置测试数据
./test-suite functional cmd/test-suite/testdata/valid
```

### 对比测试（与 ebook-meta）

```bash
./test-suite compare /path/to/epubs --output results.csv
```

## ✅ 测试结果

所有测试已通过：

```bash
# 单元测试
$ go test ./cmd/golibri/commands/
ok  	golibri/cmd/golibri/commands	0.816s

# 功能测试
$ ./test-suite functional cmd/test-suite/testdata/valid --mode all
Found 3 EPUB files. Mode: all
Result: 3 passed, 0 failed

# 对比测试
$ ./test-suite compare cmd/test-suite/testdata/valid
Total files: 3
Golibri success: 3 (100.0%)
Ebook-meta success: 3 (100.0%)
```

## 📚 详细文档

- **完整说明**：[README.md](README.md)
- **实现报告**：[IMPLEMENTATION_COMPLETE.md](IMPLEMENTATION_COMPLETE.md)
- **测试数据**：[cmd/test-suite/testdata/README.md](cmd/test-suite/testdata/README.md)

## 🎯 核心特性

| 特性 | 状态 | 说明 |
|------|------|------|
| 文本输出 | ✅ | 人类可读格式 |
| JSON 输出 | ✅ | 与 ebook-meta 兼容 |
| 元数据修改 | ✅ | 支持所有字段 |
| 功能测试 | ✅ | 4 种测试模式 |
| 对比测试 | ✅ | JSON + 文本格式 |
| TDD 开发 | ✅ | 测试先行 |

开始使用吧！ 🚀
