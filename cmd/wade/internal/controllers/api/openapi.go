// This file is completely vibecoded and needs to be properly reviewed.

package api

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Operation describes one HTTP API operation exposed as a CLI command.
type Operation struct {
	Command         string
	Method          string
	Path            string
	Summary         string
	Parameters      []Parameter
	HasBody         bool
	BodyRequired    bool
	BodyDescription string
}

// Parameter describes one path or query parameter mapped to a string flag.
type Parameter struct {
	Name        string
	Flag        string
	In          string
	Description string
	Required    bool
	Enum        []string
}

type specDocument struct {
	BasePath string                              `json:"basePath"`
	Paths    map[string]map[string]specOperation `json:"paths"`
}

type specOperation struct {
	ID         string          `json:"operationId"`
	Summary    string          `json:"summary"`
	Parameters []specParameter `json:"parameters"`
	CLIIgnored bool            `json:"x-wade-cli-ignore"`
}

type specParameter struct {
	Name        string   `json:"name"`
	In          string   `json:"in"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Type        string   `json:"type"`
	Enum        []string `json:"enum"`
}

func parseOperations(specJSON []byte) ([]Operation, error) {
	var document specDocument
	if err := json.Unmarshal(specJSON, &document); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI specification: %w", err)
	}

	var operations []Operation
	commands := map[string]string{}
	for path, item := range document.Paths {
		for method, operationSpec := range item {
			if operationSpec.CLIIgnored {
				continue
			}

			operation, err := newOperation(strings.ToUpper(method), joinBasePath(document.BasePath, path), operationSpec)
			if err != nil {
				return nil, err
			}

			if existing, taken := commands[operation.Command]; taken {
				return nil, fmt.Errorf("operations %s and %s map to the same command %s", existing, operationSpec.ID, operation.Command)
			}
			commands[operation.Command] = operationSpec.ID
			operations = append(operations, operation)
		}
	}

	sort.Slice(operations, func(i, j int) bool { return operations[i].Command < operations[j].Command })
	return operations, nil
}

func newOperation(method string, path string, operationSpec specOperation) (Operation, error) {
	if operationSpec.ID == "" {
		return Operation{}, fmt.Errorf("operation %s %s has no operationId", method, path)
	}

	operation := Operation{
		Command: kebabCase(operationSpec.ID),
		Method:  method,
		Path:    path,
		Summary: operationSpec.Summary,
	}
	flagNames := map[string]bool{addressFlag: true, bodyFlag: true}
	for _, parameterSpec := range operationSpec.Parameters {
		switch parameterSpec.In {
		case "path", "query":
			if parameterSpec.Type != "string" {
				return Operation{}, fmt.Errorf("operation %s has unsupported %s parameter type %s for %s", operationSpec.ID, parameterSpec.In, parameterSpec.Type, parameterSpec.Name)
			}
			flagName := kebabCase(parameterSpec.Name)
			if flagNames[flagName] {
				return Operation{}, fmt.Errorf("operation %s parameter %s maps to flag --%s, which is already taken", operationSpec.ID, parameterSpec.Name, flagName)
			}
			flagNames[flagName] = true
			operation.Parameters = append(operation.Parameters, Parameter{
				Name:        parameterSpec.Name,
				Flag:        flagName,
				In:          parameterSpec.In,
				Description: parameterSpec.Description,
				Required:    parameterSpec.Required,
				Enum:        append([]string(nil), parameterSpec.Enum...),
			})
		case "body":
			if operation.HasBody {
				return Operation{}, fmt.Errorf("operation %s has more than one body parameter", operationSpec.ID)
			}
			operation.HasBody = true
			operation.BodyRequired = parameterSpec.Required
			operation.BodyDescription = parameterSpec.Description
		default:
			return Operation{}, fmt.Errorf("operation %s has unsupported parameter location %s for %s", operationSpec.ID, parameterSpec.In, parameterSpec.Name)
		}
	}

	return operation, nil
}

func kebabCase(identifier string) string {
	var converted strings.Builder
	runes := []rune(identifier)
	for i, r := range runes {
		if !unicode.IsUpper(r) {
			converted.WriteRune(r)
			continue
		}

		previousIsLower := i > 0 && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]))
		nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
		if i > 0 && (previousIsLower || nextIsLower) {
			converted.WriteRune('-')
		}
		converted.WriteRune(unicode.ToLower(r))
	}
	return converted.String()
}

func joinBasePath(basePath string, path string) string {
	prefix := strings.TrimSuffix(basePath, "/")
	return prefix + path
}
