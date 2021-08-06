package blockchain

import (
	bcproto "github.com/providenetwork/tendermint/proto/tendermint/blockchain"
	"github.com/providenetwork/tendermint/types"
)

const (
	MaxMsgSize = types.MaxBlockSizeBytes +
		bcproto.BlockResponseMessagePrefixSize +
		bcproto.BlockResponseMessageFieldKeySize
)
