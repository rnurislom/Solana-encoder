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
	conn *grpc.ClientConn
	cfg  *config.Config
}

func New(conn *grpc.ClientConn, cfg *config.Config) *Subscriber {
	return &Subscriber{conn: conn, cfg: cfg}
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
		printAccountUpdate(u.Account, update.Filters)
	case *pb.SubscribeUpdate_Transaction:
		printTransactionUpdate(u.Transaction, update.Filters)
	case *pb.SubscribeUpdate_Ping:
		// heartbeat — silent
	case *pb.SubscribeUpdate_Pong:
		// pong — silent
	}
}

func printAccountUpdate(acct *pb.SubscribeUpdateAccount, filters []string) {
	if acct == nil || acct.Account == nil {
		return
	}

	info := acct.Account
	pubkey := base58.Encode(info.Pubkey)
	owner := base58.Encode(info.Owner)
	solBalance := float64(info.Lamports) / 1e9

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  ACCOUNT UPDATE")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Pubkey:        %s\n", pubkey)
	fmt.Printf("  Owner:         %s\n", owner)
	fmt.Printf("  Balance:       %.9f SOL (%d lamports)\n", solBalance, info.Lamports)
	fmt.Printf("  Slot:          %d\n", acct.Slot)
	fmt.Printf("  Executable:    %v\n", info.Executable)
	fmt.Printf("  Data size:     %d bytes\n", len(info.Data))
	fmt.Printf("  Write version: %d\n", info.WriteVersion)
	if info.TxnSignature != nil {
		fmt.Printf("  Txn signature: %s\n", base58.Encode(info.TxnSignature))
	}
	fmt.Printf("  Filters:       %v\n", filters)
	fmt.Printf("  Timestamp:     %s\n", time.Now().Format(time.RFC3339Nano))
	fmt.Println()
}

func printTransactionUpdate(tx *pb.SubscribeUpdateTransaction, filters []string) {
	if tx == nil || tx.Transaction == nil {
		return
	}

	info := tx.Transaction
	signature := base58.Encode(info.Signature)

	var msg *pb.Message
	if info.Transaction != nil {
		msg = info.Transaction.Message
	}

	accounts := decoder.ResolveAccounts(msg, info.Meta)

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  TRANSACTION")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Signature:     %s\n", signature)
	fmt.Printf("  Slot:          %d\n", tx.Slot)

	if info.Meta != nil {
		fmt.Printf("  Fee:           %d lamports (%.9f SOL)\n", info.Meta.Fee, float64(info.Meta.Fee)/1e9)
		if info.Meta.Err != nil {
			fmt.Printf("  Status:        FAILED\n")
		} else {
			fmt.Printf("  Status:        SUCCESS\n")
		}
		if cu := info.Meta.ComputeUnitsConsumed; cu != nil {
			fmt.Printf("  Compute units: %d\n", *cu)
		}
	}

	printAccountTable(accounts)
	printInstructionTree(msg, info.Meta, accounts)
	printSOLChanges(info.Meta, accounts)
	printTokenChanges(info.Meta, accounts)
	printWalletSummary(info.Meta, accounts)
	printLogs(info.Meta)

	fmt.Printf("  Filters:       %v\n", filters)
	fmt.Printf("  Timestamp:     %s\n", time.Now().Format(time.RFC3339Nano))
	fmt.Println()
}

// --- Account table ---

func printAccountTable(accounts []decoder.AccountMeta) {
	if len(accounts) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("  ┌─ Accounts ────────────────────────────────────────")
	for _, a := range accounts {
		fmt.Printf("  │ [%2d] %s  %s\n", a.Index, a.Address, a.Role)
	}
	fmt.Println("  └────────────────────────────────────────────────────")
}

// --- Instruction tree ---

func printInstructionTree(msg *pb.Message, meta *pb.TransactionStatusMeta, accounts []decoder.AccountMeta) {
	if msg == nil {
		return
	}

	innerMap := make(map[uint32][]*pb.InnerInstruction)
	if meta != nil && !meta.InnerInstructionsNone {
		for _, ii := range meta.InnerInstructions {
			innerMap[ii.Index] = ii.Instructions
		}
	}

	fmt.Println()
	for i, ix := range msg.Instructions {
		progAddr := resolveAddress(accounts, int(ix.ProgramIdIndex))
		progLabel := formatProgram(progAddr)

		fmt.Printf("  ┌─ Instruction #%d: %s ─\n", i, progLabel)

		for _, accIdx := range ix.Accounts {
			idx := int(accIdx)
			if idx < len(accounts) {
				a := accounts[idx]
				fmt.Printf("  │   [%2d] %s  %s\n", a.Index, a.Address, a.Role)
			}
		}

		if inners, ok := innerMap[uint32(i)]; ok && len(inners) > 0 {
			fmt.Println("  │")
			fmt.Println("  │   Inner (CPI) calls:")
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

				fmt.Printf("  │   %s─ CPI #%d: %s%s\n", prefix, j, innerProgLabel, stackStr)

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
					fmt.Printf("  │   %s  Accounts: %s\n", indent, strings.Join(acctParts, " "))
				}
			}
		}

		fmt.Println("  └────────────────────────────────────────────────────")
	}
}

// --- SOL balance changes ---

func printSOLChanges(meta *pb.TransactionStatusMeta, accounts []decoder.AccountMeta) {
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

	fee := int64(meta.Fee)

	fmt.Println()
	fmt.Println("  ┌─ SOL Balance Changes ─────────────────────────────")
	fmt.Printf("  │ Transaction fee: %d lamports (%.9f SOL)\n", fee, float64(fee)/1e9)
	fmt.Println("  │")
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
			fmt.Printf("  │ %s: %s SOL (total) = %s SOL (transfer) + %s SOL (fee)\n",
				addr,
				formatSOL(diff),
				formatSOL(diffExFee),
				formatSOL(-fee))
		} else {
			fmt.Printf("  │ %s: %s SOL\n", addr, formatSOL(diff))
		}
	}
	fmt.Println("  └────────────────────────────────────────────────────")
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

func printTokenChanges(meta *pb.TransactionStatusMeta, accounts []decoder.AccountMeta) {
	if meta == nil {
		return
	}

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

	fmt.Println()
	fmt.Println("  ┌─ Token Balance Changes ────────────────────────────")
	for _, mint := range mintOrder {
		mintChanges := byMint[mint]
		fmt.Printf("  │ Mint: %s\n", mint)
		for _, c := range mintChanges {
			preF, _ := strconv.ParseFloat(c.preAmount, 64)
			postF, _ := strconv.ParseFloat(c.postAmount, 64)
			diffVal := postF - preF
			ownerLabel := shortAddr(c.owner)
			fmt.Printf("  │   Owner %s: %s -> %s (%s)\n",
				ownerLabel, c.preAmount, c.postAmount, formatSignedFloat(diffVal))
		}
	}
	fmt.Println("  └────────────────────────────────────────────────────")
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

func printWalletSummary(meta *pb.TransactionStatusMeta, accounts []decoder.AccountMeta) {
	if meta == nil {
		return
	}

	fee := int64(meta.Fee)

	fmt.Println()
	fmt.Println("  ┌─ Net Changes Summary ─────────────────────────────")

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
		fmt.Printf("  │ %s SOL: %s (fee: %s)\n", addr, formatSOL(diffExFee), formatSOL(-fee))
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
		fmt.Printf("  │ %s: %s — %s\n", mintLabel, formatSignedFloat(diffAmt), action)
	}

	fmt.Println("  └────────────────────────────────────────────────────")
}

// --- Log messages ---

func printLogs(meta *pb.TransactionStatusMeta) {
	if meta == nil || meta.LogMessagesNone || len(meta.LogMessages) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("  ┌─ Program Logs ─────────────────────────────────────")
	for _, msg := range meta.LogMessages {
		fmt.Printf("  │ %s\n", msg)
	}
	fmt.Println("  └────────────────────────────────────────────────────")
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
