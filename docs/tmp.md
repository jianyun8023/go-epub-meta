## 当前元数据字段支持情况

| 字段 | 读取 | 写入 | CLI 显示 |
|------|------|------|----------|
| Title | ✅ | ✅ | ✅ |
| Author(s) | ✅ | ✅ | ✅ |
| Publisher | ✅ | ❌ | ✅ |
| Published | ✅ | ❌ | ✅ |
| Language | ✅ | ❌ | ✅ |
| Series | ✅ | ✅ | ✅ |
| SeriesIndex | ✅ | ❌ | ✅ |
| Tags | ✅ | ❌ | ✅ |
| Rating | ✅ | ❌ | ✅ |
| Identifiers | ✅ | ✅ | ✅ |
| Producer | ✅ | ❌ | ✅ |
| Comments | ✅ | ❌ | ✅ |
| Cover | ✅ | ✅ | ✅ |

### 写入支持的 CLI Flags

```
-t, --title       设置标题
-a, --author      设置作者
-s, --series      设置系列名
-c, --cover       设置封面图片
    --isbn        设置 ISBN
    --asin        设置 ASIN
-i, --identifier  设置标识符 (scheme:value)
```
