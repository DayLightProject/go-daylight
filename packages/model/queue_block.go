package model

type QueueBlock struct {
	Hash       []byte `gorm:"primary_key;not null"`
	BlockID    int64  `gorm:"not null"`
	FullNodeID int64  `gorm:"not null"`
}

func (qb *QueueBlock) Get() (bool, error) {
	return isFound(DBConn.First(qb))
}

func (qb *QueueBlock) Delete() error {
	return DBConn.Delete(qb).Error
}

func (qb *QueueBlock) Create() error {
	return DBConn.Create(qb).Error
}
