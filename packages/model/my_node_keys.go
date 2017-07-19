package model

type MyNodeKeys struct {
	ID         int32  `gorm:primary_key;not null`
	AddTime    int32  `gorm:not null`
	PublickKey []byte `gorm:not null`
	PrivateKey []byte `gorm:not null`
	Status     int8   `gorm:not null`
	MyTime     int32  `gorm:not null`
	Time       int32  `gorm:not null`
	BlockID    int64  `gorm:not null`
	RbID       int64  `gorm:not null`
}

func (mnk *MyNodeKeys) GetNodeWithMaxBlockID() error {
	if err := DBConn.Where("block_id = ?", "(SELECT max(block_id) FROM my_node_keys)").First(&mnk).Error; err != nil {
		return err
	}
	return nil
}

func (mnk *MyNodeKeys) Create() error {
	return DBConn.Create(mnk).Error
}

func (mnk *MyNodeKeys) GetZeroBlock(publicKey []byte) error {
	return DBConn.Where("block_id = 0 AND public_key = ", publicKey).First(mnk).Error
}
