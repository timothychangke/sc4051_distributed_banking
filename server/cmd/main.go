package main

import (
	"fmt"
)

func main() {
	fmt.Println("Distributed Banking Server starting...")

	// ──────────────────────────────────────────────────────
	// 1. UDP SOCKET SETUP
	// ──────────────────────────────────────────────────────
	// addr := &net.UDPAddr{IP: net.IPv4zero, Port: 2222}
	// conn, err := net.ListenUDP("udp", addr)
	// if err != nil { log.Fatalf("Failed to bind: %v", err) }
	// defer conn.Close()

	// ──────────────────────────────────────────────────────
	// 2. DEPENDENCY WIRING
	// ──────────────────────────────────────────────────────
	// memStore := store.NewMemoryStore()
	// bankingSvc := banking.NewService(memStore)

	// ──────────────────────────────────────────────────────
	// 3. CALLBACK MONITOR SETUP (zhixuan's integration point)
	//
	// The monitor.Manager needs:
	//   - conn: the server's UDP socket (reused for sending callbacks)
	//   - marshalCallbackUpdate: a function with signature:
	//       func(models.AccountUpdate) ([]byte, error)
	//     This must produce a callback packet using MsgTypeCallback (0x02).
	//     See protocol.go for the message type constants.
	//   - sweepInterval: how often to reap expired subscribers (e.g., 5s)
	//
	// NOTE FOR ZHIXUAN:
	//   You need to implement marshalCallbackUpdate() using the Encoder.
	//   The callback packet layout should be:
	//     [1 byte ] MsgTypeCallback (0x02)
	//     [1 byte ] ServiceID (which op triggered this: 1-4, 7)
	//     [4 bytes] AccountNumber (big-endian uint32)
	//     [4+N bytes] HolderName (length-prefixed string)
	//     [1 byte ] CurrencyType (uint8)
	//     [8 bytes] NewBalance (big-endian IEEE 754 float64)
	//   Coordinate with the C++ client teammate — they need to deserialize
	//   this exact layout in their monitor_server_updates() loop.
	// ──────────────────────────────────────────────────────
	// monitorMgr := monitor.NewManager(conn, marshalCallbackUpdate, 5*time.Second)
	// defer monitorMgr.Stop()

	// ══════════════════════════════════════════════════════
	// ██  CRITICAL CONTRACT: REQUEST/REPLY WIRE FORMAT    ██
	// ══════════════════════════════════════════════════════
	//
	// The invocation semantics layer (internal/semantics/) reads bytes 0–4
	// of EVERY incoming request to extract the ServiceID and RequestID.
	// If the client doesn't put these in the right place, duplicate
	// filtering breaks silently and at-most-once won't work.
	//
	// ┌─────────────────────────────────────────────────────┐
	// │                  REQUEST PACKET                     │
	// ├──────────┬──────────────┬───────────────────────────┤
	// │ Byte 0   │ Bytes 1–4   │ Bytes 5+                  │
	// │ ServiceID│ RequestID   │ Payload (varies by service)│
	// │ (uint8)  │ (uint32 BE) │ (zhixuan owns this)       │
	// └──────────┴──────────────┴───────────────────────────┘
	//
	// ZHIXUAN — YOUR HANDLER RECEIVES THE FULL PACKET (bytes 0 through N).
	// The dispatcher does NOT strip the header before calling you. So when
	// you unmarshal inside the handler, skip the first 5 bytes to get to
	// your service-specific payload. Example:
	//
	//   func handleRequest(data []byte, addr *net.UDPAddr) []byte {
	//       serviceID := data[0]
	//       // requestID := binary.BigEndian.Uint32(data[1:5]) // if you need it
	//       payload := data[semantics.HeaderSize:]  // your fields start here
	//       ...
	//   }
	//
	// REQUEST ID RULES (these matter for at-most-once to work):
	//   - The C++ client must set a unique, monotonically increasing
	//     RequestID (uint32, big-endian) in bytes 1–4 of every request.
	//   - Retransmissions of the same request MUST reuse the same RequestID.
	//   - A new logical request MUST use a new RequestID.
	//   - If the client gets this wrong, duplicates won't be detected.
	//
	// ┌─────────────────────────────────────────────────────┐
	// │                   REPLY PACKET                      │
	// ├──────────┬──────────────┬──────────┬────────────────┤
	// │ Byte 0   │ Bytes 1–4   │ Byte 5   │ Bytes 6+       │
	// │ MsgType  │ RequestID   │ Status   │ Response body   │
	// │ (0x01)   │ (uint32 BE) │ (uint8)  │ (varies)       │
	// └──────────┴──────────────┴──────────┴────────────────┘
	//
	// ZHIXUAN — YOUR HANDLER MUST RETURN THE COMPLETE REPLY (bytes 0+).
	// The dispatcher caches whatever []byte you return, verbatim. It does
	// NOT prepend any header for you. So your handler is responsible for
	// building the full reply including MsgType, RequestID, and StatusCode.
	//
	// WHY THE REQUESTID MUST BE IN THE REPLY:
	//   The C++ client needs to match replies to outstanding requests.
	//   Without it, a cached reply from a retransmission looks identical
	//   to a reply for a completely different request. Echo it back.
	//
	// HANDLER FUNCTION SIGNATURE (defined in internal/semantics/dispatcher.go):
	//
	//   type RequestHandler func(data []byte, addr *net.UDPAddr) []byte
	//
	// Your handler must ALWAYS return a non-nil []byte. Even for errors,
	// return a reply with the appropriate StatusCode (see protocol.go).
	// If you return nil, the dispatcher will send nothing and the client
	// will time out and retransmit forever.
	//
	// ══════════════════════════════════════════════════════

	// ──────────────────────────────────────────────────────
	// 4. HANDLER + INVOCATION SEMANTICS
	//
	// NOTE FOR ZHIXUAN:
	//   When building the handler, the request dispatcher needs to call
	//   monitorMgr.NotifyAll(update) after every SUCCESSFUL mutation
	//   (Open, Close, Deposit, Withdraw, Transfer).
	//
	//   Do NOT notify on:
	//     - Failed operations (wrong password, insufficient funds, etc.)
	//     - Read-only operations (GetBalance)
	//     - Monitor registration itself
	//
	//   The AccountUpdate struct lives in pkg/models/update.go.
	//   Example after a successful deposit:
	//
	//     monitorMgr.NotifyAll(models.AccountUpdate{
	//         ServiceID:     protocol.ServiceDeposit,
	//         AccountNumber: accNo,
	//         HolderName:    name,
	//         CurrencyType:  currency,
	//         NewBalance:    newBalance,
	//     })
	//
	//   For Transfer, consider notifying twice (once per affected account)
	//   so monitoring clients see both sides of the transfer.
	// ──────────────────────────────────────────────────────
	// mode, _ := semantics.ParseMode(os.Args[1])     // "at-least-once" or "at-most-once"
	// lossRate := parseLossRate(os.Args)               // e.g., 0.3 for 30% simulated loss
	// handler := buildYourHandler(bankingSvc, monitorMgr) // zhixuan's code
	// dispatcher := semantics.NewDispatcher(mode, handler)
	// replyLoss := semantics.NewLossSimulator(lossRate, 42)

	// ──────────────────────────────────────────────────────
	// 5. MAIN REQUEST LOOP
	//
	// NOTE FOR ZHIXUAN:
	//   Monitor registration (ServiceID=5) needs special handling:
	//     1. Unmarshal the interval (uint32 seconds) from the request
	//     2. Call monitorMgr.Register(clientAddr, interval)
	//     3. Send back an ack reply so the C++ client enters its
	//        blocking recvfrom loop
	//
	//   The client's addr comes from ReadFromUDP — we don't need
	//   the client to tell us their IP/port in the payload.
	//
	//   Callback packets bypass the normal reply path entirely.
	//   They're sent directly by monitorMgr.NotifyAll() through
	//   the shared conn, so the main loop doesn't touch them.
	//
	// NOTE ON LOSS SIMULATION:
	//   We simulate loss at two points:
	//     - requestLoss.ShouldDrop() BEFORE dispatching = request never processed
	//     - replyLoss.ShouldDrop() AFTER dispatching = reply never sent
	//   For the experiment, you probably only need reply loss to demonstrate
	//   the retransmission scenario. But having both gives flexibility.
	//   The seed is fixed (42) so experiments are reproducible across runs.
	// ──────────────────────────────────────────────────────
	// buf := make([]byte, 4096)
	// for {
	// 		n, clientAddr, err := conn.ReadFromUDP(buf)
	//		if err != nil {
	//			log.Printf("[Server] ReadFromUDP error: %v", err)
	//			continue
	//		}
	//		log.Printf("[Server] Received %d bytes from %s", n, clientAddr)
	//
	// 		reply, err := dispatcher.Dispatch(buf[:n], clientAddr)
	//		if err != nil {
	//			log.Printf("[Server] Dispatch error: %v", err)
	//			continue
	//		}
	//
	// 		if replyLoss.ShouldDrop() {
	//			log.Printf("[Server] Simulated reply loss for request from %s", clientAddr)
	//			continue
	//		}
	//
	//		if _, err := conn.WriteToUDP(reply, clientAddr); err != nil {
	//			log.Printf("[Server] WriteToUDP error: %v", err)
	//		}
	// }

	fmt.Println("Server shut down.")
}