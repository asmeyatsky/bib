package iso20022

import (
	"encoding/xml"
	"fmt"
	"time"
)

// FIToFICreditTransfer represents pacs.008 message.
type FIToFICreditTransfer struct {
	Header       MessageHeader
	Transactions []FICreditTransferTransaction
}

func (f FIToFICreditTransfer) Type() MessageType { return Pacs008 }

func (f FIToFICreditTransfer) ToXML() ([]byte, error) {
	doc := pacs008Document{
		XMLName: xml.Name{Local: "Document"},
		Xmlns:   "urn:iso:std:iso:20022:tech:xsd:pacs.008.001.12",
		FIToFICstmrCdtTrf: pacs008FIToFICstmrCdtTrf{
			GrpHdr: pacs008GrpHdr{
				MsgID:   f.Header.MessageID,
				CreDtTm: f.Header.CreationDate.Format(time.RFC3339),
				NbOfTxs: fmt.Sprintf("%d", len(f.Transactions)),
				SttlmInf: pacs008SttlmInf{
					SttlmMtd: "CLRG",
				},
			},
		},
	}
	return xml.MarshalIndent(doc, "", "  ")
}

// FICreditTransferTransaction contains FI-level transaction details.
type FICreditTransferTransaction struct {
	TransactionID   string
	EndToEndID      string
	Amount          string
	Currency        string
	DebtorAgent     string // BIC
	CreditorAgent   string // BIC
	DebtorName      string
	DebtorAccount   string
	CreditorName    string
	CreditorAccount string
}

// XML marshaling structs
type pacs008Document struct {
	XMLName           xml.Name                 `xml:"Document"`
	Xmlns             string                   `xml:"xmlns,attr"`
	FIToFICstmrCdtTrf pacs008FIToFICstmrCdtTrf `xml:"FIToFICstmrCdtTrf"`
}

type pacs008FIToFICstmrCdtTrf struct {
	GrpHdr pacs008GrpHdr `xml:"GrpHdr"`
}

type pacs008GrpHdr struct {
	MsgID    string          `xml:"MsgId"`
	CreDtTm  string          `xml:"CreDtTm"`
	NbOfTxs  string          `xml:"NbOfTxs"`
	SttlmInf pacs008SttlmInf `xml:"SttlmInf"`
}

type pacs008SttlmInf struct {
	SttlmMtd string `xml:"SttlmMtd"`
}
