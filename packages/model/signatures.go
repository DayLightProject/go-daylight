package model

import (
	"strconv"
)

type Signature struct {
	tableName  string
	Name       string `gorm:"primary_key;not null;size:255"`
	Value      string `gorm:"not null;type:jsonb(PostgreSQL)"`
	Conditions string `gorm:"not null"`
	RbID       int64  `gorm:"not null"`
}

func (s *Signature) SetTablePrefix(prefix string) {
	s.tableName = prefix + "_signatures"
}

func (s *Signature) TableName() string {
	return s.tableName
}

func (s *Signature) Get(name string) (bool, error) {
	return isFound(DBConn.Where("name = ?", name).First(s))
}

func (s *Signature) GetAllOredered(prefix string) ([]Signature, error) {
	result := make([]Signature, 0)
	err := DBConn.Table(prefix + "_signatures").Order("name").Find(&result).Error
	return result, err
}

func (s *Signature) ToMap() map[string]string {
	result := make(map[string]string, 0)
	result["name"] = s.Name
	result["value"] = s.Value
	result["conditions"] = s.Conditions
	result["rb_id"] = strconv.FormatInt(s.RbID, 10)
	return result
}

func CreateSignaturesStateTable(stateID string) error {
	return DBConn.Exec(`CREATE TABLE "` + stateID + `_signatures" (
				"name" varchar(100)  NOT NULL DEFAULT '',
				"value" jsonb,
				"conditions" text  NOT NULL DEFAULT '',
				"rb_id" bigint NOT NULL DEFAULT '0'
				);
				ALTER TABLE ONLY "` + stateID + `_signatures" ADD CONSTRAINT "` + stateID + `_signatures_pkey" PRIMARY KEY (name);
		    `).Error
}
