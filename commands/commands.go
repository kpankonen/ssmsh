package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/abiosoft/ishell"
	"github.com/bwhaley/ssmsh/parameterstore"
)

type fn func(*ishell.Context)

var (
	shell      *ishell.Shell
	ps         *parameterstore.ParameterStore
	outputMode string
)

// Init initializes the ssmsh subcommands
func Init(iShell *ishell.Shell, iPs *parameterstore.ParameterStore, output string) {
	shell = iShell
	ps = iPs
	outputMode = output
	registerCommand("cd", "change your relative location within the parameter store", cd, cdUsage)
	registerCommand("cp", "copy source to dest", cp, cpUsage)
	registerCommand("decrypt", "toggle parameter decryption", decrypt, decryptUsage)
	registerCommand("get", "get parameters", get, getUsage)
	registerCommand("history", "get parameter history", history, historyUsage)
	registerCommand("key", "set the KMS key", key, keyUsage)
	registerCommand("ls", "list parameters", ls, lsUsage)
	registerCommand("mv", "move parameters", mv, mvUsage)
	registerCommand("policy", "create named parameter policy", policy, policyUsage)
	registerCommand("profile", "switch to a different AWS IAM profile", profile, profileUsage)
	registerCommand("put", "set parameter", put, putUsage)
	registerCommand("region", "change region", region, regionUsage)
	registerCommand("rm", "remove parameters", rm, rmUsage)
	setPrompt(parameterstore.Delimiter)
}

// registerCommand adds a command to the shell
func registerCommand(name string, helpText string, f fn, usageText string) {
	shell.AddCmd(&ishell.Cmd{
		Name:     name,
		Help:     helpText,
		LongHelp: usageText,
		Func:     f,
	})
}

// setPrompt configures the shell prompt
func setPrompt(prompt string) {
	shell.SetPrompt(prompt + ">")
}

// remove deletes an element from a slice of strings
func remove(slice []string, i int) []string {
	return append(slice[:i], slice[i+1:]...)
}

// checkRecursion searches a slice of strings for an element matching -r or -R
func checkRecursion(paths []string) ([]string, bool) {
	for i, p := range paths {
		if strings.EqualFold(p, "-r") {
			paths = remove(paths, i)
			return paths, true
		}
	}
	return paths, false
}

// parsePath determines whether a path includes a region
func parsePath(path string) (parameterPath parameterstore.ParameterPath) {
	pathParts := strings.Split(path, ":")
	switch len(pathParts) {
	case 1:
		parameterPath.Name = pathParts[0]
		parameterPath.Region = ps.Region
	case 2:
		parameterPath.Region = pathParts[0]
		parameterPath.Name = pathParts[1]
	}
	ps.InitClient(parameterPath.Region)
	return parameterPath
}

func groupByRegion(params []parameterstore.ParameterPath) map[string][]string {
	paramsByRegion := make(map[string][]string)
	for _, p := range params {
		paramsByRegion[p.Region] = append(paramsByRegion[p.Region], p.Name)
	}
	return paramsByRegion
}

func trim(with []string) (without []string) {
	for i := range with {
		without = append(without, strings.TrimSpace(with[i]))
	}
	return without
}

func printResult(result any) {
	cleaned, err := cleanResult(result)
	if err != nil {
		shell.Println("Error with result: ", err)
		return
	}
	if outputMode != "json" {
		shell.Println(formatPlain(cleaned, 0))
		return
	}
	printJSON(cleaned)
}

func printJSON(result any) {
	indented, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		shell.Println("Error with result: ", err)
		return
	}
	shell.Println(string(indented))
}

func cleanResult(result any) (any, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return stripNulls(raw), nil
}

func stripNulls(data []byte) any {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return raw
	}
	return cleanValue(raw)
}

func cleanValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		cleaned := make(map[string]any)
		for k, v := range val {
			if v != nil {
				cleaned[k] = cleanValue(v)
			}
		}
		return cleaned
	case []any:
		for i, item := range val {
			val[i] = cleanValue(item)
		}
		return val
	default:
		return v
	}
}

func formatPlain(v any, indent int) string {
	switch val := v.(type) {
	case map[string]any:
		return formatPlainMap(val, indent)
	case []any:
		return formatPlainArray(val, indent)
	case string:
		return strconv.Quote(val)
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case nil:
		return "<nil>"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func formatPlainMap(m map[string]any, indent int) string {
	if len(m) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("{\n")
	for _, k := range keys {
		b.WriteString(strings.Repeat(" ", indent+4))
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(formatPlain(m[k], indent+4))
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat(" ", indent))
	b.WriteString("}")
	return b.String()
}

func formatPlainArray(items []any, indent int) string {
	if len(items) == 0 {
		return "[]"
	}

	var b strings.Builder
	b.WriteString("[")
	for i, item := range items {
		if i > 0 {
			b.WriteString("\n")
			b.WriteString(strings.Repeat(" ", indent+1))
		}
		b.WriteString(formatPlain(item, indent))
	}
	b.WriteString("]")
	return b.String()
}
