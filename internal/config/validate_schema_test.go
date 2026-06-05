package config

import "testing"

func TestValidateSchemaDefaults(t *testing.T) {
	cfg := DefaultConfig()
	errs := ValidateSchema(&cfg)
	if len(errs) != 0 {
		t.Fatalf("DefaultConfig должен быть валиден: %v", errs)
	}
}

func TestValidateSchemaBadParserMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Parser.Mode = "invalid"
	errs := ValidateSchema(&cfg)
	if len(errs) == 0 {
		t.Fatal("ожидалась ошибка parser.mode")
	}
}

func TestValidateSchemaExamplePolicyPacks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Policy.Packs = []string{
		"security-baseline",
		"mozilla-ssl",
		"performance-baseline",
		"caching",
		"rate-limit",
	}
	errs := ValidateSchema(&cfg)
	if len(errs) != 0 {
		t.Fatalf("packs из example-config должны быть валидны: %v", errs)
	}
}

func TestValidateSchemaUnknownPolicyPack(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Policy.Packs = []string{"unknown-pack"}
	errs := ValidateSchema(&cfg)
	if len(errs) == 0 {
		t.Fatal("ожидалась ошибка для неизвестного pack")
	}
}

func TestValidateSchemaCustomPolicyPack(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Policy.Packs = []string{"custom-my-rules"}
	errs := ValidateSchema(&cfg)
	if len(errs) != 0 {
		t.Fatalf("custom-* pack должен быть валиден: %v", errs)
	}
}

func TestValidateSchemaBadOutputFormat(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Output.Format = "xml"
	errs := ValidateSchema(&cfg)
	if len(errs) == 0 {
		t.Fatal("ожидалась ошибка output.format")
	}
}

func TestValidateSchemaBadDockerEnabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Docker.Enabled = "maybe"
	errs := ValidateSchema(&cfg)
	if len(errs) == 0 {
		t.Fatal("ожидалась ошибка docker.enabled")
	}
}
