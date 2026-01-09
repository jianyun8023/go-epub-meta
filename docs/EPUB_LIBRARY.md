# Golibri `epub` 库使用指南（面向开发者）

本指南介绍如何在 Go 项目中使用 `golibri/epub` 读取与写入 EPUB 元数据，并说明 **API 稳定性** 与 **版本锁定** 建议。

## 1. API 稳定性说明（重要）

当前仓库处于 **v0.x** 阶段，整体 API 仍可能调整。为减少破坏性变更风险，建议你遵循以下约定：

- **推荐稳定层（优先使用）**：
  - `epub.Open(path) (*epub.Reader, error)`
  - `(*epub.Reader).Save(outputPath string) error`（原子写入）
  - `(*epub.Reader).GetCoverImage() (io.ReadCloser, string, error)`
  - `(*epub.Reader).SetCover(data []byte, mediaType string)`
  - `(*epub.Package)` 上的“元数据 Getter/Setter”（见下文示例）
- **不保证稳定（尽量不要直接依赖）**：
  - `epub/opf.go` 中导出的 OPF 结构体（例如 `Metadata`, `Manifest`, `Meta` 等）：它们更接近内部表示，未来可能为了兼容性/简化而调整字段或结构。

如果你需要“长周期稳定依赖”，建议：
- 在 `go.mod` 中 **锁定版本 tag**（等项目发布稳定 tag，例如 `v0.4.0` / `v0.5.0` 等）。
- 避免直接读写 OPF 的内部结构体字段，尽量通过 `Package` 的方法完成业务。

## 2. 导入路径与依赖方式

### 2.1 在本仓库内使用（最简单）

本仓库 `go.mod` 的 module 目前是 `golibri`，因此在仓库内直接：

```go
import "golibri/epub"
```

### 2.2 在外部 Go 项目中使用（当前建议：使用 replace）

由于 module path 目前不是 GitHub 路径，外部项目若要直接 `go get`，可能会遇到 module path 不匹配的问题。现阶段建议用 `replace` 引用仓库源码：

```go
module your-app

go 1.24

require golibri v0.0.0

replace golibri => /abs/path/to/go-epub-meta
```

如果你希望“直接 go get 远端仓库”成为官方支持路径，需要后续把 module path 迁移为 GitHub 路径（这会是一次破坏性变更，通常会配合 release 说明）。

## 3. 读取元数据（示例）

```go
package main

import (
	"fmt"
	"strings"

	"golibri/epub"
)

func main() {
	book, err := epub.Open("book.epub")
	if err != nil {
		panic(err)
	}
	defer book.Close()

	// 标准字段（EPUB/Dublin Core）
	fmt.Println("Title:", book.Package.GetTitle())
	fmt.Println("Authors:", strings.Join(book.Package.GetAuthors(), ", "))
	fmt.Println("Publisher:", book.Package.GetPublisher())
	fmt.Println("Published:", book.Package.GetPublishDate())
	fmt.Println("Language:", book.Package.GetLanguage())
	fmt.Println("Tags:", strings.Join(book.Package.GetSubjects(), ", "))
	fmt.Println("Comments:", book.Package.GetDescription())

	// 标识符
	fmt.Println("Identifiers:", book.Package.GetIdentifiers())
	fmt.Println("ISBN:", book.Package.GetISBN())
	fmt.Println("ASIN:", book.Package.GetASIN())

	// Calibre 扩展（如果存在）
	fmt.Println("Series:", book.Package.GetSeries(), "#", book.Package.GetSeriesIndex())
	fmt.Println("Rating(0-5):", book.Package.GetRating(), "Raw(0-10):", book.Package.GetRatingRaw())

	// Producer（常见来自 calibre 或生成工具）
	fmt.Println("Producer:", book.Package.GetProducer())
}
```

## 4. 写入元数据并保存（示例）

### 4.1 写入并输出到新文件

```go
package main

import "golibri/epub"

func main() {
	book, err := epub.Open("book.epub")
	if err != nil {
		panic(err)
	}
	defer book.Close()

	book.Package.SetTitle("新标题")
	book.Package.SetAuthor("新作者") // 注意：SetAuthor 设置为单字符串（适合你的业务输入）
	book.Package.SetPublisher("出版社")
	book.Package.SetPublishDate("2025-01-07")
	book.Package.SetLanguage("zh-CN")
	book.Package.SetSubjects([]string{"科幻", "历史"})
	book.Package.SetDescription("简介/描述文本")

	// Calibre 扩展
	book.Package.SetSeries("系列名")
	book.Package.SetSeriesIndex("3")
	book.Package.SetRating(4) // 0-5，会写入 calibre:rating (0-10)

	// 标识符
	book.Package.SetISBN("9781234567890")
	book.Package.SetASIN("B08XYZABC1")
	book.Package.SetIdentifier("douban", "12345678")

	if err := book.Save("output.epub"); err != nil {
		panic(err)
	}
}
```

### 4.2 原地覆盖保存（in-place）

`Save()` 支持原子写入，因此可以安全地原地覆盖（建议你仍做好备份策略）：

```go
if err := book.Save("book.epub"); err != nil {
    panic(err)
}
```

## 5. 封面读写（示例）

```go
rc, mediaType, err := book.GetCoverImage()
if err == nil {
	defer rc.Close()
	_ = mediaType
	// 读取封面数据自行保存
}

// 设置封面（data 为图片 bytes，mediaType: image/jpeg 或 image/png）
book.SetCover(data, "image/jpeg")
```

## 6. 关于 Comments/Description 里的 HTML

现实中很多 EPUB（尤其由 Calibre 生成/整理）会把 HTML 以“转义文本”的形式放入 `dc:description`。

- `Package.GetDescription()` 返回的是 **OPF 中的原始内容**（可能包含转义的 HTML）。
- 如果你的程序需要纯文本展示，你可以自行做 HTML strip / 截断策略（CLI 的行为就是为了易读做了处理）。

## 7. 元数据兼容性与维护策略 (EPUB 3 vs EPUB 2)

本库在写入元数据时，遵循 **“严格符合标准，同时有条件地维护兼容性”** 的原则，具体策略为 **“有则维护，无则不加”**。

### 7.1 EPUB 3 模式下的行为
当检测到 EPUB 版本为 3.0 或更高时，写入操作（如设置系列、评分、封面等）会优先使用 EPUB 3 标准的 `property` 属性或专门的 XML 结构（如 `belongs-to-collection`）。

对于非标准的旧式兼容标签（主要是 Calibre 引入的 `calibre:series`, `calibre:rating` 或 EPUB 2 风格的 `<meta name="cover">`）：
- **如果原文件中已存在这些标签**：本库会**同步更新**它们的值，以保证在旧设备上的兼容性不退化。
- **如果原文件中不存在这些标签**：本库**不会主动添加**它们。这保持了 EPUB 3 文件的“纯净度”，避免引入不必要的非标准元数据。

### 7.2 EPUB 2 模式下的行为
当检测到 EPUB 版本为 2.0 时，本库会自动回退到使用 `name` / `content` 属性的旧式元数据写法（如 `calibre:series`），以确保最大兼容性。

### 7.3 开发者提示
你不需要手动处理这些差异，只需调用统一的 API（如 `SetSeries`, `SetCover`），库内部会自动根据文件版本和现有内容应用上述策略。
