package decoder

import (
	"github.com/mr-tron/base58"
	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
)

type AccountRole uint8

const (
	RoleReadonly AccountRole = 0
	RoleWritable AccountRole = 1 << 0
	RoleSigner   AccountRole = 1 << 1
	RoleFeePayer AccountRole = 1 << 2
)

func (r AccountRole) IsWritable() bool { return r&RoleWritable != 0 }
func (r AccountRole) IsSigner() bool   { return r&RoleSigner != 0 }
func (r AccountRole) IsFeePayer() bool { return r&RoleFeePayer != 0 }

func (r AccountRole) String() string {
	s := ""
	if r.IsSigner() {
		s += "SIGNER "
	}
	if r.IsWritable() {
		s += "WRITABLE"
	} else {
		s += "READONLY"
	}
	if r.IsFeePayer() {
		s += " (fee payer)"
	}
	return s
}

type AccountMeta struct {
	Index   int
	Address string
	Role    AccountRole
}

// ResolveAccounts builds the full ordered account list with roles
// derived from the message header and loaded addresses from versioned txs.
func ResolveAccounts(msg *pb.Message, meta *pb.TransactionStatusMeta) []AccountMeta {
	if msg == nil {
		return nil
	}

	var allKeys [][]byte
	allKeys = append(allKeys, msg.AccountKeys...)
	if meta != nil {
		allKeys = append(allKeys, meta.LoadedWritableAddresses...)
		allKeys = append(allKeys, meta.LoadedReadonlyAddresses...)
	}

	header := msg.Header
	if header == nil {
		result := make([]AccountMeta, len(allKeys))
		for i, k := range allKeys {
			result[i] = AccountMeta{Index: i, Address: base58.Encode(k)}
		}
		return result
	}

	numStatic := uint32(len(msg.AccountKeys))
	numSig := header.NumRequiredSignatures
	numReadonlySigned := header.NumReadonlySignedAccounts
	numReadonlyUnsigned := header.NumReadonlyUnsignedAccounts

	numWritableSigned := numSig - numReadonlySigned
	numWritableUnsigned := numStatic - numSig - numReadonlyUnsigned

	result := make([]AccountMeta, len(allKeys))
	for i, k := range allKeys {
		idx := uint32(i)
		addr := base58.Encode(k)
		var role AccountRole

		switch {
		case idx < numWritableSigned:
			role = RoleSigner | RoleWritable
			if idx == 0 {
				role |= RoleFeePayer
			}
		case idx < numSig:
			role = RoleSigner
		case idx < numSig+numWritableUnsigned:
			role = RoleWritable
		case idx < numStatic:
			role = RoleReadonly
		default:
			loadedWritableEnd := numStatic + uint32(len(meta.LoadedWritableAddresses))
			if idx < loadedWritableEnd {
				role = RoleWritable
			} else {
				role = RoleReadonly
			}
		}

		result[i] = AccountMeta{Index: i, Address: addr, Role: role}
	}

	return result
}
