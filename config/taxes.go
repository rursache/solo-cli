package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const taxesFileName = "taxes.json"

// TaxThreshold defines a salary-based threshold bracket
type TaxThreshold struct {
	// MinSalaries is the lower bound (inclusive) in multiples of SMB
	MinSalaries float64 `json:"min_salaries"`
	// MaxSalaries is the upper bound (exclusive), 0 means unlimited
	MaxSalaries float64 `json:"max_salaries"`
	// BaseSalaries is what to calculate % on: positive = fixed multiple of SMB, 0 = exempt, -1 = proportional (use actual net income)
	BaseSalaries float64 `json:"base_salaries"`
	// Label is a human-readable description
	Label string `json:"label"`
}

// TaxConfig holds all configurable tax parameters
type TaxConfig struct {
	Year             int            `json:"year"`
	SalariuMinimBrut float64        `json:"salariu_minim_brut"`
	IncomeTaxPercent float64        `json:"income_tax_percent"`
	CASPercent       float64        `json:"cas_percent"`
	CASThresholds    []TaxThreshold `json:"cas_thresholds"`
	CASSPercent      float64        `json:"cass_percent"`
	CASSThresholds   []TaxThreshold `json:"cass_thresholds"`
}

// DefaultTaxConfig returns the default tax configuration for 2026
func DefaultTaxConfig() *TaxConfig {
	return &TaxConfig{
		Year:             2026,
		SalariuMinimBrut: 4325,
		IncomeTaxPercent: 10,
		CASPercent:       25,
		CASThresholds: []TaxThreshold{
			{MinSalaries: 0, MaxSalaries: 12, BaseSalaries: 0, Label: "Fără CAS (sub 12 salarii)"},
			{MinSalaries: 12, MaxSalaries: 24, BaseSalaries: 12, Label: "CAS pe 12 salarii"},
			{MinSalaries: 24, MaxSalaries: 0, BaseSalaries: 24, Label: "CAS pe 24 salarii"},
		},
		CASSPercent: 10,
		CASSThresholds: []TaxThreshold{
			{MinSalaries: 0, MaxSalaries: 6, BaseSalaries: 6, Label: "CASS minim (6 salarii)"},
			{MinSalaries: 6, MaxSalaries: 72, BaseSalaries: -1, Label: "CASS proporțional"},
			{MinSalaries: 72, MaxSalaries: 0, BaseSalaries: 72, Label: "CASS plafonat (72 salarii)"},
		},
	}
}

// GetTaxesConfigPath returns the full path to the taxes config file
func GetTaxesConfigPath() (string, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(configPath), taxesFileName), nil
}

// EnsureTaxesExists creates a default taxes.json if it doesn't exist
func EnsureTaxesExists() error {
	taxesPath, err := GetTaxesConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(taxesPath); os.IsNotExist(err) {
		data, err := json.MarshalIndent(DefaultTaxConfig(), "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(taxesPath, data, 0644)
	}

	return nil
}

// LoadTaxes reads and parses the taxes config file
func LoadTaxes() (*TaxConfig, error) {
	if err := EnsureTaxesExists(); err != nil {
		return nil, err
	}

	taxesPath, err := GetTaxesConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(taxesPath)
	if err != nil {
		return nil, err
	}

	var cfg TaxConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
