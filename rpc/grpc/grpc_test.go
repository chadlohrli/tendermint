package coregrpc_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/providenetwork/tendermint/abci/example/kvstore"
	"github.com/providenetwork/tendermint/config"
	"github.com/providenetwork/tendermint/libs/service"
	core_grpc "github.com/providenetwork/tendermint/rpc/grpc"
	rpctest "github.com/providenetwork/tendermint/rpc/test"
)

func NodeSuite(t *testing.T) (service.Service, *config.Config) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	conf := rpctest.CreateConfig(t.Name())

	// start a tendermint node in the background to test against
	app := kvstore.NewApplication()

	node, closer, err := rpctest.StartTendermint(ctx, conf, app)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = closer(ctx)
		cancel()
	})
	return node, conf
}

func TestBroadcastTx(t *testing.T) {
	_, conf := NodeSuite(t)

	res, err := rpctest.GetGRPCClient(conf).BroadcastTx(
		context.Background(),
		&core_grpc.RequestBroadcastTx{Tx: []byte("this is a tx")},
	)
	require.NoError(t, err)
	require.EqualValues(t, 0, res.CheckTx.Code)
	require.EqualValues(t, 0, res.DeliverTx.Code)
}
