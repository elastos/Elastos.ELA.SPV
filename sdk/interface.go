package sdk

import (
	"github.com/elastos/Elastos.ELA.SPV/database"
	"github.com/elastos/Elastos.ELA.SPV/util"
	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/core"
)

/*
IService is an implementation for SPV features.
*/
type IService interface {
	// Start SPV service
	Start()

	// Stop SPV service
	Stop()

	// IsCurrent returns whether or not the SPV service believes it is synced with
	// the connected peers.
	IsCurrent() bool

	// UpdateFilter is a trigger to make SPV service refresh the current
	// transaction filer(in our implementation the bloom filter) and broadcast the
	// new filter to connected peers.  This will invoke the GetFilterData() method
	// in Config.
	UpdateFilter()

	// SendTransaction broadcast a transaction message to the peer to peer network.
	SendTransaction(core.Transaction) error
}

// StateNotifier exposes methods to notify status changes of transactions and blocks.
type StateNotifier interface {
	// TransactionAccepted will be invoked after a transaction sent by
	// SendTransaction() method has been accepted.  Notice: this method needs at
	// lest two connected peers to work.
	TransactionAccepted(tx *util.Tx)

	// TransactionRejected will be invoked if a transaction sent by SendTransaction()
	// method has been rejected.
	TransactionRejected(tx *util.Tx)

	// TransactionConfirmed will be invoked after a transaction sent by
	// SendTransaction() method has been packed into a block.
	TransactionConfirmed(tx *util.Tx)

	// BlockCommitted will be invoked when a block and transactions within it are
	// successfully committed into database.
	BlockCommitted(block *util.Block)
}

// Config is the configuration settings to the SPV service.
type Config struct {
	// The magic number to indicate which network to access.
	Magic uint32

	// The seed peers addresses in [host:port] or [ip:port] format.
	SeedList []string

	// The default port for public peers to provide service.
	DefaultPort uint16

	// The max peer connections.
	MaxPeers int

	// The min candidate peers count to start syncing progress.
	MinPeersForSync int

	// Foundation address of the current access blockhain network
	Foundation string

	// The database to store all block headers
	ChainStore database.ChainStore

	// GetFilterData() returns two arguments.
	// First arguments are all addresses stored in your data store.
	// Second arguments are all balance references to those addresses stored in your data store,
	// including UTXO(Unspent Transaction Output)s and STXO(Spent Transaction Output)s.
	// Outpoint is a data structure include a transaction ID and output index. It indicates the
	// reference of an transaction output. If an address ever received an transaction output,
	// there will be the outpoint reference to it. Any time you want to spend the balance of an
	// address, you must provide the reference of the balance which is an outpoint in the transaction input.
	GetFilterData func() ([]*common.Uint168, []*core.OutPoint)

	// StateNotifier is an optional config, if you don't want to receive state changes of transactions
	// or blocks, just keep it blank.
	StateNotifier StateNotifier
}

/*
NewService returns a new SPV service instance.
there are two implementations you need to do, DataStore and GetBloomFilter() method.
DataStore is an interface including all methods you need to implement placed in db/datastore.go.
Also an sample APP spvwallet is contain in this project placed in spvwallet folder.
*/
func NewService(config *Config) (IService, error) {
	return NewSPVService(config)
}