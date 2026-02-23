package iso20022

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func TestCreditTransferInitiationType(t *testing.T) {
	msg := CreditTransferInitiation{}
	if msg.Type() != Pain001 {
		t.Errorf("expected %s, got %s", Pain001, msg.Type())
	}
}

func TestCreditTransferInitiationToXML(t *testing.T) {
	msg := CreditTransferInitiation{
		Header: MessageHeader{
			MessageID:    "MSG-001",
			CreationDate: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		PaymentInfo: []PaymentInstructionInfo{
			{
				PaymentInfoID: "PI-001",
				PaymentMethod: "TRF",
				DebtorName:    "Acme Corp",
				DebtorAccount: "DE89370400440532013000",
				DebtorAgent:   "COBADEFFXXX",
				Transactions: []CreditTransferTransaction{
					{
						EndToEndID:      "E2E-001",
						Amount:          "1000.00",
						Currency:        "EUR",
						CreditorName:    "Widget Inc",
						CreditorAccount: "GB29NWBK60161331926819",
						CreditorAgent:   "NWBKGB2LXXX",
						RemittanceInfo:  "Invoice 12345",
					},
				},
			},
		},
	}

	data, err := msg.ToXML()
	if err != nil {
		t.Fatalf("ToXML() returned error: %v", err)
	}

	// Verify it is valid XML
	var doc pain001Document
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("produced invalid XML: %v", err)
	}

	// Verify namespace
	xmlStr := string(data)
	if !strings.Contains(xmlStr, "urn:iso:std:iso:20022:tech:xsd:pain.001.001.12") {
		t.Error("XML does not contain expected pain.001 namespace")
	}

	// Verify message ID
	if doc.CstmrCdtTrfInitn.GrpHdr.MsgID != "MSG-001" {
		t.Errorf("expected MsgId MSG-001, got %s", doc.CstmrCdtTrfInitn.GrpHdr.MsgID)
	}

	// Verify transaction count
	if doc.CstmrCdtTrfInitn.GrpHdr.NbOfTxs != "1" {
		t.Errorf("expected NbOfTxs 1, got %s", doc.CstmrCdtTrfInitn.GrpHdr.NbOfTxs)
	}
}

func TestFIToFICreditTransferType(t *testing.T) {
	msg := FIToFICreditTransfer{}
	if msg.Type() != Pacs008 {
		t.Errorf("expected %s, got %s", Pacs008, msg.Type())
	}
}

func TestFIToFICreditTransferToXML(t *testing.T) {
	msg := FIToFICreditTransfer{
		Header: MessageHeader{
			MessageID:    "PACS-001",
			CreationDate: time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
		},
		Transactions: []FICreditTransferTransaction{
			{
				TransactionID:   "TXN-001",
				EndToEndID:      "E2E-001",
				Amount:          "5000.00",
				Currency:        "USD",
				DebtorAgent:     "COBADEFFXXX",
				CreditorAgent:   "NWBKGB2LXXX",
				DebtorName:      "Acme Corp",
				DebtorAccount:   "DE89370400440532013000",
				CreditorName:    "Widget Inc",
				CreditorAccount: "GB29NWBK60161331926819",
			},
			{
				TransactionID:   "TXN-002",
				EndToEndID:      "E2E-002",
				Amount:          "3000.00",
				Currency:        "USD",
				DebtorAgent:     "COBADEFFXXX",
				CreditorAgent:   "CHASUS33XXX",
				DebtorName:      "Acme Corp",
				DebtorAccount:   "DE89370400440532013000",
				CreditorName:    "Global Ltd",
				CreditorAccount: "US12345678901234567890",
			},
		},
	}

	data, err := msg.ToXML()
	if err != nil {
		t.Fatalf("ToXML() returned error: %v", err)
	}

	// Verify it is valid XML
	var doc pacs008Document
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("produced invalid XML: %v", err)
	}

	// Verify namespace
	xmlStr := string(data)
	if !strings.Contains(xmlStr, "urn:iso:std:iso:20022:tech:xsd:pacs.008.001.12") {
		t.Error("XML does not contain expected pacs.008 namespace")
	}

	// Verify message ID
	if doc.FIToFICstmrCdtTrf.GrpHdr.MsgID != "PACS-001" {
		t.Errorf("expected MsgId PACS-001, got %s", doc.FIToFICstmrCdtTrf.GrpHdr.MsgID)
	}

	// Verify transaction count
	if doc.FIToFICstmrCdtTrf.GrpHdr.NbOfTxs != "2" {
		t.Errorf("expected NbOfTxs 2, got %s", doc.FIToFICstmrCdtTrf.GrpHdr.NbOfTxs)
	}

	// Verify settlement method
	if doc.FIToFICstmrCdtTrf.GrpHdr.SttlmInf.SttlmMtd != "CLRG" {
		t.Errorf("expected SttlmMtd CLRG, got %s", doc.FIToFICstmrCdtTrf.GrpHdr.SttlmInf.SttlmMtd)
	}
}

func TestPain001XMLNamespace(t *testing.T) {
	msg := CreditTransferInitiation{
		Header: MessageHeader{
			MessageID:    "NS-TEST",
			CreationDate: time.Now(),
		},
	}
	data, err := msg.ToXML()
	if err != nil {
		t.Fatalf("ToXML() returned error: %v", err)
	}
	if !strings.Contains(string(data), "urn:iso:std:iso:20022:tech:xsd:pain.001.001.12") {
		t.Error("pain.001 namespace not found in XML output")
	}
}

func TestPacs008XMLNamespace(t *testing.T) {
	msg := FIToFICreditTransfer{
		Header: MessageHeader{
			MessageID:    "NS-TEST",
			CreationDate: time.Now(),
		},
	}
	data, err := msg.ToXML()
	if err != nil {
		t.Fatalf("ToXML() returned error: %v", err)
	}
	if !strings.Contains(string(data), "urn:iso:std:iso:20022:tech:xsd:pacs.008.001.12") {
		t.Error("pacs.008 namespace not found in XML output")
	}
}
