# Corrupted EPUB Test Files

这个目录包含 ZIP 结构损坏的 EPUB 文件，用于测试错误处理能力。

## 文件列表

| 文件 | 大小 | 问题描述 |
|------|------|----------|
| 255363.epub | 3.1M | 缺少 End-of-Central-Directory 签名 |
| 268984.epub | 24M | 缺少 End-of-Central-Directory 签名 |
| 272110.epub | 25M | 缺少 End-of-Central-Directory 签名 |
| 272111.epub | 16M | 缺少 End-of-Central-Directory 签名 |

## 问题特征

- 文件头正常 (`PK\x03\x04` - 有效的 Local File Header)
- **缺少 End-of-Central-Directory (EOCD) 签名**
- `unzip -t` 报告: "End-of-central-directory signature not found"
- Python `zipfile` 无法打开
- Go `archive/zip` 无法打开

## 工具兼容性

| 工具 | 结果 |
|------|------|
| Golibri | ❌ 失败 (zip: not a valid zip file) |
| Python zipfile | ❌ 失败 |
| unzip | ❌ 失败 |
| ebook-meta (Calibre) | ✅ 成功 |

## 为什么 ebook-meta 能解析？

Calibre 使用容错 ZIP 解析器，可以通过扫描本地文件头（而非中央目录）来恢复文件列表。

## 元数据信息 (由 ebook-meta 提取)

### 255363.epub
- Title: 合同审查精要与实务指南（第二版）
- Author: 雷霆
- Publisher: 法律出版社

### 272111.epub
- Title: 翁贝托·埃科重要代表作品集（套装共12册）
- Author: 翁贝托·埃科(Umberto Eco)
- Publisher: 上海译文出版社

### 272110.epub
- Title: 高品质摄影全流程解析（套装全9册）
- Author: 斯科特·凯尔比(Scott Kelby)
- Publisher: 人民邮电出版社

### 268984.epub
- Title: 疫情、人类与社会（套装共15册）
- Author: 加缪 & 普雷斯顿 & 桑塔格等
- Publisher: 上海译文出版社

## 用途

这些文件可用于：
1. 测试 Golibri 的错误处理行为
2. 未来实现容错 ZIP 解析功能时的测试用例
3. 与 ebook-meta 进行兼容性对比测试

