package taxes

import (
	"math"
	"testing"

	"solo-cli/config"
)

func defaultCfg() *config.TaxConfig {
	return config.DefaultTaxConfig()
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func TestCalculateZeroAndNegativeNetIncome(t *testing.T) {
	cfg := defaultCfg()

	for _, tc := range []struct {
		name               string
		revenues, expenses float64
	}{
		{"zero", 0, 0},
		{"negative", 1000, 5000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Calculate(tc.revenues, tc.expenses, cfg)
			if r.NetIncome != 0 {
				t.Errorf("NetIncome = %f, want 0", r.NetIncome)
			}
			if r.CAS.Amount != 0 {
				t.Errorf("CAS.Amount = %f, want 0", r.CAS.Amount)
			}
			if r.IncomeTax != 0 {
				t.Errorf("IncomeTax = %f, want 0", r.IncomeTax)
			}
			if r.EffectiveRate != 0 {
				t.Errorf("EffectiveRate = %f, want 0", r.EffectiveRate)
			}
		})
	}
}

// Net income below 6 salarii: CAS exempt, CASS on minimum base of 6 SMB
func TestCalculateBelowAllThresholds(t *testing.T) {
	cfg := defaultCfg()
	smb := cfg.SalariuMinimBrut

	r := Calculate(3*smb, 0, cfg)

	if r.CAS.Amount != 0 {
		t.Errorf("CAS.Amount = %f, want 0 (exempt under 12 salarii)", r.CAS.Amount)
	}
	wantCASS := math.Round(6*smb*cfg.CASSPercent) / 100
	if !almostEqual(r.CASS.Amount, wantCASS) {
		t.Errorf("CASS.Amount = %f, want %f (minimum on 6 salarii)", r.CASS.Amount, wantCASS)
	}
	if r.CASS.Base != 6*smb {
		t.Errorf("CASS.Base = %f, want %f", r.CASS.Base, 6*smb)
	}
}

// Net income between 12 and 24 salarii: CAS on fixed 12 SMB base, CASS proportional
func TestCalculateMidBracket(t *testing.T) {
	cfg := defaultCfg()
	smb := cfg.SalariuMinimBrut
	netIncome := 15 * smb

	r := Calculate(netIncome, 0, cfg)

	wantCAS := math.Round(12*smb*cfg.CASPercent) / 100
	if !almostEqual(r.CAS.Amount, wantCAS) {
		t.Errorf("CAS.Amount = %f, want %f", r.CAS.Amount, wantCAS)
	}
	wantCASS := math.Round(netIncome*cfg.CASSPercent) / 100
	if !almostEqual(r.CASS.Amount, wantCASS) {
		t.Errorf("CASS.Amount = %f, want %f (proportional)", r.CASS.Amount, wantCASS)
	}

	wantTaxable := netIncome - wantCAS - wantCASS
	wantIncomeTax := math.Round(wantTaxable*cfg.IncomeTaxPercent) / 100
	if !almostEqual(r.IncomeTax, wantIncomeTax) {
		t.Errorf("IncomeTax = %f, want %f", r.IncomeTax, wantIncomeTax)
	}
	if !almostEqual(r.TotalTaxes, wantCAS+wantCASS+wantIncomeTax) {
		t.Errorf("TotalTaxes = %f, want %f", r.TotalTaxes, wantCAS+wantCASS+wantIncomeTax)
	}
}

// Above the top CAS threshold (24+ salarii): CAS on fixed 24 SMB base
func TestCalculateTopCASBracket(t *testing.T) {
	cfg := defaultCfg()
	smb := cfg.SalariuMinimBrut

	r := Calculate(30*smb, 0, cfg)

	wantCAS := math.Round(24*smb*cfg.CASPercent) / 100
	if !almostEqual(r.CAS.Amount, wantCAS) {
		t.Errorf("CAS.Amount = %f, want %f", r.CAS.Amount, wantCAS)
	}
	if r.CAS.NextLabel != "" {
		t.Errorf("CAS.NextLabel = %q, want empty (already at top bracket)", r.CAS.NextLabel)
	}
}

// Above the CASS cap: CASS frozen at the capped base regardless of income
func TestCalculateCASSCapped(t *testing.T) {
	cfg := defaultCfg()
	smb := cfg.SalariuMinimBrut
	capSalaries := cfg.CASSThresholds[len(cfg.CASSThresholds)-1].MinSalaries

	r := Calculate((capSalaries+10)*smb, 0, cfg)

	wantCASS := math.Round(capSalaries*smb*cfg.CASSPercent) / 100
	if !almostEqual(r.CASS.Amount, wantCASS) {
		t.Errorf("CASS.Amount = %f, want %f (capped)", r.CASS.Amount, wantCASS)
	}
	if r.CASS.Base != capSalaries*smb {
		t.Errorf("CASS.Base = %f, want %f", r.CASS.Base, capSalaries*smb)
	}
}

// Buffer must point at the boundary where the contribution actually changes
func TestBufferToNext(t *testing.T) {
	cfg := defaultCfg()
	smb := cfg.SalariuMinimBrut
	netIncome := 11.2 * smb

	r := Calculate(netIncome, 0, cfg)

	wantCASBuffer := 12*smb - netIncome
	if !almostEqual(r.CAS.BufferToNext, wantCASBuffer) {
		t.Errorf("CAS.BufferToNext = %f, want %f", r.CAS.BufferToNext, wantCASBuffer)
	}

	capSalaries := cfg.CASSThresholds[len(cfg.CASSThresholds)-1].MinSalaries
	wantCASSBuffer := capSalaries*smb - netIncome
	if !almostEqual(r.CASS.BufferToNext, wantCASSBuffer) {
		t.Errorf("CASS.BufferToNext = %f, want %f", r.CASS.BufferToNext, wantCASSBuffer)
	}
}

// Surplus hint: fires for CAS where dropping a bracket saves more than the
// expense costs, suppressed for proportional CASS where it is a net loss
func TestSurplusHint(t *testing.T) {
	cfg := defaultCfg()
	smb := cfg.SalariuMinimBrut
	netIncome := 12.5 * smb

	r := Calculate(netIncome, 0, cfg)

	if r.CAS.PrevLabel == "" {
		t.Fatal("CAS.PrevLabel empty, want surplus hint just above 12 salarii")
	}
	wantExpenses := math.Round((netIncome-12*smb)*100)/100 + 1
	if !almostEqual(r.CAS.ExpensesToPrev, wantExpenses) {
		t.Errorf("CAS.ExpensesToPrev = %f, want %f", r.CAS.ExpensesToPrev, wantExpenses)
	}
	// CAS saving (12 SMB * 25%) must beat the expense needed
	if saving := math.Round(12*smb*cfg.CASPercent) / 100; saving <= wantExpenses {
		t.Errorf("hint fired but saving %f <= expense %f", saving, wantExpenses)
	}

	if r.CASS.PrevLabel != "" {
		t.Errorf("CASS.PrevLabel = %q, want empty (dropping proportional bracket is a net loss)", r.CASS.PrevLabel)
	}
}

// Regression: net income just under the 12 salarii CAS threshold. CAS buffer
// points to 12 salarii; CASS buffer points to the cap because CASS is
// proportional in between, so no CASS hint should reference 12 salarii
func TestCalculateJustUnderCASThreshold(t *testing.T) {
	cfg := defaultCfg()
	netIncome := 11.3 * cfg.SalariuMinimBrut

	r := Calculate(netIncome+25000, 25000, cfg)

	if !almostEqual(r.NetIncome, netIncome) {
		t.Fatalf("NetIncome = %f, want %f", r.NetIncome, netIncome)
	}
	if r.CAS.Amount != 0 {
		t.Errorf("CAS.Amount = %f, want 0", r.CAS.Amount)
	}
	if !almostEqual(r.CAS.BufferToNext, 12*cfg.SalariuMinimBrut-netIncome) {
		t.Errorf("CAS.BufferToNext = %f, want %f", r.CAS.BufferToNext, 12*cfg.SalariuMinimBrut-netIncome)
	}
	wantCASS := math.Round(netIncome*cfg.CASSPercent) / 100
	if !almostEqual(r.CASS.Amount, wantCASS) {
		t.Errorf("CASS.Amount = %f, want %f", r.CASS.Amount, wantCASS)
	}
	capSalaries := cfg.CASSThresholds[len(cfg.CASSThresholds)-1].MinSalaries
	if !almostEqual(r.CASS.BufferToNext, capSalaries*cfg.SalariuMinimBrut-netIncome) {
		t.Errorf("CASS.BufferToNext = %f, want distance to cap", r.CASS.BufferToNext)
	}
}

func TestFormatHelpers(t *testing.T) {
	if got := FormatRON(1234.5); got != "1234.50 RON" {
		t.Errorf("FormatRON = %q", got)
	}
	if got := FormatBuffer(0); got != "plafonul a fost atins" {
		t.Errorf("FormatBuffer(0) = %q", got)
	}
	if got := FormatBuffer(100); got == "plafonul a fost atins" {
		t.Errorf("FormatBuffer(100) = %q, want remaining amount", got)
	}
}
