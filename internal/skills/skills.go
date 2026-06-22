package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed content/*/SKILL.md content/*/references/*.md
var content embed.FS

type Info struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version,omitempty"`
	CLIHelp     string `json:"cli_help,omitempty"`
}

func List() ([]Info, error) {
	entries, err := content.ReadDir("content")
	if err != nil {
		return nil, err
	}
	out := make([]Info, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		raw, err := Read(entry.Name(), "")
		if err != nil {
			return nil, err
		}
		out = append(out, parseInfo(entry.Name(), string(raw)))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func Read(name string, relPath string) ([]byte, error) {
	name = cleanName(name)
	if name == "" {
		return nil, fmt.Errorf("skill name is required")
	}
	if err := validateName(name); err != nil {
		return nil, err
	}
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		relPath = "SKILL.md"
	}
	path, err := cleanSkillPath(name, relPath)
	if err != nil {
		return nil, err
	}
	raw, err := content.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read skill %s %s: %w", name, relPath, err)
	}
	return raw, nil
}

func ListPath(name string, relPath string) ([]string, error) {
	name = cleanName(name)
	if name == "" {
		return nil, fmt.Errorf("skill name is required")
	}
	if err := validateName(name); err != nil {
		return nil, err
	}
	path, err := cleanSkillPath(name, relPath)
	if err != nil {
		return nil, err
	}
	entries, err := content.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}

func cleanName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, "/")
	if strings.Contains(name, "/") {
		name = strings.SplitN(name, "/", 2)[0]
	}
	return name
}

func validateName(name string) error {
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' {
			continue
		}
		return fmt.Errorf("invalid skill name %q", name)
	}
	return nil
}

func cleanSkillPath(name string, relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return path.Join("content", name), nil
	}
	if strings.HasPrefix(relPath, "/") || strings.Contains(relPath, "\\") {
		return "", fmt.Errorf("invalid skill path %q", relPath)
	}
	clean := path.Clean(relPath)
	if clean == "." {
		clean = ""
	}
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid skill path %q", relPath)
	}
	for _, part := range strings.Split(clean, "/") {
		if part == "." || part == ".." {
			return "", fmt.Errorf("invalid skill path %q", relPath)
		}
	}
	embedPath := path.Join("content", name, clean)
	if _, err := fs.Stat(content, embedPath); err != nil {
		return "", err
	}
	return embedPath, nil
}

func parseInfo(name string, raw string) Info {
	info := Info{Name: name}
	lines := strings.Split(raw, "\n")
	inFrontMatter := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if i == 0 && line == "---" {
			inFrontMatter = true
			continue
		}
		if inFrontMatter && line == "---" {
			break
		}
		if !inFrontMatter {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch key {
		case "name":
			info.Name = value
		case "description":
			info.Description = value
		case "version":
			info.Version = value
		case "cliHelp":
			info.CLIHelp = value
		}
	}
	return info
}
