package marshal

import "testing"

func TestFieldIDValues(t *testing.T) {

	tests := []struct {
		name string
		got  uint8
		want uint8
	}{
		{"Service", FieldService, 0x01},
		{"AccountNumber", FieldAccountNumber, 0x02},
		{"AccountOwnerName", FieldAccountOwnerName, 0x03},
		{"AccountPassword", FieldAccountPassword, 0x04},
		{"TxAccountNumber", FieldTxAccountNumber, 0x05},
		{"TxAccountOwnerName", FieldTxAccountOwnerName, 0x06},
		{"MonetaryValue", FieldMonetaryValue, 0x07},
		{"Currency", FieldCurrency, 0x08},
		{"MonitorUpdates", FieldMonitorUpdates, 0x09},
		{"MonitorTimeoutSeconds", FieldMonitorTimeoutSeconds, 0x0A},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("Field%s = 0x%02X, want 0x%02X", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestTLVHeaderSize(t *testing.T) {
	
	if TLVHeaderSize != 5 {
		t.Errorf("TLVHeaderSize = %d, want 5", TLVHeaderSize)
	}
}

func TestIsValidFieldID(t *testing.T) {
	
	validIDs := []uint8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
	for _, id := range validIDs {
		if !IsValidFieldID(id) {
			t.Errorf("IsValidFieldID(0x%02X) = false, want true", id)
		}
	}

	
	invalidIDs := []uint8{0x00, 0x0B, 0x0C, 0xFF, 0x10, 0x80}
	for _, id := range invalidIDs {
		if IsValidFieldID(id) {
			t.Errorf("IsValidFieldID(0x%02X) = true, want false", id)
		}
	}
}

func TestFieldIDsAreUnique(t *testing.T) {
	
	all := []uint8{
		FieldService, FieldAccountNumber, FieldAccountOwnerName,
		FieldAccountPassword, FieldTxAccountNumber, FieldTxAccountOwnerName,
		FieldMonetaryValue, FieldCurrency,
		FieldMonitorUpdates, FieldMonitorTimeoutSeconds,
	}

	seen := make(map[uint8]bool)
	for _, id := range all {
		if seen[id] {
			t.Errorf("duplicate FieldID value: 0x%02X", id)
		}
		seen[id] = true
	}
}