package theme

import (
	"strings"
	"testing"
)

func TestDecodeConfigRejectsInvalidHex(t *testing.T) {
	input := `{
  "schemaVersion": 1,
  "seedColor": "#GGGGGG",
  "colors": {
    "light": {
      "background": "#000000",
      "surface": "#000000",
      "primary": "#000000",
      "onPrimary": "#000000",
      "secondary": "#000000",
      "onSecondary": "#000000",
      "text": "#000000",
      "muted": "#000000",
      "error": "#000000",
      "onError": "#000000",
      "success": "#000000",
      "warning": "#000000",
      "surfaceAlt": "#000000",
      "outline": "#000000"
    },
    "dark": {
      "background": "#000000",
      "surface": "#000000",
      "primary": "#000000",
      "onPrimary": "#000000",
      "secondary": "#000000",
      "onSecondary": "#000000",
      "text": "#000000",
      "muted": "#000000",
      "error": "#000000",
      "onError": "#000000",
      "success": "#000000",
      "warning": "#000000",
      "surfaceAlt": "#000000",
      "outline": "#000000"
    }
  }
}`

	_, err := DecodeConfig([]byte(input))
	if err == nil {
		t.Fatal("expected invalid hex error")
	}
	if !strings.Contains(err.Error(), "$.seedColor") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeConfigRejectsUnknownField(t *testing.T) {
	input := `{
  "schemaVersion": 1,
  "seedColor": "#111111",
  "colors": {
    "light": {
      "background": "#000000",
      "surface": "#000000",
      "primary": "#000000",
      "onPrimary": "#000000",
      "secondary": "#000000",
      "onSecondary": "#000000",
      "text": "#000000",
      "muted": "#000000",
      "error": "#000000",
      "onError": "#000000",
      "success": "#000000",
      "warning": "#000000",
      "surfaceAlt": "#000000",
      "outline": "#000000",
      "extra": "#000000"
    },
    "dark": {
      "background": "#000000",
      "surface": "#000000",
      "primary": "#000000",
      "onPrimary": "#000000",
      "secondary": "#000000",
      "onSecondary": "#000000",
      "text": "#000000",
      "muted": "#000000",
      "error": "#000000",
      "onError": "#000000",
      "success": "#000000",
      "warning": "#000000",
      "surfaceAlt": "#000000",
      "outline": "#000000"
    }
  }
}`

	_, err := DecodeConfig([]byte(input))
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeConfigRejectsDuplicateKey(t *testing.T) {
	input := `{
  "schemaVersion": 1,
  "seedColor": "#111111",
  "colors": {
    "light": {
      "background": "#000000",
      "surface": "#000000",
      "primary": "#000000",
      "primary": "#111111",
      "onPrimary": "#000000",
      "secondary": "#000000",
      "onSecondary": "#000000",
      "text": "#000000",
      "muted": "#000000",
      "error": "#000000",
      "onError": "#000000",
      "success": "#000000",
      "warning": "#000000",
      "surfaceAlt": "#000000",
      "outline": "#000000"
    },
    "dark": {
      "background": "#000000",
      "surface": "#000000",
      "primary": "#000000",
      "onPrimary": "#000000",
      "secondary": "#000000",
      "onSecondary": "#000000",
      "text": "#000000",
      "muted": "#000000",
      "error": "#000000",
      "onError": "#000000",
      "success": "#000000",
      "warning": "#000000",
      "surfaceAlt": "#000000",
      "outline": "#000000"
    }
  }
}`

	_, err := DecodeConfig([]byte(input))
	if err == nil {
		t.Fatal("expected duplicate key error")
	}
	if !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeConfigRejectsMissingRole(t *testing.T) {
	input := `{
  "schemaVersion": 1,
  "seedColor": "#111111",
  "colors": {
    "light": {
      "background": "#000000",
      "surface": "#000000",
      "primary": "#000000",
      "onPrimary": "#000000",
      "secondary": "#000000",
      "onSecondary": "#000000",
      "text": "#000000",
      "muted": "#000000",
      "error": "#000000",
      "onError": "#000000",
      "success": "#000000",
      "warning": "#000000",
      "surfaceAlt": "#000000"
    },
    "dark": {
      "background": "#000000",
      "surface": "#000000",
      "primary": "#000000",
      "onPrimary": "#000000",
      "secondary": "#000000",
      "onSecondary": "#000000",
      "text": "#000000",
      "muted": "#000000",
      "error": "#000000",
      "onError": "#000000",
      "success": "#000000",
      "warning": "#000000",
      "surfaceAlt": "#000000",
      "outline": "#000000"
    }
  }
}`

	_, err := DecodeConfig([]byte(input))
	if err == nil {
		t.Fatal("expected missing role error")
	}
	if !strings.Contains(err.Error(), "$.colors.light.outline") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeConfigRejectsSchemaVersionMismatch(t *testing.T) {
	input := `{
  "schemaVersion": 2,
  "seedColor": "#111111",
  "colors": {
    "light": {
      "background": "#000000",
      "surface": "#000000",
      "primary": "#000000",
      "onPrimary": "#000000",
      "secondary": "#000000",
      "onSecondary": "#000000",
      "text": "#000000",
      "muted": "#000000",
      "error": "#000000",
      "onError": "#000000",
      "success": "#000000",
      "warning": "#000000",
      "surfaceAlt": "#000000",
      "outline": "#000000"
    },
    "dark": {
      "background": "#000000",
      "surface": "#000000",
      "primary": "#000000",
      "onPrimary": "#000000",
      "secondary": "#000000",
      "onSecondary": "#000000",
      "text": "#000000",
      "muted": "#000000",
      "error": "#000000",
      "onError": "#000000",
      "success": "#000000",
      "warning": "#000000",
      "surfaceAlt": "#000000",
      "outline": "#000000"
    }
  }
}`

	_, err := DecodeConfig([]byte(input))
	if err == nil {
		t.Fatal("expected schemaVersion mismatch error")
	}
	if !strings.Contains(err.Error(), "$.schemaVersion must be 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}
