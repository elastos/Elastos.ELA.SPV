package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/elastos/Elastos.ELA/common"
	"github.com/elastos/Elastos.ELA/core/types/payload"

	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
)

func TestUpgrade(t *testing.T) {
	dataDir := "upgrade_test"
	os.RemoveAll(dataDir)

	db, err := leveldb.OpenFile(filepath.Join(dataDir, "store"), nil)
	if err != nil {
		println(err.Error())
	}
	upgradeDB := NewUpgrade(db)

	batch := upgradeDB.b

	_, err = upgradeDB.GetByHeight(100)
	assert.EqualError(t, err, "current positions is nil")

	// batch put controversial upgrade
	ph := common.Uint256{1}
	up := &payload.UpgradeCodeInfo{
		WorkingHeight:   100,
		NodeVersion:     "0.7.0",
		NodeDownLoadUrl: "",
		NodeBinHash:     &common.Uint256{2},
		ForceUpgrade:    true,
	}
	err = upgradeDB.BatchPutControversialUpgrade(ph, up, 0x00, batch)
	assert.NoError(t, err)
	err = upgradeDB.Commit()
	assert.NoError(t, err)

	// get upgrade code information from db, upgrade proposal is controversial.
	_, err = upgradeDB.GetByHeight(100)
	assert.EqualError(t, err, "current positions is nil")

	// batch put proposal result: true
	result := payload.ProposalResult{
		ProposalHash: ph,
		ProposalType: 0x0201,
		Result:       true,
	}
	err = upgradeDB.BatchPutUpgradeProposalResult(result, batch)
	assert.NoError(t, err)
	err = upgradeDB.Commit()
	assert.NoError(t, err)

	// get upgrade code information from db
	info, err := upgradeDB.GetByHeight(100)
	assert.NoError(t, err)
	assert.Equal(t, *info, *up)

	// second test, batch put upgrade proposal again
	// batch put controversial upgrade
	ph2 := common.Uint256{3}
	up2 := &payload.UpgradeCodeInfo{
		WorkingHeight:   1000,
		NodeVersion:     "0.8.0",
		NodeDownLoadUrl: "",
		NodeBinHash:     &common.Uint256{4},
		ForceUpgrade:    true,
	}
	err = upgradeDB.BatchPutControversialUpgrade(ph2, up2, 0x00, batch)
	assert.NoError(t, err)
	err = upgradeDB.Commit()
	assert.NoError(t, err)

	// get upgrade code information from db, upgrade proposal is controversial.
	info, err = upgradeDB.GetByHeight(1000)
	assert.NoError(t, err)
	assert.Equal(t, *info, *up)

	// batch put proposal result: true
	result = payload.ProposalResult{
		ProposalHash: ph2,
		ProposalType: 0x0201,
		Result:       true,
	}
	err = upgradeDB.BatchPutUpgradeProposalResult(result, batch)
	assert.NoError(t, err)
	err = upgradeDB.Commit()
	assert.NoError(t, err)

	// get upgrade code information from db
	info, err = upgradeDB.GetByHeight(1000)
	assert.NoError(t, err)
	assert.Equal(t, *info, *up2)

	// third test, batch put upgrade proposal again with result: false
	// batch put controversial upgrade
	ph3 := common.Uint256{5}
	up3 := &payload.UpgradeCodeInfo{
		WorkingHeight:   10000,
		NodeVersion:     "0.9.0",
		NodeDownLoadUrl: "",
		NodeBinHash:     &common.Uint256{6},
		ForceUpgrade:    true,
	}
	err = upgradeDB.BatchPutControversialUpgrade(ph3, up3, 0x00, batch)
	assert.NoError(t, err)
	err = upgradeDB.Commit()
	assert.NoError(t, err)

	// get upgrade code information from db, upgrade proposal is controversial.
	info, err = upgradeDB.GetByHeight(10000)
	assert.NoError(t, err)
	assert.Equal(t, *info, *up2)

	// batch put proposal result: false
	result = payload.ProposalResult{
		ProposalHash: ph2,
		ProposalType: 0x0201,
		Result:       false,
	}
	err = upgradeDB.BatchPutUpgradeProposalResult(result, batch)
	assert.NoError(t, err)
	err = upgradeDB.Commit()
	assert.NoError(t, err)

	// get upgrade code information from db
	info, err = upgradeDB.GetByHeight(10000)
	assert.NoError(t, err)
	assert.Equal(t, *info, *up2)
}
