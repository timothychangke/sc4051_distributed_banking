package marshal

import (
	"encoding/binary"
	"math"
	"testing"

	"bank-server/pkg/models"
)

// ─────────────────────────────────────────────────────────────────────
// Callback packet tests.
//
// The callback format is flat (not TLV), so we verify each field at
// its exact byte offset. The C++ client's monitor loop reads these
// bytes sequentially: if anything is misaligned, it's game over.
// ─────────────────────────────────────────────────────────────────────

func TestMarshalCallbackUpdate_MinimumSize(t *testing.T) {
	// With an empty holder name, the minimum size is:
	// 1 (MsgType) + 1 (ServiceID) + 4 (AccNo) + 4 (name len) + 0 (name) + 1 (Currency) + 8 (Balance) = 19
	update := models.AccountUpdate{
		ServiceID:     1,
		AccountNumber: 10000,
		HolderName:    "",
		CurrencyType:  models.SGD,
		NewBalance:    0.0,
	}

	data, err := MarshalCallbackUpdate(update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) != 19 {
		t.Errorf("minimum callback size: want 19 bytes, got %d", len(data))
	}
}

func TestMarshalCallbackUpdate_MsgType(t *testing.T) {
	update := models.AccountUpdate{
		ServiceID:     3,
		AccountNumber: 10001,
		HolderName:    "Alice",
		CurrencyType:  models.SGD,
		NewBalance:    1000.0,
	}

	data, err := MarshalCallbackUpdate(update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Byte 0: Must be 0x02 (MsgTypeCallback)
	if data[0] != MsgTypeCallback {
		t.Errorf("byte 0 (MsgType): want 0x%02X, got 0x%02X", MsgTypeCallback, data[0])
	}
}

func TestMarshalCallbackUpdate_ServiceID(t *testing.T) {
	update := models.AccountUpdate{
		ServiceID:     4, // Withdraw
		AccountNumber: 10001,
		HolderName:    "Bob",
		CurrencyType:  models.USD,
		NewBalance:    500.0,
	}

	data, err := MarshalCallbackUpdate(update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Byte 1: ServiceID
	if data[1] != 4 {
		t.Errorf("byte 1 (ServiceID): want 4, got %d", data[1])
	}
}

func TestMarshalCallbackUpdate_AccountNumber(t *testing.T) {
	update := models.AccountUpdate{
		ServiceID:     1,
		AccountNumber: 0x0000CAFE,
		HolderName:    "Test",
		CurrencyType:  models.EUR,
		NewBalance:    0.0,
	}

	data, err := MarshalCallbackUpdate(update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bytes 2-5: AccountNumber in big-endian
	gotAccNo := binary.BigEndian.Uint32(data[2:6])
	if gotAccNo != 0x0000CAFE {
		t.Errorf("bytes 2-5 (AccountNumber): want 0x0000CAFE, got 0x%08X", gotAccNo)
	}
}

func TestMarshalCallbackUpdate_HolderName(t *testing.T) {
	update := models.AccountUpdate{
		ServiceID:     1,
		AccountNumber: 10001,
		HolderName:    "Charlie",
		CurrencyType:  models.SGD,
		NewBalance:    100.0,
	}

	data, err := MarshalCallbackUpdate(update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Bytes 6-9: Name length as big-endian uint32
	nameLen := binary.BigEndian.Uint32(data[6:10])
	if nameLen != 7 { // "Charlie" = 7 bytes
		t.Errorf("name length: want 7, got %d", nameLen)
	}

	// Bytes 10-16: The name string itself
	gotName := string(data[10 : 10+nameLen])
	if gotName != "Charlie" {
		t.Errorf("name: want 'Charlie', got '%s'", gotName)
	}
}

func TestMarshalCallbackUpdate_FullPacket(t *testing.T) {
	// Verify the complete packet layout with a realistic example
	update := models.AccountUpdate{
		ServiceID:     3, // Deposit
		AccountNumber: 10042,
		HolderName:    "Alice",
		CurrencyType:  models.USD, // 2
		NewBalance:    1500.75,
	}

	data, err := MarshalCallbackUpdate(update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expected total: 1 + 1 + 4 + 4 + 5("Alice") + 1 + 8 = 24 bytes
	expectedSize := 1 + 1 + 4 + 4 + 5 + 1 + 8
	if len(data) != expectedSize {
		t.Fatalf("total size: want %d, got %d", expectedSize, len(data))
	}

	offset := 0

	// MsgType
	if data[offset] != MsgTypeCallback {
		t.Errorf("MsgType: want 0x02, got 0x%02X", data[offset])
	}
	offset++

	// ServiceID
	if data[offset] != 3 {
		t.Errorf("ServiceID: want 3, got %d", data[offset])
	}
	offset++

	// AccountNumber
	accNo := binary.BigEndian.Uint32(data[offset : offset+4])
	if accNo != 10042 {
		t.Errorf("AccountNumber: want 10042, got %d", accNo)
	}
	offset += 4

	// Name length
	nameLen := binary.BigEndian.Uint32(data[offset : offset+4])
	if nameLen != 5 {
		t.Errorf("NameLen: want 5, got %d", nameLen)
	}
	offset += 4

	// Name
	name := string(data[offset : offset+int(nameLen)])
	if name != "Alice" {
		t.Errorf("Name: want 'Alice', got '%s'", name)
	}
	offset += int(nameLen)

	// Currency
	if data[offset] != 2 { // USD = 2
		t.Errorf("Currency: want 2, got %d", data[offset])
	}
	offset++

	// Balance
	bits := binary.BigEndian.Uint64(data[offset : offset+8])
	balance := math.Float64frombits(bits)
	if balance != 1500.75 {
		t.Errorf("Balance: want 1500.75, got %f", balance)
	}
}

func TestMarshalCallbackUpdate_AllCurrencyTypes(t *testing.T) {
	currencies := []struct {
		currency models.Currency
		expected uint8
	}{
		{models.SGD, 1},
		{models.USD, 2},
		{models.EUR, 3},
	}

	for _, tc := range currencies {
		update := models.AccountUpdate{
			ServiceID:     1,
			AccountNumber: 10000,
			HolderName:    "X",
			CurrencyType:  tc.currency,
			NewBalance:    0.0,
		}

		data, err := MarshalCallbackUpdate(update)
		if err != nil {
			t.Fatalf("error for currency %d: %v", tc.expected, err)
		}

		// Currency byte is after: MsgType(1) + ServiceID(1) + AccNo(4) + NameLen(4) + Name(1)
		currOffset := 1 + 1 + 4 + 4 + 1
		if data[currOffset] != tc.expected {
			t.Errorf("currency %d: want %d at offset %d, got %d",
				tc.expected, tc.expected, currOffset, data[currOffset])
		}
	}
}

func TestMarshalCallbackUpdate_LongName(t *testing.T) {
	longName := "Bartholomew Jebediah Worthington III"
	update := models.AccountUpdate{
		ServiceID:     1,
		AccountNumber: 10000,
		HolderName:    longName,
		CurrencyType:  models.SGD,
		NewBalance:    0.0,
	}

	data, err := MarshalCallbackUpdate(update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSize := 1 + 1 + 4 + 4 + len(longName) + 1 + 8
	if len(data) != expectedSize {
		t.Errorf("total size: want %d, got %d", expectedSize, len(data))
	}

	// Read the name back out
	nameLen := binary.BigEndian.Uint32(data[6:10])
	if nameLen != uint32(len(longName)) {
		t.Errorf("name length: want %d, got %d", len(longName), nameLen)
	}
	gotName := string(data[10 : 10+nameLen])
	if gotName != longName {
		t.Errorf("name: want '%s', got '%s'", longName, gotName)
	}
}

func TestMarshalCallbackUpdate_ZeroBalance(t *testing.T) {
	update := models.AccountUpdate{
		ServiceID:     2, // Close
		AccountNumber: 10001,
		HolderName:    "A",
		CurrencyType:  models.SGD,
		NewBalance:    0.0,
	}

	data, err := MarshalCallbackUpdate(update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Balance is the last 8 bytes
	balanceOffset := len(data) - 8
	bits := binary.BigEndian.Uint64(data[balanceOffset : balanceOffset+8])
	balance := math.Float64frombits(bits)
	if balance != 0.0 {
		t.Errorf("balance: want 0.0, got %f", balance)
	}
}
