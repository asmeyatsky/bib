package iso20022

import "time"

// MessageType represents ISO 20022 message types.
type MessageType string

const (
	// Payment Initiation
	Pain001 MessageType = "pain.001.001.12" // CustomerCreditTransferInitiation
	Pain002 MessageType = "pain.002.001.14" // CustomerPaymentStatusReport
	Pain008 MessageType = "pain.008.001.11" // CustomerDirectDebitInitiation

	// Payment Clearing and Settlement
	Pacs002 MessageType = "pacs.002.001.14" // FIToFIPaymentStatusReport
	Pacs004 MessageType = "pacs.004.001.13" // PaymentReturn
	Pacs008 MessageType = "pacs.008.001.12" // FIToFICustomerCreditTransfer
	Pacs009 MessageType = "pacs.009.001.11" // FIToFIFinancialInstitutionCreditTransfer

	// Cash Management
	Camt053 MessageType = "camt.053.001.11" // BankToCustomerStatement
	Camt054 MessageType = "camt.054.001.11" // BankToCustomerDebitCreditNotification
)

// Message is the base interface for all ISO 20022 messages.
type Message interface {
	Type() MessageType
	ToXML() ([]byte, error)
}

// MessageHeader contains the Business Application Header (BAH) fields.
type MessageHeader struct {
	MessageID    string
	CreationDate time.Time
	From         PartyIdentification
	To           PartyIdentification
	MessageType  MessageType
}

// PartyIdentification identifies a party in the message.
type PartyIdentification struct {
	BIC  string // Business Identifier Code (SWIFT code)
	Name string
	ID   string // LEI or other identifier
}
