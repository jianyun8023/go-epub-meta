## 当前元数据字段支持情况

| 字段 | 读取 | 写入 | CLI 显示 |
|------|------|------|----------|
| Title | ✅ | ✅ | ✅ |
| Author(s) | ✅ | ✅ | ✅ |
| Publisher | ✅ | ✅ | ✅ |
| Published | ✅ | ✅ | ✅ |
| Language | ✅ | ✅ | ✅ |
| Series | ✅ | ✅ | ✅ |
| SeriesIndex | ✅ | ✅ | ✅ |
| Tags | ✅ | ✅ | ✅ |
| Rating | ✅ | ✅ | ✅ |
| Identifiers | ✅ | ✅ | ✅ |
| Producer | ✅ | ❌ | ✅ |
| Comments | ✅ | ✅ | ✅ |
| Cover | ✅ | ✅ | ✅ |

### 写入支持的 CLI Flags

```
-t, --title         设置标题
-a, --author        设置作者
-s, --series        设置系列名
    --series-index  设置系列索引 (Calibre 扩展)
-c, --cover         设置封面图片
    --isbn          设置 ISBN
    --asin          设置 ASIN
-i, --identifier    设置标识符 (scheme:value)
    --publisher     设置出版社
    --date          设置出版日期
    --language      设置语言 (e.g., zh, en, zh-CN)
    --tags          设置标签 (逗号分隔)
    --comments      设置简介/描述
    --rating        设置评分 (0-5, Calibre 扩展)
```

### 规范说明

| 字段 | 规范来源 |
|------|----------|
| Title, Author, Publisher, Language, Comments, Tags, Identifiers | **EPUB 标准** (Dublin Core) |
| Series, SeriesIndex, Rating | **Calibre 扩展** |
| Producer | 只读（工具自动生成） |
