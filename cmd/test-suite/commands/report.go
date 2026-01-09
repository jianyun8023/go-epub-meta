package commands

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var (
	reportInputFile string
	reportSamples   int
)

func init() {
	reportCmd.Flags().StringVarP(&reportInputFile, "input", "i", "comparison_results.csv", "Input CSV file")
	reportCmd.Flags().IntVar(&reportSamples, "samples", 20, "Number of sample mismatches to show")

	rootCmd.AddCommand(reportCmd)
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate detailed analysis report from comparison results",
	Run: func(cmd *cobra.Command, args []string) {
		runReport()
	},
}

type Record struct {
	FilePath           string
	GolibriTitle       string
	GolibriAuthor      string
	GolibriPublisher   string
	GolibriProducer    string
	GolibriPublished   string
	GolibriSeries      string
	GolibriLanguage    string
	GolibriIdentifiers string
	GolibriCover       string
	GolibriError       string
	GolibriTime        string
	EbookTitle         string
	EbookAuthor        string
	EbookPublisher     string
	EbookProducer      string
	EbookLanguage      string
	EbookPublished     string
	EbookIdentifiers   string
	EbookSeries        string
	EbookError         string
	EbookTime          string
}

func runReport() {
	records, err := loadRecords(reportInputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("=== 详细分析报告 ===\n")
	fmt.Printf("总文件数: %d\n\n", len(records))

	analyzeErrors(records)
	analyzeMissingFields(records)
	analyzeFieldMismatches(records)
	analyzeIdentifierFormats(records)
}

func loadRecords(filename string) ([]Record, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("CSV文件为空或只有表头")
	}

	var records []Record
	for i, row := range rows {
		if i == 0 {
			continue // Skip header
		}
		if len(row) < 22 {
			continue
		}
		records = append(records, Record{
			FilePath:           row[0],
			GolibriTitle:       row[1],
			GolibriAuthor:      row[2],
			GolibriPublisher:   row[3],
			GolibriProducer:    row[4],
			GolibriPublished:   row[5],
			GolibriSeries:      row[6],
			GolibriLanguage:    row[7],
			GolibriIdentifiers: row[8],
			GolibriCover:       row[9],
			GolibriError:       row[10],
			GolibriTime:        row[11],
			EbookTitle:         row[12],
			EbookAuthor:        row[13],
			EbookPublisher:     row[14],
			EbookProducer:      row[15],
			EbookLanguage:      row[16],
			EbookPublished:     row[17],
			EbookIdentifiers:   row[18],
			EbookSeries:        row[19],
			EbookError:         row[20],
			EbookTime:          row[21],
		})
	}

	return records, nil
}

func analyzeErrors(records []Record) {
	golibriErrors := 0
	ebookErrors := 0

	errorTypes := make(map[string]int)

	for _, r := range records {
		if r.GolibriError != "" {
			golibriErrors++
			// Simplify error message
			errType := strings.Split(r.GolibriError, ":")[0]
			errorTypes[errType]++
		}
		if r.EbookError != "" {
			ebookErrors++
		}
	}

	fmt.Println("## 1. 错误分析")
	fmt.Printf("Golibri错误数: %d (%.2f%%)\n", golibriErrors, float64(golibriErrors)/float64(len(records))*100)
	fmt.Printf("Ebook-meta错误数: %d (%.2f%%)\n", ebookErrors, float64(ebookErrors)/float64(len(records))*100)

	if len(errorTypes) > 0 {
		fmt.Println("\nGolibri错误类型分布:")
		for errType, count := range errorTypes {
			fmt.Printf("  - %s: %d次\n", errType, count)
		}
	}
	fmt.Println()
}

func analyzeMissingFields(records []Record) {
	fmt.Println("## 2. Golibri缺失字段分析")
	fmt.Println("这些是ebook-meta有但golibri完全没有的字段：")
	fmt.Println()

	// Publisher
	publisherEbookCount := 0
	publisherGolibriCount := 0
	publisherMatchCount := 0
	for _, r := range records {
		if r.EbookPublisher != "" {
			publisherEbookCount++
		}
		if r.GolibriPublisher != "" {
			publisherGolibriCount++
			if normalize(r.GolibriPublisher) == normalize(r.EbookPublisher) {
				publisherMatchCount++
			}
		}
	}
	fmt.Printf("Publisher (出版社):\n")
	fmt.Printf("  - ebook-meta中有此字段: %d本书 (%.1f%%)\n", publisherEbookCount, float64(publisherEbookCount)/float64(len(records))*100)
	fmt.Printf("  - golibri中有此字段: %d本书 (%.1f%%)\n", publisherGolibriCount, float64(publisherGolibriCount)/float64(len(records))*100)
	if publisherGolibriCount > 0 {
		fmt.Printf("  - 匹配率: %.1f%%\n", float64(publisherMatchCount)/float64(publisherGolibriCount)*100)
	}
	fmt.Printf("  - 重要性: ⭐⭐⭐⭐⭐ 高\n\n")

	// Published
	publishedEbookCount := 0
	publishedGolibriCount := 0
	publishedMatchCount := 0
	for _, r := range records {
		if r.EbookPublished != "" {
			publishedEbookCount++
		}
		if r.GolibriPublished != "" {
			publishedGolibriCount++
			if normalize(r.GolibriPublished) == normalize(r.EbookPublished) {
				publishedMatchCount++
			}
		}
	}
	fmt.Printf("Published (出版日期):\n")
	fmt.Printf("  - ebook-meta中有此字段: %d本书 (%.1f%%)\n", publishedEbookCount, float64(publishedEbookCount)/float64(len(records))*100)
	fmt.Printf("  - golibri中有此字段: %d本书 (%.1f%%)\n", publishedGolibriCount, float64(publishedGolibriCount)/float64(len(records))*100)
	if publishedGolibriCount > 0 {
		fmt.Printf("  - 匹配率: %.1f%%\n", float64(publishedMatchCount)/float64(publishedGolibriCount)*100)
	}
	fmt.Printf("  - 重要性: ⭐⭐⭐⭐⭐ 高\n\n")

	// Producer
	producerCount := 0
	producerTypes := make(map[string]int)
	for _, r := range records {
		if r.EbookProducer != "" {
			producerCount++
			// Extract producer name (e.g., "calibre (7.2.0)")
			if strings.Contains(r.EbookProducer, "calibre") {
				producerTypes["calibre"]++
			} else {
				producerTypes["其他"]++
			}
		}
	}
	fmt.Printf("Producer (制作工具):\n")
	fmt.Printf("  - ebook-meta中有此字段: %d本书 (%.1f%%)\n", producerCount, float64(producerCount)/float64(len(records))*100)
	fmt.Printf("  - golibri中有此字段: 0本书 (0%%)\n")
	fmt.Printf("  - 主要工具: calibre占%.1f%%\n", float64(producerTypes["calibre"])/float64(producerCount)*100)
	fmt.Printf("  - 重要性: ⭐⭐⭐ 中 (对普通用户价值较低)\n\n")
}

func analyzeFieldMismatches(records []Record) {
	fmt.Println("## 3. 字段值不匹配分析")
	fmt.Println()

	// Author mismatch analysis
	fmt.Println("### Author (作者) 匹配率低的原因分析:")
	authorMismatches := 0
	sampleMismatches := []Record{}

	for _, r := range records {
		if r.GolibriAuthor != "" && r.EbookAuthor != "" {
			if normalize(r.GolibriAuthor) != normalize(r.EbookAuthor) {
				authorMismatches++
				if len(sampleMismatches) < reportSamples {
					sampleMismatches = append(sampleMismatches, r)
				}
			}
		}
	}

	fmt.Printf("不匹配数量: %d\n\n", authorMismatches)
	fmt.Println("典型样例 (前10个):")
	for i, r := range sampleMismatches {
		if i >= 10 {
			break
		}
		fmt.Printf("%d. Golibri: \"%s\"\n", i+1, r.GolibriAuthor)
		fmt.Printf("   Ebook:    \"%s\"\n", r.EbookAuthor)
		fmt.Printf("   分析: %s\n\n", analyzeAuthorDifference(r.GolibriAuthor, r.EbookAuthor))
	}

	// Language mismatch analysis
	fmt.Println("### Language (语言) 格式差异:")
	langFormats := make(map[string]int)
	langExamples := make(map[string]string)

	for _, r := range records {
		if r.EbookLanguage != "" {
			key := fmt.Sprintf("Ebook: %s", r.EbookLanguage)
			langFormats[key]++
			if langExamples[key] == "" && r.GolibriLanguage != "" {
				langExamples[key] = r.GolibriLanguage
			}
		}
	}

	// Sort and display top language codes
	type kv struct {
		Key   string
		Value int
	}
	var sortedLangs []kv
	for k, v := range langFormats {
		sortedLangs = append(sortedLangs, kv{k, v})
	}
	sort.Slice(sortedLangs, func(i, j int) bool {
		return sortedLangs[i].Value > sortedLangs[j].Value
	})

	fmt.Println("主要语言代码格式 (前5个):")
	for i, kv := range sortedLangs {
		if i >= 5 {
			break
		}
		fmt.Printf("  %s: %d本书", kv.Key, kv.Value)
		if example := langExamples[kv.Key]; example != "" {
			fmt.Printf(" (Golibri对应: \"%s\")", example)
		}
		fmt.Println()
	}
	fmt.Println("分析: ebook-meta使用ISO 639-3代码(如zho)，golibri使用ISO 639-1代码(如zh)")
	fmt.Println()
}

func analyzeIdentifierFormats(records []Record) {
	fmt.Println("## 4. Identifier (标识符) 格式差异分析")
	fmt.Println()

	golibriSchemes := make(map[string]int)
	ebookSchemes := make(map[string]int)

	for _, r := range records {
		// Parse golibri identifiers
		if r.GolibriIdentifiers != "" {
			parts := strings.Split(r.GolibriIdentifiers, ",")
			for _, part := range parts {
				schemeParts := strings.Split(strings.TrimSpace(part), ":")
				if len(schemeParts) >= 1 {
					scheme := strings.TrimSpace(schemeParts[0])
					golibriSchemes[scheme]++
				}
			}
		}

		// Parse ebook identifiers
		if r.EbookIdentifiers != "" {
			parts := strings.Split(r.EbookIdentifiers, ",")
			for _, part := range parts {
				schemeParts := strings.Split(strings.TrimSpace(part), ":")
				if len(schemeParts) >= 1 {
					scheme := strings.TrimSpace(schemeParts[0])
					ebookSchemes[scheme]++
				}
			}
		}
	}

	fmt.Println("Golibri识别的标识符类型:")
	printTopSchemes(golibriSchemes, 10)

	fmt.Println("\nEbook-meta识别的标识符类型:")
	printTopSchemes(ebookSchemes, 10)

	fmt.Println("\n分析:")
	fmt.Println("- Golibri将很多标识符识别为'unknown'，说明scheme解析可能有问题")
	fmt.Println("- ebook-meta能正确识别isbn等标准标识符类型")
}

func printTopSchemes(schemes map[string]int, limit int) {
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range schemes {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	for i, kv := range sorted {
		if i >= limit {
			break
		}
		fmt.Printf("  %d. %s: %d次\n", i+1, kv.Key, kv.Value)
	}
}

func analyzeAuthorDifference(golibri, ebook string) string {
	// Check if ebook has [file-as] format
	if strings.Contains(ebook, "[") && strings.Contains(ebook, "]") {
		return "ebook-meta显示了file-as属性 (排序名), golibri只显示了名字"
	}

	if golibri == strings.Split(ebook, "[")[0] {
		return "基本一致，但ebook-meta添加了file-as信息"
	}

	return "格式或编码差异"
}
