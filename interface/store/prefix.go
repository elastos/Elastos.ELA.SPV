package store

var (
	// addresses
	BKTAddrs = []byte("addrs")

	// headers
	BKTHeaders  = []byte("headers")
	BKTIndexes  = []byte("indexes")
	BKTChainTip = []byte("chaintip")

	// ops
	BKTOps = []byte("ops")

	// que
	BKTQue    = []byte("que")
	BKTQueIdx = []byte("qindex")

	// transactions
	BKTTxs       = []byte("transactions")
	BKTHeightTxs = []byte("heighttxs")
	BKTForkTxs   = []byte("forktxs")

	// arbiters
	BKTArbiters          = []byte("arbiters")
	BKTArbPosition       = []byte("arbptn")
	BKTArbPositions      = []byte("arbpts")
	BKTArbitersData      = []byte("arbdata")
	BKTTransactionHeight = []byte("txheight")

	// revert to pow
	BKTRevertPosition  = []byte("revertp")
	BKTRevertPositions = []byte("revertps")
	BKTConsensus       = []byte("consensus")

	// custom ID
	BKTReservedCustomID  = []byte("RS")
	BKTReceivedCustomID  = []byte("RC")
	BKTChangeCustomIDFee = []byte("CF")
	BKTLastCustomIDFee   = []byte("CH")

	// upgrade code
	BKTUpgradeControversial = []byte("upgradectl")
	BKTUpgradeCode          = []byte("upgradecode")
	BKTUpgradePositions     = []byte("upgradepts")
)
