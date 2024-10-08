package types

import "encoding/json"

type TxInput interface {
	GetBlockchain() Blockchain
	TxInputConflicts
	TxInputGasFeeMultiplier
}

// TxInputWithPublicKey is input data to a tx for chains that need to explicitly set the public key, e.g. Cosmos
type TxInputWithPublicKey interface {
	TxInput
	SetPublicKey([]byte) error
	SetPublicKeyFromStr(string) error
}

// TxInputWithAmount for chains that can optimize the tx input if they know the amount being transferred.
type TxInputWithAmount interface {
	SetAmount(BigInt)
}

// For chains/transactions that leverage memo field
type TxInputWithMemo interface {
	SetMemo(string)
}

// For chains/transactions that can benefit from knowing the timestamp
type TxInputWithUnix interface {
	SetUnix(int64)
}

type TxInputGasFeeMultiplier interface {
	SetGasFeePriority(priority GasFeePriority) error
}

// This interface determines whether if different tx inputs conflict with one another.
type TxInputConflicts interface {
	// Test independence of two tx-inputs, assuming the same address is used.
	// Examples:
	// - using the same nonce or sequence is NOT independent
	// - spending the same resources or utxo's is NOT independent
	// - solana (using recent_block_hash) is pretty much always independent
	// This is used to determine if a transaction needs to be queued or if it can be immediately signed & broadcasted.
	IndependentOf(other TxInput) (independent bool)

	// Test if tx-input could possibly result in a "double-send" given the history of past attempts.
	// A double send is a user re-signing their transaction (to overcome a timeout or use new fees), but then risk multiple transactions
	// landing on chain.  A valid re-sign should only occur if it's only possible for one transaction to land out of the total set of attempts.
	// Examples:
	// - Solana typically has no conflicts, but need to ensure (a) new blockhash is used, and (b) sufficient time has passed
	//   to be sure a double send won't occur (return `true`).
	// - If tx-inputs are not independent (spending same resources), then typically double-sends are impossible (and should return `true` here).
	// - If there exists tx-inputs that are fully independent (and not timed out), then a double-send is possible and this should return false.
	SafeFromDoubleSend(previousAttempts ...TxInput) (safe bool)
}

func SameTxInputTypes[T TxInput](as T, inputs ...TxInput) bool {
	for _, input := range inputs {
		if _, ok := input.(T); !ok {
			return false
		}
	}
	return true
}

type TxInputEnvelope struct {
	Type    Blockchain      `json:"type"`
	TxInput json.RawMessage `json:"tx_input"`
}

func NewTxInputEnvelope(envType Blockchain) *TxInputEnvelope {
	return &TxInputEnvelope{
		Type: envType,
	}
}
