package sync

import (
	"github.com/elastos/Elastos.ELA.SPV/blockchain"
	"github.com/elastos/Elastos.ELA.SPV/util"

	"github.com/elastos/Elastos.ELA/filter"
)

const (
	defaultMaxPeers = 125
)

// Config is a configuration struct used to initialize a new SyncManager.
type Config struct {
	Chain *blockchain.BlockChain

	MaxPeers        int

	UpdateFilter        func() filter.TxFilter
	TransactionAnnounce func(tx util.Transaction)
}

func NewDefaultConfig(chain *blockchain.BlockChain,
	updateFilter func() filter.TxFilter) *Config {
	return &Config{
		Chain:        chain,
		MaxPeers:     defaultMaxPeers,
		UpdateFilter: updateFilter,
	}
}
