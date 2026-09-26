package bpmn

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ValidateFormInput validates initial execution input map against Generated Task Form
// fields and constraints defined in the workflow's StartEvent node.
func ValidateFormInput(fields []*FormField, input map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	if input == nil {
		input = make(map[string]interface{})
	}

	for _, field := range fields {
		if field == nil || field.ID == "" {
			continue
		}

		rawVal, exists := input[field.ID]
		valStr := ""
		if exists && rawVal != nil {
			valStr = strings.TrimSpace(fmt.Sprintf("%v", rawVal))
		}

		// Extract constraints
		var isRequired bool
		var isReadonly bool
		var minLen, maxLen int = -1, -1
		var minVal, maxVal *float64
		var patternStr string

		for _, c := range field.Constraints {
			cName := strings.ToLower(strings.TrimSpace(c.Name))
			cConfig := strings.TrimSpace(c.Config)

			switch cName {
			case "required":
				if cConfig == "" || cConfig == "true" || cConfig == "1" {
					isRequired = true
				}
			case "readonly":
				if cConfig == "" || cConfig == "true" || cConfig == "1" {
					isReadonly = true
				}
			case "minlength", "min_length":
				if v, err := strconv.Atoi(cConfig); err == nil {
					minLen = v
				}
			case "maxlength", "max_length":
				if v, err := strconv.Atoi(cConfig); err == nil {
					maxLen = v
				}
			case "min":
				if v, err := strconv.ParseFloat(cConfig, 64); err == nil {
					minVal = &v
					if minLen < 0 {
						minLen = int(v)
					}
				}
			case "max":
				if v, err := strconv.ParseFloat(cConfig, 64); err == nil {
					maxVal = &v
					if maxLen < 0 {
						maxLen = int(v)
					}
				}
			case "pattern":
				patternStr = cConfig
			}
		}

		// Normalize field type
		fieldTypeNorm := strings.ToLower(strings.TrimSpace(field.Type))
		if fieldTypeNorm == "" || fieldTypeNorm == "text" || fieldTypeNorm == "longtext" {
			fieldTypeNorm = "string"
		} else if fieldTypeNorm == "number" || fieldTypeNorm == "float" || fieldTypeNorm == "double" || fieldTypeNorm == "int" {
			fieldTypeNorm = "integer"
		}

		// 1. Validate Required
		if isRequired {
			if !exists || rawVal == nil {
				return fmt.Errorf("field '%s' is required", field.ID)
			}
			if fieldTypeNorm == "boolean" {
				bVal, ok := rawVal.(bool)
				if !ok || !bVal {
					return fmt.Errorf("field '%s' must be true", field.ID)
				}
			} else if valStr == "" {
				return fmt.Errorf("field '%s' is required", field.ID)
			}
		}

		// 2. Validate Readonly
		if isReadonly && exists && rawVal != nil {
			if field.DefaultValue != "" && valStr != field.DefaultValue {
				return fmt.Errorf("field '%s' is read-only and cannot be modified", field.ID)
			}
		}

		// Skip remaining validations if value is empty and not required
		if !exists || valStr == "" {
			continue
		}

		// 3. Type-specific validations
		switch fieldTypeNorm {
		case "string":
			runeCount := len([]rune(valStr))
			effectiveMin := minLen
			if effectiveMin < 0 && minVal != nil {
				effectiveMin = int(*minVal)
			}
			effectiveMax := maxLen
			if effectiveMax < 0 && maxVal != nil {
				effectiveMax = int(*maxVal)
			}

			if effectiveMin >= 0 && runeCount < effectiveMin {
				return fmt.Errorf("field '%s' length must be at least %d characters", field.ID, effectiveMin)
			}
			if effectiveMax >= 0 && runeCount > effectiveMax {
				return fmt.Errorf("field '%s' length cannot exceed %d characters", field.ID, effectiveMax)
			}
			if patternStr != "" {
				re, err := regexp.Compile(patternStr)
				if err == nil && !re.MatchString(valStr) {
					return fmt.Errorf("field '%s' does not match required pattern (%s)", field.ID, patternStr)
				}
			}

		case "long", "integer":
			numVal, err := strconv.ParseFloat(valStr, 64)
			if err != nil {
				return fmt.Errorf("field '%s' must be a valid number", field.ID)
			}
			effectiveMin := minVal
			if effectiveMin == nil && minLen >= 0 {
				v := float64(minLen)
				effectiveMin = &v
			}
			effectiveMax := maxVal
			if effectiveMax == nil && maxLen >= 0 {
				v := float64(maxLen)
				effectiveMax = &v
			}

			if effectiveMin != nil && numVal < *effectiveMin {
				return fmt.Errorf("field '%s' value must be greater than or equal to %g", field.ID, *effectiveMin)
			}
			if effectiveMax != nil && numVal > *effectiveMax {
				return fmt.Errorf("field '%s' value must be less than or equal to %g", field.ID, *effectiveMax)
			}

		case "enum":
			if len(field.Values) > 0 {
				found := false
				for _, enumOpt := range field.Values {
					if enumOpt.ID == valStr {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("field '%s' value '%s' is not a valid enum option", field.ID, valStr)
				}
			}
		}
	}

	return nil
}

var (
	blockCommentRegex = regexp.MustCompile(`(?s)/\*.*?\*/`)
	lineCommentRegex  = regexp.MustCompile(`(?m)//.*$`)
	returnValRegex    = regexp.MustCompile(`\breturn\b\s*[^;\s}]+`)
)

// IsSupportedScriptFormat returns true if the script format is JavaScript.
func IsSupportedScriptFormat(format string) bool {
	f := strings.ToLower(strings.TrimSpace(format))
	switch f {
	case "javascript", "java script", "js", "ecmascript":
		return true
	default:
		return false
	}
}

// HasMandatoryReturnStatement strips comments and checks if the script contains a return statement with an expression.
func HasMandatoryReturnStatement(script string) bool {
	cleaned := blockCommentRegex.ReplaceAllString(script, "")
	cleaned = lineCommentRegex.ReplaceAllString(cleaned, "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return false
	}
	return returnValRegex.MatchString(cleaned)
}

// ValidateScriptConfig validates that a script task uses the 'javascript' format, has a non-empty script body
// with a mandatory return statement, and defines a resultVariable to map the return value.
func ValidateScriptConfig(nodeLabel, scriptFormat, scriptBody, resultVariable string) error {
	if !IsSupportedScriptFormat(scriptFormat) {
		return fmt.Errorf("script task %s: unsupported or missing scriptFormat %q (only 'javascript' is supported)", nodeLabel, scriptFormat)
	}
	if strings.TrimSpace(scriptBody) == "" {
		return fmt.Errorf("script task %s: script body is required", nodeLabel)
	}
	if !HasMandatoryReturnStatement(scriptBody) {
		return fmt.Errorf("script task %s: a 'return <value>;' statement is mandatory in the script", nodeLabel)
	}
	if strings.TrimSpace(resultVariable) == "" {
		return fmt.Errorf("script task %s: output resultVariable is required to map the script return value", nodeLabel)
	}
	return nil
}

// ValidateScriptTasks validates all ScriptTask nodes (or tasks configured with ScriptTask template/properties) in a BPMNModel.
func ValidateScriptTasks(model *BPMNModel) error {
	if model == nil {
		return nil
	}
	for _, node := range model.Nodes {
		if node == nil {
			continue
		}
		tpl := strings.ToLower(strings.TrimSpace(node.Properties["modelerTemplate"]))
		hasScriptProp := strings.TrimSpace(node.Properties["script"]) != "" || strings.TrimSpace(node.Properties["scriptBody"]) != ""
		isScriptNode := node.Type == ElementScriptTask || strings.Contains(tpl, "scripttask") || hasScriptProp
		if !isScriptNode {
			continue
		}

		// If the node uses a custom ActivityTemplate (not built-in ScriptTask and no inline script),
		// full validation will occur after template inheritance resolution in AdvanceTokenCommand / ScriptExecutor.
		if tpl != "" && !strings.Contains(tpl, "scripttask") && !hasScriptProp && node.Type != ElementScriptTask {
			continue
		}

		format := strings.TrimSpace(node.Properties["scriptFormat"])
		if format == "" && strings.Contains(tpl, "scripttask") {
			format = "javascript"
		}
		body := strings.TrimSpace(node.Properties["script"])
		if body == "" {
			body = strings.TrimSpace(node.Properties["scriptBody"])
		}
		resultVar := strings.TrimSpace(node.Properties["resultVariable"])
		if resultVar == "" {
			resultVar = strings.TrimSpace(node.Properties["camunda:resultVariable"])
		}
		if resultVar == "" {
			resultVar = strings.TrimSpace(node.Properties["outputVariable"])
		}

		label := node.ID
		if node.Name != "" {
			label = fmt.Sprintf("%q (%s)", node.Name, node.ID)
		}

		if err := ValidateScriptConfig(label, format, body, resultVar); err != nil {
			return err
		}
	}
	return nil
}

