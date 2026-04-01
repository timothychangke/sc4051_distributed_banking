package handler

import (
	"encoding/binary"
	"net"
	"testing"
	"time"

	"bank-server/internal/banking"
	marshal "bank-server/internal/marshalling"
	"bank-server/internal/monitor"
	"bank-server/internal/semantics"
	"bank-server/internal/store"
	"bank-server/pkg/models"
	protocol "bank-server/util"
)

// ─────────────────────────────────────────────────────────────────────
// Handler integration tests.
//
// These use a real MemoryStore and banking.Service so we're testing the
// full path from raw bytes → handler → banking → raw reply bytes.
// The only thing mocked is the UDP connection (monitor notifications
// go to a dummy conn).
// ─────────────────────────────────────────────────────────────────────

// testSetup creates a fresh banking service and monitor for each test.
// Returns the handler function and the banking service (for state inspection).
func testSetup(t *testing.T) (semantics.RequestHandler, banking.Service) {
	t.Helper()

	memStore := store.NewMemoryStore()
	svc := banking.NewService(memStore)

	// Monitor needs a real UDPConn. Bind to a random port on loopback
	// so we don't conflict with anything. We never actually read from it.
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("failed to create monitor UDP socket: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	mon := monitor.NewManager(conn, marshal.MarshalCallbackUpdate, 1*time.Second)
	t.Cleanup(func() { mon.Stop() })

	handler := BuildHandler(svc, mon)
	return handler, svc
}

// clientAddr returns a deterministic fake client address for tests.
func clientAddr() *net.UDPAddr {
	return &net.UDPAddr{IP: net.IPv4(192, 168, 1, 50), Port: 9999}
}

// buildRequest constructs a full request packet with the semantics header
// prepended, exactly as the C++ client sends on the wire.
//
//	[Type(1B)] [Flag(1B)] [RequestID(4B BE)] [IPv4(4B BE)] [Port(2B BE)] [StatusCode(2B BE)] [ContentLen(4B BE)] [TLV payload...]
func buildRequest(requestID uint32, tlvFields []marshal.TLVField) []byte {
	tlvBytes := marshal.EncodeTLVFields(tlvFields)
	contentLen := len(tlvBytes)

	// Pre-allocate the full 18-byte header + payload
	enc := marshal.NewEncoderWithCap(semantics.HeaderSize + contentLen)

	// 18-byte header
	enc.PutUint8(marshal.MsgTypeReply) // Type (0x01)
	enc.PutUint8(0)                  // Flag
	enc.PutUint32(requestID)         // RequestID
	enc.PutUint32(0)                  // IPv4 (empty for test)
	enc.PutUint16(0)                  // Port
	enc.PutUint16(0)                  // StatusCode
	enc.PutUint32(uint32(contentLen)) // ContentLen

	// Payload
	if contentLen > 0 {
		enc.PutBytes(tlvBytes)
	}

	return enc.Bytes()
}

// parseReply pulls apart the fixed reply header so tests can check individual fields.
type parsedReply struct {
	MsgType    uint8
	RequestID  uint32
	IPv4       uint32
	Port       uint16
	StatusCode uint16
	ContentLen uint32
	Content    []byte
}

func parseReplyBytes(t *testing.T, data []byte) parsedReply {
	t.Helper()
	if len(data) < semantics.HeaderSize {
		t.Fatalf("reply too short: %d bytes (need at least %d)", len(data), semantics.HeaderSize)
	}

	r := parsedReply{
		MsgType:    data[0],
		RequestID:  binary.BigEndian.Uint32(data[2:6]),
		IPv4:       binary.BigEndian.Uint32(data[6:10]),
		Port:       binary.BigEndian.Uint16(data[10:12]),
		StatusCode: binary.BigEndian.Uint16(data[12:14]),
		ContentLen: binary.BigEndian.Uint32(data[14:18]),
	}
	if len(data) > semantics.HeaderSize {
		r.Content = data[18:]
	}
	return r
}

// --- OpenAccount tests ---

func TestHandleOpen_Success(t *testing.T) {
	handler, _ := testSetup(t)

	req := buildRequest(1, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceOpenAccount),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Alice"),
		marshal.TLVString(marshal.FieldAccountPassword, "secret12"),
		marshal.TLVUint8(marshal.FieldCurrency, uint8(models.SGD)),
		marshal.TLVFloat64(marshal.FieldMonetaryValue, 500.0),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode != uint16(protocol.StatusSuccess) {
		t.Fatalf("expected success, got status %d", r.StatusCode)
	}
	if r.RequestID != 1 {
		t.Errorf("RequestID: want 1, got %d", r.RequestID)
	}

	// Content should contain the new account number
	cmd, err := marshal.DecodeTLV(r.Content)
	if err != nil {
		t.Fatalf("failed to decode reply content: %v", err)
	}
	if cmd.AccountNumber == nil {
		t.Fatal("expected account number in reply")
	}
	// MemoryStore starts at 10000
	if *cmd.AccountNumber < 10000 {
		t.Errorf("account number %d seems too low", *cmd.AccountNumber)
	}
}

func TestHandleOpen_MissingFields(t *testing.T) {
	handler, _ := testSetup(t)

	// Missing password and currency
	req := buildRequest(2, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceOpenAccount),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Bob"),
		marshal.TLVFloat64(marshal.FieldMonetaryValue, 100.0),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode == uint16(protocol.StatusSuccess) {
		t.Fatal("expected error for missing fields, got success")
	}
}

// --- CloseAccount tests ---

func TestHandleClose_Success(t *testing.T) {
	handler, svc := testSetup(t)

	// First create an account to close
	pw := marshal.PasswordStringToFixed("close_me")
	accNo := svc.OpenAccount("Dave", pw, models.USD, 0.0)

	req := buildRequest(10, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceCloseAccount),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Dave"),
		marshal.TLVUint32(marshal.FieldAccountNumber, accNo),
		marshal.TLVString(marshal.FieldAccountPassword, "close_me"),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode != uint16(protocol.StatusSuccess) {
		t.Fatalf("expected success, got status %d", r.StatusCode)
	}
	// Close returns no content
	if r.ContentLen != 0 {
		t.Errorf("expected empty content, got %d bytes", r.ContentLen)
	}
}

func TestHandleClose_WrongPassword(t *testing.T) {
	handler, svc := testSetup(t)

	pw := marshal.PasswordStringToFixed("rightpwd")
	accNo := svc.OpenAccount("Eve", pw, models.SGD, 100.0)

	req := buildRequest(11, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceCloseAccount),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Eve"),
		marshal.TLVUint32(marshal.FieldAccountNumber, accNo),
		marshal.TLVString(marshal.FieldAccountPassword, "wrongpwd"),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode == uint16(protocol.StatusSuccess) {
		t.Fatal("expected error for wrong password")
	}
}

// --- Deposit tests ---

func TestHandleDeposit_Success(t *testing.T) {
	handler, svc := testSetup(t)

	pw := marshal.PasswordStringToFixed("deposit1")
	accNo := svc.OpenAccount("Frank", pw, models.SGD, 100.0)

	req := buildRequest(20, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceDeposit),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Frank"),
		marshal.TLVUint32(marshal.FieldAccountNumber, accNo),
		marshal.TLVString(marshal.FieldAccountPassword, "deposit1"),
		marshal.TLVUint8(marshal.FieldCurrency, uint8(models.SGD)),
		marshal.TLVFloat64(marshal.FieldMonetaryValue, 250.50),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode != uint16(protocol.StatusSuccess) {
		t.Fatalf("expected success, got status %d", r.StatusCode)
	}

	// Content should contain the new balance: 100.0 + 250.50 = 350.50
	cmd, err := marshal.DecodeTLV(r.Content)
	if err != nil {
		t.Fatalf("failed to decode reply content: %v", err)
	}
	if cmd.MonetaryValue == nil {
		t.Fatal("expected balance in reply")
	}
	if *cmd.MonetaryValue != 350.50 {
		t.Errorf("balance: want 350.50, got %f", *cmd.MonetaryValue)
	}
}

func TestHandleDeposit_CurrencyMismatch(t *testing.T) {
	handler, svc := testSetup(t)

	pw := marshal.PasswordStringToFixed("currmis1")
	accNo := svc.OpenAccount("Grace", pw, models.SGD, 100.0)

	// Try to deposit USD into an SGD account
	req := buildRequest(21, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceDeposit),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Grace"),
		marshal.TLVUint32(marshal.FieldAccountNumber, accNo),
		marshal.TLVString(marshal.FieldAccountPassword, "currmis1"),
		marshal.TLVUint8(marshal.FieldCurrency, uint8(models.USD)),
		marshal.TLVFloat64(marshal.FieldMonetaryValue, 50.0),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode != uint16(protocol.StatusErrCurrMismatch) {
		t.Errorf("expected currency mismatch error (status %d), got %d",
			protocol.StatusErrCurrMismatch, r.StatusCode)
	}
}

// --- Withdraw tests ---

func TestHandleWithdraw_Success(t *testing.T) {
	handler, svc := testSetup(t)

	pw := marshal.PasswordStringToFixed("withdraw")
	accNo := svc.OpenAccount("Hank", pw, models.EUR, 1000.0)

	req := buildRequest(30, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceWithdraw),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Hank"),
		marshal.TLVUint32(marshal.FieldAccountNumber, accNo),
		marshal.TLVString(marshal.FieldAccountPassword, "withdraw"),
		marshal.TLVUint8(marshal.FieldCurrency, uint8(models.EUR)),
		marshal.TLVFloat64(marshal.FieldMonetaryValue, 300.0),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode != uint16(protocol.StatusSuccess) {
		t.Fatalf("expected success, got status %d", r.StatusCode)
	}

	cmd, _ := marshal.DecodeTLV(r.Content)
	if *cmd.MonetaryValue != 700.0 {
		t.Errorf("balance: want 700.0, got %f", *cmd.MonetaryValue)
	}
}

func TestHandleWithdraw_InsufficientFunds(t *testing.T) {
	handler, svc := testSetup(t)

	pw := marshal.PasswordStringToFixed("insuffi1")
	accNo := svc.OpenAccount("Ivy", pw, models.SGD, 50.0)

	req := buildRequest(31, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceWithdraw),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Ivy"),
		marshal.TLVUint32(marshal.FieldAccountNumber, accNo),
		marshal.TLVString(marshal.FieldAccountPassword, "insuffi1"),
		marshal.TLVUint8(marshal.FieldCurrency, uint8(models.SGD)),
		marshal.TLVFloat64(marshal.FieldMonetaryValue, 100.0),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode != uint16(protocol.StatusErrInsuffFunds) {
		t.Errorf("expected insufficient funds (status %d), got %d",
			protocol.StatusErrInsuffFunds, r.StatusCode)
	}
}

// --- GetBalance tests ---

func TestHandleGetBalance_Success(t *testing.T) {
	handler, svc := testSetup(t)

	pw := marshal.PasswordStringToFixed("balance1")
	accNo := svc.OpenAccount("Jack", pw, models.USD, 777.77)

	req := buildRequest(40, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceGetBalance),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Jack"),
		marshal.TLVUint32(marshal.FieldAccountNumber, accNo),
		marshal.TLVString(marshal.FieldAccountPassword, "balance1"),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode != uint16(protocol.StatusSuccess) {
		t.Fatalf("expected success, got status %d", r.StatusCode)
	}

	cmd, _ := marshal.DecodeTLV(r.Content)
	if *cmd.MonetaryValue != 777.77 {
		t.Errorf("balance: want 777.77, got %f", *cmd.MonetaryValue)
	}
}

// --- Transfer tests ---

func TestHandleTransfer_Success(t *testing.T) {
	handler, svc := testSetup(t)

	pw := marshal.PasswordStringToFixed("xfer_src")
	srcAccNo := svc.OpenAccount("Kate", pw, models.SGD, 1000.0)

	dstPw := marshal.PasswordStringToFixed("xfer_dst")
	dstAccNo := svc.OpenAccount("Leo", dstPw, models.SGD, 500.0)

	req := buildRequest(50, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceTransferFunds),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Kate"),
		marshal.TLVUint32(marshal.FieldAccountNumber, srcAccNo),
		marshal.TLVString(marshal.FieldAccountPassword, "xfer_src"),
		marshal.TLVUint32(marshal.FieldTxAccountNumber, dstAccNo),
		marshal.TLVString(marshal.FieldTxAccountOwnerName, "Leo"),
		marshal.TLVUint8(marshal.FieldCurrency, uint8(models.SGD)),
		marshal.TLVFloat64(marshal.FieldMonetaryValue, 200.0),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode != uint16(protocol.StatusSuccess) {
		t.Fatalf("expected success, got status %d", r.StatusCode)
	}

	// Sender's new balance: 1000 - 200 = 800
	cmd, _ := marshal.DecodeTLV(r.Content)
	if *cmd.MonetaryValue != 800.0 {
		t.Errorf("sender balance: want 800.0, got %f", *cmd.MonetaryValue)
	}
}

func TestHandleTransfer_SameAccount(t *testing.T) {
	handler, svc := testSetup(t)

	pw := marshal.PasswordStringToFixed("self_xfr")
	accNo := svc.OpenAccount("Mike", pw, models.SGD, 500.0)

	req := buildRequest(51, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceTransferFunds),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Mike"),
		marshal.TLVUint32(marshal.FieldAccountNumber, accNo),
		marshal.TLVString(marshal.FieldAccountPassword, "self_xfr"),
		marshal.TLVUint32(marshal.FieldTxAccountNumber, accNo), // same account!
		marshal.TLVString(marshal.FieldTxAccountOwnerName, "Mike"),
		marshal.TLVUint8(marshal.FieldCurrency, uint8(models.SGD)),
		marshal.TLVFloat64(marshal.FieldMonetaryValue, 100.0),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode != uint16(protocol.StatusErrSameAccount) {
		t.Errorf("expected same-account error (status %d), got %d",
			protocol.StatusErrSameAccount, r.StatusCode)
	}
}

// --- Monitor tests ---

func TestHandleMonitor_Success(t *testing.T) {
	handler, _ := testSetup(t)

	req := buildRequest(60, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceMonitor),
		marshal.TLVUint32(marshal.FieldMonitorTimeoutSeconds, 30),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode != uint16(protocol.StatusSuccess) {
		t.Fatalf("expected success, got status %d", r.StatusCode)
	}
	// Monitor ack has no content
	if r.ContentLen != 0 {
		t.Errorf("expected empty content, got %d bytes", r.ContentLen)
	}
}

func TestHandleMonitor_MissingTimeout(t *testing.T) {
	handler, _ := testSetup(t)

	// No MonitorTimeoutSeconds field — should fail
	req := buildRequest(61, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceMonitor),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.StatusCode == uint16(protocol.StatusSuccess) {
		t.Fatal("expected error for missing timeout, got success")
	}
}

// --- Reply format sanity checks ---

func TestReplyFormat_AlwaysHas18ByteHeader(t *testing.T) {
	handler, _ := testSetup(t)

	// Even a bad request should produce a well-formed reply
	req := buildRequest(99, nil) // unknown service ID

	reply := handler(req, clientAddr())
	if len(reply) < semantics.HeaderSize {
		t.Fatalf("reply too short: %d bytes", len(reply))
	}

	r := parseReplyBytes(t, reply)
	if r.MsgType != marshal.MsgTypeReply {
		t.Errorf("MsgType: want 0x%02X, got 0x%02X", marshal.MsgTypeReply, r.MsgType)
	}
	if r.RequestID != 99 {
		t.Errorf("RequestID: want 99, got %d", r.RequestID)
	}
}

func TestReplyFormat_RequestIDEchoed(t *testing.T) {
	handler, _ := testSetup(t)

	// Use a distinctive request ID and make sure it comes back
	req := buildRequest(0xCAFEBABE, []marshal.TLVField{
		marshal.TLVUint8(marshal.FieldService, protocol.ServiceOpenAccount),
		marshal.TLVString(marshal.FieldAccountOwnerName, "Test"),
		marshal.TLVString(marshal.FieldAccountPassword, "testtest"),
		marshal.TLVUint8(marshal.FieldCurrency, uint8(models.SGD)),
		marshal.TLVFloat64(marshal.FieldMonetaryValue, 0.0),
	})

	reply := handler(req, clientAddr())
	r := parseReplyBytes(t, reply)

	if r.RequestID != 0xCAFEBABE {
		t.Errorf("RequestID: want 0xCAFEBABE, got 0x%08X", r.RequestID)
	}
}

// --- Error mapping tests ---

func TestMapBankingError_AllCases(t *testing.T) {
	tests := []struct {
		err      error
		expected uint8
	}{
		{banking.ErrInvalidCredentials, protocol.StatusErrInvalidCreds},
		{banking.ErrAccountMismatch, protocol.StatusErrAccMismatch},
		{banking.ErrCurrencyMismatch, protocol.StatusErrCurrMismatch},
		{banking.ErrInsufficientFunds, protocol.StatusErrInsuffFunds},
		{banking.ErrTransferSameAccount, protocol.StatusErrSameAccount},
		{banking.ErrAccountNotFound, protocol.StatusErrAccNotFound},
	}

	for _, tt := range tests {
		got := mapBankingError(tt.err)
		if got != tt.expected {
			t.Errorf("mapBankingError(%v): want %d, got %d", tt.err, tt.expected, got)
		}
	}
}