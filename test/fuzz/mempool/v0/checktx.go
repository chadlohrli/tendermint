package v0

import (
	"context"

	"github.com/providenetwork/tendermint/abci/example/kvstore"
	"github.com/providenetwork/tendermint/config"
	"github.com/providenetwork/tendermint/internal/mempool"
	mempoolv0 "github.com/providenetwork/tendermint/internal/mempool/v0"
	"github.com/providenetwork/tendermint/proxy"
)

var mp mempool.Mempool

func init() {
	app := kvstore.NewApplication()
	cc := proxy.NewLocalClientCreator(app)
	appConnMem, _ := cc.NewABCIClient()
	err := appConnMem.Start()
	if err != nil {
		panic(err)
	}

	cfg := config.DefaultMempoolConfig()
	cfg.Broadcast = false

	mp = mempoolv0.NewCListMempool(cfg, appConnMem, 0)
}

func Fuzz(data []byte) int {
	err := mp.CheckTx(context.Background(), data, nil, mempool.TxInfo{})
	if err != nil {
		return 0
	}

	return 1
}
