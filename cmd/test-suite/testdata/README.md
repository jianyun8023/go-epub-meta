# Test Data for Golibri Test Suite

This directory contains test EPUB files for validating golibri functionality.

## Directory Structure

```
testdata/
├── valid/              # Valid EPUB files for normal testing
│   ├── simple.epub
│   ├── with_metadata.epub
│   └── chinese.epub
├── invalid/            # Invalid/broken EPUB files for error handling tests
│   └── broken.epub
├── covers/             # Test cover images
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

### Invalid EPUBs

1. **broken.epub**
   - Malformed EPUB for error handling tests
   - Invalid container.xml

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

- Test files are intentionally small (~2KB each)
- Cover images should be added to `covers/` directory manually if needed
- All test EPUBs are valid EPUB 2.0 format
- The invalid EPUB is designed to test error handling, not crash the parser
