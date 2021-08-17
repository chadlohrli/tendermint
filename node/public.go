// Package node provides a high level wrapper around tendermint services.
package node

import (
	"fmt"

	uuid "github.com/kthomas/go.uuid"
	"github.com/providenetwork/tendermint/config"
	"github.com/providenetwork/tendermint/libs/log"
	"github.com/providenetwork/tendermint/libs/service"
	"github.com/providenetwork/tendermint/p2p"
	"github.com/providenetwork/tendermint/privval"
	"github.com/providenetwork/tendermint/proxy"
	"github.com/providenetwork/tendermint/types"
)

// NewDefault constructs a tendermint node service for use in go
// process that host their own process-local tendermint node. This is
// equivalent to running tendermint in it's own process communicating
// to an external ABCI application.
func NewDefault(conf *config.Config, logger log.Logger) (service.Service, error) {
	return DefaultNewNode(conf, logger)
}

// New constructs a tendermint node. The ClientCreator makes it
// possible to construct an ABCI application that runs in the same
// process as the tendermint node.  The final option is a pointer to a
// Genesis document: if the value is nil, the genesis document is read
// from the file specified in the config, and otherwise the node uses
// value of the final argument.
func New(
	conf *config.Config,
	logger log.Logger,
	cf proxy.ClientCreator,
	gen *types.GenesisDoc,
) (service.Service, error) {
	var nodeKey *p2p.NodeKey
	var err error

	var vaultID *uuid.UUID
	var vaultKeyID *uuid.UUID

	if vaultUUID, err := uuid.FromString(conf.VaultID); err == nil {
		vaultID = &vaultUUID
	}

	if vaultKeyUUID, err := uuid.FromString(conf.VaultKeyID); err == nil {
		vaultKeyID = &vaultKeyUUID
	}

	nodeKey, err = p2p.LoadOrGenNodeKey(conf.NodeKey, conf.VaultRefreshToken, vaultID, vaultKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to load or gen node key %s; %s", conf.NodeKeyFile(), err.Error())
	}

	var genesisProvider GenesisDocProvider
	switch gen {
	case nil:
		genesisProvider = DefaultGenesisDocProviderFunc(conf)
	default:
		genesisProvider = func() (*types.GenesisDoc, error) { return gen, nil }
	}

	var pval types.PrivValidator

	switch conf.Mode {
	case "full", "validator":
		pval, err = privval.LoadOrGenValidator(conf.RootDir, conf.VaultRefreshToken, *vaultID, vaultKeyID)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("%s is not a valid mode", conf.Mode)
	}

	return NewNode(
		conf,
		pval,
		nodeKey,
		cf,
		genesisProvider,
		DefaultDBProvider,
		DefaultMetricsProvider(conf.Instrumentation),
		logger,
	)
}
