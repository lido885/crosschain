package signer

import (
	"context"

	"github.com/openweb3-io/crosschain/types"
)

type Signer interface {
	PublicKey(ctx context.Context) ([]byte, error)
	SharedKey(theirKey []byte) ([]byte, error)
	Sign(ctx context.Context, payload types.TxDataToSign) (types.TxSignature, error)
}
