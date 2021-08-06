package blocksync

import (
	bcproto "github.com/providenetwork/tendermint/proto/tendermint/blocksync"
	"github.com/providenetwork/tendermint/types"
)

const (
	MaxMsgSize = types.MaxBlockSizeBytes +
		bcproto.BlockResponseMessagePrefixSize +
		bcproto.BlockResponseMessageFieldKeySize
)
