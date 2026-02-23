package iso20022

import (
	"encoding/xml"
	"fmt"
	"time"
)

// CreditTransferInitiation represents pain.001 message.
type CreditTransferInitiation struct {
	Header      MessageHeader
	PaymentInfo []PaymentInstructionInfo
}

func (c CreditTransferInitiation) Type() MessageType { return Pain001 }

func (c CreditTransferInitiation) ToXML() ([]byte, error) {
	doc := pain001Document{
		XMLName: xml.Name{Local: "Document"},
		Xmlns:   "urn:iso:std:iso:20022:tech:xsd:pain.001.001.12",
		CstmrCdtTrfInitn: pain001CstmrCdtTrfInitn{
			GrpHdr: pain001GrpHdr{
				MsgID:   c.Header.MessageID,
				CreDtTm: c.Header.CreationDate.Format(time.RFC3339),
				NbOfTxs: countTransactions(c.PaymentInfo),
			},
		},
	}
	return xml.MarshalIndent(doc, "", "  ")
}

// PaymentInstructionInfo contains payment instruction details.
type PaymentInstructionInfo struct {
	PaymentInfoID string
	PaymentMethod string // "TRF" for credit transfer
	DebtorName    string
	DebtorAccount string
	DebtorAgent   string // BIC
	Transactions  []CreditTransferTransaction
}

// CreditTransferTransaction contains individual transaction details.
type CreditTransferTransaction struct {
	EndToEndID      string
	Amount          string // decimal string
	Currency        string
	CreditorName    string
	CreditorAccount string
	CreditorAgent   string // BIC
	RemittanceInfo  string
}

// XML marshaling structs (internal)
type pain001Document struct {
	XMLName          xml.Name                `xml:"Document"`
	Xmlns            string                  `xml:"xmlns,attr"`
	CstmrCdtTrfInitn pain001CstmrCdtTrfInitn `xml:"CstmrCdtTrfInitn"`
}

type pain001CstmrCdtTrfInitn struct {
	GrpHdr pain001GrpHdr `xml:"GrpHdr"`
}

type pain001GrpHdr struct {
	MsgID   string `xml:"MsgId"`
	CreDtTm string `xml:"CreDtTm"`
	NbOfTxs string `xml:"NbOfTxs"`
}

func countTransactions(infos []PaymentInstructionInfo) string {
	count := 0
	for _, info := range infos {
		count += len(info.Transactions)
	}
	return fmt.Sprintf("%d", count)
}
