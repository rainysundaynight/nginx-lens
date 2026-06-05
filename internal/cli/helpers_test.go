package cli

import (
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/analyzer"
	"github.com/rainysundaynight/nginx-lens/internal/config"
)

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  a ", "b"); got != "a" {
		t.Fatalf("got %q, want a", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestValidateFailOn(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Validate.FailOn = "medium"
	if got := validateFailOn(cfg); got != "medium" {
		t.Fatalf("got %q", got)
	}
	cfg.Validate.FailOn = ""
	cfg.Validate.FailOnLow = true
	if got := validateFailOn(cfg); got != "low" {
		t.Fatalf("got %q", got)
	}
}

func TestRequireParam(t *testing.T) {
	if err := requireParam("url", ""); err == nil {
		t.Fatal("expected error for empty param")
	}
	if err := requireParam("url", "http://x"); err != nil {
		t.Fatal(err)
	}
}

func TestRequireNonEmpty(t *testing.T) {
	if err := requireNonEmpty("field", ""); err == nil {
		t.Fatal("expected error")
	}
}

func TestAnalyzeFilterFromCfg(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Analyze.MinSeverity = "high"
	cfg.Analyze.SkipTypes = []string{"duplicate_directive"}
	opts := analyzeFilter(cfg)
	if opts.MinSeverity != analyzer.SeverityHigh {
		t.Fatalf("min severity=%v", opts.MinSeverity)
	}
	if _, ok := opts.SkipTypes["duplicate_directive"]; !ok {
		t.Fatal("expected skip type")
	}
}
