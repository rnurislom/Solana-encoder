package decoder

var knownPrograms = map[string]string{
	"11111111111111111111111111111111":                        "System Program",
	"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA":           "Token Program",
	"TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb":           "Token-2022",
	"ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL":          "Associated Token Program",
	"ComputeBudget111111111111111111111111111111":              "Compute Budget",
	"JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4":           "Jupiter v6",
	"JUP4Fb2cqiRUcaTHdrPC8h2gNsA2ETXiPDD33WcGuJB":           "Jupiter v4",
	"675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8":          "Raydium AMM",
	"CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK":          "Raydium CLMM",
	"CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C":          "Raydium CPMM",
	"routeUGWgWzqBWFcrCfv8tritsqukccJPu3q5GPP3xS":           "Raydium Route",
	"whirLbMiicVdio4qvUfM5KAg6Ct8VwpYzGff3uctyCc":           "Orca Whirlpool",
	"9W959DqEETiGZocYWCQPaJ6sBmUzgfxXfqGeTEdp3aQP":          "Orca Swap v2",
	"PhoeNiXZ8ByJGLkxNfZRnkUfjvmuYqLR89jjFHGqdXY":           "Phoenix",
	"LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo":           "Meteora DLMM",
	"Eo7WjKq67rjJQSZxS6z3YkapzY3eMj6Xy8X5EQVn5UaB":          "Meteora Pools",
	"srmqPvymJeFKQ4zGQed1GFppgkRHL9kaELCbyksJtPX":            "Serum/Openbook v1",
	"opnb2LAfJYbRMAHHvqjCwQxanZn7ReEHp1k81EQMQvR":            "Openbook v2",
	"MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr":           "Memo v2",
	"Memo1UhkJBfCR4NPCKxSTRep82mnjCPzaKBCGadBfHc3":           "Memo v1",
	"Vote111111111111111111111111111111111111111111":            "Vote Program",
	"Stake11111111111111111111111111111111111111":               "Stake Program",
	"Config1111111111111111111111111111111111111111":            "Config Program",
	"AddressLookupTab1e1111111111111111111111111":              "Address Lookup Table",
	"BPFLoaderUpgradeab1e11111111111111111111111":              "BPF Upgradeable Loader",
	"namesLPneVptA9Z5rqUDD9tMTWEJwofgaYwp8cawRkX":            "Name Service",
	"metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s":            "Metaplex Token Metadata",
	"p1exdMJcjVao65QdewkaZRUnU6VPSXhus9n2GzWfh98":            "Metaplex Auction",
	"6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P":            "Pump.fun",
	"pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA":            "Pump.fun AMM",
}

func ProgramName(address string) string {
	if name, ok := knownPrograms[address]; ok {
		return name
	}
	return ""
}
