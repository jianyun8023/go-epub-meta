# EPUB Data Analysis Report

**Date:** 2026-01-08  
**Source:** Analyzed 10,000 files from test dataset.

## Summary

The dataset is predominantly **EPUB 2.0**, with a significant minority of **EPUB 3.x**. The vast majority of EPUB 3 files are backward compatible (Hybrid) with EPUB 2 readers.

| Metric | Count | Percentage |
| :--- | :--- | :--- |
| **Total Files** | 10,000 | 100% |
| **EPUB 2.x** | 8,747 | 87.5% |
| **EPUB 3.x** | 1,243 | 12.4% |
| **OEBPS 1.x** | 6 | < 0.1% |
| **Errors** | 4 | < 0.1% |

## EPUB 3.x Breakdown

We analyzed the EPUB 3 files for the presence of an NCX file (the navigation center for EPUB 2).

- **Hybrid (Backward Compatible)**: 1,115 (89.7% of all EPUB 3)
  - contain `toc.ncx`
  - readable by older devices
- **Pure EPUB 3**: 128 (10.3% of all EPUB 3)
  - rely solely on `nav` document
  - require modern reader

## Data Quality Issues

- **Corruption**: 4 files were invalid ZIP archives.
- **Legacy Formats**: 6 files identified as version "1.0" (OEBPS 1.0/1.2), pre-dating the EPUB 2.0 standard.

## Conclusion

Support for EPUB 2 is critical. EPUB 3 support should prioritize Hybrid compatibility handling.
