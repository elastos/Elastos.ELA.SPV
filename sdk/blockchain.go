package sdk

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/elastos/Elastos.ELA.SPV/log"
	"github.com/elastos/Elastos.ELA.SPV/store"

	"github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA/core"
)

type ChainState int

const (
	SYNCING = ChainState(0)
	WAITING = ChainState(1)
)

func (s ChainState) String() string {
	switch s {
	case SYNCING:
		return "SYNCING"
	case WAITING:
		return "WAITING"
	default:
		return "UNKNOWN"
	}
}

const (
	MaxBlockLocatorHashes = 100
)

var PowLimit = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))

/*
Blockchain is the database of blocks, also when a new transaction or block commit,
Blockchain will verify them with stored blocks.
*/
type Blockchain struct {
	lock  *sync.RWMutex
	state ChainState
	store.HeaderStore
}

// Create a instance of *Blockchain
func NewBlockchain(foundation string, headerStore store.HeaderStore) (*Blockchain, error) {
	blockchain := &Blockchain{
		lock:        new(sync.RWMutex),
		state:       WAITING,
		HeaderStore: headerStore,
	}

	// Init genesis header
	_, err := blockchain.GetBestHeader()
	if err != nil {
		var err error
		var foundationAddress *common.Uint168
		if len(foundation) == 34 {
			foundationAddress, err = common.Uint168FromAddress(foundation)
		} else {
			foundationAddress, err = common.Uint168FromAddress("8VYXVxKKSAxkmRrfmGpQR2Kc66XhG6m3ta")
		}
		if err != nil {
			return nil, errors.New("parse foundation address failed")
		}
		genesisHeader := GenesisHeader(foundationAddress)
		storeHeader := &store.StoreHeader{Header: *genesisHeader, TotalWork: new(big.Int)}
		blockchain.PutHeader(storeHeader, true)
	}

	return blockchain, nil
}

// Close the blockchain
func (bc *Blockchain) Close() {
	bc.lock.Lock()
	bc.HeaderStore.Close()
}

// Set the current state of blockchain
func (bc *Blockchain) SetChainState(state ChainState) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	bc.state = state
}

// Return a bool value if blockchain is in syncing state
func (bc *Blockchain) IsSyncing() bool {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	return bc.state == SYNCING
}

// Get current blockchain height
func (bc *Blockchain) Height() uint32 {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	tip, err := bc.GetBestHeader()
	if err != nil {
		return 0
	}
	return tip.Height
}

// Get current blockchain tip
func (bc *Blockchain) ChainTip() *store.StoreHeader {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	return bc.chainTip()
}

func (bc *Blockchain) chainTip() *store.StoreHeader {
	tip, err := bc.GetBestHeader()
	if err != nil { // Empty blockchain, return empty header
		return &store.StoreHeader{TotalWork: new(big.Int)}
	}
	return tip
}

// Create a block locator which is a array of block hashes stored in blockchain
func (bc *Blockchain) GetBlockLocatorHashes() []*common.Uint256 {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	var ret []*common.Uint256
	parent, err := bc.GetBestHeader()
	if err != nil { // No headers stored return empty locator
		return ret
	}

	rollback := func(parent *store.StoreHeader, n int) (*store.StoreHeader, error) {
		for i := 0; i < n; i++ {
			parent, err = bc.GetPrevious(parent)
			if err != nil {
				return parent, err
			}
		}
		return parent, nil
	}

	step := 1
	start := 0
	for {
		if start >= 9 {
			step *= 2
			start = 0
		}
		hash := parent.Hash()
		ret = append(ret, &hash)
		if len(ret) >= MaxBlockLocatorHashes {
			break
		}
		parent, err = rollback(parent, step)
		if err != nil {
			break
		}
		start += 1
	}
	return ret
}

// IsKnownHeader returns if a header is already stored in database by it's hash
func (bc *Blockchain) IsKnownHeader(hash *common.Uint256) bool {
	header, _ := bc.HeaderStore.GetHeader(hash)
	return header != nil
}

// Commit header add a header into blockchain, return if this header
// is a new tip, or meet a reorganize (reorgFrom > 0), and error
func (bc *Blockchain) CommitHeader(header core.Header) (newTip bool, reorgFrom uint32, err error) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	err = bc.CheckProofOfWork(header)
	if err != nil {
		return newTip, reorgFrom, err
	}

	commitHeader := &store.StoreHeader{Header: header}

	// Get current chain tip
	tip := bc.chainTip()
	tipHash := tip.Hash()

	// Lookup of the parent header. Otherwise (ophan?) we need to fetch the parent.
	// If the tip is also the parent of this header, then we can save a database read by skipping
	var parentHeader *store.StoreHeader
	if header.Previous.IsEqual(tipHash) {
		parentHeader = tip
	} else {
		parentHeader, err = bc.GetPrevious(commitHeader)
		if err != nil {
			return newTip, reorgFrom, fmt.Errorf("Header %s does not extend any known headers", header.Hash().String())
		}
	}

	// If this block is already the tip, return
	if tipHash.IsEqual(header.Hash()) {
		return newTip, reorgFrom, err
	}
	// Add the work of this header to the total work stored at the previous header
	cumulativeWork := new(big.Int).Add(parentHeader.TotalWork, CalcWork(header.Bits))
	commitHeader.TotalWork = cumulativeWork

	// If the cumulative work is greater than the total work of our best header
	// then we have a new best header. Update the chain tip and check for a reorg.
	var forkPoint *store.StoreHeader
	if cumulativeWork.Cmp(tip.TotalWork) == 1 {
		newTip = true
		// If this header is not extending the previous best header then we have a reorg.
		if !tipHash.IsEqual(parentHeader.Hash()) {
			commitHeader.Height = parentHeader.Height + 1
			forkPoint, err = bc.getCommonAncestor(commitHeader, tip)
			if err != nil {
				log.Errorf("error calculating common ancestor: %s", err.Error())
				return newTip, reorgFrom, err
			}
			fmt.Printf("Reorganize At block %d, Wiped out %d blocks\n",
				int(tip.Height), int(tip.Height-forkPoint.Height))
		}
	}

	// If common ancestor exists, means we have an fork chan
	// so we need to rollback to the last good point.
	if forkPoint != nil {
		reorgFrom = tip.Height
		// Save reorganize point as the new tip
		err = bc.PutHeader(forkPoint, newTip)
		if err != nil {
			return newTip, reorgFrom, err
		}
		return newTip, reorgFrom, err
	}

	// Save header to db
	err = bc.PutHeader(commitHeader, newTip)
	if err != nil {
		return newTip, reorgFrom, err
	}

	return newTip, reorgFrom, err
}

// Returns last header before reorg point
func (bc *Blockchain) getCommonAncestor(bestHeader, prevTip *store.StoreHeader) (*store.StoreHeader, error) {
	var err error
	rollback := func(parent *store.StoreHeader, n int) (*store.StoreHeader, error) {
		for i := 0; i < n; i++ {
			parent, err = bc.GetPrevious(parent)
			if err != nil {
				return parent, err
			}
		}
		return parent, nil
	}

	majority := bestHeader
	minority := prevTip
	if bestHeader.Height > prevTip.Height {
		majority, err = rollback(majority, int(bestHeader.Height-prevTip.Height))
		if err != nil {
			return nil, err
		}
	} else if prevTip.Height > bestHeader.Height {
		minority, err = rollback(minority, int(prevTip.Height-bestHeader.Height))
		if err != nil {
			return nil, err
		}
	}

	for {
		majorityHash := majority.Hash()
		minorityHash := minority.Hash()
		if majorityHash.IsEqual(minorityHash) {
			return majority, nil
		}
		majority, err = bc.GetPrevious(majority)
		if err != nil {
			return nil, err
		}
		minority, err = bc.GetPrevious(minority)
		if err != nil {
			return nil, err
		}
	}
}

func CalcWork(bits uint32) *big.Int {
	// Return a work value of zero if the passed difficulty bits represent
	// a negative number. Note this should not happen in practice with valid
	// blocks, but an invalid block could trigger it.
	difficultyNum := CompactToBig(bits)
	if difficultyNum.Sign() <= 0 {
		return big.NewInt(0)
	}

	// (1 << 256) / (difficultyNum + 1)
	denominator := new(big.Int).Add(difficultyNum, big.NewInt(1))
	return new(big.Int).Div(new(big.Int).Lsh(big.NewInt(1), 256), denominator)
}

func (bc *Blockchain) CheckProofOfWork(header core.Header) error {
	// The target difficulty must be larger than zero.
	target := CompactToBig(header.Bits)
	if target.Sign() <= 0 {
		return errors.New("[Blockchain], block target difficulty is too low.")
	}

	// The target difficulty must be less than the maximum allowed.
	if target.Cmp(PowLimit) > 0 {
		return errors.New("[Blockchain], block target difficulty is higher than max of limit.")
	}

	// The block hash must be less than the claimed target.
	hash := header.AuxPow.ParBlockHeader.Hash()
	hashNum := HashToBig(&hash)
	if hashNum.Cmp(target) > 0 {
		return errors.New("[Blockchain], block target difficulty is higher than expected difficulty.")
	}

	return nil
}

func HashToBig(hash *common.Uint256) *big.Int {
	// A Hash is in little-endian, but the big package wants the bytes in
	// big-endian, so reverse them.
	buf := *hash
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

func CompactToBig(compact uint32) *big.Int {
	// Extract the mantissa, sign bit, and exponent.
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes to represent the full 256-bit number.  So,
	// treat the exponent as the number of bytes and shift the mantissa
	// right or left accordingly.  This is equivalent to:
	// N = mantissa * 256^(exponent-3)
	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	// Make it negative if the sign bit is set.
	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}
