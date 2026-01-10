# Project Golibri - EPUB 元数据编辑器

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

一个高性能、轻量级的 Go 语言 EPUB 元数据编辑器，支持 EPUB 2.0 和 EPUB 3.x。

**核心优势**：
- **轻量级**：单个二进制文件，<10MB，零外部依赖（无需 Python/Qt）。
- **安全**：非破坏性编辑，保证未修改数据不被重新压缩。
- **兼容**：支持 EPUB 2/3 标准，自动适配。

## 📦 快速开始

### 安装

从源码编译：

```bash
git clone https://github.com/jianyun8023/go-epub-meta.git
cd go-epub-meta
go build -o golibri ./cmd/golibri/
```

### 命令行使用

`golibri` 旨在替代 calibre 的 `ebook-meta`，提供简单直观的命令行接口。

#### 1. 查看元数据

```bash
./golibri meta book.epub
```

#### 2. 修改元数据

```bash
# 默认：直接修改原文件（原子写入，安全覆盖）
./golibri meta book.epub \
  -t "新标题" \
  -a "新作者" \
  -s "新系列"

# 也可以指定输出到新文件
./golibri meta book.epub \
  -t "新标题" \
  -a "新作者" \
  -s "新系列" \
  -o output.epub
```

支持写入的更多字段：

```bash
./golibri meta book.epub \
  --publisher "出版社" \
  --date "2025-01-07" \
  --language "zh-CN" \
  --tags "科幻,历史,文学" \
  --comments "简介/描述文本" \
  --series-index "3" \
  --rating 4
```

注：`--series-index` / `--rating` 为 **Calibre 扩展字段**（存储在 OPF 的 `meta name="calibre:*"`）；其余字段遵循 EPUB/Dublin Core 标准。

#### 3. JSON 输出（新功能）

输出 JSON 格式的元数据（字段风格对齐 Calibre/ebook-meta 的常见语义）。注意：`ebook-meta` 通常只输出文本，不保证提供稳定的 JSON 输出开关：

```bash
./golibri meta book.epub --json
```

输出示例：

```json
{
  "title": "示例书籍",
  "authors": ["作者甲", "作者乙"],
  "publisher": "出版社",
  "published": "2024-01-01",
  "language": "zh",
  "series": "系列名",
  "series_index": "3",
  "tags": ["科幻", "历史", "文学"],
  "comments": "简介/描述（JSON 保留原始内容，可能包含转义的 HTML）",
  "rating": 4,
  "producer": "calibre (7.21.0) [https://calibre-ebook.com]",
  "identifiers": {
    "isbn": "9781234567890",
    "calibre": "123"
  },
  "cover": true
}
```

#### 4. 替换封面

```bash
./golibri meta book.epub -c cover.jpg

# 或输出到新文件
./golibri meta book.epub -c cover.jpg -o output.epub
```

#### 5. 导出封面

```bash
# 将封面导出到指定文件
./golibri meta book.epub --get-cover cover.jpg
```

## 🧪 测试套件

Golibri 提供了独立的测试套件 `test-suite`，用于功能验证和与 ebook-meta 对比。

### 编译测试套件

```bash
go build -o test-suite ./cmd/test-suite/
```

### 功能测试

测试 golibri 的核心功能（读取、JSON 输出、往返测试、元数据修改）：

```bash
# 运行所有测试（推荐 - 完整的读+写覆盖）
./test-suite functional /path/to/epub/files --mode all

# 仅测试基本读取
./test-suite functional /path/to/epub/files --mode read

# 仅测试 JSON 输出
./test-suite functional /path/to/epub/files --mode json

# 仅测试往返（读取→保存→再读取，无修改）
./test-suite functional /path/to/epub/files --mode roundtrip

# 仅测试元数据修改（写入功能）
./test-suite functional /path/to/epub/files --mode write
```

### 对比测试

与 ebook-meta 进行对比测试（需要安装 Calibre）：

```bash
./test-suite compare /path/to/epub/files --output results.csv
```

对比测试会生成 CSV 报告，包含详细的字段覆盖率和匹配率统计。

### 测试数据

项目包含测试 EPUB 样本文件：

```bash
- **内置合成数据**: 位于 `cmd/test-suite/testdata/valid`，由脚本生成，体积小，适合快速功能验证。
- **真实样本数据**: 位于 `cmd/test-suite/testdata/samples`，按 EPUB 版本（2.0, 3.x, Hybrid, OEBPS 1.0）分类，包含真实复杂的元数据场景。

```bash
# 使用真实样本运行综合测试
./test-suite functional cmd/test-suite/testdata/samples --mode all
```

详见 [测试数据说明](cmd/test-suite/testdata/README.md) 和 [样本说明](cmd/test-suite/testdata/samples/README.md)。

## 🛠️ 作为库使用

Golibri 可作为标准的 Go 包导入，用于在该 Go 项目中处理 EPUB 文件。

更完整的开发者指南见：[`docs/EPUB_LIBRARY.md`](docs/EPUB_LIBRARY.md)。

### 安装依赖

```bash
go get github.com/jianyun8023/go-epub-meta/epub
```

### 示例代码

```go
package main

import (
    "fmt"
    "strings"
    "github.com/jianyun8023/go-epub-meta/epub"
)

func main() {
    // 1. 打开 EPUB 文件
    book, err := epub.Open("book.epub")
    if err != nil {
        panic(err)
    }
    defer book.Close()

    // 2. 读取元数据
    fmt.Println("标题:", book.Package.GetTitle())
    fmt.Println("作者:", strings.Join(book.Package.GetAuthors(), ", "))

    // 3. 修改元数据
    book.Package.SetTitle("新标题")
    
    // 4. 保存修改
    if err := book.Save("output.epub"); err != nil {
        panic(err)
    }
}
```

## ✨ 特性总结

### 元数据读写

- ✅ **文本格式输出**：人类可读的元数据展示
- ✅ **JSON 格式输出**：与 ebook-meta 兼容的 JSON 格式（`--json`）
- ✅ **完整字段支持**：Title, Authors, Publisher, Published, Language, Series, SeriesIndex, Tags, Comments, Rating, Producer, Identifiers, Cover
- ✅ **Identifier 智能识别**：支持 ISBN, ASIN, Calibre ID 等多种格式
- ✅ **多作者支持**：EPUB2/EPUB3 多 `dc:creator`，以及单字段内常见分隔符智能拆分
- ✅ **原子写入**：不指定 `-o` 时直接修改原文件，使用临时文件 + 重命名保证安全

### 测试与质量保证

- ✅ **功能测试**：读取、JSON 输出、往返测试、元数据修改
- ✅ **对比测试**：与 ebook-meta 对比验证（支持文本和 JSON 两种格式）
- ✅ **测试覆盖**：包含内置测试数据集和生成工具
- ✅ **TDD 开发**：测试驱动开发，保证代码质量

### 技术优势

- ✅ **纯 Go 实现**：无需 Python/Qt 依赖
- ✅ **非破坏性编辑**：使用 `zip.CreateRaw()` 保证未修改数据不被重新压缩
- ✅ **EPUB 2/3 兼容**：自动识别和适配两种版本
- ✅ **高性能**：并发处理，适合大规模批量操作
- ✅ **容错解析**：高容错 XML 解析，支持 Latin1/Windows-1252 编码

## 📊 与 ebook-meta 对比

| 特性 | golibri | ebook-meta (Calibre) |
|------|---------|---------------------|
| 大小 | <10MB | >300MB (含依赖) |
| 依赖 | 零依赖 | Python + Qt |
| JSON 输出 | ✅ `--json` | ⚠️ 通常仅文本输出（不同版本行为不一） |
| 性能 | 高（纯 Go） | 中（Python） |
| 安装 | 单文件 | 需要 Calibre |
| 跨平台 | ✅ | ✅ |

## 📄 许可证

[Apache-2.0 License](LICENSE)

## 🙏 致谢

本项目遵循 TDD（测试驱动开发）原则，所有功能均有完整测试覆盖。

感谢所有为 EPUB 标准和开源社区做出贡献的开发者。
