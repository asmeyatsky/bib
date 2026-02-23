package service

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/bibbank/bib/services/reporting-service/internal/domain/valueobject"
)

// ReportData holds the financial data needed to generate a regulatory report.
type ReportData struct {
	TenantID           uuid.UUID
	Period             string
	TotalAssets        decimal.Decimal
	TotalLiabilities   decimal.Decimal
	TotalEquity        decimal.Decimal
	NetIncome          decimal.Decimal
	RiskWeightedAssets decimal.Decimal
	CET1Ratio          decimal.Decimal
	LCRRatio           decimal.Decimal
}

// XBRLGenerator is a domain service that generates XBRL content for regulatory reports.
type XBRLGenerator struct{}

// NewXBRLGenerator creates a new XBRLGenerator.
func NewXBRLGenerator() *XBRLGenerator {
	return &XBRLGenerator{}
}

// Generate creates XBRL content for the given report type and data.
func (g *XBRLGenerator) Generate(reportType valueobject.ReportType, data ReportData) (string, error) {
	if reportType.IsZero() {
		return "", fmt.Errorf("report type must not be empty")
	}

	switch {
	case reportType.Equal(valueobject.ReportTypeCOREP):
		return g.generateCOREP(data), nil
	case reportType.Equal(valueobject.ReportTypeFINREP):
		return g.generateFINREP(data), nil
	case reportType.Equal(valueobject.ReportTypeMREL):
		return g.generateMREL(data), nil
	case reportType.Equal(valueobject.ReportTypeCUSTOM):
		return g.generateCustom(data), nil
	default:
		return "", fmt.Errorf("unsupported report type: %s", reportType)
	}
}

func (g *XBRLGenerator) generateCOREP(data ReportData) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString("\n")
	b.WriteString(`<xbrli:xbrl`)
	b.WriteString(` xmlns:xbrli="http://www.xbrl.org/2003/instance"`)
	b.WriteString(` xmlns:link="http://www.xbrl.org/2003/linkbase"`)
	b.WriteString(` xmlns:xlink="http://www.w3.org/1999/xlink"`)
	b.WriteString(` xmlns:iso4217="http://www.xbrl.org/2003/iso4217"`)
	b.WriteString(` xmlns:corep="http://www.eba.europa.eu/xbrl/crr/dict/met"`)
	b.WriteString(` xmlns:find="http://www.eurofiling.info/xbrl/ext/filing-indicators">`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <xbrli:context id="ctx_%s">`, data.Period))
	b.WriteString("\n")
	b.WriteString(`    <xbrli:entity>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`      <xbrli:identifier scheme="http://www.bibbank.com">%s</xbrli:identifier>`, data.TenantID))
	b.WriteString("\n")
	b.WriteString(`    </xbrli:entity>`)
	b.WriteString("\n")
	b.WriteString(`    <xbrli:period>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`      <xbrli:instant>%s</xbrli:instant>`, periodToInstant(data.Period)))
	b.WriteString("\n")
	b.WriteString(`    </xbrli:period>`)
	b.WriteString("\n")
	b.WriteString(`  </xbrli:context>`)
	b.WriteString("\n")
	b.WriteString(`  <xbrli:unit id="u_EUR">
    <xbrli:measure>iso4217:EUR</xbrli:measure>
  </xbrli:unit>`)
	b.WriteString("\n")
	b.WriteString(`  <xbrli:unit id="u_pure">
    <xbrli:measure>xbrli:pure</xbrli:measure>
  </xbrli:unit>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <corep:RiskWeightedAssets contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</corep:RiskWeightedAssets>`,
		data.Period, data.RiskWeightedAssets.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <corep:CET1Ratio contextRef="ctx_%s" unitRef="u_pure" decimals="4">%s</corep:CET1Ratio>`,
		data.Period, data.CET1Ratio.StringFixed(4)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <corep:TotalEquity contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</corep:TotalEquity>`,
		data.Period, data.TotalEquity.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <corep:LCRRatio contextRef="ctx_%s" unitRef="u_pure" decimals="4">%s</corep:LCRRatio>`,
		data.Period, data.LCRRatio.StringFixed(4)))
	b.WriteString("\n")
	b.WriteString(`</xbrli:xbrl>`)
	b.WriteString("\n")
	return b.String()
}

func (g *XBRLGenerator) generateFINREP(data ReportData) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString("\n")
	b.WriteString(`<xbrli:xbrl`)
	b.WriteString(` xmlns:xbrli="http://www.xbrl.org/2003/instance"`)
	b.WriteString(` xmlns:link="http://www.xbrl.org/2003/linkbase"`)
	b.WriteString(` xmlns:xlink="http://www.w3.org/1999/xlink"`)
	b.WriteString(` xmlns:iso4217="http://www.xbrl.org/2003/iso4217"`)
	b.WriteString(` xmlns:finrep="http://www.eba.europa.eu/xbrl/crr/dict/finrep"`)
	b.WriteString(` xmlns:find="http://www.eurofiling.info/xbrl/ext/filing-indicators">`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <xbrli:context id="ctx_%s">`, data.Period))
	b.WriteString("\n")
	b.WriteString(`    <xbrli:entity>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`      <xbrli:identifier scheme="http://www.bibbank.com">%s</xbrli:identifier>`, data.TenantID))
	b.WriteString("\n")
	b.WriteString(`    </xbrli:entity>`)
	b.WriteString("\n")
	b.WriteString(`    <xbrli:period>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`      <xbrli:instant>%s</xbrli:instant>`, periodToInstant(data.Period)))
	b.WriteString("\n")
	b.WriteString(`    </xbrli:period>`)
	b.WriteString("\n")
	b.WriteString(`  </xbrli:context>`)
	b.WriteString("\n")
	b.WriteString(`  <xbrli:unit id="u_EUR">
    <xbrli:measure>iso4217:EUR</xbrli:measure>
  </xbrli:unit>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <finrep:TotalAssets contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</finrep:TotalAssets>`,
		data.Period, data.TotalAssets.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <finrep:TotalLiabilities contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</finrep:TotalLiabilities>`,
		data.Period, data.TotalLiabilities.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <finrep:TotalEquity contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</finrep:TotalEquity>`,
		data.Period, data.TotalEquity.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <finrep:NetIncome contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</finrep:NetIncome>`,
		data.Period, data.NetIncome.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(`</xbrli:xbrl>`)
	b.WriteString("\n")
	return b.String()
}

func (g *XBRLGenerator) generateMREL(data ReportData) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString("\n")
	b.WriteString(`<xbrli:xbrl`)
	b.WriteString(` xmlns:xbrli="http://www.xbrl.org/2003/instance"`)
	b.WriteString(` xmlns:link="http://www.xbrl.org/2003/linkbase"`)
	b.WriteString(` xmlns:xlink="http://www.w3.org/1999/xlink"`)
	b.WriteString(` xmlns:iso4217="http://www.xbrl.org/2003/iso4217"`)
	b.WriteString(` xmlns:mrel="http://www.eba.europa.eu/xbrl/crr/dict/mrel"`)
	b.WriteString(` xmlns:find="http://www.eurofiling.info/xbrl/ext/filing-indicators">`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <xbrli:context id="ctx_%s">`, data.Period))
	b.WriteString("\n")
	b.WriteString(`    <xbrli:entity>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`      <xbrli:identifier scheme="http://www.bibbank.com">%s</xbrli:identifier>`, data.TenantID))
	b.WriteString("\n")
	b.WriteString(`    </xbrli:entity>`)
	b.WriteString("\n")
	b.WriteString(`    <xbrli:period>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`      <xbrli:instant>%s</xbrli:instant>`, periodToInstant(data.Period)))
	b.WriteString("\n")
	b.WriteString(`    </xbrli:period>`)
	b.WriteString("\n")
	b.WriteString(`  </xbrli:context>`)
	b.WriteString("\n")
	b.WriteString(`  <xbrli:unit id="u_EUR">
    <xbrli:measure>iso4217:EUR</xbrli:measure>
  </xbrli:unit>`)
	b.WriteString("\n")
	b.WriteString(`  <xbrli:unit id="u_pure">
    <xbrli:measure>xbrli:pure</xbrli:measure>
  </xbrli:unit>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <mrel:TotalEquity contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</mrel:TotalEquity>`,
		data.Period, data.TotalEquity.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <mrel:TotalLiabilities contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</mrel:TotalLiabilities>`,
		data.Period, data.TotalLiabilities.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <mrel:RiskWeightedAssets contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</mrel:RiskWeightedAssets>`,
		data.Period, data.RiskWeightedAssets.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <mrel:CET1Ratio contextRef="ctx_%s" unitRef="u_pure" decimals="4">%s</mrel:CET1Ratio>`,
		data.Period, data.CET1Ratio.StringFixed(4)))
	b.WriteString("\n")
	b.WriteString(`</xbrli:xbrl>`)
	b.WriteString("\n")
	return b.String()
}

func (g *XBRLGenerator) generateCustom(data ReportData) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	b.WriteString("\n")
	b.WriteString(`<xbrli:xbrl`)
	b.WriteString(` xmlns:xbrli="http://www.xbrl.org/2003/instance"`)
	b.WriteString(` xmlns:link="http://www.xbrl.org/2003/linkbase"`)
	b.WriteString(` xmlns:xlink="http://www.w3.org/1999/xlink"`)
	b.WriteString(` xmlns:iso4217="http://www.xbrl.org/2003/iso4217"`)
	b.WriteString(` xmlns:custom="http://www.bibbank.com/xbrl/custom">`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <xbrli:context id="ctx_%s">`, data.Period))
	b.WriteString("\n")
	b.WriteString(`    <xbrli:entity>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`      <xbrli:identifier scheme="http://www.bibbank.com">%s</xbrli:identifier>`, data.TenantID))
	b.WriteString("\n")
	b.WriteString(`    </xbrli:entity>`)
	b.WriteString("\n")
	b.WriteString(`    <xbrli:period>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`      <xbrli:instant>%s</xbrli:instant>`, periodToInstant(data.Period)))
	b.WriteString("\n")
	b.WriteString(`    </xbrli:period>`)
	b.WriteString("\n")
	b.WriteString(`  </xbrli:context>`)
	b.WriteString("\n")
	b.WriteString(`  <xbrli:unit id="u_EUR">
    <xbrli:measure>iso4217:EUR</xbrli:measure>
  </xbrli:unit>`)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <custom:TotalAssets contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</custom:TotalAssets>`,
		data.Period, data.TotalAssets.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <custom:TotalLiabilities contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</custom:TotalLiabilities>`,
		data.Period, data.TotalLiabilities.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <custom:TotalEquity contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</custom:TotalEquity>`,
		data.Period, data.TotalEquity.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf(`  <custom:NetIncome contextRef="ctx_%s" unitRef="u_EUR" decimals="0">%s</custom:NetIncome>`,
		data.Period, data.NetIncome.StringFixed(0)))
	b.WriteString("\n")
	b.WriteString(`</xbrli:xbrl>`)
	b.WriteString("\n")
	return b.String()
}

// periodToInstant converts a period like "2025-Q1" to an instant date.
func periodToInstant(period string) string {
	parts := strings.Split(period, "-")
	if len(parts) != 2 {
		return period
	}
	year := parts[0]
	quarter := parts[1]
	switch quarter {
	case "Q1":
		return year + "-03-31"
	case "Q2":
		return year + "-06-30"
	case "Q3":
		return year + "-09-30"
	case "Q4":
		return year + "-12-31"
	default:
		return period
	}
}
