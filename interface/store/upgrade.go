package store

import (
	"bytes"
	"errors"
	"github.com/elastos/Elastos.ELA/core/types/payload"
	"sync"

	"github.com/elastos/Elastos.ELA/common"

	"github.com/syndtr/goleveldb/leveldb"
)

// Ensure arbiters implement arbiters interface.
var _ Upgrade = (*upgrade)(nil)

type upgrade struct {
	batch
	sync.RWMutex
	db       *leveldb.DB
	b        *leveldb.Batch
	posCache []uint32
}

func (u *upgrade) Clear() error {
	panic("implement me")
}

func (u *upgrade) Close() error {
	return nil
}

func NewUpgrade(db *leveldb.DB) *upgrade {
	return &upgrade{
		db: db,
		b:  new(leveldb.Batch),
	}
}

func (u *upgrade) BatchPutControversialUpgrade(proposalHash common.Uint256, info *payload.UpgradeCodeInfo,
	version byte, batch *leveldb.Batch) error {
	// store reserved custom ID.
	w := new(bytes.Buffer)
	if err := w.WriteByte(version); err != nil {
		return err
	}
	if err := info.Serialize(w, version); err != nil {
		return err
	}
	batch.Put(toKey(BKTUpgradeControversial, proposalHash.Bytes()...), w.Bytes())
	return nil
}

func (u *upgrade) getCurrentPositions() []uint32 {
	pos, err := u.db.Get(BKTUpgradePositions, nil)
	if err == nil {
		return bytesToUint32Array(pos)
	}
	return nil
}

func (u *upgrade) BatchDeleteControversialUpgrade(proposalHash common.Uint256, batch *leveldb.Batch) error {
	batch.Delete(toKey(BKTUpgradeControversial, proposalHash.Bytes()...))
	return nil
}

func (u *upgrade) batchPutUpgradeResult(proposalHash common.Uint256, batch *leveldb.Batch) error {
	info, data, err := u.getControversialUpgradeInfoByProposalHash(proposalHash)
	if err != nil {
		return err
	}

	// update current positions
	posCache := u.getCurrentPositions()
	newPosCache := make([]uint32, 0)
	for _, p := range posCache {
		if p < info.WorkingHeight {
			newPosCache = append(newPosCache, p)
		}
	}
	newPosCache = append(newPosCache, info.WorkingHeight)
	u.posCache = newPosCache
	batch.Put(BKTArbPositions, uint32ArrayToBytes(u.posCache))

	index := getIndex(info.WorkingHeight)
	batch.Put(toKey(BKTUpgradeCode, index...), data)
	return nil
}

func (u *upgrade) BatchPutUpgradeProposalResult(
	result payload.ProposalResult, batch *leveldb.Batch) error {
	u.Lock()
	defer u.Unlock()

	return u.batchPutUpgradeProposalResults(result, batch)
}

func (u *upgrade) batchPutUpgradeProposalResults(
	result payload.ProposalResult, batch *leveldb.Batch) error {
	if result.Result == true {
		err := u.batchPutUpgradeResult(result.ProposalHash, batch)
		if err != nil {
			return err
		}
	}

	// remove controversial upgrade information
	if err := u.BatchDeleteControversialUpgrade(result.ProposalHash, batch); err != nil {
		return err
	}
	return nil
}

func (u *upgrade) GetByHeight(height uint32) (info *payload.UpgradeCodeInfo, err error) {
	u.RLock()
	defer u.RUnlock()
	var pos []uint32
	if len(u.posCache) == 0 {
		pos = u.getCurrentPositions()
		u.posCache = pos
	} else {
		pos = u.posCache
	}
	currentHeight, err := findHeight(pos, height)
	if err != nil {
		return nil, err
	}

	return u.getUpgradeInfoByHeight(currentHeight)
}

func (u *upgrade) getUpgradeInfoByHeight(height uint32) (*payload.UpgradeCodeInfo, error) {
	index := getIndex(height)
	data, err := u.db.Get(toKey(BKTUpgradeCode, index...), nil)
	if err != nil {
		return nil, err
	}
	r := bytes.NewBuffer(data)
	versionBytes, err := common.ReadBytes(r, 1)
	if err != nil {
		return nil, err
	}
	info := &payload.UpgradeCodeInfo{}
	if err = info.Deserialize(r, versionBytes[0]); err != nil {
		return nil, err
	}
	return info, nil
}

func findHeight(pos []uint32, height uint32) (uint32, error) {
	if len(pos) == 0 {
		return 0, errors.New("current positions is nil")
	}

	for i := len(pos) - 1; i >= 0; i-- {
		if height >= pos[i] {
			return pos[i], nil
		}
	}

	return 0, errors.New("invalid height")
}

func (u *upgrade) getControversialUpgradeInfoByProposalHash(proposalHash common.Uint256) (*payload.UpgradeCodeInfo, []byte, error) {
	data, err := u.db.Get(toKey(BKTUpgradeControversial, proposalHash.Bytes()...), nil)
	if err != nil {
		return nil, nil, err
	}
	r := bytes.NewBuffer(data)
	versionBytes, err := common.ReadBytes(r, 1)
	if err != nil {
		return nil, nil, err
	}
	info := &payload.UpgradeCodeInfo{}
	if err = info.Deserialize(r, versionBytes[0]); err != nil {
		return nil, nil, err
	}

	return info, nil, nil
}
