package bpmn

import (
	"testing"
)

func TestValidateFormInput(t *testing.T) {
	fields := []*FormField{
		{
			ID:    "monto",
			Label: "Monto Solicitado",
			Type:  "long",
			Constraints: []FormFieldConstraint{
				{Name: "required", Config: "true"},
				{Name: "min", Config: "100"},
				{Name: "max", Config: "5000"},
			},
		},
		{
			ID:    "codigo",
			Label: "Código Cliente",
			Type:  "string",
			Constraints: []FormFieldConstraint{
				{Name: "required", Config: "true"},
				{Name: "minlength", Config: "3"},
				{Name: "maxlength", Config: "10"},
				{Name: "pattern", Config: "^[A-Z]{3}-\\d{3}$"},
			},
		},
		{
			ID:    "tipo",
			Label: "Tipo Crédito",
			Type:  "enum",
			Values: []FormFieldValue{
				{ID: "personal", Name: "Personal"},
				{ID: "hipotecario", Name: "Hipotecario"},
			},
			Constraints: []FormFieldConstraint{
				{Name: "required", Config: "true"},
			},
		},
	}

	t.Run("Valid input passes", func(t *testing.T) {
		input := map[string]interface{}{
			"monto":  500,
			"codigo": "ABC-123",
			"tipo":   "personal",
		}
		err := ValidateFormInput(fields, input)
		if err != nil {
			t.Fatalf("expected valid input to pass, got error: %v", err)
		}
	})

	t.Run("Missing required field fails", func(t *testing.T) {
		input := map[string]interface{}{
			"codigo": "ABC-123",
			"tipo":   "personal",
		}
		err := ValidateFormInput(fields, input)
		if err == nil {
			t.Fatalf("expected error for missing required field 'monto'")
		}
	})

	t.Run("Value out of min/max range fails", func(t *testing.T) {
		input := map[string]interface{}{
			"monto":  50, // min is 100
			"codigo": "ABC-123",
			"tipo":   "personal",
		}
		err := ValidateFormInput(fields, input)
		if err == nil {
			t.Fatalf("expected error for monto < 100")
		}
	})

	t.Run("Invalid pattern fails", func(t *testing.T) {
		input := map[string]interface{}{
			"monto":  500,
			"codigo": "invalid_code",
			"tipo":   "personal",
		}
		err := ValidateFormInput(fields, input)
		if err == nil {
			t.Fatalf("expected error for invalid regex pattern")
		}
	})

	t.Run("Invalid enum option fails", func(t *testing.T) {
		input := map[string]interface{}{
			"monto":  500,
			"codigo": "ABC-123",
			"tipo":   "invalid_enum_option",
		}
		err := ValidateFormInput(fields, input)
		if err == nil {
			t.Fatalf("expected error for invalid enum option")
		}
	})
}
