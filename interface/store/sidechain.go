package store

import (
	"bytes"
	"errors"
	"math/big"
	"sync"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Ensure sideChain implement sideChain interface.
var _ SideChain = (*sideChain)(nil)

type sideChain struct {
	batch
	sync.RWMutex
	db               *leveldb.DB
	b                *leveldb.Batch
	gasPricePosCache map[common.Uint256][]uint32
}

func NewSideChain(db *leveldb.DB) *sideChain {
	return &sideChain{
		db:               db,
		b:                new(leveldb.Batch),
		gasPricePosCache: make(map[common.Uint256][]uint32, 0),
	}
}

func (c *sideChain) BatchPutControversialSetMinGasPrice(
	genesisBlockHash common.Uint256, gasPrice *big.Int,
	proposalHash common.Uint256, workingHeight uint32, batch *leveldb.Batch) error {
	c.Lock()
	defer c.Unlock()

	return c.batchPutChangeMinGasPrice(
		genesisBlockHash, gasPrice, workingHeight, proposalHash, batch)
}

func (c *sideChain) BatchDeleteControversialChangeMinGasPrice(
	proposalHash common.Uint256, batch *leveldb.Batch) {
	c.Lock()
	defer c.Unlock()

	batch.Delete(toKey(BKTChangeSideChainMinGasPrice, proposalHash.Bytes()...))
}

func (c *sideChain) PutSideChainRelatedProposalResults(
	results []payload.ProposalResult, height uint32) error {
	c.Lock()
	defer c.Unlock()
	batch := new(leveldb.Batch)
	if err := c.batchPutSideChainRelatedProposalResults(results, height, batch); err != nil {
		return err
	}
	return c.db.Write(batch, nil)
}

func (c *sideChain) BatchPutSideChainRelatedProposalResults(
	results []payload.ProposalResult, height uint32, batch *leveldb.Batch) error {
	c.Lock()
	defer c.Unlock()

	return c.batchPutSideChainRelatedProposalResults(results, height, batch)
}

func (c *sideChain) batchPutSideChainRelatedProposalResults(
	results []payload.ProposalResult, height uint32, batch *leveldb.Batch) error {
	// add new reserved custom ID into cache.
	for _, r := range results {
		switch r.ProposalType {
		case payload.ChangeSideChainMinGasPrice:
			if r.Result == true {
				genesisBlockHash, gasPrice, workingHeight, err :=
					c.getControversialMinGasPriceByProposalHash(r.ProposalHash)
				if err != nil {
					return err
				}
				if err := c.batchPuMinGasPrice(batch, genesisBlockHash,
					gasPrice, workingHeight); err != nil {
					return err
				}
			} else {
				// if you need to remove data from db, you need to consider rollback.
				//c.removeControversialCustomIDFeeRate(r.ProposalHash, batch)
			}
		}
	}
	return nil
}

func (c *sideChain) batchPutChangeMinGasPrice(
	genesisBlockHash common.Uint256, gasPrice *big.Int, workingHeight uint32,
	proposalHash common.Uint256, batch *leveldb.Batch) error {
	w := new(bytes.Buffer)
	if err := genesisBlockHash.Serialize(w); err != nil {
		return err
	}
	if err := common.WriteVarBytes(w, gasPrice.Bytes()); err != nil {
		return err
	}
	if err := common.WriteUint32(w, workingHeight); err != nil {
		return err
	}
	batch.Put(toKey(BKTChangeSideChainMinGasPrice, proposalHash.Bytes()...), w.Bytes())
	return nil
}

func (c *sideChain) getCurrentMinGasPricePositions(genesisHash common.Uint256) []uint32 {
	pos, err := c.db.Get(toKey(BKTSideChainMinGasPricePositions, genesisHash.Bytes()...), nil)
	if err == nil {
		return bytesToUint32Array(pos)
	}
	return nil
}

func (c *sideChain) batchPuMinGasPrice(batch *leveldb.Batch,
	genesisHash common.Uint256, gasPrice big.Int, workingHeight uint32) error {
	posCache := c.getCurrentMinGasPricePositions(genesisHash)
	newPosCache := make([]uint32, 0)
	for _, p := range posCache {
		if p < workingHeight {
			newPosCache = append(newPosCache, p)
		}
	}
	newPosCache = append(newPosCache, workingHeight)
	c.gasPricePosCache[genesisHash] = newPosCache
	batch.Put(toKey(BKTSideChainMinGasPricePositions, genesisHash.Bytes()...),
		uint32ArrayToBytes(c.gasPricePosCache[genesisHash]))

	buf := new(bytes.Buffer)
	if err := common.WriteUint32(buf, workingHeight); err != nil {
		return err
	}
	key := toKey(toKey(BKTChangeSideChainMinGasPrice, genesisHash.Bytes()...), buf.Bytes()...)
	w := new(bytes.Buffer)
	if err := common.WriteVarBytes(w, gasPrice.Bytes()); err != nil {
		return err
	}
	batch.Put(key, w.Bytes())
	return nil
}

func (c *sideChain) GetMinGasPrice(height uint32, genesisBlockHash common.Uint256) (*big.Int, error) {
	c.RLock()
	defer c.RUnlock()
	return c.getMinGasPrice(height, genesisBlockHash)
}

func (c *sideChain) getMinGasPrice(height uint32, genesisBlockHash common.Uint256) (*big.Int, error) {
	workingHeight, err := c.findGasPriceWorkingHeightByCurrentHeight(height, genesisBlockHash)
	if err != nil {
		return nil, err
	}

	return c.getControversialMinGasPriceByHeight(genesisBlockHash, workingHeight)
}

func (c *sideChain) findGasPriceWorkingHeightByCurrentHeight(
	height uint32, genesisBlockHash common.Uint256) (uint32, error) {
	var pos []uint32
	if _, ok := c.gasPricePosCache[genesisBlockHash]; !ok {
		pos = c.getCurrentMinGasPricePositions(genesisBlockHash)
		c.gasPricePosCache[genesisBlockHash] = pos
	} else {
		pos = c.gasPricePosCache[genesisBlockHash]
	}

	if len(pos) == 0 {
		return 0, errors.New("have no min fee from main chain proposal")
	}

	for i := len(pos) - 1; i >= 0; i-- {
		if height > pos[i] {
			return pos[i], nil
		}
	}

	return 0, nil
}

func (c *sideChain) getControversialMinGasPriceByProposalHash(
	proposalHash common.Uint256) (genesisBlockHash common.Uint256,
	gasPrice big.Int, workingHeight uint32, err error) {
	var val []byte
	val, err = c.db.Get(toKey(BKTChangeSideChainMinGasPrice, proposalHash.Bytes()...), nil)
	if err != nil {
		return
	}
	r := bytes.NewReader(val)

	if err = genesisBlockHash.Deserialize(r); err != nil {
		return
	}

	var minGasPriceBytes []byte
	if minGasPriceBytes, err = common.ReadVarBytes(r, payload.MaxSideChainGasPriceLength, "gas price"); err != nil {
		return
	}
	gasPrice.SetBytes(minGasPriceBytes)

	workingHeight, err = common.ReadUint32(r)

	return
}

func (c *sideChain) getControversialMinGasPriceByHeight(genesisHash common.Uint256, workingHeight uint32) (*big.Int, error) {
	buf := new(bytes.Buffer)
	if err := common.WriteUint32(buf, workingHeight); err != nil {
		return nil, err
	}
	var val []byte
	val, err := c.db.Get(toKey(toKey(BKTChangeSideChainMinGasPrice, genesisHash.Bytes()...), buf.Bytes()...), nil)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(val)
	var gasPriceBytes []byte
	if gasPriceBytes, err = common.ReadVarBytes(r, payload.MaxSideChainGasPriceLength, "gas price"); err != nil {
		return nil, err
	}
	gasPrice := new(big.Int)
	gasPrice.SetBytes(gasPriceBytes)
	return gasPrice, nil
}

func (c *sideChain) Close() error {
	c.Lock()
	return nil
}

func (c *sideChain) Clear() error {
	c.Lock()
	defer c.Unlock()

	batch := new(leveldb.Batch)
	it := c.db.NewIterator(util.BytesPrefix(BKTChangeSideChainMinGasPrice), nil)
	defer it.Release()
	for it.Next() {
		batch.Delete(it.Key())
	}

	return c.db.Write(c.b, nil)
}

func (c *sideChain) Commit() error {
	return c.db.Write(c.b, nil)
}

func (c *sideChain) Rollback() error {
	c.b.Reset()
	return nil
}

func (c *sideChain) CommitBatch(batch *leveldb.Batch) error {
	return c.db.Write(batch, nil)
}

func (c *sideChain) RollbackBatch(batch *leveldb.Batch) error {
	batch.Reset()
	return nil
}
