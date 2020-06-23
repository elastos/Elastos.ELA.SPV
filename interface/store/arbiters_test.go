package store

import (
	"bytes"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

func TestArbiters(t *testing.T) {
	dataDir := "/tmp"
	db, err := leveldb.OpenFile(filepath.Join(dataDir, "store"), nil)
	if err != nil {
		println(err.Error())
	}
	arbiters := NewArbiters(db)
	crcPublicKey := []string{
		"03C3A4A137EB63B05E9F14070639E680DF78616D70EE1BA52B0759236B4B698CDB",
		"03B97154758B8B1A044DB774A4A19E1591DC165A0FA24F74388FBDF0EFDB919CFA",
	}

	normalPublicKey := []string{
		"02713D40469D5AAF54FB622791936B4C21DABB62315041C292E2DCEC97AE1FBA69",
		"0276305327217E42CF6892536251354A029A9B814C3A65492B033504D29844CCB1",
		"03D3787D8904E82AFC1B83687AC0FEF919A1E96A1C78FB049904F553C3102049B4",
		"03DD46B1E064A0BD0BA9A0FEFE58E4703EB44189D137462F4FA5181EE42A8F61AE",
	}
	var crcs [][]byte
	for _, v := range crcPublicKey {
		crc, _ := hex.DecodeString(v)
		crcs = append(crcs, crc)
	}
	var normal [][]byte
	for _, v := range normalPublicKey {
		nor, _ := hex.DecodeString(v)
		normal = append(normal, nor)
	}
	err = arbiters.Put(402, crcs, normal)
	if err != nil {
		t.Errorf("put arbiter error %s", err.Error())
		return
	}

	crc, nor, err := arbiters.Get()
	if err != nil {
		t.Errorf("get arbiter error %s", err.Error())
		return
	}
	if !checkExist(crc, crcs) {
		t.Errorf("crc arbiter can not be found")
		return
	}
	if !checkExist(nor, normal) {
		t.Errorf("normal arbiter can not be found")
		return
	}

	err = arbiters.Put(403, crcs, normal)
	if err != nil {
		t.Errorf("put arbiter error %s", err.Error())
		return
	}
	crc, nor, err = arbiters.Get()
	if err != nil {
		t.Errorf("get arbiter error %s", err.Error())
		return
	}
	if !checkExist(crc, crcs) {
		t.Errorf("crc arbiter can not be found")
		return
	}
	if !checkExist(nor, normal) {
		t.Errorf("normal arbiter can not be found")
		return
	}

	append1, _ := hex.DecodeString("02ECF46B0DE8435DD4E4A93341763F3DDBF12C106C0BE00363B114EFE90F5D2F58")
	crcs = append(crcs, append1)

	err = arbiters.Put(405, crcs, normal)
	if err != nil {
		t.Errorf("put arbiter error %s", err.Error())
		return
	}
	crc, nor, err = arbiters.Get()
	if err != nil {
		t.Errorf("get arbiter error %s", err.Error())
		return
	}
	if !checkExist(crc, crcs) {
		t.Errorf("crc arbiter can not be found")
		return
	}
	if !checkExist(nor, normal) {
		t.Errorf("normal arbiter can not be found")
		return
	}

	err = arbiters.Put(407, crcs, normal)
	if err != nil {
		t.Errorf("put arbiter error %s", err.Error())
		return
	}
	crc, nor, err = arbiters.Get()
	if err != nil {
		t.Errorf("get arbiter error %s", err.Error())
		return
	}
	if !checkExist(crc, crcs) {
		t.Errorf("crc arbiter can not be found")
		return
	}
	if !checkExist(nor, normal) {
		t.Errorf("normal arbiter can not be found")
		return
	}

}

func checkExist(target [][]byte, src [][]byte) bool {
	for _, v := range target {
		var find bool
		for _, _v := range src {
			if bytes.Equal(v, _v) {
				find = true
				break
			}
		}
		if find == false {
			return false
		}
	}
	return true
}
