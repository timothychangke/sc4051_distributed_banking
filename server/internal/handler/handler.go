package handler

import (
	"errors"
	"log"
	"net"
	"time"

	"bank-server/internal/banking"
	marshal "bank-server/internal/marshalling"
	"bank-server/internal/monitor"
	"bank-server/internal/semantics"
	"bank-server/pkg/models"
	protocol "bank-server/util" // directory is util/, package declares itself as protocol
)

// ─────────────────────────────────────────────────────────────────────
// Request handler — the central dispatch point for all banking operations.
//
// This is the function the semantics.Dispatcher calls for every incoming
// request. It sits between the raw UDP bytes and the banking service,
// translating in both directions:
//
//   [C++ client bytes] → TLV decode → banking logic → TLV encode → [reply bytes]
//
// The handler receives the FULL packet (including the 5-byte semantics
// header). It does NOT strip the header — the plan explicitly says the
// dispatcher passes the entire packet through. We skip past it ourselves
// to reach the TLV payload.
// ─────────────────────────────────────────────────────────────────────

// BuildHandler constructs the RequestHandler closure that gets wired
// into the dispatcher. It captures the banking service and monitor manager
// so individual handler functions can use them without global state.
func BuildHandler(svc banking.Service, mon *monitor.Manager) semantics.RequestHandler {
	return func(data []byte, addr *net.UDPAddr) []byte {
		// 1. Parse the semantics header to get ServiceID and RequestID.
		//    The dispatcher already validated this, but we need the requestID
		//    to echo it back in every reply.
		header, err := semantics.ParseHeader(data)
		if err != nil {
			log.Printf("[Handler] Failed to parse header from %s: %v", addr, err)
			// Can't even read the requestID, so use 0. The client will time
			// out on this — there's nothing better we can do.
			return marshal.BuildErrorReply(0, addr, protocol.StatusErrInternal)
		}

		// 2. Extract the TLV payload sitting after the 5-byte header.
		payload := data[semantics.HeaderSize:]

		// 3. Decode the TLV fields into a structured command.
		cmd, err := marshal.DecodeTLV(payload)
		if err != nil {
			log.Printf("[Handler] TLV decode error for request %d from %s: %v",
				header.RequestID, addr, err)
			return marshal.BuildErrorReply(header.RequestID, addr, protocol.StatusErrInternal)
		}

		// 4. Route to the appropriate service handler based on the Service field
		//    in the TLV payload. In the new protocol, the 18-byte header no
		//    longer contains the ServiceID; it's now a TLV field inside the
		//    command payload (Tag 0x01).
		if cmd.Service == nil {
			log.Printf("[Handler] Request %d from %s: missing ServiceID in TLV payload",
				header.RequestID, addr)
			return marshal.BuildErrorReply(header.RequestID, addr, protocol.StatusErrInternal)
		}

		serviceID := *cmd.Service
		switch serviceID {
		case protocol.ServiceOpenAccount:
			return handleOpen(cmd, header.RequestID, addr, svc, mon)
		case protocol.ServiceCloseAccount:
			return handleClose(cmd, header.RequestID, addr, svc, mon)
		case protocol.ServiceDeposit:
			return handleDeposit(cmd, header.RequestID, addr, svc, mon)
		case protocol.ServiceWithdraw:
			return handleWithdraw(cmd, header.RequestID, addr, svc, mon)
		case protocol.ServiceMonitor:
			return handleMonitor(cmd, header.RequestID, addr, mon)
		case protocol.ServiceGetBalance:
			return handleGetBalance(cmd, header.RequestID, addr, svc)
		case protocol.ServiceTransferFunds:
			return handleTransfer(cmd, header.RequestID, addr, svc, mon)
		default:
			log.Printf("[Handler] Unknown service ID %d from %s", serviceID, addr)
			return marshal.BuildErrorReply(header.RequestID, addr, protocol.StatusErrInternal)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────
// Individual service handlers.
//
// Each one follows the same pattern:
//   1. Validate that all required TLV fields are present
//   2. Convert wire types to what the banking service expects
//   3. Call the banking service
//   4. Map banking errors to wire status codes
//   5. Build the reply with TLV-encoded content
//   6. Notify the monitor if the operation mutated state
// ─────────────────────────────────────────────────────────────────────

func handleOpen(cmd *marshal.ParsedCommand, reqID uint32, addr *net.UDPAddr,
	svc banking.Service, mon *monitor.Manager) []byte {

	// Open requires: name, password, currency, initial balance
	if cmd.AccountOwnerName == nil || cmd.AccountPassword == nil ||
		cmd.Currency == nil || cmd.MonetaryValue == nil {
		log.Printf("[Handler] Open: missing required fields from %s", addr)
		return marshal.BuildErrorReply(reqID, addr, protocol.StatusErrInternal)
	}

	pw := marshal.PasswordStringToFixed(*cmd.AccountPassword)

	accNo := svc.OpenAccount(
		*cmd.AccountOwnerName,
		pw,
		models.Currency(*cmd.Currency),
		*cmd.MonetaryValue,
	)

	log.Printf("[Handler] Open: created account %d for '%s'", accNo, *cmd.AccountOwnerName)

	// Reply content: the newly generated account number
	content := marshal.EncodeTLVFields([]marshal.TLVField{
		marshal.TLVUint32(marshal.FieldAccountNumber, accNo),
	})

	// Notify monitoring clients about the new account
	mon.NotifyAll(models.AccountUpdate{
		ServiceID:     protocol.ServiceOpenAccount,
		AccountNumber: accNo,
		HolderName:    *cmd.AccountOwnerName,
		CurrencyType:  models.Currency(*cmd.Currency),
		NewBalance:    *cmd.MonetaryValue,
	})

	return marshal.BuildReply(reqID, addr, protocol.StatusSuccess, content)
}

func handleClose(cmd *marshal.ParsedCommand, reqID uint32, addr *net.UDPAddr,
	svc banking.Service, mon *monitor.Manager) []byte {

	// Close requires: name, account number, password
	if cmd.AccountOwnerName == nil || cmd.AccountNumber == nil || cmd.AccountPassword == nil {
		log.Printf("[Handler] Close: missing required fields from %s", addr)
		return marshal.BuildErrorReply(reqID, addr, protocol.StatusErrInternal)
	}

	pw := marshal.PasswordStringToFixed(*cmd.AccountPassword)

	err := svc.CloseAccount(*cmd.AccountOwnerName, *cmd.AccountNumber, pw)
	if err != nil {
		status := mapBankingError(err)
		log.Printf("[Handler] Close: account %d failed: %v", *cmd.AccountNumber, err)
		return marshal.BuildErrorReply(reqID, addr, status)
	}

	log.Printf("[Handler] Close: account %d closed", *cmd.AccountNumber)

	// Notify monitoring clients. Balance is 0 since the account is gone.
	mon.NotifyAll(models.AccountUpdate{
		ServiceID:     protocol.ServiceCloseAccount,
		AccountNumber: *cmd.AccountNumber,
		HolderName:    *cmd.AccountOwnerName,
		CurrencyType:  0, // account no longer exists, currency doesn't matter
		NewBalance:    0.0,
	})

	// No content body for close — status code alone signals success.
	return marshal.BuildReply(reqID, addr, protocol.StatusSuccess, nil)
}

func handleDeposit(cmd *marshal.ParsedCommand, reqID uint32, addr *net.UDPAddr,
	svc banking.Service, mon *monitor.Manager) []byte {

	// Deposit requires: name, account number, password, currency, amount
	if cmd.AccountOwnerName == nil || cmd.AccountNumber == nil ||
		cmd.AccountPassword == nil || cmd.Currency == nil || cmd.MonetaryValue == nil {
		log.Printf("[Handler] Deposit: missing required fields from %s", addr)
		return marshal.BuildErrorReply(reqID, addr, protocol.StatusErrInternal)
	}

	pw := marshal.PasswordStringToFixed(*cmd.AccountPassword)

	newBalance, err := svc.Deposit(
		*cmd.AccountOwnerName,
		*cmd.AccountNumber,
		pw,
		models.Currency(*cmd.Currency),
		*cmd.MonetaryValue,
	)
	if err != nil {
		status := mapBankingError(err)
		log.Printf("[Handler] Deposit: account %d failed: %v", *cmd.AccountNumber, err)
		return marshal.BuildErrorReply(reqID, addr, status)
	}

	log.Printf("[Handler] Deposit: account %d new balance %.2f", *cmd.AccountNumber, newBalance)

	// Reply content: the updated balance
	content := marshal.EncodeTLVFields([]marshal.TLVField{
		marshal.TLVFloat64(marshal.FieldMonetaryValue, newBalance),
	})

	mon.NotifyAll(models.AccountUpdate{
		ServiceID:     protocol.ServiceDeposit,
		AccountNumber: *cmd.AccountNumber,
		HolderName:    *cmd.AccountOwnerName,
		CurrencyType:  models.Currency(*cmd.Currency),
		NewBalance:    newBalance,
	})

	return marshal.BuildReply(reqID, addr, protocol.StatusSuccess, content)
}

func handleWithdraw(cmd *marshal.ParsedCommand, reqID uint32, addr *net.UDPAddr,
	svc banking.Service, mon *monitor.Manager) []byte {

	// Withdraw has the same required fields as Deposit
	if cmd.AccountOwnerName == nil || cmd.AccountNumber == nil ||
		cmd.AccountPassword == nil || cmd.Currency == nil || cmd.MonetaryValue == nil {
		log.Printf("[Handler] Withdraw: missing required fields from %s", addr)
		return marshal.BuildErrorReply(reqID, addr, protocol.StatusErrInternal)
	}

	pw := marshal.PasswordStringToFixed(*cmd.AccountPassword)

	newBalance, err := svc.Withdraw(
		*cmd.AccountOwnerName,
		*cmd.AccountNumber,
		pw,
		models.Currency(*cmd.Currency),
		*cmd.MonetaryValue,
	)
	if err != nil {
		status := mapBankingError(err)
		log.Printf("[Handler] Withdraw: account %d failed: %v", *cmd.AccountNumber, err)
		return marshal.BuildErrorReply(reqID, addr, status)
	}

	log.Printf("[Handler] Withdraw: account %d new balance %.2f", *cmd.AccountNumber, newBalance)

	content := marshal.EncodeTLVFields([]marshal.TLVField{
		marshal.TLVFloat64(marshal.FieldMonetaryValue, newBalance),
	})

	mon.NotifyAll(models.AccountUpdate{
		ServiceID:     protocol.ServiceWithdraw,
		AccountNumber: *cmd.AccountNumber,
		HolderName:    *cmd.AccountOwnerName,
		CurrencyType:  models.Currency(*cmd.Currency),
		NewBalance:    newBalance,
	})

	return marshal.BuildReply(reqID, addr, protocol.StatusSuccess, content)
}

func handleMonitor(cmd *marshal.ParsedCommand, reqID uint32, addr *net.UDPAddr,
	mon *monitor.Manager) []byte {

	// The new client sends the interval as a proper TLV field, not flat bytes.
	if cmd.MonitorTimeoutSeconds == nil {
		log.Printf("[Handler] Monitor: missing timeout from %s", addr)
		return marshal.BuildErrorReply(reqID, addr, protocol.StatusErrInternal)
	}

	interval := time.Duration(*cmd.MonitorTimeoutSeconds) * time.Second
	mon.Register(addr, interval)

	log.Printf("[Handler] Monitor: registered %s for %v", addr, interval)

	// Ack reply with no content — the C++ client uses receipt of this reply
	// as the signal to enter its blocking recvfrom loop for callbacks.
	return marshal.BuildReply(reqID, addr, protocol.StatusSuccess, nil)
}

func handleGetBalance(cmd *marshal.ParsedCommand, reqID uint32, addr *net.UDPAddr,
	svc banking.Service) []byte {

	// GetBalance (idempotent op): requires name, account number, password
	if cmd.AccountOwnerName == nil || cmd.AccountNumber == nil || cmd.AccountPassword == nil {
		log.Printf("[Handler] GetBalance: missing required fields from %s", addr)
		return marshal.BuildErrorReply(reqID, addr, protocol.StatusErrInternal)
	}

	pw := marshal.PasswordStringToFixed(*cmd.AccountPassword)

	balance, err := svc.CheckBalance(*cmd.AccountOwnerName, *cmd.AccountNumber, pw)
	if err != nil {
		status := mapBankingError(err)
		log.Printf("[Handler] GetBalance: account %d failed: %v", *cmd.AccountNumber, err)
		return marshal.BuildErrorReply(reqID, addr, status)
	}

	log.Printf("[Handler] GetBalance: account %d balance %.2f", *cmd.AccountNumber, balance)

	content := marshal.EncodeTLVFields([]marshal.TLVField{
		marshal.TLVFloat64(marshal.FieldMonetaryValue, balance),
	})

	// No monitor notification — GetBalance is read-only
	return marshal.BuildReply(reqID, addr, protocol.StatusSuccess, content)
}

func handleTransfer(cmd *marshal.ParsedCommand, reqID uint32, addr *net.UDPAddr,
	svc banking.Service, mon *monitor.Manager) []byte {

	// Transfer (non-idempotent op): requires sender name, sender account,
	// password, destination account, and amount
	if cmd.AccountOwnerName == nil || cmd.AccountNumber == nil ||
		cmd.AccountPassword == nil || cmd.TxAccountNumber == nil || cmd.MonetaryValue == nil {
		log.Printf("[Handler] Transfer: missing required fields from %s", addr)
		return marshal.BuildErrorReply(reqID, addr, protocol.StatusErrInternal)
	}

	pw := marshal.PasswordStringToFixed(*cmd.AccountPassword)

	senderNewBalance, receiverNewBalance, err := svc.Transfer(
		*cmd.AccountOwnerName,
		*cmd.AccountNumber,
		pw,
		*cmd.TxAccountNumber,
		models.Currency(*cmd.Currency),
		*cmd.MonetaryValue,
	)
	if err != nil {
		status := mapBankingError(err)
		log.Printf("[Handler] Transfer: %d -> %d failed: %v",
			*cmd.AccountNumber, *cmd.TxAccountNumber, err)
		return marshal.BuildErrorReply(reqID, addr, status)
	}

	log.Printf("[Handler] Transfer: %d -> %d amount %.2f, sender balance %.2f",
		*cmd.AccountNumber, *cmd.TxAccountNumber, *cmd.MonetaryValue, senderNewBalance)

	content := marshal.EncodeTLVFields([]marshal.TLVField{
		marshal.TLVFloat64(marshal.FieldMonetaryValue, senderNewBalance),
	})

	// Notify for the sender's side of the transfer. We know their full info
	// because they authenticated. The receiver's details would require an
	// extra store lookup that the banking.Service interface doesn't expose,
	// so we only notify for the sender.
	// TODO: If the demo requires both sides, add a GetAccountInfo method to
	//       the Service interface and send a second notification for toAccNo.
	mon.NotifyAll(models.AccountUpdate{
		ServiceID:     protocol.ServiceTransferFunds,
		AccountNumber: *cmd.AccountNumber,
		HolderName:    *cmd.AccountOwnerName,
		CurrencyType:  0, // sender's currency isn't in the Transfer params
		NewBalance:    senderNewBalance,
	})

	mon.NotifyAll(models.AccountUpdate{
		ServiceID:     protocol.ServiceTransferFunds,
		AccountNumber: *cmd.TxAccountNumber,
		HolderName:    *cmd.TxAccountOwnerName,
		CurrencyType:  0, // receivers's currency isn't in the Transfer params
		NewBalance:    receiverNewBalance,
	})

	return marshal.BuildReply(reqID, addr, protocol.StatusSuccess, content)
}

// ─────────────────────────────────────────────────────────────────────
// Error mapping
// ─────────────────────────────────────────────────────────────────────

// mapBankingError translates a banking layer error into the corresponding
// wire status code. The C++ client uses these codes to display the right
// error message, so they must stay in sync with protocol.go.
func mapBankingError(err error) uint8 {
	switch {
	case errors.Is(err, banking.ErrInvalidCredentials):
		return protocol.StatusErrInvalidCreds
	case errors.Is(err, banking.ErrAccountMismatch):
		return protocol.StatusErrAccMismatch
	case errors.Is(err, banking.ErrCurrencyMismatch):
		return protocol.StatusErrCurrMismatch
	case errors.Is(err, banking.ErrInsufficientFunds):
		return protocol.StatusErrInsuffFunds
	case errors.Is(err, banking.ErrTransferSameAccount):
		return protocol.StatusErrSameAccount
	case errors.Is(err, banking.ErrAccountNotFound):
		return protocol.StatusErrAccNotFound
	default:
		return protocol.StatusErrInternal
	}
}