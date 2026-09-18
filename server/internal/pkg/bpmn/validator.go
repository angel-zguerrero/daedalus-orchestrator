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
