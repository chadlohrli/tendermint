package mempool

import (
	"testing"

	"github.com/providenetwork/tendermint/p2p"
	"github.com/stretchr/testify/require"
)

func TestMempoolIDsBasic(t *testing.T) {
	ids := NewMempoolIDs()

	peerID, err := p2p.NewNodeID("0011223344556677889900112233445566778899")
	require.NoError(t, err)

	ids.ReserveForPeer(peerID)
	require.EqualValues(t, 1, ids.GetForPeer(peerID))
	ids.Reclaim(peerID)

	ids.ReserveForPeer(peerID)
	require.EqualValues(t, 2, ids.GetForPeer(peerID))
	ids.Reclaim(peerID)
}
