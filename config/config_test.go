package config

import (
	"os"
	"path/filepath"
	"testing"
)

// useTempConfig points the package at a temp config file and restores after
func useTempConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	SetConfigPath(path)
	t.Cleanup(func() { SetConfigPath("") })
	return path
}

func TestEnsureExistsCreatesDefaultConfig(t *testing.T) {
	path := useTempConfig(t)

	if err := EnsureExists(); err != nil {
		t.Fatalf("EnsureExists: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("config permissions = %o, want 0600", perm)
	}

	// Empty credentials must be rejected by Load
	if _, err := Load(); err != ErrCredentialsMissing {
		t.Errorf("Load on empty config = %v, want ErrCredentialsMissing", err)
	}
}

func TestLoadValidConfig(t *testing.T) {
	path := useTempConfig(t)

	content := `{"username":"user@example.com","password":"secret","page_size":50,"user_agent":"test-agent"}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Username != "user@example.com" || cfg.Password != "secret" {
		t.Errorf("credentials not parsed: %+v", cfg)
	}
	if cfg.PageSize != 50 {
		t.Errorf("PageSize = %d, want 50", cfg.PageSize)
	}
	if cfg.UserAgent != "test-agent" {
		t.Errorf("UserAgent = %q", cfg.UserAgent)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	path := useTempConfig(t)

	if err := os.WriteFile(path, []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Error("Load on invalid JSON should fail")
	}
}

func TestEnsureTaxesExistsAndLoad(t *testing.T) {
	useTempConfig(t)

	cfg, err := LoadTaxes()
	if err != nil {
		t.Fatalf("LoadTaxes: %v", err)
	}

	def := DefaultTaxConfig()
	if cfg.SalariuMinimBrut != def.SalariuMinimBrut {
		t.Errorf("SalariuMinimBrut = %f, want %f", cfg.SalariuMinimBrut, def.SalariuMinimBrut)
	}
	if len(cfg.CASThresholds) != len(def.CASThresholds) {
		t.Errorf("CASThresholds count = %d, want %d", len(cfg.CASThresholds), len(def.CASThresholds))
	}
	if len(cfg.CASSThresholds) != len(def.CASSThresholds) {
		t.Errorf("CASSThresholds count = %d, want %d", len(cfg.CASSThresholds), len(def.CASSThresholds))
	}
}

// Brackets must tile the income range with no gaps: each bracket's max equals
// the next bracket's min, starting at 0 and ending open-ended (max = 0)
func TestDefaultThresholdsAreContiguous(t *testing.T) {
	def := DefaultTaxConfig()

	for name, thresholds := range map[string][]TaxThreshold{
		"CAS":  def.CASThresholds,
		"CASS": def.CASSThresholds,
	} {
		if thresholds[0].MinSalaries != 0 {
			t.Errorf("%s: first bracket starts at %f, want 0", name, thresholds[0].MinSalaries)
		}
		last := thresholds[len(thresholds)-1]
		if last.MaxSalaries != 0 {
			t.Errorf("%s: last bracket max = %f, want 0 (unlimited)", name, last.MaxSalaries)
		}
		for i := 0; i < len(thresholds)-1; i++ {
			if thresholds[i].MaxSalaries != thresholds[i+1].MinSalaries {
				t.Errorf("%s: gap between bracket %d (max %f) and %d (min %f)",
					name, i, thresholds[i].MaxSalaries, i+1, thresholds[i+1].MinSalaries)
			}
		}
	}
}
