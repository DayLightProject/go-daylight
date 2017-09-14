package model

import (
	"strconv"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/jinzhu/gorm"
)

type Block struct {
	ID         int64  `gorm:"primary_key;not_null"`
	Hash       []byte `gorm:"not null"`
	Data       []byte `gorm:"not null"`
	StateID    int64  `gorm:"not null"`
	WalletID   int64  `gorm:"not null"`
	Time       int64  `gorm:"not null"`
	Tx         int32  `gorm:"not null"`
	MaxMinerID int32  `gorm:"not null"`
}

func GetBlockchain(startBlockID int64, endblockID int64) ([]Block, error) {
	var err error
	blockchain := new([]Block)
	if endblockID > 0 {
		err = DBConn.Model(&Block{}).Order("id asc").Where("id > ? AND id <= ?", startBlockID, endblockID).Find(&blockchain).Error
	} else {
		err = DBConn.Model(&Block{}).Order("id asc").Where("id > ?", startBlockID).Find(&blockchain).Error
	}
	if err != nil {
		return nil, err
	}
	return *blockchain, nil
}

func (Block) TableName() string {
	return "block_chain"
}

func (b *Block) IsExists() (bool, error) {
	query := DBConn.First(b)
	if query.Error == gorm.ErrRecordNotFound {
		return false, nil
	}
	return !query.RecordNotFound(), handleError(query.Error)
}

func (b *Block) IsExistsID(blockID int64) (bool, error) {
	query := DBConn.Where("id = ?").First(b)
	if query.Error == gorm.ErrRecordNotFound {
		return false, nil
	}
	return !query.RecordNotFound(), handleError(query.Error)
}

func (b *Block) Create(transaction *DbTransaction) error {
	return getDB(transaction).Create(b).Error
}

func (b *Block) GetBlock(blockID int64) error {
	return handleError(DBConn.Where("id = ?", blockID).First(b).Error)
}

func (b *Block) GetMaxBlock() error {
	return handleError(DBConn.Last(b).Error)
}

func (b *Block) GetBlocksFrom(startFromID int64, ordering string) ([]Block, error) {
	var err error
	blockchain := new([]Block)
	err = DBConn.Order("id "+ordering).Where("id > ?", startFromID).Find(&blockchain).Error
	return *blockchain, handleError(err)
}

func (b *Block) GetBlocks(startFromID int64, limit int32) ([]Block, error) {
	var err error
	blockchain := new([]Block)
	if startFromID > 0 {
		err = DBConn.Order("id desc").Limit(limit).Where("id > ?", startFromID).Find(&blockchain).Error
	} else {
		err = DBConn.Order("id desc").Limit(limit).Find(&blockchain).Error
	}
	return *blockchain, handleError(err)
}

func (b *Block) Delete() error {
	return DBConn.Delete(b).Error
}

func (b *Block) DeleteById(transaction *DbTransaction, id int64) error {
	return getDB(transaction).Where("id = ?", id).Delete(Block{}).Error
}

func (b *Block) DeleteChain() error {
	return DBConn.Where("id > ?", b.ID).Delete(Block{}).Error
}

func (b *Block) GetLastBlockData() (map[string]int64, error) {
	result := make(map[string]int64)
	confirmation := &Confirmation{}
	err := confirmation.GetMaxGoodBlock()
	if err != nil {
		return result, err
	}
	if confirmation.BlockID == 0 {
		confirmation.BlockID = 1
	}

	err = b.GetBlock(confirmation.BlockID)
	if err != nil || b.ID == 0 {
		return result, err
	}
	result["blockId"] = int64(converter.BinToDec(b.Data[1:5]))
	result["lastBlockTime"] = int64(converter.BinToDec(b.Data[5:9]))
	return result, nil
}

func (b *Block) ToMap() map[string]string {
	result := make(map[string]string)
	result["hash"] = string(converter.BinToHex(b.Hash))
	result["state_id"] = strconv.FormatInt(b.StateID, 10)
	result["wallet_id"] = converter.AddressToString(b.WalletID)
	result["time"] = strconv.FormatInt(b.Time, 10)
	result["tx"] = strconv.FormatInt(int64(b.Tx), 10)
	result["id"] = strconv.FormatInt(b.ID, 10)
	return result
}

func BlockChainCreateTable() error {
	return DBConn.CreateTable(&Block{}).Error
}
