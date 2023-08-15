package store

import (
	"bytes"
	"errors"
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
	db                *leveldb.DB
	b                 *leveldb.Batch
	escMinFeePosCache []uint32
}

func NewSideChain(db *leveldb.DB) *sideChain {
	return &sideChain{
		db: db,
		b:  new(leveldb.Batch),
	}
}

func (c *sideChain) BatchPutControversialSetESCMinGasPrice(gasPrice common.Fixed64,
	proposalHash common.Uint256, workingHeight uint32, batch *leveldb.Batch) error {
	c.Lock()
	defer c.Unlock()

	return c.batchPutChangeESCMinGasPrice(gasPrice, workingHeight, proposalHash, batch)
}

func (c *sideChain) BatchDeleteControversialChangeESCMinGasPrice(
	proposalHash common.Uint256, batch *leveldb.Batch) {
	c.Lock()
	defer c.Unlock()

	batch.Delete(toKey(BKTChangeESCMinGasPrice, proposalHash.Bytes()...))
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
		case payload.ChangeESCMinGasPrice:
			if r.Result == true {
				minFee, workingHeight, err := c.getControversialESCMinFeeByProposalHash(r.ProposalHash)
				if err != nil {
					return err
				}
				if err := c.batchPuESCMinGasPrice(batch, minFee, workingHeight); err != nil {
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

func (c *sideChain) batchPutChangeESCMinGasPrice(gasPrice common.Fixed64,
	workingHeight uint32, proposalHash common.Uint256, batch *leveldb.Batch) error {
	w := new(bytes.Buffer)
	if err := gasPrice.Serialize(w); err != nil {
		return err
	}
	if err := common.WriteUint32(w, workingHeight); err != nil {
		return err
	}
	batch.Put(toKey(BKTChangeESCMinGasPrice, proposalHash.Bytes()...), w.Bytes())
	return nil
}

func (c *sideChain) getCurrentESCMinGasPricePositions() []uint32 {
	pos, err := c.db.Get(BKTESCMinGasPricePositions, nil)
	if err == nil {
		return bytesToUint32Array(pos)
	}
	return nil
}

func (c *sideChain) batchPuESCMinGasPrice(batch *leveldb.Batch, minFee common.Fixed64, workingHeight uint32) error {
	posCache := c.getCurrentESCMinGasPricePositions()
	newPosCache := make([]uint32, 0)
	for _, p := range posCache {
		if p < workingHeight {
			newPosCache = append(newPosCache, p)
		}
	}
	newPosCache = append(newPosCache, workingHeight)
	c.escMinFeePosCache = newPosCache
	batch.Put(BKTESCMinGasPricePositions, uint32ArrayToBytes(c.escMinFeePosCache))

	buf := new(bytes.Buffer)
	if err := common.WriteUint32(buf, workingHeight); err != nil {
		return err
	}
	key := toKey(BKTChangeESCMinGasPrice, buf.Bytes()...)
	w := new(bytes.Buffer)
	if err := minFee.Serialize(w); err != nil {
		return err
	}
	batch.Put(key, w.Bytes())
	return nil
}

func (c *sideChain) GetESCMinGasPrice(height uint32, genesisBlockHash common.Uint256) (common.Fixed64, error) {
	c.RLock()
	defer c.RUnlock()
	return c.getEscMinGasPrice(height, genesisBlockHash)
}

func (c *sideChain) getEscMinGasPrice(height uint32, genesisBlockHash common.Uint256) (common.Fixed64, error) {
	workingHeight, err := c.findESCMinFeeWorkingHeightByCurrentHeight(height)
	if err != nil {
		return 0, err
	}

	return c.getControversialESCMinGasPriceByHeight(workingHeight)
}

func (c *sideChain) findESCMinFeeWorkingHeightByCurrentHeight(height uint32) (uint32, error) {
	var pos []uint32
	if len(c.escMinFeePosCache) == 0 {
		pos = c.getCurrentESCMinGasPricePositions()
		c.escMinFeePosCache = pos
	} else {
		pos = c.escMinFeePosCache
	}

	if len(c.escMinFeePosCache) == 0 {
		return 0, errors.New("have no esc min fee from main chain proposal")
	}

	for i := len(c.escMinFeePosCache) - 1; i >= 0; i-- {
		if height > c.escMinFeePosCache[i] {
			return c.escMinFeePosCache[i], nil
		}
	}

	return 0, nil
}

func (c *sideChain) getControversialESCMinFeeByProposalHash(proposalHash common.Uint256) (common.Fixed64, uint32, error) {
	var val []byte
	val, err := c.db.Get(toKey(BKTChangeESCMinGasPrice, proposalHash.Bytes()...), nil)
	if err != nil {
		return 0, 0, err
	}
	r := bytes.NewReader(val)
	var minFee common.Fixed64
	if err := minFee.Deserialize(r); err != nil {
		return 0, 0, err
	}
	workingHeight, err := common.ReadUint32(r)
	if err != nil {
		return 0, 0, err
	}
	return minFee, workingHeight, nil
}

func (c *sideChain) getControversialESCMinGasPriceByHeight(workingHeight uint32) (common.Fixed64, error) {
	buf := new(bytes.Buffer)
	if err := common.WriteUint32(buf, workingHeight); err != nil {
		return 0, err
	}
	var val []byte
	val, err := c.db.Get(toKey(BKTChangeESCMinGasPrice, buf.Bytes()...), nil)
	if err != nil {
		return 0, err
	}
	r := bytes.NewReader(val)
	var minFee common.Fixed64
	if err := minFee.Deserialize(r); err != nil {
		return 0, err
	}
	return minFee, nil
}

func (c *sideChain) Close() error {
	c.Lock()
	return nil
}

func (c *sideChain) Clear() error {
	c.Lock()
	defer c.Unlock()

	batch := new(leveldb.Batch)
	it := c.db.NewIterator(util.BytesPrefix(BKTChangeESCMinGasPrice), nil)
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
