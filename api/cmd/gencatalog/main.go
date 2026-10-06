package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type entry struct {
	Severity  string `yaml:"severity"`
	Retryable bool   `yaml:"retryable"`
	Action    string `yaml:"action"`
	Message   string `yaml:"message"`
}

// main reads openapi/errors.yaml and writes the Go and TypeScript catalogs.
// -check exits 1 when either generated file would change.
func main() {
	check := flag.Bool("check", false, "exit 1 when generated files differ")
	flag.Parse()

	root, err := moduleRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	catalog, err := readCatalog(filepath.Join(root, "openapi", "errors.yaml"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	files := map[string][]byte{
		filepath.Join(root, "internal", "platform", "catalog", "catalog.gen.go"): goSource(catalog),
		filepath.Join(root, "..", "src", "domain", "errorCatalog.gen.ts"):        tsSource(catalog),
	}

	failed := false
	for path, body := range files {
		if *check {
			existing, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(existing, body) {
				fmt.Fprintf(os.Stderr, "generated file is stale: %s\n", path)
				failed = true
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if failed {
		os.Exit(1)
	}
}

func moduleRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
		return wd, nil
	}
	parent := filepath.Dir(wd)
	if _, err := os.Stat(filepath.Join(parent, "go.mod")); err == nil {
		return parent, nil
	}
	return "", fmt.Errorf("run gencatalog from the api module")
}

func readCatalog(path string) (map[string]entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Errors map[string]entry `yaml:"errors"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if len(doc.Errors) == 0 {
		return nil, fmt.Errorf("errors.yaml has no entries")
	}
	return doc.Errors, nil
}

func sortedKeys(catalog map[string]entry) []string {
	keys := make([]string, 0, len(catalog))
	for key := range catalog {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func goSource(catalog map[string]entry) []byte {
	var b strings.Builder
	b.WriteString("// Code generated from openapi/errors.yaml. DO NOT EDIT.\n\n")
	b.WriteString("package catalog\n\n")
	b.WriteString("type Entry struct {\n")
	b.WriteString("\tCode      string\n")
	b.WriteString("\tSeverity  string\n")
	b.WriteString("\tRetryable bool\n")
	b.WriteString("\tAction    string\n")
	b.WriteString("\tMessage   string\n")
	b.WriteString("}\n\n")
	b.WriteString("var ByCode = map[string]Entry{\n")
	for _, code := range sortedKeys(catalog) {
		item := catalog[code]
		fmt.Fprintf(&b, "\t%q: {Code: %q, Severity: %q, Retryable: %t, Action: %q, Message: %q},\n",
			code, code, item.Severity, item.Retryable, item.Action, item.Message)
	}
	b.WriteString("}\n")
	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return []byte(b.String())
	}
	return formatted
}

func tsSource(catalog map[string]entry) []byte {
	var b strings.Builder
	b.WriteString("// Code generated from api/openapi/errors.yaml. DO NOT EDIT.\n\n")
	b.WriteString("export type ErrorCatalogEntry = {\n")
	b.WriteString("  code: string;\n")
	b.WriteString("  severity: \"error\" | \"warning\";\n")
	b.WriteString("  retryable: boolean;\n")
	b.WriteString("  action: string;\n")
	b.WriteString("  message: string;\n")
	b.WriteString("};\n\n")
	b.WriteString("export const errorCatalog = {\n")
	for _, code := range sortedKeys(catalog) {
		item := catalog[code]
		fmt.Fprintf(&b, "  %s: { code: %q, severity: %q, retryable: %t, action: %q, message: %q },\n",
			code, code, item.Severity, item.Retryable, item.Action, item.Message)
	}
	b.WriteString("} as const satisfies Record<string, ErrorCatalogEntry>;\n\n")
	b.WriteString("export type ErrorCode = keyof typeof errorCatalog;\n")
	return []byte(b.String())
}
