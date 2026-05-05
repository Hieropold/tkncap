package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/hieropold/tkncap/internal/account"
	"github.com/hieropold/tkncap/internal/provider"
)

func TestTableRenderer_ColorCoding(t *testing.T) {
	tr := &TableRenderer{}
	used50 := int64(50)
	limit100 := int64(100)
	used95 := int64(95)

	quotas := []provider.Quota{
		{
			Account: account.Account{Provider: account.ProviderClaude, Name: "work"},
			Name:    "5-hour",
			Status:  provider.StatusOK,
			Used:    &used50,
			Limit:   &limit100,
		},
		{
			Account: account.Account{Provider: account.ProviderGemini, Name: "main"},
			Name:    "RPM",
			Status:  provider.StatusOK,
			Used:    &used95,
			Limit:   &limit100,
		},
	}

	var buf bytes.Buffer
	err := tr.Render(&buf, quotas)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := buf.String()

	// Check for Green color prefix (\x1b[38;5;46m) for 50% usage
	if !strings.Contains(output, "\x1b[38;5;46m") {
		t.Errorf("Expected green color code for 50%% usage not found in output")
	}

	// Check for Red color prefix (\x1b[38;5;196m) for 95% usage
	if !strings.Contains(output, "\x1b[38;5;196m") {
		t.Errorf("Expected red color code for 95%% usage not found in output")
	}

	// Check for Reset color code (\x1b[0m)
	if !strings.Contains(output, "\x1b[0m") {
		t.Errorf("Expected reset color code not found in output")
	}
}
