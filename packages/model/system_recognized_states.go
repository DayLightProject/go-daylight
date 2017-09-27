package model

type SystemRecognizedState struct {
	Name             string `gorm:"not null;size:255"`
	StateID          int64  `gorm:"not null;primary_key"`
	Host             string `gorm:"not null;size:255"`
	NodePublicKey    []byte `gorm:"not null"`
	DelegateWalletID int64  `gorm:"not null"`
	DelegateStateID  int64  `gorm:"not null"`
	RbID             int64  `gorm:"not null"`
}

func (srs *SystemRecognizedState) GetState(stateID int64) (bool, error) {
	return isFound(DBConn.Where("state_id = ?", stateID).First(srs))
}

func (srs *SystemRecognizedState) IsDelegated(stateID int64) (bool, error) {
	found, err := srs.GetState(stateID)
	if !found {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return srs.DelegateStateID > 0 || srs.DelegateWalletID > 0, nil
}

func SystemRecognizedStatesCreateTable() error {
	return DBConn.CreateTable(&SystemRecognizedState{}).Error
}
