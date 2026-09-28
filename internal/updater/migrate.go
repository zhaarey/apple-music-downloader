package updater

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
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

// commentColumn is where inline comments start in config.yaml.example.
const commentColumn = 42

type MissingOption struct {
	Key          string // full dotted path, e.g. "media.mv.audio-type"
	DefaultValue any    // parsed default from the example
	DefaultText  string // default exactly as written in the example, e.g. `"atmos"`
	Comment      string
}

// yamlKeyLine is one `key: value` line of a block-mapping YAML document.
type yamlKeyLine struct {
	path    []string
	indent  int
	line    int
	value   string // raw value text, without inline comment; "" for a nested mapping
	comment string // inline comment text
}

// scanYAMLKeys returns the block-mapping keys of a YAML document in order, with
// their full paths. It covers what config files use: nested mappings with scalar
// values plus full-line and inline comments. Lines that are not `key: value`
// (sequence items, block scalar content) are skipped with everything under them.
func scanYAMLKeys(lines []string) []yamlKeyLine {
	var out, stack []yamlKeyLine
	skipDeeperThan := -1

	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))

		if skipDeeperThan >= 0 {
			if indent > skipDeeperThan || (indent == skipDeeperThan && strings.HasPrefix(trimmed, "-")) {
				continue
			}
			skipDeeperThan = -1
		}

		key, rest, ok := splitKey(trimmed)
		if !ok {
			skipDeeperThan = indent
			continue
		}
		value, comment := splitInlineComment(rest)

		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		path := make([]string, 0, len(stack)+1)
		for _, parent := range stack {
			path = append(path, parent.path[len(parent.path)-1])
		}
		path = append(path, key)

		kl := yamlKeyLine{path: path, indent: indent, line: i, value: value, comment: comment}
		out = append(out, kl)

		switch {
		case value == "":
			stack = append(stack, kl)
		case strings.HasPrefix(value, "|") || strings.HasPrefix(value, ">"):
			skipDeeperThan = indent
		}
	}
	return out
}

// splitKey splits `key: rest` and reports whether the line is a mapping entry.
func splitKey(trimmed string) (key, rest string, ok bool) {
	if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		return "", "", false
	}
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == ':' && (i+1 == len(trimmed) || trimmed[i+1] == ' ' || trimmed[i+1] == '\t') {
			key = strings.Trim(strings.TrimSpace(trimmed[:i]), `"'`)
			return key, trimmed[i+1:], key != ""
		}
	}
	return "", "", false
}

// splitInlineComment separates a value from its trailing `# comment`,
// ignoring '#' inside quoted strings or not preceded by whitespace.
func splitInlineComment(s string) (value, comment string) {
	inSingle, inDouble := false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && inDouble:
			i++
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '#' && !inSingle && !inDouble && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t'):
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
		}
	}
	return strings.TrimSpace(s), ""
}

// commentAbove returns the contiguous full-line comment block directly above line i.
func commentAbove(lines []string, i int) string {
	var parts []string
	for j := i - 1; j >= 0; j-- {
		t := strings.TrimSpace(lines[j])
		if !strings.HasPrefix(t, "#") {
			break
		}
		t = strings.TrimSpace(strings.TrimPrefix(t, "#"))
		if strings.Trim(t, "-=# ") == "" {
			break // section banner, not a description
		}
		parts = append([]string{t}, parts...)
	}
	return strings.Join(parts, " ")
}

func splitLines(content string) []string {
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
}

func parseYAML(content string) (*koanf.Koanf, error) {
	k := koanf.New(".")
	if err := k.Load(bytesProvider(content), yaml.Parser()); err != nil {
		return nil, err
	}
	return k, nil
}

// FindMissingOptions compares the user config against the example config and
// returns the missing options in the example's document order.
func FindMissingOptions(userConfigPath string, exampleContent string) ([]MissingOption, error) {
	data, err := os.ReadFile(userConfigPath)
	if err != nil {
		return nil, err
	}
	userK, err := parseYAML(normalizeNewlines(string(data)))
	if err != nil {
		return nil, fmt.Errorf("parse user config: %w", err)
	}

	exampleContent = normalizeNewlines(exampleContent)
	exampleK, err := parseYAML(exampleContent)
	if err != nil {
		return nil, fmt.Errorf("parse example config: %w", err)
	}

	exampleLines := splitLines(exampleContent)
	byKey := make(map[string]yamlKeyLine)
	for _, kl := range scanYAMLKeys(exampleLines) {
		byKey[strings.Join(kl.path, ".")] = kl
	}

	var missing []MissingOption
	for _, k := range exampleK.Keys() {
		if userK.Exists(k) {
			continue
		}
		opt := MissingOption{Key: k, DefaultValue: exampleK.Get(k)}
		if kl, ok := byKey[k]; ok {
			opt.DefaultText = kl.value
			opt.Comment = kl.comment
			if opt.Comment == "" {
				opt.Comment = commentAbove(exampleLines, kl.line)
			}
		} else {
			opt.DefaultText = formatScalar(opt.DefaultValue)
		}
		missing = append(missing, opt)
	}

	line := func(k string) int {
		if kl, ok := byKey[k]; ok {
			return kl.line
		}
		return len(exampleLines)
	}
	sort.SliceStable(missing, func(i, j int) bool { return line(missing[i].Key) < line(missing[j].Key) })
	return missing, nil
}

// formatScalar renders a parsed scalar as YAML text.
func formatScalar(v any) string {
	if s, ok := v.(string); ok {
		return strconv.Quote(s)
	}
	return fmt.Sprintf("%v", v)
}

// parseCustomValue parses user input for an option and checks it against the
// default's type, returning the YAML text to write and the value it parses to.
func parseCustomValue(input string, def any) (string, any, error) {
	switch def.(type) {
	case string:
		s := input
		switch {
		case len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"':
			u, err := strconv.Unquote(input)
			if err != nil {
				return "", nil, fmt.Errorf("invalid quoted string %s", input)
			}
			s = u
		case len(input) >= 2 && input[0] == '\'' && input[len(input)-1] == '\'':
			s = strings.ReplaceAll(input[1:len(input)-1], "''", "'")
		}
		// Always quote, so input like `true` or `123` stays a string.
		return strconv.Quote(s), s, nil
	case bool:
		switch strings.ToLower(input) {
		case "true":
			return "true", true, nil
		case "false":
			return "false", false, nil
		}
		return "", nil, fmt.Errorf("expected true or false, got %q", input)
	case int, int64:
		n, err := strconv.Atoi(input)
		if err != nil {
			return "", nil, fmt.Errorf("expected an integer, got %q", input)
		}
		return strconv.Itoa(n), n, nil
	case float64:
		if _, err := strconv.ParseFloat(input, 64); err != nil {
			return "", nil, fmt.Errorf("expected a number, got %q", input)
		}
		// Let the YAML parser decide int vs float, as it will when loading the file.
		k, err := parseYAML("v: " + input)
		if err != nil {
			return "", nil, fmt.Errorf("expected a number, got %q", input)
		}
		return input, k.Get("v"), nil
	}
	return "", nil, fmt.Errorf("custom values are not supported for this option; edit the config file manually")
}

// detectIndentUnit returns the indentation step used by a document (default 2).
func detectIndentUnit(keys []yamlKeyLine) int {
	unit := 0
	for i := 1; i < len(keys); i++ {
		if len(keys[i].path) == len(keys[i-1].path)+1 {
			if d := keys[i].indent - keys[i-1].indent; d > 0 && (unit == 0 || d < unit) {
				unit = d
			}
		}
	}
	if unit == 0 {
		unit = 2
	}
	return unit
}

func pathEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// InsertOption inserts `key: valueText  # comment` at the nested position given by
// path, creating any missing parent mappings, and returns the new content.
// The entry goes after the last existing child of its deepest existing parent,
// so it never lands among the next section's leading comments.
func InsertOption(content string, path []string, valueText, comment string) string {
	lines := splitLines(content)
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	keys := scanYAMLKeys(lines)
	unit := detectIndentUnit(keys)

	// Deepest existing mapping that is an ancestor of path.
	depth := 0
	var parent *yamlKeyLine
	for d := len(path) - 1; d >= 1 && parent == nil; d-- {
		for i := range keys {
			if keys[i].value == "" && pathEqual(keys[i].path, path[:d]) {
				parent, depth = &keys[i], d
				break
			}
		}
	}

	baseIndent := 0
	insertAt := len(lines)
	if parent != nil {
		baseIndent = parent.indent + unit
		insertAt = parent.line + 1
		firstChild := true
		for j := parent.line + 1; j < len(lines); j++ {
			t := strings.TrimSpace(lines[j])
			if t == "" || strings.HasPrefix(t, "#") {
				continue
			}
			indent := len(lines[j]) - len(strings.TrimLeft(lines[j], " "))
			if indent <= parent.indent {
				break
			}
			if firstChild {
				baseIndent = indent
				firstChild = false
			}
			insertAt = j + 1
		}
	}

	var newLines []string
	if parent == nil && len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		newLines = append(newLines, "")
	}
	for j := depth; j < len(path); j++ {
		prefix := strings.Repeat(" ", baseIndent+(j-depth)*unit) + path[j] + ":"
		if j < len(path)-1 {
			newLines = append(newLines, prefix)
			continue
		}
		line := prefix + " " + valueText
		if comment != "" {
			pad := commentColumn - len(line)
			if pad < 2 {
				pad = 2
			}
			line += strings.Repeat(" ", pad) + "# " + comment
		}
		newLines = append(newLines, line)
	}

	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:insertAt]...)
	result = append(result, newLines...)
	result = append(result, lines[insertAt:]...)
	return strings.Join(result, "\n") + "\n"
}

// verifyMigration checks that updated keeps every existing option unchanged and
// contains exactly the added options with their intended values.
func verifyMigration(original, updated string, added map[string]any) error {
	origK, err := parseYAML(original)
	if err != nil {
		return fmt.Errorf("parse original config: %w", err)
	}
	newK, err := parseYAML(updated)
	if err != nil {
		return fmt.Errorf("updated config is not valid YAML: %w", err)
	}
	for _, k := range origK.Keys() {
		if !reflect.DeepEqual(origK.Get(k), newK.Get(k)) {
			return fmt.Errorf("existing option %s would change", k)
		}
	}
	for k, want := range added {
		if got := newK.Get(k); !reflect.DeepEqual(got, want) {
			return fmt.Errorf("option %s would be %v, want %v", k, got, want)
		}
	}
	if got, want := len(newK.Keys()), len(origK.Keys())+len(added); got != want {
		return fmt.Errorf("updated config has %d options, want %d", got, want)
	}
	return nil
}

func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
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

// RunConfigMigrationGuide runs the interactive or automated config migration.
// A missing config file is left alone: normal startup creates it from the template.
func RunConfigMigrationGuide(configPath, exampleContent, versionTag string, autoYes bool) (bool, error) {
	displayPath := configPath
	if abs, err := filepath.Abs(configPath); err == nil {
		displayPath = abs
	}

	raw, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		fmt.Printf("Config file %s not found; skipping migration (it is created from the default template on first run).\n", displayPath)
		return false, nil
	}
	if err != nil {
		return false, err
	}

	missing, err := FindMissingOptions(configPath, exampleContent)
	if err != nil {
		return false, err
	}

	if len(missing) == 0 {
		fmt.Printf("✔ Configuration %s is already up to date. No new options needed.\n", displayPath)
		return false, nil
	}

	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("==================================================================")
	fmt.Printf("🔍 Detected %d new configuration options introduced in %s\n", len(missing), versionTag)
	fmt.Printf("   Config file: %s\n", displayPath)
	fmt.Println("==================================================================")

	useCRLF := strings.Contains(string(raw), "\r\n")
	original := normalizeNewlines(string(raw))
	current := original
	added := make(map[string]any)

	for i, opt := range missing {
		desc := opt.Comment
		if desc == "" {
			desc = "No description available"
		}

		fmt.Printf("[%d/%d] Option: \033[1;36m%s\033[0m\n", i+1, len(missing), opt.Key)
		fmt.Printf("      Description: %s\n", desc)
		fmt.Printf("      Default:     \033[1;32m%s\033[0m\n", opt.DefaultText)

		valueText, value := opt.DefaultText, opt.DefaultValue
		skip := false

		if !autoYes && isTTY {
			for {
				fmt.Printf("      Accept default? [Y/n/custom value]: ")
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				if strings.EqualFold(input, "n") {
					fmt.Println("      -> Skipped.")
					skip = true
				} else if input != "" && !strings.EqualFold(input, "y") {
					text, v, err := parseCustomValue(input, opt.DefaultValue)
					if err != nil {
						fmt.Printf("      -> Invalid value: %v\n", err)
						continue
					}
					valueText, value = text, v
					fmt.Printf("      -> Using custom value: %s\n", valueText)
				} else {
					fmt.Printf("      -> Accepted default: %s\n", valueText)
				}
				break
			}
		} else {
			fmt.Println("      -> Auto-accepted default value.")
		}

		if !skip {
			current = InsertOption(current, strings.Split(opt.Key, "."), valueText, opt.Comment)
			added[opt.Key] = value
		}
		fmt.Println()
	}

	if len(added) == 0 {
		fmt.Println("✔ No options were added. Configuration remained unchanged.")
		return false, nil
	}

	if err := verifyMigration(original, current, added); err != nil {
		return false, fmt.Errorf("could not safely update %s (%v); please add the options above manually", displayPath, err)
	}

	backupPath, err := BackupConfigFile(configPath, versionTag)
	if err != nil {
		return false, fmt.Errorf("failed to create config backup: %w", err)
	}
	fmt.Printf("📦 Original configuration backed up to: %s\n", backupPath)

	if useCRLF {
		current = strings.ReplaceAll(current, "\n", "\r\n")
	}
	if err := os.WriteFile(configPath, []byte(current), 0644); err != nil {
		return false, fmt.Errorf("write updated config: %w", err)
	}
	fmt.Printf("✔ Successfully updated %s with %d new options!\n", displayPath, len(added))
	return true, nil
}
