package subscriber

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"wallet-monitor/internal/config"
	"wallet-monitor/internal/decoder"

	"github.com/mr-tron/base58"
	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type Subscriber struct {
	conn   *grpc.ClientConn
	cfg    *config.Config
	output io.Writer
}

func New(conn *grpc.ClientConn, cfg *config.Config, output io.Writer) *Subscriber {
	if output == nil {
		output = io.Discard
	}
	return &Subscriber{conn: conn, cfg: cfg, output: output}
}

func (s *Subscriber) Run(ctx context.Context) error {
	client := pb.NewGeyserClient(s.conn)

	if s.cfg.Token != "" {
		md := metadata.New(map[string]string{"x-token": s.cfg.Token})
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	stream, err := client.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("subscribe failed: %w", err)
	}

	req := s.buildRequest()
	if err := stream.Send(req); err != nil {
		return fmt.Errorf("send subscription request failed: %w", err)
	}

	log.Printf("Monitoring wallets: %s", strings.Join(s.cfg.Wallets, ", "))
	log.Println("Waiting for updates...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Shutting down gracefully...")
			return ctx.Err()
		default:
		}

		resp, err := stream.Recv()
		if err == io.EOF {
			log.Println("Stream closed by server")
			return nil
		}
		if err != nil {
			return fmt.Errorf("stream receive error: %w", err)
		}

		s.handleUpdate(resp)
	}
}

func (s *Subscriber) buildRequest() *pb.SubscribeRequest {
	commitment := pb.CommitmentLevel_CONFIRMED

	return &pb.SubscribeRequest{
		Accounts: map[string]*pb.SubscribeRequestFilterAccounts{
			"wallet_accounts": {
				Account: s.cfg.Wallets,
			},
		},
		Transactions: map[string]*pb.SubscribeRequestFilterTransactions{
			"wallet_txns": {
				AccountInclude: s.cfg.Wallets,
				Vote:           boolPtr(false),
			},
		},
		Commitment: &commitment,
	}
}

func (s *Subscriber) handleUpdate(update *pb.SubscribeUpdate) {
	switch u := update.UpdateOneof.(type) {
	case *pb.SubscribeUpdate_Account:
		s.printAccountUpdate(u.Account, update.Filters)
	case *pb.SubscribeUpdate_Transaction:
		s.printTransactionUpdate(u.Transaction, update.Filters)
	case *pb.SubscribeUpdate_Ping:
		// heartbeat — silent
	case *pb.SubscribeUpdate_Pong:
		// pong — silent
	}
}

func (s *Subscriber) printAccountUpdate(acct *pb.SubscribeUpdateAccount, filters []string) {
	if acct == nil || acct.Account == nil {
		return
	}

	info := acct.Account
	pubkey := base58.Encode(info.Pubkey)
	owner := base58.Encode(info.Owner)
	solBalance := float64(info.Lamports) / 1e9

	w := s.output
	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(w, "  ACCOUNT UPDATE")
	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(w, "  Pubkey:        %s\n", pubkey)
	fmt.Fprintf(w, "  Owner:         %s\n", owner)
	fmt.Fprintf(w, "  Balance:       %.9f SOL (%d lamports)\n", solBalance, info.Lamports)
	fmt.Fprintf(w, "  Slot:          %d\n", acct.Slot)
	fmt.Fprintf(w, "  Executable:    %v\n", info.Executable)
	fmt.Fprintf(w, "  Data size:     %d bytes\n", len(info.Data))
	fmt.Fprintf(w, "  Write version: %d\n", info.WriteVersion)
	if info.TxnSignature != nil {
		fmt.Fprintf(w, "  Txn signature: %s\n", base58.Encode(info.TxnSignature))
	}
	fmt.Fprintf(w, "  Filters:       %v\n", filters)
	fmt.Fprintf(w, "  Timestamp:     %s\n", time.Now().Format(time.RFC3339Nano))
	fmt.Fprintln(w)
}

func (s *Subscriber) printTransactionUpdate(tx *pb.SubscribeUpdateTransaction, filters []string) {
	if tx == nil || tx.Transaction == nil {
		return
	}

	w := s.output
	info := tx.Transaction
	signature := base58.Encode(info.Signature)

	var msg *pb.Message
	if info.Transaction != nil {
		msg = info.Transaction.Message
	}

	accounts := decoder.ResolveAccounts(msg, info.Meta)

	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(w, "  TRANSACTION")
	fmt.Fprintln(w, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(w, "  Signature:     %s\n", signature)
	fmt.Fprintf(w, "  Slot:          %d\n", tx.Slot)

	if info.Meta != nil {
		fmt.Fprintf(w, "  Fee:           %d lamports (%.9f SOL)\n", info.Meta.Fee, float64(info.Meta.Fee)/1e9)
		if info.Meta.Err != nil {
			fmt.Fprintf(w, "  Status:        FAILED\n")
		} else {
			fmt.Fprintf(w, "  Status:        SUCCESS\n")
		}
		if cu := info.Meta.ComputeUnitsConsumed; cu != nil {
			fmt.Fprintf(w, "  Compute units: %d\n", *cu)
		}
	}

	s.printAccountTable(accounts)
	s.printInstructionTree(msg, info.Meta, accounts)
	s.printSOLChanges(info.Meta, accounts)
	s.printTokenChanges(info.Meta, accounts)
	s.printWalletSummary(info.Meta, accounts)
	s.printLogs(info.Meta)

	fmt.Fprintf(w, "  Filters:       %v\n", filters)
	fmt.Fprintf(w, "  Timestamp:     %s\n", time.Now().Format(time.RFC3339Nano))
	fmt.Fprintln(w)
}

// --- Account table ---

func (s *Subscriber) printAccountTable(accounts []decoder.AccountMeta) {
	if len(accounts) == 0 {
		return
	}
	w := s.output
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  ┌─ Accounts ────────────────────────────────────────")
	for _, a := range accounts {
		fmt.Fprintf(w, "  │ [%2d] %s  %s\n", a.Index, a.Address, a.Role)
	}
	fmt.Fprintln(w, "  └────────────────────────────────────────────────────")
}

// --- Instruction tree ---

func (s *Subscriber) printInstructionTree(msg *pb.Message, meta *pb.TransactionStatusMeta, accounts []decoder.AccountMeta) {
	if msg == nil {
		return
	}
	w := s.output
	innerMap := make(map[uint32][]*pb.InnerInstruction)
	if meta != nil && !meta.InnerInstructionsNone {
		for _, ii := range meta.InnerInstructions {
			innerMap[ii.Index] = ii.Instructions
		}
	}

	fmt.Fprintln(w)
	for i, ix := range msg.Instructions {
		progAddr := resolveAddress(accounts, int(ix.ProgramIdIndex))
		progLabel := formatProgram(progAddr)

		fmt.Fprintf(w, "  ┌─ Instruction #%d: %s ─\n", i, progLabel)

		for _, accIdx := range ix.Accounts {
			idx := int(accIdx)
			if idx < len(accounts) {
				a := accounts[idx]
				fmt.Fprintf(w, "  │   [%2d] %s  %s\n", a.Index, a.Address, a.Role)
			}
		}

		if inners, ok := innerMap[uint32(i)]; ok && len(inners) > 0 {
			fmt.Fprintln(w, "  │")
			fmt.Fprintln(w, "  │   Inner (CPI) calls:")
			for j, inner := range inners {
				innerProgAddr := resolveAddress(accounts, int(inner.ProgramIdIndex))
				innerProgLabel := formatProgram(innerProgAddr)

				prefix := "├"
				if j == len(inners)-1 {
					prefix = "└"
				}

				stackStr := ""
				if inner.StackHeight != nil {
					stackStr = fmt.Sprintf(" [depth=%d]", *inner.StackHeight)
				}

				fmt.Fprintf(w, "  │   %s─ CPI #%d: %s%s\n", prefix, j, innerProgLabel, stackStr)

				var acctParts []string
				for _, accIdx := range inner.Accounts {
					idx := int(accIdx)
					if idx < len(accounts) {
						acctParts = append(acctParts, fmt.Sprintf("[%d]%s", idx, shortAddr(accounts[idx].Address)))
					}
				}
				if len(acctParts) > 0 {
					indent := "│"
					if j == len(inners)-1 {
						indent = " "
					}
					fmt.Fprintf(w, "  │   %s  Accounts: %s\n", indent, strings.Join(acctParts, " "))
				}
			}
		}

		fmt.Fprintln(w, "  └────────────────────────────────────────────────────")
	}
}

// --- SOL balance changes ---

func (s *Subscriber) printSOLChanges(meta *pb.TransactionStatusMeta, accounts []decoder.AccountMeta) {
	if meta == nil || len(meta.PreBalances) == 0 || len(meta.PostBalances) == 0 {
		return
	}

	hasChanges := false
	for i := 0; i < len(meta.PreBalances) && i < len(meta.PostBalances); i++ {
		if meta.PreBalances[i] != meta.PostBalances[i] {
			hasChanges = true
			break
		}
	}
	if !hasChanges {
		return
	}

	w := s.output
	fee := int64(meta.Fee)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "  ┌─ SOL Balance Changes ─────────────────────────────")
	fmt.Fprintf(w, "  │ Transaction fee: %d lamports (%.9f SOL)\n", fee, float64(fee)/1e9)
	fmt.Fprintln(w, "  │")
	for i := 0; i < len(meta.PreBalances) && i < len(meta.PostBalances); i++ {
		pre := meta.PreBalances[i]
		post := meta.PostBalances[i]
		if pre == post {
			continue
		}
		diff := int64(post) - int64(pre)

		isFeePayer := i < len(accounts) && accounts[i].Role.IsFeePayer()
		diffExFee := diff
		if isFeePayer {
			diffExFee = diff + fee
		}

		addr := fmt.Sprintf("[%d]", i)
		if i < len(accounts) {
			addr = shortAddr(accounts[i].Address)
		}

		if isFeePayer {
			fmt.Fprintf(w, "  │ %s: %s SOL (total) = %s SOL (transfer) + %s SOL (fee)\n",
				addr,
				formatSOL(diff),
				formatSOL(diffExFee),
				formatSOL(-fee))
		} else {
			fmt.Fprintf(w, "  │ %s: %s SOL\n", addr, formatSOL(diff))
		}
	}
	fmt.Fprintln(w, "  └────────────────────────────────────────────────────")
}

// --- Token balance changes ---

type tokenChange struct {
	accountIndex uint32
	owner        string
	mint         string
	preAmount    string
	postAmount   string
	preUi        float64
	postUi       float64
	decimals     uint32
}

func (s *Subscriber) printTokenChanges(meta *pb.TransactionStatusMeta, accounts []decoder.AccountMeta) {
	if meta == nil {
		return
	}
	w := s.output
	changes := computeTokenChanges(meta)
	if len(changes) == 0 {
		return
	}

	byMint := make(map[string][]tokenChange)
	var mintOrder []string
	for _, c := range changes {
		if _, exists := byMint[c.mint]; !exists {
			mintOrder = append(mintOrder, c.mint)
		}
		byMint[c.mint] = append(byMint[c.mint], c)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "  ┌─ Token Balance Changes ────────────────────────────")
	for _, mint := range mintOrder {
		mintChanges := byMint[mint]
		fmt.Fprintf(w, "  │ Mint: %s\n", mint)
		for _, c := range mintChanges {
			preF, _ := strconv.ParseFloat(c.preAmount, 64)
			postF, _ := strconv.ParseFloat(c.postAmount, 64)
			diffVal := postF - preF
			ownerLabel := shortAddr(c.owner)
			fmt.Fprintf(w, "  │   Owner %s: %s -> %s (%s)\n",
				ownerLabel, c.preAmount, c.postAmount, formatSignedFloat(diffVal))
		}
	}
	fmt.Fprintln(w, "  └────────────────────────────────────────────────────")
}

func computeTokenChanges(meta *pb.TransactionStatusMeta) []tokenChange {
	type balKey struct {
		accountIndex uint32
		mint         string
	}

	preMap := make(map[balKey]*pb.TokenBalance)
	for _, tb := range meta.PreTokenBalances {
		preMap[balKey{tb.AccountIndex, tb.Mint}] = tb
	}

	postMap := make(map[balKey]*pb.TokenBalance)
	for _, tb := range meta.PostTokenBalances {
		postMap[balKey{tb.AccountIndex, tb.Mint}] = tb
	}

	seen := make(map[balKey]bool)
	var allKeys []balKey
	for k := range preMap {
		allKeys = append(allKeys, k)
		seen[k] = true
	}
	for k := range postMap {
		if !seen[k] {
			allKeys = append(allKeys, k)
		}
	}

	var result []tokenChange
	for _, k := range allKeys {
		pre := preMap[k]
		post := postMap[k]

		var preUi, postUi float64
		var preAmt, postAmt, owner string
		var decimals uint32

		if pre != nil && pre.UiTokenAmount != nil {
			preUi = pre.UiTokenAmount.UiAmount
			preAmt = pre.UiTokenAmount.UiAmountString
			decimals = pre.UiTokenAmount.Decimals
			owner = pre.Owner
		} else {
			preAmt = "0"
		}
		if post != nil && post.UiTokenAmount != nil {
			postUi = post.UiTokenAmount.UiAmount
			postAmt = post.UiTokenAmount.UiAmountString
			decimals = post.UiTokenAmount.Decimals
			if owner == "" {
				owner = post.Owner
			}
		} else {
			postAmt = "0"
		}

		if preAmt == postAmt {
			continue
		}

		result = append(result, tokenChange{
			accountIndex: k.accountIndex,
			owner:        owner,
			mint:         k.mint,
			preAmount:    preAmt,
			postAmount:   postAmt,
			preUi:        preUi,
			postUi:       postUi,
			decimals:     decimals,
		})
	}
	return result
}

// --- Wallet-centric summary (net SOL + token changes per program) ---

func (s *Subscriber) printWalletSummary(meta *pb.TransactionStatusMeta, accounts []decoder.AccountMeta) {
	if meta == nil {
		return
	}
	w := s.output
	fee := int64(meta.Fee)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "  ┌─ Net Changes Summary ─────────────────────────────")

	for i := 0; i < len(meta.PreBalances) && i < len(meta.PostBalances); i++ {
		if i >= len(accounts) || !accounts[i].Role.IsSigner() {
			continue
		}
		pre := meta.PreBalances[i]
		post := meta.PostBalances[i]
		if pre == post {
			continue
		}
		diff := int64(post) - int64(pre)
		diffExFee := diff
		if accounts[i].Role.IsFeePayer() {
			diffExFee = diff + fee
		}

		addr := shortAddr(accounts[i].Address)
		fmt.Fprintf(w, "  │ %s SOL: %s (fee: %s)\n", addr, formatSOL(diffExFee), formatSOL(-fee))
	}

	// Net tokens owned by signers
	changes := computeTokenChanges(meta)
	signerAddrs := make(map[string]bool)
	for _, a := range accounts {
		if a.Role.IsSigner() {
			signerAddrs[a.Address] = true
		}
	}

	for _, c := range changes {
		if !signerAddrs[c.owner] {
			continue
		}

		preF, _ := strconv.ParseFloat(c.preAmount, 64)
		postF, _ := strconv.ParseFloat(c.postAmount, 64)
		diffAmt := postF - preF
		if math.Abs(diffAmt) < 1e-12 {
			continue
		}

		action := "RECEIVED"
		if diffAmt < 0 {
			action = "SENT"
		}
		mintLabel := shortAddr(c.mint)
		name := decoder.ProgramName(c.mint)
		if name != "" {
			mintLabel = name
		}
		fmt.Fprintf(w, "  │ %s: %s — %s\n", mintLabel, formatSignedFloat(diffAmt), action)
	}

	fmt.Fprintln(w, "  └────────────────────────────────────────────────────")
}

// --- Log messages ---

func (s *Subscriber) printLogs(meta *pb.TransactionStatusMeta) {
	if meta == nil || meta.LogMessagesNone || len(meta.LogMessages) == 0 {
		return
	}
	w := s.output
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  ┌─ Program Logs ─────────────────────────────────────")
	for _, msg := range meta.LogMessages {
		fmt.Fprintf(w, "  │ %s\n", msg)
	}
	fmt.Fprintln(w, "  └────────────────────────────────────────────────────")
}

// --- Helpers ---

func resolveAddress(accounts []decoder.AccountMeta, idx int) string {
	if idx < len(accounts) {
		return accounts[idx].Address
	}
	return fmt.Sprintf("unknown[%d]", idx)
}

func formatProgram(addr string) string {
	name := decoder.ProgramName(addr)
	if name != "" {
		return fmt.Sprintf("%s (%s)", name, shortAddr(addr))
	}
	return addr
}

func shortAddr(addr string) string {
	if len(addr) <= 12 {
		return addr
	}
	return addr[:4] + "..." + addr[len(addr)-4:]
}

func formatSOL(lamports int64) string {
	sign := "+"
	if lamports < 0 {
		sign = "-"
	}
	abs := lamports
	if abs < 0 {
		abs = -abs
	}
	return fmt.Sprintf("%s%.9f", sign, float64(abs)/1e9)
}

func formatSignedFloat(f float64) string {
	s := strconv.FormatFloat(math.Abs(f), 'f', -1, 64)
	if f < 0 {
		return "-" + s
	}
	return "+" + s
}

func boolPtr(b bool) *bool {
	return &b
}
