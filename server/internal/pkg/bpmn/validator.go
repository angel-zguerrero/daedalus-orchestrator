package bpmn

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
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

type elementContext struct {
	tag  string
	id   string
	name string
}

// ValidateDiagramFormFields parses a BPMN XML payload and verifies that all form fields
// defined within any formData extensions have a valid, non-empty ID. It also checks that
// form field IDs are unique within each BPMN element.
func ValidateDiagramFormFields(xmlData []byte) error {
	if len(xmlData) == 0 {
		return nil
	}

	decoder := xml.NewDecoder(strings.NewReader(string(xmlData)))
	var elementStack []elementContext
	seenFieldIDs := make(map[string]map[string]bool)

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error decoding BPMN XML: %w", err)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			local := stripPrefix(t.Name.Local)
			if local != "definitions" && local != "extensionElements" &&
				local != "formData" && local != "formField" &&
				local != "validation" && local != "constraint" &&
				local != "value" && local != "properties" &&
				local != "property" {
				elemID := getAttr(t.Attr, "id")
				elemName := getAttr(t.Attr, "name")
				if elemID != "" {
					elementStack = append(elementStack, elementContext{
						tag:  local,
						id:   elemID,
						name: elemName,
					})
				}
			}

			if local == "formField" {
				var currentElem elementContext
				if len(elementStack) > 0 {
					currentElem = elementStack[len(elementStack)-1]
				}
				nodeDesc := currentElem.id
				if currentElem.name != "" {
					nodeDesc = fmt.Sprintf("%q (%s)", currentElem.name, currentElem.id)
				}
				if nodeDesc == "" {
					nodeDesc = "unknown element"
				}

				fID := strings.TrimSpace(getAttr(t.Attr, "id"))
				if fID == "" {
					fLabel := strings.TrimSpace(getAttr(t.Attr, "label"))
					if fLabel != "" {
						return fmt.Errorf("form field %q in element %s is missing required 'id'", fLabel, nodeDesc)
					}
					return fmt.Errorf("form field in element %s is missing required 'id'", nodeDesc)
				}

				if seenFieldIDs[currentElem.id] == nil {
					seenFieldIDs[currentElem.id] = make(map[string]bool)
				}
				if seenFieldIDs[currentElem.id][fID] {
					return fmt.Errorf("duplicate form field id %q in element %s", fID, nodeDesc)
				}
				seenFieldIDs[currentElem.id][fID] = true
			}

		case xml.EndElement:
			local := stripPrefix(t.Name.Local)
			if len(elementStack) > 0 && elementStack[len(elementStack)-1].tag == local {
				elementStack = elementStack[:len(elementStack)-1]
			}
		}
	}

	return nil
}

// ValidateEventInput strictly validates an incoming payload against the ExpectedInput schema.
func ValidateEventInput(expectedInput map[string]interface{}, payload map[string]interface{}) error {
	if expectedInput == nil {
		return nil
	}

	fieldsRaw, ok := expectedInput["fields"]
	if !ok || fieldsRaw == nil {
		return nil
	}

	var fieldsList []map[string]interface{}
	switch f := fieldsRaw.(type) {
	case []map[string]interface{}:
		fieldsList = f
	case []interface{}:
		for _, item := range f {
			if m, ok := item.(map[string]interface{}); ok {
				fieldsList = append(fieldsList, m)
			}
		}
	}

	if len(fieldsList) == 0 {
		return nil
	}

	if payload == nil {
		payload = make(map[string]interface{})
	}

	for _, field := range fieldsList {
		fieldID, _ := field["id"].(string)
		if fieldID == "" {
			fieldID, _ = field["name"].(string)
		}
		if fieldID == "" {
			continue
		}

		fieldType, _ := field["type"].(string)
		fieldType = strings.ToLower(strings.TrimSpace(fieldType))

		// Check if required
		isRequired, _ := field["required"].(bool)
		constraintsRaw := field["constraints"]
		var constraints []map[string]string
		if cList, ok := constraintsRaw.([]map[string]string); ok {
			constraints = cList
		} else if cList, ok := constraintsRaw.([]interface{}); ok {
			for _, item := range cList {
				if cm, ok := item.(map[string]interface{}); ok {
					cName, _ := cm["name"].(string)
					cConfig, _ := cm["config"].(string)
					constraints = append(constraints, map[string]string{"name": cName, "config": cConfig})
				} else if cm, ok := item.(map[string]string); ok {
					constraints = append(constraints, cm)
				}
			}
		}

		for _, c := range constraints {
			if strings.EqualFold(c["name"], "required") {
				isRequired = true
				break
			}
		}

		val, exists := payload[fieldID]
		if !exists && strings.Contains(fieldID, ".") {
			if v, ok := ResolveVariablePathWithExists(fieldID, payload); ok {
				val = v
				exists = true
			}
		}

		// 1. Required Check
		if isRequired {
			if !exists || val == nil {
				return fmt.Errorf("validation error: field %q is required", fieldID)
			}
			if strVal, isStr := val.(string); isStr && strings.TrimSpace(strVal) == "" {
				return fmt.Errorf("validation error: field %q is required and cannot be empty", fieldID)
			}
		}

		if !exists || val == nil {
			continue
		}

		// 2. Type Checking
		switch fieldType {
		case "string", "text":
			if _, ok := val.(string); !ok {
				return fmt.Errorf("validation error: field %q must be a string, got %T", fieldID, val)
			}

		case "long", "int", "integer":
			isInt := false
			switch n := val.(type) {
			case int, int32, int64:
				isInt = true
			case float64:
				if n == float64(int64(n)) {
					isInt = true
				}
			case float32:
				if n == float32(int64(n)) {
					isInt = true
				}
			case string:
				if _, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64); err == nil {
					isInt = true
				}
			}
			if !isInt {
				return fmt.Errorf("validation error: field %q must be an integer, got %T (%v)", fieldID, val, val)
			}

		case "double", "float", "number":
			isNum := false
			switch n := val.(type) {
			case int, int32, int64, float32, float64:
				isNum = true
			case string:
				if _, err := strconv.ParseFloat(strings.TrimSpace(n), 64); err == nil {
					isNum = true
				}
			}
			if !isNum {
				return fmt.Errorf("validation error: field %q must be a numeric value, got %T (%v)", fieldID, val, val)
			}

		case "boolean", "bool":
			isBool := false
			switch b := val.(type) {
			case bool:
				isBool = true
			case string:
				lower := strings.ToLower(strings.TrimSpace(b))
				if lower == "true" || lower == "false" {
					isBool = true
				}
			}
			if !isBool {
				return fmt.Errorf("validation error: field %q must be a boolean, got %T (%v)", fieldID, val, val)
			}

		case "date":
			strVal, ok := val.(string)
			if !ok {
				return fmt.Errorf("validation error: field %q must be a date string (YYYY-MM-DD), got %T", fieldID, val)
			}
			strVal = strings.TrimSpace(strVal)
			parsedDate := false
			for _, layout := range []string{"2006-01-02", "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05", "2006/01/02"} {
				if _, err := strconv.ParseInt(strVal, 10, 64); err == nil {
					// pure number is not a date string
					break
				}
				if _, err := time.Parse(layout, strVal); err == nil {
					parsedDate = true
					break
				}
			}
			if !parsedDate {
				return fmt.Errorf("validation error: field %q has invalid date format: %q", fieldID, strVal)
			}

		case "enum":
			var enumValues []string
			if valsRaw, ok := field["values"]; ok && valsRaw != nil {
				if vList, ok := valsRaw.([]map[string]string); ok {
					for _, item := range vList {
						enumValues = append(enumValues, item["id"], item["name"])
					}
				} else if vList, ok := valsRaw.([]interface{}); ok {
					for _, item := range vList {
						if vm, ok := item.(map[string]interface{}); ok {
							if id, ok := vm["id"].(string); ok {
								enumValues = append(enumValues, id)
							}
							if name, ok := vm["name"].(string); ok {
								enumValues = append(enumValues, name)
							}
						}
					}
				}
			}

			if len(enumValues) > 0 {
				strVal := fmt.Sprintf("%v", val)
				matched := false
				for _, allowed := range enumValues {
					if allowed != "" && strings.EqualFold(allowed, strVal) {
						matched = true
						break
					}
				}
				if !matched {
					return fmt.Errorf("validation error: field %q value %q is not in allowed enum options", fieldID, strVal)
				}
			}
		}

		// 3. Constraints Checking
		for _, c := range constraints {
			cName := strings.ToLower(strings.TrimSpace(c["name"]))
			cConfig := strings.TrimSpace(c["config"])

			switch cName {
			case "minlength":
				if minLen, err := strconv.Atoi(cConfig); err == nil {
					strVal := fmt.Sprintf("%v", val)
					if len([]rune(strVal)) < minLen {
						return fmt.Errorf("validation error: field %q length must be at least %d characters", fieldID, minLen)
					}
				}

			case "maxlength":
				if maxLen, err := strconv.Atoi(cConfig); err == nil {
					strVal := fmt.Sprintf("%v", val)
					if len([]rune(strVal)) > maxLen {
						return fmt.Errorf("validation error: field %q length cannot exceed %d characters", fieldID, maxLen)
					}
				}

			case "min":
				if minVal, err := strconv.ParseFloat(cConfig, 64); err == nil {
					strVal := fmt.Sprintf("%v", val)
					if num, err2 := strconv.ParseFloat(strVal, 64); err2 == nil && num < minVal {
						return fmt.Errorf("validation error: field %q must be >= %v", fieldID, minVal)
					}
				}

			case "max":
				if maxVal, err := strconv.ParseFloat(cConfig, 64); err == nil {
					strVal := fmt.Sprintf("%v", val)
					if fieldType == "string" || fieldType == "text" {
						if float64(len([]rune(strVal))) > maxVal {
							return fmt.Errorf("validation error: field %q length cannot exceed %d characters", fieldID, int(maxVal))
						}
					} else if num, err2 := strconv.ParseFloat(strVal, 64); err2 == nil && num > maxVal {
						return fmt.Errorf("validation error: field %q must be <= %v", fieldID, maxVal)
					}
				}

			case "pattern":
				if cConfig != "" {
					re, err := regexp.Compile(cConfig)
					if err == nil {
						strVal := fmt.Sprintf("%v", val)
						if !re.MatchString(strVal) {
							return fmt.Errorf("validation error: field %q does not match pattern %s", fieldID, cConfig)
						}
					}
				}
			}
		}
	}

	return nil
}

// ValidateDiagram runs comprehensive validation on a BPMN diagram payload:
// 1. Verifies all form fields have valid IDs and uniqueness per element.
// 2. Parses the BPMN model and verifies script task configurations.
func ValidateDiagram(xmlData []byte) error {
	if len(xmlData) == 0 {
		return nil
	}

	if err := ValidateDiagramFormFields(xmlData); err != nil {
		return err
	}

	model, err := ParseBPMN(xmlData)
	if err != nil {
		return err
	}

	if err := ValidateScriptTasks(model); err != nil {
		return err
	}

	return nil
}

