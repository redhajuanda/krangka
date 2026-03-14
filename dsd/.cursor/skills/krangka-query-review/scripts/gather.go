// gather.go extracts qwery RunRaw and Run queries from the repository for review.
// Uses qwery.Build() to get the actual generated query and params.
// Run: go run .cursor/skills/krangka-query-review/scripts/gather.go
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/qwery"
	"github.com/redhajuanda/qwery/pagination"
)

type BuiltVariant struct {
	Query            string   `json:"query"`
	Params           []any    `json:"params"`
	OptionalIncluded []string `json:"optional_included,omitempty"` // which optional params were passed
	PaginationType   string   `json:"pagination_type,omitempty"`   // "offset", "cursor_first", "cursor_next"
}

type QueryInfo struct {
	File           string         `json:"file"`
	Method         string         `json:"method"`
	Source         string         `json:"source"`               // "RunRaw" or "Run"
	QueryName      string         `json:"query_name,omitempty"` // for Run: e.g. "user.GetUser"
	Query          string         `json:"query"`
	Params         []string       `json:"params"`
	OptionalParams []string       `json:"optional_params,omitempty"` // params in {{ if .param }}
	Pagination     string         `json:"pagination,omitempty"`
	OrderBy        []string       `json:"order_by,omitempty"`
	LineStart      int            `json:"line_start"`
	BuiltVariants  []BuiltVariant `json:"built_variants,omitempty"` // all query possibilities from Build
	BuildError     string         `json:"build_error,omitempty"`    // if Build() failed
	HasPagination  bool           `json:"has_pagination"`           // true if chain has WithPagination
	HasOrderBy     bool           `json:"has_order_by"`             // true if chain has WithOrderBy
	PurposeHint    string         `json:"purpose_hint,omitempty"`   // from func doc comment above query
	QueryHash      string         `json:"query_hash,omitempty"`     // sha256 for diff (file+method+query+params)
	Change         string         `json:"change,omitempty"`         // "New"|"Updated"|"Unchanged" (set when last_hashes provided)
	chain          string         // raw chain (not in JSON) for pagination check
}

func main() {
	root := "."
	lastHashesPath := ""
	writeHashesPath := ""
	filterQuery := ""
	var posArgs []string
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--write-hashes":
			if i+1 < len(os.Args) {
				writeHashesPath = os.Args[i+1]
				i++
			}
		case "--filter":
			if i+1 < len(os.Args) {
				filterQuery = os.Args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(os.Args[i], "-") {
				posArgs = append(posArgs, os.Args[i])
			}
		}
	}
	if len(posArgs) >= 1 {
		root = posArgs[0]
	}
	if len(posArgs) >= 2 {
		lastHashesPath = posArgs[1]
	}

	searchDir := filepath.Join(root, "internal", "adapter", "outbound", "mariadb")
	queryFiles := loadQueryFiles(root)

	var results []QueryInfo

	err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		queries := extractQueries(path, queryFiles)
		for _, q := range queries {
			if filterQuery != "" {
				// Filter by QueryName (e.g. "ticket.ListTicket"), repository prefix (e.g. "ticket"), or "ListTicket"
				filterNorm := strings.TrimSuffix(strings.TrimSpace(filterQuery), ".sql")
				filterNorm = strings.ReplaceAll(filterNorm, "/", ".")
				match := q.Source == "Run" && (q.QueryName == filterNorm ||
					q.QueryName == filterQuery ||
					strings.HasSuffix(q.QueryName, "."+filterNorm) || // e.g. "ListTicket" matches "ticket.ListTicket"
					strings.HasPrefix(q.QueryName, filterNorm+".")) // e.g. "ticket" matches "ticket.ListTicket", "ticket.SimpleListTicket"
				if !match {
					continue
				}
			}
			results = append(results, q)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "walk error: %v\n", err)
		os.Exit(1)
	}

	// Use qwery.Build() to get actual generated query and params for each
	log := logger.New("gather")
	client := qwery.NewTestClient(log, queryFiles, qwery.Question)
	ctx := context.Background()
	for i := range results {
		buildQuery(&results[i], client, ctx)
	}

	// Populate purpose hints, hashes, and change status
	for i := range results {
		results[i].PurposeHint = extractPurposeHint(results[i])
		results[i].QueryHash = computeQueryHash(&results[i])
		results[i].HasPagination = results[i].Pagination != ""
		results[i].HasOrderBy = len(results[i].OrderBy) > 0
	}
	if lastHashesPath != "" {
		applyChangeStatus(results, lastHashesPath)
	}

	suggestedFilename := time.Now().Format("20060102150405") + "_query.md"
	out := map[string]any{
		"suggested_filename": suggestedFilename,
		"queries":            results,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "encode error: %v\n", err)
		os.Exit(1)
	}

	if writeHashesPath != "" {
		hashes := make(map[string]string)
		for _, q := range results {
			key := q.File + ":" + q.Method
			hashes[key] = q.QueryHash
		}
		data, _ := json.MarshalIndent(hashes, "", "  ")
		_ = os.WriteFile(writeHashesPath, data, 0644)
	}
}

// loadQueryFiles builds a map from qwery query name (e.g. "user.GetUser") to SQL content.
// Searches for .sql files in queries/ dirs (excluding migrations).
func loadQueryFiles(root string) map[string]string {
	m := make(map[string]string)
	queryDirs := []string{
		filepath.Join(root, "internal", "adapter", "outbound", "mariadb", "queries"),
		filepath.Join(root, "queries"),
	}
	for _, dir := range queryDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".sql") {
				return err
			}
			if strings.Contains(path, "migrations") {
				return nil
			}
			rel, _ := filepath.Rel(dir, path)
			rel = filepath.ToSlash(rel)
			// qwery: last dir + filename without .sql → "dir.Name"
			dirPart := filepath.Dir(rel)
			base := strings.TrimSuffix(filepath.Base(rel), ".sql")
			lastDir := filepath.Base(dirPart)
			queryName := lastDir + "." + base
			content, _ := os.ReadFile(path)
			m[queryName] = strings.TrimSpace(string(content))
			return nil
		})
	}
	return m
}

// sampleParam returns a sample value for a param name (for qwery.Build).
// Uses naming conventions instead of hardcoded field names.
func sampleParam(name string) any {
	n := strings.ToLower(name)
	switch {
	case n == "id" || strings.HasSuffix(n, "_id"):
		return "01HXXX0000000000000000000"
	case strings.HasPrefix(n, "is_") || strings.HasPrefix(n, "has_"):
		return false
	case strings.HasSuffix(n, "_at") || strings.Contains(n, "retry") || strings.Contains(n, "attempt"):
		return 0
	case strings.Contains(n, "date") || strings.HasSuffix(n, "_start") || strings.HasSuffix(n, "_end"):
		return "2026-01-01 00:00:00"
	case strings.HasSuffix(n, "stages") || strings.HasSuffix(n, "statuses") || strings.HasSuffix(n, "_ids"):
		return []string{"sample"}
	case n == "payload" || n == "data" || strings.Contains(n, "json"):
		return "{}"
	case n == "search" || n == "query" || strings.Contains(n, "term"):
		return "search term"
	case n == "status" || n == "stage" || n == "type" || n == "code" || n == "topic" || n == "target":
		return "sample"
	case strings.Contains(n, "error") || strings.Contains(n, "message"):
		return "error"
	default:
		return "?"
	}
}

// extractOptionalParams finds params used in {{ if .param }} conditionals.
func extractOptionalParams(query string) []string {
	seen := make(map[string]bool)
	// {{ if .param }} or {{- if .param }} or {{ if .param }}
	re := regexp.MustCompile(`\{\{-?\s*if\s+\.(\w+)\s*\}\}`)
	for _, m := range re.FindAllStringSubmatch(query, -1) {
		if len(m) > 1 {
			seen[m[1]] = true
		}
	}
	var out []string
	for k := range seen {
		out = append(out, k)
	}
	return out
}

// powerSet generates all subsets of optional params (2^n combinations).
func powerSet(opts []string) [][]string {
	n := len(opts)
	total := 1 << n
	result := make([][]string, total)
	for i := 0; i < total; i++ {
		var subset []string
		for j := 0; j < n; j++ {
			if i&(1<<j) != 0 {
				subset = append(subset, opts[j])
			}
		}
		result[i] = subset
	}
	return result
}

const maxOptionalSets = 8 // cap to avoid 2^n explosion for queries with many optional params

func buildQuery(q *QueryInfo, client *qwery.Client, ctx context.Context) {
	q.OptionalParams = extractOptionalParams(q.Query)

	// Build params for each combination of optional params (2^n variants)
	// Cap to avoid OOM for queries with many optional params (e.g. ListTicket has 40+)
	var optionalSets [][]string
	if len(q.OptionalParams) > 10 {
		// Use only empty set + single-param combinations
		optionalSets = [][]string{{}}
		for _, p := range q.OptionalParams {
			if len(optionalSets) >= maxOptionalSets {
				break
			}
			optionalSets = append(optionalSets, []string{p})
		}
	} else {
		optionalSets = powerSet(q.OptionalParams)
		if len(optionalSets) > maxOptionalSets {
			optionalSets = optionalSets[:maxOptionalSets]
		}
	}

	hasPagination := strings.Contains(q.chain, "WithPagination")
	hasOrderBy := strings.Contains(q.chain, "WithOrderBy")
	queryNorm := strings.TrimSpace(strings.ToUpper(strings.Split(q.Query, "\n")[0]))
	isSelect := strings.HasPrefix(queryNorm, "SELECT")
	generatePaginationVariants := hasPagination && hasOrderBy && isSelect && q.Pagination != "" && len(q.OrderBy) > 0

	// Pagination types to generate when query has WithPagination+WithOrderBy
	paginationConfigs := []struct {
		typ string
		pag *pagination.Pagination
	}{
		{"offset", &pagination.Pagination{Type: "offset", Page: 1, PerPage: 10}},
		{"cursor_first", &pagination.Pagination{Type: "cursor", PerPage: 10}},
	}

	for _, included := range optionalSets {
		params := make(map[string]any)
		for _, p := range q.Params {
			params[p] = sampleParam(p)
		}
		for _, p := range q.OptionalParams {
			if !contains(included, p) {
				delete(params, p)
			}
		}

		var runner qwery.Runnerer
		if q.Source == "Run" {
			runner = client.Run(q.QueryName)
		} else {
			runner = client.RunRaw(q.Query)
		}
		runner = runner.WithParams(params)

		if generatePaginationVariants {
			for _, cfg := range paginationConfigs {
				r := runner.WithPagination(cfg.pag).WithOrderBy(q.OrderBy...)
				result, err := r.Build(ctx)
				if err != nil {
					q.BuildError = err.Error()
					return
				}
				q.BuiltVariants = append(q.BuiltVariants, BuiltVariant{
					Query:            result.Query,
					Params:           result.Params,
					OptionalIncluded: included,
					PaginationType:   cfg.typ,
				})
				// Derive cursor_next from cursor_first (Build panics with cursor; construct manually)
				if cfg.typ == "cursor_first" {
					nextQuery, nextParams := deriveCursorNext(result.Query, result.Params, q.OrderBy)
					if nextQuery != "" {
						q.BuiltVariants = append(q.BuiltVariants, BuiltVariant{
							Query:            nextQuery,
							Params:           nextParams,
							OptionalIncluded: included,
							PaginationType:   "cursor_next",
						})
					}
				}
			}
		} else {
			result, err := runner.Build(ctx)
			if err != nil {
				q.BuildError = err.Error()
				return
			}
			q.BuiltVariants = append(q.BuiltVariants, BuiltVariant{
				Query:            result.Query,
				Params:           result.Params,
				OptionalIncluded: included,
			})
		}
	}
}

func contains(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

// deriveCursorNext constructs cursor-next-page query from cursor-first query.
// Inserts " AND (col > ?)" or " AND (col < ?)" before ORDER BY per reference.md.
// Supports single-column OrderBy only (e.g. "id", "+id", "-created_at").
func deriveCursorNext(firstQuery string, firstParams []any, orderBy []string) (string, []any) {
	if len(orderBy) == 0 {
		return "", nil
	}
	// Parse first column: "+id" -> id, asc; "-created_at" -> created_at, desc
	col := orderBy[0]
	asc := true
	if strings.HasPrefix(col, "-") {
		col = strings.TrimPrefix(col, "-")
		asc = false
	} else if strings.HasPrefix(col, "+") {
		col = strings.TrimPrefix(col, "+")
	}
	if col == "" {
		return "", nil
	}
	cond := col + " > ?"
	if !asc {
		cond = col + " < ?"
	}
	cursorWhere := " AND (" + cond + ")"

	// Find " ORDER BY" and insert cursor WHERE before it
	orderIdx := strings.Index(firstQuery, " ORDER BY ")
	if orderIdx < 0 {
		return "", nil
	}
	nextQuery := firstQuery[:orderIdx] + cursorWhere + firstQuery[orderIdx:]

	// Params: insert cursor value before last param (limit)
	// firstParams = [..., limit]; nextParams = [..., cursor_val, limit]
	cursorVal := "01HXXX0000000000000000000" // sample ULID for id; use 100 for numeric
	if col == "id" {
		cursorVal = "01HXXX0000000000000000000"
	} else if col == "created_at" {
		cursorVal = "2026-01-01 00:00:00"
	}
	nextParams := make([]any, 0, len(firstParams)+1)
	nextParams = append(nextParams, firstParams[:len(firstParams)-1]...)
	nextParams = append(nextParams, cursorVal)
	nextParams = append(nextParams, firstParams[len(firstParams)-1])

	return nextQuery, nextParams
}

func extractQueries(filePath string, queryFiles map[string]string) []QueryInfo {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	content := string(data)
	lines := strings.Split(content, "\n")

	var results []QueryInfo

	// Pattern 1: query := `...` then RunRaw(query)
	queryVarRe := regexp.MustCompile(`(?s)query\s*:=\s*` + "`" + `([^` + "`" + `]*)` + "`")
	runRawRe := regexp.MustCompile(`RunRaw\s*\(\s*` + "`" + `([^` + "`" + `]*)` + "`")
	runRawVarRe := regexp.MustCompile(`RunRaw\s*\(\s*query\s*\)`)

	// Find query variable and its content
	queryVarMatches := queryVarRe.FindAllStringSubmatchIndex(content, -1)
	for _, m := range queryVarMatches {
		if len(m) < 4 {
			continue
		}
		query := content[m[2]:m[3]]
		lineNum := 1 + strings.Count(content[:m[0]], "\n")

		// Find RunRaw(query) after this
		after := content[m[1]:]
		if !runRawVarRe.MatchString(after) {
			continue
		}

		// Find method name (func containing this)
		method := findMethod(lines, lineNum)

		// Find chain
		chainStart := m[1] + runRawVarRe.FindStringIndex(after)[1]
		chain := content[chainStart:]
		chainEnd := findChainEnd(chain)
		chain = chain[:chainEnd]

		params := extractParams(chain, query)
		pagination, orderBy := extractPaginationOrder(chain)

		results = append(results, QueryInfo{
			File:       filePath,
			Method:     method,
			Source:     "RunRaw",
			Query:      strings.TrimSpace(query),
			Params:     params,
			Pagination: pagination,
			OrderBy:    orderBy,
			LineStart:  lineNum,
			chain:      chain,
		})
	}

	// Pattern 2: RunRaw(`...`) inline
	inlineMatches := runRawRe.FindAllStringSubmatchIndex(content, -1)
	for _, m := range inlineMatches {
		if len(m) < 4 {
			continue
		}
		query := content[m[2]:m[3]]
		lineNum := 1 + strings.Count(content[:m[0]], "\n")

		// Skip if already captured (query var)
		already := false
		for _, r := range results {
			if r.LineStart == lineNum && r.Query == strings.TrimSpace(query) {
				already = true
				break
			}
		}
		if already {
			continue
		}

		method := findMethod(lines, lineNum)
		chainStart := m[1]
		chain := content[chainStart:]
		chainEnd := findChainEnd(chain)
		chain = chain[:chainEnd]

		params := extractParams(chain, query)
		pagination, orderBy := extractPaginationOrder(chain)

		results = append(results, QueryInfo{
			File:       filePath,
			Method:     method,
			Source:     "RunRaw",
			Query:      strings.TrimSpace(query),
			Params:     params,
			Pagination: pagination,
			OrderBy:    orderBy,
			LineStart:  lineNum,
			chain:      chain,
		})
	}

	// Pattern 3: Run("queryName") — SQL from .sql files
	runRe := regexp.MustCompile(`Run\s*\(\s*"([^"]+)"\s*\)`)
	runMatches := runRe.FindAllStringSubmatchIndex(content, -1)
	for _, m := range runMatches {
		if len(m) < 4 {
			continue
		}
		queryName := content[m[2]:m[3]]
		lineNum := 1 + strings.Count(content[:m[0]], "\n")

		query := queryFiles[queryName]
		if query == "" {
			query = "(query file not found: " + queryName + ")"
		}

		method := findMethod(lines, lineNum)
		chainStart := m[1]
		chain := content[chainStart:]
		chainEnd := findChainEnd(chain)
		chain = chain[:chainEnd]

		params := extractParams(chain, query)
		pagination, orderBy := extractPaginationOrder(chain)

		results = append(results, QueryInfo{
			File:       filePath,
			Method:     method,
			Source:     "Run",
			QueryName:  queryName,
			Query:      query,
			Params:     params,
			Pagination: pagination,
			OrderBy:    orderBy,
			LineStart:  lineNum,
			chain:      chain,
		})
	}

	return results
}

func findMethod(lines []string, lineNum int) string {
	funcRe := regexp.MustCompile(`func\s+\([^)]+\)\s+(\w+)\s*\(`)
	for i := lineNum - 1; i >= 0; i-- {
		if m := funcRe.FindStringSubmatch(lines[i]); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

func findChainEnd(s string) int {
	// Chain ends at .Query(...), .Exec(...), or .Build(...) - find first such call and its closing paren.
	// Also bound by next "func " to avoid including content from other functions (e.g. when chain
	// extraction spans multiple RunRaw blocks in the same file).
	if nextFunc := strings.Index(s, "\nfunc "); nextFunc >= 0 {
		s = s[:nextFunc]
	}
	// Allow whitespace/newlines between dot and method (e.g. ".ScanStruct(...).\n\t\tQuery(ctx)")
	endRe := regexp.MustCompile(`\.\s*(Query|Exec|Build)\s*\(`)
	matches := endRe.FindStringIndex(s)
	if matches != nil {
		open := matches[1] - 1 // position of (
		depth := 1
		for i := open + 1; i < len(s); i++ {
			switch s[i] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
	}
	return len(s)
}

func extractParams(chain string, query string) []string {
	seen := make(map[string]bool)
	// From WithParam("key", value)
	re := regexp.MustCompile(`WithParam\s*\(\s*"([^"]+)"`)
	for _, m := range re.FindAllStringSubmatch(chain, -1) {
		if len(m) > 1 {
			seen[m[1]] = true
		}
	}
	// From query template {{ .param }}
	templateRe := regexp.MustCompile(`\{\{\s*\.(\w+)\s*\}\}`)
	for _, m := range templateRe.FindAllStringSubmatch(query, -1) {
		if len(m) > 1 {
			seen[m[1]] = true
		}
	}
	var params []string
	for k := range seen {
		params = append(params, k)
	}
	return params
}

// extractPurposeHint extracts the first line of the func doc comment above the query.
func extractPurposeHint(q QueryInfo) string {
	data, err := os.ReadFile(q.File)
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	// Find the func line containing this query (search backward from lineNum)
	lineNum := q.LineStart - 1 // 0-based
	funcLine := -1
	funcRe := regexp.MustCompile(`func\s+\([^)]+\)\s+` + regexp.QuoteMeta(q.Method) + `\s*\(`)
	for i := lineNum; i >= 0; i-- {
		if funcRe.MatchString(lines[i]) {
			funcLine = i
			break
		}
	}
	if funcLine < 0 {
		return ""
	}
	// Collect doc comment lines immediately above the func
	var doc []string
	for i := funcLine - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "//") {
			doc = append([]string{strings.TrimPrefix(strings.TrimPrefix(line, "//"), " ")}, doc...)
		} else if line == "" || strings.HasPrefix(line, "/*") {
			continue
		} else {
			break
		}
	}
	if len(doc) == 0 {
		return ""
	}
	return doc[0]
}

// applyChangeStatus sets Change (New/Updated/Unchanged) by comparing with last hashes.
// lastHashesPath should point to a JSON file: { "file:method": "hash", ... }.
func applyChangeStatus(results []QueryInfo, lastHashesPath string) {
	data, err := os.ReadFile(lastHashesPath)
	if err != nil {
		return
	}
	var last map[string]string
	if err := json.Unmarshal(data, &last); err != nil {
		return
	}
	for i := range results {
		key := results[i].File + ":" + results[i].Method
		prevHash, ok := last[key]
		if !ok {
			results[i].Change = "New"
		} else if prevHash == results[i].QueryHash {
			results[i].Change = "Unchanged"
		} else {
			results[i].Change = "Updated"
		}
	}
}

// computeQueryHash returns a stable hash for diff (file+method+query+params).
func computeQueryHash(q *QueryInfo) string {
	h := sha256.New()
	h.Write([]byte(q.File))
	h.Write([]byte("\x00"))
	h.Write([]byte(q.Method))
	h.Write([]byte("\x00"))
	h.Write([]byte(q.Query))
	h.Write([]byte("\x00"))
	for _, p := range q.Params {
		h.Write([]byte(p))
		h.Write([]byte("\x00"))
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func extractPaginationOrder(chain string) (string, []string) {
	var pagination string
	var orderBy []string

	if strings.Contains(chain, "WithPagination") {
		if strings.Contains(chain, `Type:\s*"cursor"`) || strings.Contains(chain, `"cursor"`) {
			pagination = "cursor"
		} else {
			pagination = "offset"
		}
	}

	// WithOrderBy("id") or WithOrderBy("+id", "-created_at")
	orderRe := regexp.MustCompile(`WithOrderBy\s*\(\s*([^)]+)\)`)
	if m := orderRe.FindStringSubmatch(chain); len(m) > 1 {
		parts := strings.Split(m[1], ",")
		for _, p := range parts {
			p = strings.Trim(p, `" \t`)
			if p != "" {
				orderBy = append(orderBy, p)
			}
		}
	}

	return pagination, orderBy
}