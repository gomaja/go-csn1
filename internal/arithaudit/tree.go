package arithaudit

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func ScanTree(root string, dirs []string) ([]Finding, error) {
	var findings []Finding
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fileFindings, err := ScanSource(filepath.ToSlash(rel), contents)
			if err != nil {
				return err
			}
			findings = append(findings, fileFindings...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].Key() < findings[j].Key() })
	return findings, nil
}

func ReadAllowlist(path string) (map[string]Classification, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	allowed := make(map[string]Classification)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 || strings.TrimSpace(parts[2]) == "" {
			return nil, fmt.Errorf("invalid arithmetic allowlist line %q", line)
		}
		count, err := strconv.Atoi(parts[1])
		if err != nil || count <= 0 {
			return nil, fmt.Errorf("invalid arithmetic allowlist count %q", parts[1])
		}
		if _, duplicate := allowed[parts[0]]; duplicate {
			return nil, fmt.Errorf("duplicate arithmetic allowlist key %q", parts[0])
		}
		allowed[parts[0]] = Classification{Count: count, Reason: parts[2]}
	}
	return allowed, scanner.Err()
}
