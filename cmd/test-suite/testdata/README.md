# Test Data for Golibri Test Suite

This directory contains test EPUB files for validating golibri functionality.

## Directory Structure

```
testdata/
├── valid/              # Valid EPUB files for normal testing
│   ├── simple.epub
│   ├── with_metadata.epub
│   ├── chinese.epub
│   ├── high_rating_tags.epub
│   ├── multi_author_7.epub
│   ├── multi_author_tags.epub
│   ├── rich_metadata_full.epub
│   └── rich_metadata_series.epub
├── invalid/            # Invalid/broken EPUB files for error handling tests
│   └── (reserved for error handling tests)
├── corrupted/          # Corrupted EPUB files for fault tolerance tests
├── samples/            # Real-world EPUB samples by version
│   ├── epub2/          # EPUB 2.x format (5 files)
│   ├── epub3-pure/     # EPUB 3.x pure format (5 files)
│   ├── epub3-hybrid/   # EPUB 3.x with NCX (5 files)
│   └── oebps1/         # OEBPS 1.x format (5 files)
└── generate_test_epubs.go  # Script to regenerate test files
```

## Test Files

### Valid EPUBs

1. **simple.epub**
   - Basic EPUB with minimal metadata
   - Title: "Simple Test Book"
   - Author: "Test Author"
   - Language: en
   - ISBN: 9781234567890

2. **with_metadata.epub**
   - Complete EPUB with all metadata fields
   - Title: "Complete Test Book"
   - Author: "John Doe"
   - Language: en
   - ISBN: 9780987654321
   - Series: "Test Series"
   - Publisher: "Test Publisher"
   - Published: 2024-01-01

3. **chinese.epub**
   - EPUB with Chinese metadata (UTF-8 support test)
   - Title: "测试书籍"
   - Author: "张三"
   - Language: zh
   - ISBN: 9787544798501
   - Series: "测试系列"
   - Publisher: "测试出版社"

4. **high_rating_tags.epub**
   - Real-world EPUB with high rating and rich tags

5. **multi_author_7.epub**
   - EPUB with 7 authors (multi-author parsing test)

6. **multi_author_tags.epub**
   - EPUB with multiple authors and tags

7. **rich_metadata_full.epub**
   - EPUB with comprehensive metadata

8. **rich_metadata_series.epub**
   - EPUB with detailed series information

### Corrupted EPUBs

See `corrupted/README.md` for details on fault tolerance test cases.

## Regenerating Test Files

To regenerate all test EPUB files:

```bash
cd testdata
go run generate_test_epubs.go
```

## Using Test Data

### Functional Tests

```bash
# Run all functional tests
../test-suite functional testdata/valid

# Run specific test mode
../test-suite functional testdata/valid --mode json
../test-suite functional testdata/valid --mode roundtrip
```

### Comparison Tests

```bash
# Compare with ebook-meta (text format)
../test-suite compare testdata/valid

# Compare with ebook-meta (JSON format, more accurate)
../test-suite compare testdata/valid --format json
```

## Notes

- Test files range from ~2KB (synthetic) to ~20MB (real-world)
- All test EPUBs are valid EPUB 2.0 format unless otherwise noted
- The corrupted EPUBs are designed to test error handling and fault tolerance
