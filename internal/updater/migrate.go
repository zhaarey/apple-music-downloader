package updater

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"golang.org/x/term"
)

type bytesProvider []byte

func (b bytesProvider) ReadBytes() ([]byte, error) {
	return []byte(b), nil
}

func (b bytesProvider) Read() (map[string]any, error) {
	return nil, nil
}

type MissingOption struct {
	Key          string
	Section      string
	SubKey       string
	DefaultValue any
	Comment      string
	ExampleLine  string
}

// FindMissingOptions compares the user config against the example config and returns missing options.
func FindMissingOptions(userConfigPath string, exampleContent string) ([]MissingOption, error) {
	if _, err := os.Stat(userConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", userConfigPath)
	}

	userK := koanf.New(".")
	if err := userK.Load(file.Provider(userConfigPath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("parse user config: %w", err)
	}

	exampleK := koanf.New(".")
	if err := exampleK.Load(bytesProvider(exampleContent), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("parse example config: %w", err)
	}

	var missing []MissingOption
	for _, k := range exampleK.Keys() {
		if !userK.Exists(k) {
			val := exampleK.Get(k)
			comment, exampleLine := extractCommentAndLine(exampleContent, k)
			parts := strings.Split(k, ".")
			section := parts[0]
			subKey := strings.Join(parts[1:], ".")

			missing = append(missing, MissingOption{
				Key:          k,
				Section:      section,
				SubKey:       subKey,
				DefaultValue: val,
				Comment:      comment,
				ExampleLine:  exampleLine,
			})
		}
	}

	return missing, nil
}

func extractCommentAndLine(exampleContent, fullKey string) (string, string) {
	parts := strings.Split(fullKey, ".")
	targetLeaf := parts[len(parts)-1]

	scanner := bufio.NewScanner(strings.NewReader(exampleContent))
	var prevComment string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "#") {
			prevComment = strings.TrimPrefix(trimmed, "#")
			prevComment = strings.TrimSpace(prevComment)
			continue
		}

		if strings.Contains(line, ":") {
			keyPart := strings.TrimSpace(strings.SplitN(line, ":", 2)[0])
			if keyPart == targetLeaf {
				inlineComment := ""
				if idx := strings.Index(line, "#"); idx != -1 {
					inlineComment = strings.TrimSpace(line[idx+1:])
				}

				comment := inlineComment
				if comment == "" {
					comment = prevComment
				}
				return comment, line
			}
			prevComment = ""
		}
	}

	return "", ""
}

// BackupConfigFile creates a timestamped backup of the config file.
func BackupConfigFile(configPath string, versionTag string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("%s.bak_%s_%s", base, strings.TrimPrefix(versionTag, "v"), timestamp)
	backupPath := filepath.Join(dir, backupName)

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("write backup file %s: %w", backupPath, err)
	}

	return backupPath, nil
}

// InsertOptionIntoYAML inserts a new key into the corresponding YAML section or appends it.
func InsertOptionIntoYAML(yamlContent, section, lineToInsert string) string {
	lines := strings.Split(yamlContent, "\n")
	var newLines []string
	inserted := false

	sectionHeader := section + ":"

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		newLines = append(newLines, line)

		if !inserted && strings.HasPrefix(strings.TrimSpace(line), sectionHeader) {
			// Found the section header! Find the end of this section
			j := i + 1
			for j < len(lines) {
				nextLine := lines[j]
				trimmed := strings.TrimSpace(nextLine)
				// If line is not empty and not indented, we hit the next section
				if trimmed != "" && !strings.HasPrefix(nextLine, " ") && !strings.HasPrefix(nextLine, "\t") && !strings.HasPrefix(trimmed, "#") {
					break
				}
				j++
			}

			// Insert before j
			for k := i + 1; k < j; k++ {
				newLines = append(newLines, lines[k])
			}
			newLines = append(newLines, lineToInsert)
			i = j - 1
			inserted = true
		}
	}

	if !inserted {
		// Section not found, append section and line to the bottom
		if len(newLines) > 0 && newLines[len(newLines)-1] != "" {
			newLines = append(newLines, "")
		}
		newLines = append(newLines, sectionHeader)
		newLines = append(newLines, lineToInsert)
	}

	return strings.Join(newLines, "\n")
}

// RunConfigMigrationGuide runs the interactive or automated config migration.
func RunConfigMigrationGuide(configPath, exampleContent, versionTag string, autoYes bool) (bool, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Config file %s does not exist, writing default template.\n", configPath)
		return true, os.WriteFile(configPath, []byte(exampleContent), 0644)
	}

	missing, err := FindMissingOptions(configPath, exampleContent)
	if err != nil {
		return false, err
	}

	if len(missing) == 0 {
		fmt.Println("✔ Configuration is already up to date. No new options needed.")
		return false, nil
	}

	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("==================================================================")
	fmt.Printf("🔍 Detected %d new configuration options introduced in %s:\n", len(missing), versionTag)
	fmt.Println("==================================================================")

	backupPath, err := BackupConfigFile(configPath, versionTag)
	if err != nil {
		return false, fmt.Errorf("failed to create config backup: %w", err)
	}
	fmt.Printf("📦 Original configuration backed up to: %s\n\n", backupPath)

	contentBytes, err := os.ReadFile(configPath)
	if err != nil {
		return false, err
	}
	currentContent := string(contentBytes)
	modifiedCount := 0

	for i, opt := range missing {
		desc := opt.Comment
		if desc == "" {
			desc = "No description available"
		}

		fmt.Printf("[%d/%d] Option: \033[1;36m%s\033[0m\n", i+1, len(missing), opt.Key)
		fmt.Printf("      Description: %s\n", desc)
		fmt.Printf("      Default:     \033[1;32m%v\033[0m\n", opt.DefaultValue)

		applyVal := fmt.Sprintf("%v", opt.DefaultValue)
		skip := false

		if !autoYes && isTTY {
			fmt.Printf("      Accept default? [Y/n/custom value]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if strings.EqualFold(input, "n") {
				fmt.Println("      -> Skipped.")
				skip = true
			} else if input != "" && !strings.EqualFold(input, "y") {
				applyVal = input
				fmt.Printf("      -> Using custom value: %s\n", applyVal)
			} else {
				fmt.Printf("      -> Accepted default: %s\n", applyVal)
			}
		} else {
			fmt.Println("      -> Auto-accepted default value.")
		}

		if !skip {
			// Construct formatted line with indentation and comment
			insertLine := fmt.Sprintf("  %s: %s", opt.SubKey, formatYAMLValue(applyVal, opt.DefaultValue))
			if opt.Comment != "" {
				insertLine += fmt.Sprintf("   # %s", opt.Comment)
			}

			currentContent = InsertOptionIntoYAML(currentContent, opt.Section, insertLine)
			modifiedCount++
		}
		fmt.Println()
	}

	if modifiedCount > 0 {
		if err := os.WriteFile(configPath, []byte(currentContent), 0644); err != nil {
			return false, fmt.Errorf("write updated config: %w", err)
		}
		fmt.Printf("✔ Successfully updated %s with %d new options!\n", configPath, modifiedCount)
		return true, nil
	}

	fmt.Println("✔ No options were added. Configuration remained unchanged.")
	return false, nil
}

func formatYAMLValue(valStr string, originalVal any) string {
	switch originalVal.(type) {
	case string:
		if !strings.HasPrefix(valStr, "\"") && !strings.HasPrefix(valStr, "'") {
			return fmt.Sprintf("%q", valStr)
		}
		return valStr
	default:
		return valStr
	}
}
