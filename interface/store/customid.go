package store

import (
	"bytes"
	"sync"

	"github.com/elastos/Elastos.ELA/common"

	"github.com/syndtr/goleveldb/leveldb"
)

// Ensure customID implement CustomID interface.
var _ CustomID = (*customID)(nil)

var BKTReservedCustomID = []byte("RS")
var BKTReceivedCustomID = []byte("RC")
var BKTChangeCustomIDFee = []byte("CF")

type customID struct {
	batch
	sync.RWMutex
	db                             *leveldb.DB
	b                              *leveldb.Batch
	cache                          map[common.Uint256]uint32
	controversialReservedCustomIDs map[string]struct{}
	controversialReceivedCustomIDs map[string]common.Uint168
}

func NewCustomID(db *leveldb.DB) *customID {
	return &customID{
		db:                             db,
		b:                              new(leveldb.Batch),
		cache:                          make(map[common.Uint256]uint32),
		controversialReservedCustomIDs: make(map[string]struct{}, 0),
		controversialReceivedCustomIDs: make(map[string]common.Uint168, 0),
	}
}

func (c *customID) PutControversialReservedCustomIDs(reservedCustomIDs []string) error {
	c.Lock()
	defer c.Unlock()
	batch := new(leveldb.Batch)
	if err := c.batchPutReservedCustomIDs(reservedCustomIDs, batch); err != nil {
		return err
	}
	return c.db.Write(batch, nil)
}

func (c *customID) PutControversialReceivedCustomIDs(reservedCustomIDs []string, did common.Uint168) error {
	c.Lock()
	defer c.Unlock()
	batch := new(leveldb.Batch)
	if err := c.batchPutReceivedCustomIDs(reservedCustomIDs, did, batch); err != nil {
		return err
	}
	return c.db.Write(batch, nil)
}

func (c *customID) PutRChangeCustomIDFee(rate common.Fixed64) error {
	c.Lock()
	defer c.Unlock()
	batch := new(leveldb.Batch)
	if err := c.batchPutPutChangeCustomIDFee(rate, batch); err != nil {
		return err
	}
	return c.db.Write(batch, nil)
}

func (c *customID) BatchPutControversialReservedCustomIDs(reservedCustomIDs []string, batch *leveldb.Batch) error {
	c.Lock()
	defer c.Unlock()

	return c.batchPutReservedCustomIDs(reservedCustomIDs, batch)
}

func (c *customID) BatchPutControversialReceivedCustomIDs(receeivedCustomIDs []string,
	did common.Uint168, batch *leveldb.Batch) error {
	c.Lock()
	defer c.Unlock()

	return c.batchPutReceivedCustomIDs(receeivedCustomIDs, did, batch)
}

func (c *customID) BatchPutChangeCustomIDFee(rate common.Fixed64, batch *leveldb.Batch) error {
	c.Lock()
	defer c.Unlock()

	return c.batchPutPutChangeCustomIDFee(rate, batch)
}

func (c *customID) batchPutReservedCustomIDs(reservedCustomIDs []string, batch *leveldb.Batch) error {
	// initialize cache.
	if len(c.controversialReservedCustomIDs) == 0 {
		existedCustomIDs, err := c.getReservedCustomIDsFromDB()
		if err != nil {
			return err
		}
		c.controversialReservedCustomIDs = existedCustomIDs
	}
	// add new reserved custom ID into cache.
	for _, id := range reservedCustomIDs {
		c.controversialReservedCustomIDs[id] = struct{}{}
	}

	// store reserved custom ID.
	w := new(bytes.Buffer)
	err := common.WriteUint32(w, uint32(len(c.controversialReservedCustomIDs)+len(reservedCustomIDs)))
	if err != nil {
		return err
	}
	for k, _ := range c.controversialReservedCustomIDs {
		if err := common.WriteVarString(w, k); err != nil {
			return err
		}
	}
	for _, id := range reservedCustomIDs {
		if err := common.WriteVarString(w, id); err != nil {
			return err
		}
	}
	batch.Put(BKTReservedCustomID, w.Bytes())
	return nil
}

func (c *customID) batchPutReceivedCustomIDs(receivedCustomIDs []string,
	did common.Uint168, batch *leveldb.Batch) error {
	// initialize cache.
	if len(c.controversialReceivedCustomIDs) == 0 {
		receivedCustomIDs, err := c.getReceivedCustomIDsFromDB()
		if err != nil {
			return err
		}
		c.controversialReceivedCustomIDs = receivedCustomIDs
	}
	// add new received custom ID into cache.
	for _, id := range receivedCustomIDs {
		c.controversialReceivedCustomIDs[id] = did
	}

	// store received custom ID.
	w := new(bytes.Buffer)
	err := common.WriteUint32(w, uint32(len(c.controversialReceivedCustomIDs)+len(receivedCustomIDs)))
	if err != nil {
		return err
	}
	for k, v := range c.controversialReceivedCustomIDs {
		if err := common.WriteVarString(w, k); err != nil {
			return err
		}
		if err := v.Serialize(w); err != nil {
			return err
		}
	}
	for _, id := range receivedCustomIDs {
		if err := common.WriteVarString(w, id); err != nil {
			return err
		}
		if err := did.Serialize(w); err != nil {
			return err
		}
	}
	batch.Put(BKTReceivedCustomID, w.Bytes())
	return nil
}

func (c *customID) batchPutPutChangeCustomIDFee(rate common.Fixed64, batch *leveldb.Batch) error {
	w := new(bytes.Buffer)
	if err := rate.Serialize(w); err != nil {
		return err
	}
	batch.Put(BKTChangeCustomIDFee, w.Bytes())
	return nil
}

func (c *customID) GetControversialReservedCustomIDs() (map[string]struct{}, error) {
	c.RLock()
	defer c.RUnlock()
	return c.getControversialReservedCustomIDs()
}

func (c *customID) GetControversialReceivedCustomIDs() (map[string]common.Uint168, error) {
	c.RLock()
	defer c.RUnlock()
	return c.getControversialReceivedCustomIDs()
}

func (c *customID) GetCustomIDFeeRate() (common.Fixed64, error) {
	c.RLock()
	defer c.RUnlock()
	return c.getCustomIDFeeRate()
}

func (c *customID) getControversialReservedCustomIDs() (map[string]struct{}, error) {
	if len(c.controversialReservedCustomIDs) != 0 {
		return c.controversialReservedCustomIDs, nil
	}

	ids, err := c.getReservedCustomIDsFromDB()
	if err != nil {
		return nil, err
	}
	// refresh the cache.
	c.controversialReservedCustomIDs = ids
	return ids, nil
}

func (c *customID) getReservedCustomIDsFromDB() (map[string]struct{}, error) {
	var val []byte
	val, err := c.db.Get(BKTReservedCustomID, nil)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(val)
	count, err := common.ReadUint32(r)
	if err != nil {
		return nil, err
	}
	reservedCustomIDs := make(map[string]struct{}, 0)
	for i := uint32(0); i < count; i++ {
		id, err := common.ReadVarString(r)
		if err != nil {
			return nil, err
		}
		reservedCustomIDs[id] = struct{}{}
	}
	return reservedCustomIDs, nil
}

func (c *customID) getReceivedCustomIDsFromDB() (map[string]common.Uint168, error) {
	var val []byte
	val, err := c.db.Get(BKTReceivedCustomID, nil)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(val)
	count, err := common.ReadUint32(r)
	if err != nil {
		return nil, err
	}
	receiedCustomIDs := make(map[string]common.Uint168, 0)
	for i := uint32(0); i < count; i++ {
		id, err := common.ReadVarString(r)
		if err != nil {
			return nil, err
		}
		var did common.Uint168
		if err = did.Deserialize(r); err != nil {
			return nil, err
		}
		receiedCustomIDs[id] = did
	}
	return receiedCustomIDs, nil
}

func (c *customID) getControversialReceivedCustomIDs() (map[string]common.Uint168, error) {
	if len(c.controversialReceivedCustomIDs) != 0 {
		return c.controversialReceivedCustomIDs, nil
	}

	ids, err := c.getReceivedCustomIDsFromDB()
	if err != nil {
		return nil, err
	}
	// refresh the cache.
	c.controversialReceivedCustomIDs = ids
	return ids, nil
}

func (c *customID) getCustomIDFeeRate() (common.Fixed64, error) {
	var val []byte
	val, err := c.db.Get(BKTChangeCustomIDFee, nil)
	if err != nil {
		return 0, err
	}
	r := bytes.NewReader(val)
	var rate common.Fixed64
	if err := rate.Deserialize(r); err != nil {
		return 0, err
	}
	return rate, nil
}

func (c *customID) Close() error {
	c.Lock()
	return nil
}

func (c *customID) Clear() error {
	c.Lock()
	defer c.Unlock()
	c.b.Delete(BKTReservedCustomID)
	return c.db.Write(c.b, nil)
}

func (c *customID) Commit() error {
	return c.db.Write(c.b, nil)
}

func (c *customID) Rollback() error {
	c.b.Reset()
	return nil
}

func (c *customID) CommitBatch(batch *leveldb.Batch) error {
	return c.db.Write(batch, nil)
}

func (c *customID) RollbackBatch(batch *leveldb.Batch) error {
	batch.Reset()
	return nil
}
