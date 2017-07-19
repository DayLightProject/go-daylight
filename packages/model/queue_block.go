package model

type QueueBlocks struct {
	Hash       []byte `gorm:"primary_key;not null"`
	BlockID    int64  `gorm:"not null"`
	FullNodeID int64  `gorm:"not null"`
}

func (qb *QueueBlocks) GetQueueBlock() error {
	return DBConn.First(&qb).Error
}

func (qb *QueueBlocks) Delete() error {
	return DBConn.Delete(qb).Error
}

func (qb *QueueBlocks) Create() error {
	return DBConn.Create(qb).Error
}
