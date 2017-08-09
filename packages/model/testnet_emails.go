package model

type TestnetEmail struct {
	ID       int64  `gorm:"primary_key;not null"`
	Email    string `gorm:"not null;size:128"`
	Country  string `gorm:"not null;size:128"`
	Currency string `gorm:"not null;size:64"`
	Private  []byte `gorm:"not null"`
	Wallet   int64  `gorm:"not null"`
	Status   int32  `gorm:"not null"`
	Code     int32  `gorm:"not null"`
	Validate int32  `gorm:"not null"`
}

func (TestnetEmail) TableName() string {
	return "testnet_emails"
}

func (te *TestnetEmail) Get(ID int64) error {
	return DBConn.Where("id = ?").First(te).Error
}

func (te *TestnetEmail) Save() error {
	return DBConn.Save(te).Error
}
