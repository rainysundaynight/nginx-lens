package cli

import (
	"os"
	"testing"

	"github.com/rainysundaynight/nginx-lens/internal/config"
)

func TestStylerColorsEnabled(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	cfg := config.DefaultConfig()
	cfg.Output.Colors = true
	s := newStyler(cfg)
	out := s.green("ok")
	if out == "ok" {
		t.Fatal("expected ansi codes when colors enabled")
	}
}

func TestStylerNoColorEnv(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	cfg := config.DefaultConfig()
	cfg.Output.Colors = true
	s := newStyler(cfg)
	if s.green("ok") != "ok" {
		t.Fatal("NO_COLOR должен отключать ansi")
	}
}

func TestStylerSeverityAndDiffType(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Colors = false
	s := newStyler(cfg)
	if s.severity("high") == "" || s.diffType("added") == "" {
		t.Fatal("expected non-empty labels")
	}
}

func TestStylerPathParts(t *testing.T) {
	cfg := config.DefaultConfig()
	s := newStyler(cfg)
	out := s.pathParts([]string{"upstream", "api"})
	if out == "" {
		t.Fatal("expected path output")
	}
}

func TestStylerHeader(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Output.Colors = true
	s := newStyler(cfg)
	_ = s.header("test")
	_ = s.ok("ok")
	_ = s.fail("fail")
	os.Unsetenv("NO_COLOR")
}
