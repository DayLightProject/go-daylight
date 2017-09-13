package model

type QueueTx struct {
	Hash     []byte `gorm:"primary_key;not null"`
	Data     []byte `gorm:"not null"`
	FromGate int    `gorm:"not null"`
}

func (qt *QueueTx) TableName() string {
	return "queue_tx"
}

func DeleteQueueTx() error {
	return DBConn.Delete(&QueueTx{}).Error
}

func (qt *QueueTx) DeleteTx() error {
	return DBConn.Delete(qt).Error
}

func (qt *QueueTx) Save(transaction *DbTransaction) error {
	db := DBConn
	if transaction != nil {
		db = transaction.conn
	}
	return db.Save(qt).Error
}

func (qt *QueueTx) Create() error {
	return DBConn.Create(qt).Error
}

func (qt *QueueTx) GetByHash(hash []byte) (bool, error) {
	query := DBConn.Where("hash = ?", hash).First(qt)
	if query.RecordNotFound() {
		return false, nil
	}
	return true, query.Error
}

func DeleteQueueTxByHash(transaction *DbTransaction, hash []byte) (int64, error) {
	db := DBConn
	if transaction != nil {
		db = transaction.conn
	}
	query := db.Exec("DELETE FROM queue_tx WHERE hash = ?", hash)
	return query.RowsAffected, query.Error
}

func DeleteQueuedTransaction(hash []byte) error {
	return DBConn.Exec("DELETE FROM queue_tx WHERE hash = ?", hash).Error
}

func GetQueuedTransactionsCount(hash []byte) (int64, error) {
	var rowsCount int64
	err := DBConn.Table("queue_tx").Where("hash = ?", hash).Count(&rowsCount).Error
	return rowsCount, err
}

func GetAllUnverifiedAndUnusedTransactions() ([]*QueueTx, error) {
	query := `SELECT *
		  FROM (
	              SELECT data,
	                     hash
	              FROM queue_tx
		      UNION
		      SELECT data,
			     hash
		      FROM transactions
		      WHERE verified = 0 AND used = 0
			)  AS x`
	rows, err := DBConn.Raw(query).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var data, hash []byte
	result := []*QueueTx{}
	for rows.Next() {
		if err := rows.Scan(&data, &hash); err != nil {
			return nil, err
		}
		result = append(result, &QueueTx{Data: data, Hash: hash})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
