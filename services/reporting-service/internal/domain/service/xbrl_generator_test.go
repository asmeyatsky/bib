package service_test

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/reporting-service/internal/domain/service"
	"github.com/bibbank/bib/services/reporting-service/internal/domain/valueobject"
)

func sampleReportData() service.ReportData {
	return service.ReportData{
		TenantID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		Period:             "2025-Q1",
		TotalAssets:        decimal.NewFromInt(1_500_000_000),
		TotalLiabilities:   decimal.NewFromInt(1_350_000_000),
		TotalEquity:        decimal.NewFromInt(150_000_000),
		NetIncome:          decimal.NewFromInt(25_000_000),
		RiskWeightedAssets: decimal.NewFromInt(800_000_000),
		CET1Ratio:          decimal.NewFromFloat(0.1475),
		LCRRatio:           decimal.NewFromFloat(1.2500),
	}
}

func TestXBRLGenerator_GenerateCOREP(t *testing.T) {
	gen := service.NewXBRLGenerator()
	data := sampleReportData()

	content, err := gen.Generate(valueobject.ReportTypeCOREP, data)
	require.NoError(t, err)

	// Verify it is well-formed XML.
	assert.True(t, xml.Unmarshal([]byte(content), new(interface{})) == nil, "output should be valid XML")

	// Verify required XBRL elements.
	assert.Contains(t, content, `<?xml version="1.0"`)
	assert.Contains(t, content, `xbrli:xbrl`)
	assert.Contains(t, content, `xbrli:context`)
	assert.Contains(t, content, `xbrli:period`)
	assert.Contains(t, content, `xbrli:instant`)
	assert.Contains(t, content, `2025-03-31`)

	// Verify COREP-specific content.
	assert.Contains(t, content, `corep:RiskWeightedAssets`)
	assert.Contains(t, content, `corep:CET1Ratio`)
	assert.Contains(t, content, `corep:LCRRatio`)
	assert.Contains(t, content, `800000000`)
	assert.Contains(t, content, `0.1475`)
	assert.Contains(t, content, `1.2500`)

	// Verify namespaces.
	assert.Contains(t, content, `xmlns:corep=`)
	assert.Contains(t, content, `xmlns:xbrli=`)
	assert.Contains(t, content, `xmlns:iso4217=`)
}

func TestXBRLGenerator_GenerateFINREP(t *testing.T) {
	gen := service.NewXBRLGenerator()
	data := sampleReportData()

	content, err := gen.Generate(valueobject.ReportTypeFINREP, data)
	require.NoError(t, err)

	// Verify well-formed XML.
	assert.True(t, xml.Unmarshal([]byte(content), new(interface{})) == nil, "output should be valid XML")

	// Verify FINREP-specific content.
	assert.Contains(t, content, `finrep:TotalAssets`)
	assert.Contains(t, content, `finrep:TotalLiabilities`)
	assert.Contains(t, content, `finrep:TotalEquity`)
	assert.Contains(t, content, `finrep:NetIncome`)
	assert.Contains(t, content, `1500000000`)
	assert.Contains(t, content, `1350000000`)
	assert.Contains(t, content, `150000000`)
	assert.Contains(t, content, `25000000`)

	// Verify namespaces.
	assert.Contains(t, content, `xmlns:finrep=`)
}

func TestXBRLGenerator_GenerateMREL(t *testing.T) {
	gen := service.NewXBRLGenerator()
	data := sampleReportData()

	content, err := gen.Generate(valueobject.ReportTypeMREL, data)
	require.NoError(t, err)

	// Verify well-formed XML.
	assert.True(t, xml.Unmarshal([]byte(content), new(interface{})) == nil, "output should be valid XML")

	// Verify MREL-specific content.
	assert.Contains(t, content, `mrel:TotalEquity`)
	assert.Contains(t, content, `mrel:TotalLiabilities`)
	assert.Contains(t, content, `mrel:RiskWeightedAssets`)
	assert.Contains(t, content, `mrel:CET1Ratio`)

	// Verify namespaces.
	assert.Contains(t, content, `xmlns:mrel=`)
}

func TestXBRLGenerator_GenerateCustom(t *testing.T) {
	gen := service.NewXBRLGenerator()
	data := sampleReportData()

	content, err := gen.Generate(valueobject.ReportTypeCUSTOM, data)
	require.NoError(t, err)

	// Verify well-formed XML.
	assert.True(t, xml.Unmarshal([]byte(content), new(interface{})) == nil, "output should be valid XML")

	// Verify custom namespace elements.
	assert.Contains(t, content, `custom:TotalAssets`)
	assert.Contains(t, content, `custom:TotalLiabilities`)
	assert.Contains(t, content, `custom:TotalEquity`)
	assert.Contains(t, content, `custom:NetIncome`)
	assert.Contains(t, content, `xmlns:custom=`)
}

func TestXBRLGenerator_EmptyReportType(t *testing.T) {
	gen := service.NewXBRLGenerator()
	data := sampleReportData()

	_, err := gen.Generate(valueobject.ReportType{}, data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestXBRLGenerator_PeriodConversion(t *testing.T) {
	gen := service.NewXBRLGenerator()

	quarters := map[string]string{
		"2025-Q1": "2025-03-31",
		"2025-Q2": "2025-06-30",
		"2025-Q3": "2025-09-30",
		"2025-Q4": "2025-12-31",
	}

	for period, expectedDate := range quarters {
		data := sampleReportData()
		data.Period = period

		content, err := gen.Generate(valueobject.ReportTypeCOREP, data)
		require.NoError(t, err)
		assert.Contains(t, content, expectedDate,
			"period %s should map to instant %s", period, expectedDate)
	}
}

func TestXBRLGenerator_ContextAndEntity(t *testing.T) {
	gen := service.NewXBRLGenerator()
	data := sampleReportData()

	content, err := gen.Generate(valueobject.ReportTypeCOREP, data)
	require.NoError(t, err)

	// Verify entity identifier contains tenant ID.
	assert.Contains(t, content, data.TenantID.String())

	// Verify context ID references the period.
	assert.Contains(t, content, `ctx_2025-Q1`)

	// Verify unit definitions.
	assert.Contains(t, content, `iso4217:EUR`)
	assert.True(t, strings.Contains(content, `xbrli:pure`) || strings.Contains(content, `xbrli:measure`))
}
