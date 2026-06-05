package cli

import (
	"strings"
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/config"
)

func TestVisibleLenIgnoresANSI(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Colors = true
	s := newStyler(cfg)
	colored := s.green("OK")
	if visibleLen(colored) != 2 {
		t.Fatalf("visibleLen(%q) = %d, want 2", colored, visibleLen(colored))
	}
}

func TestPadVisible(t *testing.T) {
	out := padVisible("ab", 6)
	if visibleLen(out) != 6 {
		t.Fatalf("padVisible width mismatch: %q", out)
	}
}

func TestFormatTableRow(t *testing.T) {
	row := formatTableRow([]int{10, 6}, []string{"upstream", "OK"})
	if !strings.Contains(row, "upstream") || !strings.Contains(row, "OK") {
		t.Fatalf("unexpected row: %q", row)
	}
}
