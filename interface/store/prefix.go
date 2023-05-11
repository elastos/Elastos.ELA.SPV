package store

var (
	// addresses
	BKTAddrs = []byte("addrs")

	// transaction types
	BKTTxTypes = []byte("txtypes")

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
	BKTArbiters             = []byte("arbiters")
	BKTArbPosition          = []byte("arbptn")
	BKTArbPositions         = []byte("arbpts")
	BKTArbitersData         = []byte("arbdata")
	BKTTransactionHeight    = []byte("txheight")
	BKTCompleteArbitersData = []byte("arbcomdata")

	// revert to pow
	BKTRevertPosition  = []byte("revertp")
	BKTRevertPositions = []byte("revertps")

	//ReturnSideChainDepositCoin
	BKTReturnSideChainDepositCoin = []byte("retschdepositcoin")

	BKTReservedCustomID     = []byte("rscid")
	BKTReceivedCustomID     = []byte("rccid")
	BKTChangeCustomIDFee    = []byte("ccidf")
	BKTCustomIDFeePositions = []byte("cidfps")
)
