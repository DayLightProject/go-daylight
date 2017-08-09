package model

import "github.com/shopspring/decimal"

type DltTransaction struct {
	ID                     int64           `gorm:"primary_key;not null"`
	SenderWalletID         int64           `gorm:"not null"`
	RecepientWalletID      int64           `gorm:"not null"`
	RecepientWalletAddress string          `gorm:"not null;size:32"`
	Amount                 decimal.Decimal `gorm:"not null"`
	Comission              decimal.Decimal `gorm:"not null"`
	Time                   int64           `gorm:"not null"`
	Comment                string          `gorm:"not null"`
	BlockID                int64           `gorm:"not null"`
	RbID                   int64           `gorm:"not null"`
}

func (dt *DltTransaction) Create() error {
	return DBConn.Create(dt).Error
}

func (dt *DltTransaction) GetTransaction(senderWalletID, recipientWalletID int64, recipientWalletAddress string) error {
	return DBConn.Where("sender_wallet_id = ? OR recipient_wallet_id = ? OR recipient_wallet_address = ?",
		senderWalletID, recipientWalletID, recipientWalletAddress).First(dt).Error
}

func (dt *DltTransaction) GetIncomingTransactions(recipientWalletID int64) error {
	return DBConn.Where("recipient_wallet_id=?", recipientWalletID).First(dt).Error
}

func (dt *DltTransaction) GetCount(senderWalletID, recipientWalletID int64, recipientWalletAddress string) (int64, error) {
	var count int64
	err := DBConn.Where("sender_wallet_id = ? OR recipient_wallet_id = ? OR recipient_wallet_address = ?",
		senderWalletID, recipientWalletID, recipientWalletAddress).Count(&count).Error
	return count, err
}
