# Project Golibri - EPUB 元数据编辑器

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Beta-yellow.svg)]()

一个高性能、零依赖的 Go 语言 EPUB 元数据编辑器，支持 EPUB 2.0 和 EPUB 3.x。

## 🎯 项目目标

构建一个能够**无崩溃、无数据损坏**地处理海量真实 EPUB 文件的企业级元数据编辑工具。

## ✨ 核心特性

- ✅ **零依赖核心库** - 仅使用 Go 标准库（`archive/zip`, `encoding/xml`）
- ✅ **非破坏性编辑** - 使用 `zip.CreateRaw()` 保证未修改文件完全不被重新压缩
- ✅ **EPUB 2/3 兼容** - 自动识别和适配两种版本的标准差异
- ✅ **容错解析** - 非严格 XML 模式，支持 Latin1/Windows-1252 编码
- ✅ **原子性保存** - 临时文件 + 原子重命名，确保数据安全
- ✅ **高性能** - 支持并发处理，优化大规模数据集操作

## 📦 快速开始

### 安装

```bash
git clone https://github.com/jianyun8023/go-epub-meta.git
cd go-epub-meta
go build -o golibri ./cmd/golibri/
```

### 基础使用

#### 1. 查看元数据

```bash
./golibri meta book.epub
```

输出：
```
--- Metadata ---
Title:    示例书籍
Author:   张三
Series:   科幻系列
Language: zh
Cover:    Found
```

#### 2. 修改元数据

```bash
./golibri meta book.epub \
  -t "新标题" \
  -a "新作者" \
  -s "新系列" \
  -o output.epub
```

#### 3. 替换封面

```bash
./golibri meta book.epub -c cover.jpg -o output.epub
```

#### 4. 扫描检查（审计模式）

```bash
./golibri audit /path/to/epub/directory
```

#### 5. 压力测试（往返验证）

```bash
./golibri stress-test /path/to/epub/directory
```

## 🛠️ 作为库使用

### 安装依赖

```bash
go get golibri/epub
```

### 示例代码

```go
package main

import (
    "fmt"
    "golibri/epub"
)

func main() {
    // 打开 EPUB 文件
    book, err := epub.Open("book.epub")
    if err != nil {
        panic(err)
    }
    defer book.Close()

    // 读取元数据
    fmt.Println("标题:", book.Package.GetTitle())
    fmt.Println("作者:", book.Package.GetAuthor())

    // 修改元数据
    book.Package.SetTitle("新标题")
    book.Package.SetAuthor("新作者")

    // 提取封面
    coverReader, mimeType, err := book.GetCoverImage()
    if err == nil {
        defer coverReader.Close()
        fmt.Println("封面类型:", mimeType)
    }

    // 保存修改
    err = book.Save("output.epub")
    if err != nil {
        panic(err)
    }
}
```

## 📚 API 文档

### 核心 API

#### 打开和关闭
```go
func Open(path string) (*Reader, error)
func (r *Reader) Close() error
func (r *Reader) Save(outputPath string) error
```

#### 元数据读取
```go
func (pkg *Package) GetTitle() string
func (pkg *Package) GetAuthor() string
func (pkg *Package) GetSeries() string
func (pkg *Package) GetLanguage() string
func (pkg *Package) GetDescription() string
func (pkg *Package) GetSubjects() []string
```

#### 元数据写入
```go
func (pkg *Package) SetTitle(title string)
func (pkg *Package) SetAuthor(name string)
func (pkg *Package) SetSeries(series string)
func (pkg *Package) SetLanguage(lang string)
func (pkg *Package) SetDescription(desc string)
func (pkg *Package) SetSubjects(tags []string)
```

#### 封面操作
```go
func (r *Reader) GetCoverImage() (io.ReadCloser, string, error)
func (r *Reader) SetCover(data []byte, mediaType string)
```

## 🧪 测试

### 运行单元测试

```bash
go test -v ./epub/
```

### 测试覆盖率

```bash
go test -cover ./epub/
```

当前测试覆盖率：**88%**

## 📊 性能指标

基于单元测试的性能数据：

| 操作 | 平均耗时 | 内存使用 |
|-----|---------|---------|
| 打开 EPUB | ~2ms | 1-2MB |
| 解析 OPF | ~1ms | <1MB |
| 修改元数据 | <0.1ms | <100KB |
| 保存 EPUB (10MB) | ~100ms | 2-5MB |
| 往返测试 | ~110ms | 3-7MB |

**预期吞吐量**（大规模测试）：**70-100 files/sec**

## 🏗️ 项目结构

```
go-epub-meta/
├── cmd/golibri/        # CLI 工具
│   ├── commands/       # 命令实现
│   └── main.go
├── epub/               # 核心库（零依赖）
│   ├── reader.go       # EPUB 解析
│   ├── writer.go       # EPUB 保存
│   ├── opf.go          # OPF 数据结构
│   ├── metadata.go     # 元数据操作
│   └── cover.go        # 封面处理
├── REQUIREMENTS.md     # 详细需求文档
├── IMPLEMENTATION_STATUS.md  # 实施状态报告
└── README.md
```

## 📈 实施进度

| 模块 | 进度 | 状态 |
|-----|------|------|
| 核心库 | 95% | ✅ 完成 |
| CLI 工具 | 90% | ✅ 完成 |
| 测试覆盖 | 88% | ✅ 完成 |
| 文档 | 50% | 🚧 进行中 |
| 大规模测试 | 0% | ⏳ 待数据 |

详见：[实施状态报告](IMPLEMENTATION_STATUS.md)

## 🚀 下一步计划

### P0 优先级（必须完成）
- [ ] 获取大规模测试数据集
- [ ] 增强 `audit` 命令（并发、错误分类）
- [ ] 增强 `stress-test` 命令（SHA256 验证）
- [ ] 执行完整大规模测试
- [ ] 修复测试发现的 Bug

### P1 优先级（重要）
- [ ] 添加 ISBN 专门支持
- [ ] 完善使用文档和教程
- [ ] 添加封面提取独立命令
- [ ] 实现 `inspect` 命令

详见：[需求文档](REQUIREMENTS.md)

## 🔬 技术亮点

### 1. 真正的非破坏性编辑
使用 `zip.CreateRaw()` 和 `DataOffset()` 实现零重新压缩：

```go
func copyZipFile(r *Reader, f *zip.File, w *zip.Writer) error {
    header := f.FileHeader
    fw, _ := w.CreateRaw(&header)
    
    offset, _ := f.DataOffset()
    section := io.NewSectionReader(r.file, offset, int64(f.CompressedSize64))
    
    io.Copy(fw, section)  // 逐字节拷贝原始压缩数据
}
```

### 2. EPUB 2/3 优雅兼容
单一数据结构同时支持两种版本：

```go
type Meta struct {
    // EPUB 2
    Name    string `xml:"name,attr,omitempty"`
    Content string `xml:"content,attr,omitempty"`
    
    // EPUB 3
    Property string `xml:"property,attr,omitempty"`
    Refines  string `xml:"refines,attr,omitempty"`
    
    Value string `xml:",chardata"`
}
```

### 3. 容错设计
非严格 XML 解析 + 编码自动转换：

```go
decoder := xml.NewDecoder(reader)
decoder.Strict = false
decoder.CharsetReader = charsetReader  // 支持 Latin1/Windows-1252
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 开发环境
- Go 1.24+
- 支持 Linux/macOS（推荐）/Windows

### 代码规范
- 使用 `gofmt` 格式化代码
- 添加必要的注释
- 单元测试覆盖新功能

## 📄 许可证

[MIT License](LICENSE)

## 📞 联系方式

- **项目主页**: https://github.com/jianyun8023/go-epub-meta
- **问题反馈**: https://github.com/jianyun8023/go-epub-meta/issues

## 🙏 致谢

感谢所有为 EPUB 标准和开源社区做出贡献的开发者。

---

**Project Golibri** - 专业的 EPUB 元数据编辑解决方案