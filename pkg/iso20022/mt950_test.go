package iso20022

import (
	"testing"
	"time"
)

func TestParseMT950_ValidMessage(t *testing.T) {
	raw := `:20:STMT230115001
:25:DEUTDEFFXXX/DE89370400440532013000
:28C:15/1
:60F:C230114USD1000000,00
:61:230115C50000,00NTRF REF001
:86:Payment from Widget Inc Invoice 12345
:61:230115D25000,00NTRF REF002
:86:Transfer to Global Ltd
:62F:C230115USD1025000,00`

	msg, err := ParseMT950(raw)
	if err != nil {
		t.Fatalf("ParseMT950() returned error: %v", err)
	}

	// Verify transaction reference
	if msg.TransactionReference != "STMT230115001" {
		t.Errorf("expected TransactionReference STMT230115001, got %s", msg.TransactionReference)
	}

	// Verify account identification
	if msg.AccountIdentification != "DEUTDEFFXXX/DE89370400440532013000" {
		t.Errorf("expected AccountIdentification DEUTDEFFXXX/DE89370400440532013000, got %s", msg.AccountIdentification)
	}

	// Verify statement number
	if msg.StatementNumber != "15/1" {
		t.Errorf("expected StatementNumber 15/1, got %s", msg.StatementNumber)
	}

	// Verify opening balance
	if msg.OpeningBalance.DebitCredit != "C" {
		t.Errorf("expected opening balance C, got %s", msg.OpeningBalance.DebitCredit)
	}
	if msg.OpeningBalance.Currency != "USD" {
		t.Errorf("expected opening balance currency USD, got %s", msg.OpeningBalance.Currency)
	}
	if msg.OpeningBalance.Amount != "1000000.00" {
		t.Errorf("expected opening balance amount 1000000.00, got %s", msg.OpeningBalance.Amount)
	}
	expectedDate := time.Date(2023, 1, 14, 0, 0, 0, 0, time.UTC)
	if !msg.OpeningBalance.Date.Equal(expectedDate) {
		t.Errorf("expected opening balance date %v, got %v", expectedDate, msg.OpeningBalance.Date)
	}

	// Verify entries
	if len(msg.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(msg.Entries))
	}

	// First entry - credit
	entry1 := msg.Entries[0]
	if entry1.DebitCredit != "C" {
		t.Errorf("entry[0]: expected C, got %s", entry1.DebitCredit)
	}
	if entry1.Amount != "50000.00" {
		t.Errorf("entry[0]: expected amount 50000.00, got %s", entry1.Amount)
	}
	if entry1.TransactionType != "NTRF" {
		t.Errorf("entry[0]: expected transaction type NTRF, got %s", entry1.TransactionType)
	}
	if entry1.Reference != "REF001" {
		t.Errorf("entry[0]: expected reference REF001, got %s", entry1.Reference)
	}
	if entry1.SupplementaryDetails != "Payment from Widget Inc Invoice 12345" {
		t.Errorf("entry[0]: unexpected supplementary details: %s", entry1.SupplementaryDetails)
	}

	// Second entry - debit
	entry2 := msg.Entries[1]
	if entry2.DebitCredit != "D" {
		t.Errorf("entry[1]: expected D, got %s", entry2.DebitCredit)
	}
	if entry2.Amount != "25000.00" {
		t.Errorf("entry[1]: expected amount 25000.00, got %s", entry2.Amount)
	}
	if entry2.Reference != "REF002" {
		t.Errorf("entry[1]: expected reference REF002, got %s", entry2.Reference)
	}

	// Verify closing balance
	if msg.ClosingBalance.DebitCredit != "C" {
		t.Errorf("expected closing balance C, got %s", msg.ClosingBalance.DebitCredit)
	}
	if msg.ClosingBalance.Amount != "1025000.00" {
		t.Errorf("expected closing balance amount 1025000.00, got %s", msg.ClosingBalance.Amount)
	}
}

func TestParseMT950_EmptyMessage(t *testing.T) {
	_, err := ParseMT950("")
	if err == nil {
		t.Error("expected error for empty message")
	}
}

func TestParseMT950_MissingTransactionReference(t *testing.T) {
	raw := `:25:DEUTDEFFXXX/DE89370400440532013000
:28C:15/1
:60F:C230114USD1000000,00
:62F:C230115USD1000000,00`

	_, err := ParseMT950(raw)
	if err == nil {
		t.Error("expected error for missing transaction reference")
	}
}

func TestParseMT950_MissingAccountIdentification(t *testing.T) {
	raw := `:20:STMT230115001
:28C:15/1
:60F:C230114USD1000000,00
:62F:C230115USD1000000,00`

	_, err := ParseMT950(raw)
	if err == nil {
		t.Error("expected error for missing account identification")
	}
}

func TestParseSWIFTDate(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
		wantErr  bool
	}{
		{"230115", time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC), false},
		{"991231", time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC), false},
		{"", time.Time{}, true},
		{"12345", time.Time{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseSWIFTDate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestParseStatementBalance(t *testing.T) {
	bal, err := parseStatementBalance("D230115EUR500000,50")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bal.DebitCredit != "D" {
		t.Errorf("expected D, got %s", bal.DebitCredit)
	}
	if bal.Currency != "EUR" {
		t.Errorf("expected EUR, got %s", bal.Currency)
	}
	if bal.Amount != "500000.50" {
		t.Errorf("expected 500000.50, got %s", bal.Amount)
	}
}
