package iso20022

import (
	"fmt"
	"strings"
	"time"
)

// MT950StatementMessage represents a parsed SWIFT MT950 statement message.
// MT950 is used for nostro account statement reporting between correspondent banks.
type MT950StatementMessage struct {
	// TransactionReference is the sender's reference (field :20:).
	TransactionReference string
	// AccountIdentification is the nostro account number (field :25:).
	AccountIdentification string
	// StatementNumber is the statement/sequence number (field :28C:).
	StatementNumber string
	// OpeningBalance is the opening balance from field :60F:.
	OpeningBalance StatementBalance
	// Entries contains the individual statement lines from field :61:.
	Entries []StatementEntry
	// ClosingBalance is the closing balance from field :62F:.
	ClosingBalance StatementBalance
}

// StatementBalance represents an opening or closing balance in an MT950 message.
type StatementBalance struct {
	// DebitCredit indicates whether the balance is a debit ("D") or credit ("C").
	DebitCredit string
	// Date is the balance date.
	Date time.Time
	// Currency is the ISO 4217 currency code.
	Currency string
	// Amount is the balance amount as a string (decimal).
	Amount string
}

// StatementEntry represents a single transaction line in an MT950 statement (field :61:).
type StatementEntry struct {
	// ValueDate is the value date of the transaction.
	ValueDate time.Time
	// EntryDate is the booking date (may differ from value date).
	EntryDate time.Time
	// DebitCredit indicates "D" for debit or "C" for credit.
	DebitCredit string
	// Amount is the transaction amount as a decimal string.
	Amount string
	// TransactionType is the SWIFT transaction type code (e.g. "NTRF").
	TransactionType string
	// Reference is the account servicing institution's reference.
	Reference string
	// SupplementaryDetails holds additional information from field :86:.
	SupplementaryDetails string
}

// ParseMT950 parses a simplified MT950 SWIFT message string into a structured
// MT950StatementMessage. The parser handles the core fields used for nostro
// reconciliation: :20:, :25:, :28C:, :60F:, :61:, :86:, and :62F:.
//
// This is a simplified parser that handles the most common MT950 format.
// Production use would require a full SWIFT FIN message parser.
func ParseMT950(raw string) (MT950StatementMessage, error) {
	if raw == "" {
		return MT950StatementMessage{}, fmt.Errorf("empty MT950 message")
	}

	msg := MT950StatementMessage{}
	lines := strings.Split(strings.TrimSpace(raw), "\n")

	var currentField string
	var currentEntry *StatementEntry

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		// Detect field tags (e.g. ":20:", ":25:", ":61:", etc.)
		if strings.HasPrefix(line, ":") && len(line) > 3 {
			// Extract field tag
			endIdx := strings.Index(line[1:], ":")
			if endIdx > 0 {
				tag := line[1 : endIdx+1]
				value := line[endIdx+2:]

				switch tag {
				case "20":
					currentField = "20"
					msg.TransactionReference = value
				case "25":
					currentField = "25"
					msg.AccountIdentification = value
				case "28C":
					currentField = "28C"
					msg.StatementNumber = value
				case "60F":
					currentField = "60F"
					bal, err := parseStatementBalance(value)
					if err != nil {
						return MT950StatementMessage{}, fmt.Errorf("parse opening balance: %w", err)
					}
					msg.OpeningBalance = bal
				case "61":
					currentField = "61"
					// Flush previous entry if any
					if currentEntry != nil {
						msg.Entries = append(msg.Entries, *currentEntry)
					}
					entry, err := parseStatementEntry(value)
					if err != nil {
						return MT950StatementMessage{}, fmt.Errorf("parse statement entry: %w", err)
					}
					currentEntry = &entry
				case "86":
					currentField = "86"
					if currentEntry != nil {
						currentEntry.SupplementaryDetails = value
					}
				case "62F":
					currentField = "62F"
					// Flush any pending entry
					if currentEntry != nil {
						msg.Entries = append(msg.Entries, *currentEntry)
						currentEntry = nil
					}
					bal, err := parseStatementBalance(value)
					if err != nil {
						return MT950StatementMessage{}, fmt.Errorf("parse closing balance: %w", err)
					}
					msg.ClosingBalance = bal
				default:
					currentField = tag
				}
				continue
			}
		}

		// Continuation line for supplementary details
		if currentField == "86" && currentEntry != nil {
			currentEntry.SupplementaryDetails += " " + line
		}
	}

	// Flush last entry if no closing balance was found after it
	if currentEntry != nil {
		msg.Entries = append(msg.Entries, *currentEntry)
	}

	if msg.TransactionReference == "" {
		return MT950StatementMessage{}, fmt.Errorf("missing transaction reference field :20:")
	}
	if msg.AccountIdentification == "" {
		return MT950StatementMessage{}, fmt.Errorf("missing account identification field :25:")
	}

	return msg, nil
}

// parseStatementBalance parses a balance field value.
// Format: D/C + YYMMDD + Currency + Amount
// Example: "C230115USD1234567,89"
func parseStatementBalance(s string) (StatementBalance, error) {
	if len(s) < 14 {
		return StatementBalance{}, fmt.Errorf("balance field too short: %q", s)
	}

	dc := string(s[0])
	if dc != "C" && dc != "D" {
		return StatementBalance{}, fmt.Errorf("invalid debit/credit indicator: %q", dc)
	}

	dateStr := s[1:7]
	date, err := parseSWIFTDate(dateStr)
	if err != nil {
		return StatementBalance{}, fmt.Errorf("parse balance date: %w", err)
	}

	currency := s[7:10]
	amountStr := strings.ReplaceAll(s[10:], ",", ".")

	return StatementBalance{
		DebitCredit: dc,
		Date:        date,
		Currency:    currency,
		Amount:      amountStr,
	}, nil
}

// parseStatementEntry parses a :61: field value.
// Simplified format: YYMMDD[MMDD] D/C Amount TransactionType Reference
// Example: "230115 C1000,00NTRF REF001"
func parseStatementEntry(s string) (StatementEntry, error) {
	if len(s) < 16 {
		return StatementEntry{}, fmt.Errorf("statement entry too short: %q", s)
	}

	// Parse value date (YYMMDD)
	valueDateStr := s[0:6]
	valueDate, err := parseSWIFTDate(valueDateStr)
	if err != nil {
		return StatementEntry{}, fmt.Errorf("parse value date: %w", err)
	}

	// Check for optional entry date (MMDD) after value date
	pos := 6
	entryDate := valueDate
	if len(s) > 10 && s[6] >= '0' && s[6] <= '1' {
		entryDateStr := fmt.Sprintf("%s%s", valueDateStr[:2], s[6:10])
		ed, err := parseSWIFTDate(entryDateStr)
		if err == nil {
			entryDate = ed
			pos = 10
		}
	}

	// Parse debit/credit indicator
	dc := string(s[pos])
	if dc != "C" && dc != "D" && dc != "R" {
		return StatementEntry{}, fmt.Errorf("invalid debit/credit indicator at position %d: %q", pos, dc)
	}
	// RC = reversal credit, RD = reversal debit
	if dc == "R" && pos+1 < len(s) {
		dc = s[pos : pos+2]
		pos += 2
	} else {
		pos++
	}

	// Parse amount (digits and comma/period until a letter is found)
	amountStart := pos
	for pos < len(s) && (s[pos] >= '0' && s[pos] <= '9' || s[pos] == ',' || s[pos] == '.') {
		pos++
	}
	amountStr := strings.ReplaceAll(s[amountStart:pos], ",", ".")

	// Parse transaction type (4 alpha characters)
	txnType := ""
	if pos+4 <= len(s) {
		txnType = s[pos : pos+4]
		pos += 4
	}

	// Remaining is reference
	reference := ""
	if pos < len(s) {
		reference = strings.TrimSpace(s[pos:])
	}

	return StatementEntry{
		ValueDate:       valueDate,
		EntryDate:       entryDate,
		DebitCredit:     dc,
		Amount:          amountStr,
		TransactionType: txnType,
		Reference:       reference,
	}, nil
}

// parseSWIFTDate parses a date in YYMMDD format.
func parseSWIFTDate(s string) (time.Time, error) {
	if len(s) != 6 {
		return time.Time{}, fmt.Errorf("SWIFT date must be 6 characters, got %d", len(s))
	}
	t, err := time.Parse("060102", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse SWIFT date %q: %w", s, err)
	}
	return t, nil
}
