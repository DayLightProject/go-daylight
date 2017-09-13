// Copyright 2016 The go-daylight Authors
// This file is part of the go-daylight library.
//
// The go-daylight library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-daylight library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-daylight library. If not, see <http://www.gnu.org/licenses/>.

package parser

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
	"github.com/EGaaS/go-egaas-mvp/packages/utils/tx"

	"github.com/EGaaS/go-egaas-mvp/packages/config/syspar"
	"gopkg.in/vmihailenco/msgpack.v2"
)

/*
Adding state tables should be spelled out in state settings
*/

type NewTableParser struct {
	*Parser
	NewTable *tx.NewTable
}

type Permissions struct {
	GeneralUpdate string            `json:"general_update"`
	Update        map[string]string `json:"update"`
	Insert        string            `json:"insert"`
	NewColumn     string            `json:"new_column"`
}

func (p *NewTableParser) Init() error {
	newTable := &tx.NewTable{}
	if err := msgpack.Unmarshal(p.TxBinaryData, newTable); err != nil {
		return p.ErrInfo(err)
	}
	p.NewTable = newTable
	return nil
}

func (p *NewTableParser) Validate() error {
	err := p.generalCheck(`add_table`, &p.NewTable.Header, map[string]string{})
	if err != nil {
		return p.ErrInfo(err)
	}

	// Check the system limits. You can not send more than X time a day this TX
	// ...

	// Check InputData
	verifyData := map[string][]interface{}{"int64": []interface{}{p.NewTable.Global}, "string": []interface{}{p.NewTable.Name}}
	err = p.CheckInputData(verifyData)
	if err != nil {
		return p.ErrInfo(err)
	}

	var cols [][]string
	err = json.Unmarshal([]byte(p.NewTable.Columns), &cols)
	if err != nil {
		return p.ErrInfo(err)
	}
	if len(cols) == 0 {
		return p.ErrInfo(`len(cols) == 0`)
	}
	if len(cols) > syspar.GetMaxColumns() {
		return fmt.Errorf(`Too many columns. Limit is %d`, syspar.GetMaxColumns())
	}
	var indexes int
	for _, data := range cols {
		if len(data) != 3 {
			return p.ErrInfo(`len(data)!=3`)
		}
		if data[1] != `text` && data[1] != `int64` && data[1] != `time` && data[1] != `hash` && data[1] != `double` && data[1] != `money` {
			return p.ErrInfo(`incorrect type`)
		}
		if data[2] == `1` {
			if data[1] == `text` {
				return p.ErrInfo(`incorrect index type`)
			}
			indexes++
		}
	}
	if indexes > syspar.GetMaxIndexes() {
		return fmt.Errorf(`Too many indexes. Limit is %d`, syspar.GetMaxIndexes())
	}

	prefix := converter.Int64ToStr(p.NewTable.Header.StateID)
	global, err := strconv.Atoi(p.NewTable.Global)
	if err != nil {
		return fmt.Errorf("Global is not int")
	}
	if global == 1 {
		prefix = `global`
	}

	t := &model.Table{}
	t.SetTablePrefix(prefix)
	exists, err := t.ExistsByName(prefix + "_" + p.NewTable.Name)
	if err != nil {
		return p.ErrInfo(err)
	}
	if exists {
		return p.ErrInfo(`table exists`)
	}

	// must be supplemented
	CheckSignResult, err := utils.CheckSign(p.PublicKeys, p.NewTable.ForSign(), p.NewTable.BinSignatures, false)
	if err != nil {
		return p.ErrInfo(err)
	}
	if !CheckSignResult {
		return p.ErrInfo("incorrect sign")
	}
	if err := p.AccessRights("new_table", false); err != nil {
		return p.ErrInfo(err)
	}

	return nil
}

func (p *NewTableParser) Action() error {
	prefix, err := GetTablePrefix(p.NewTable.Global, p.NewTable.Header.StateID)
	if err != nil {
		return p.ErrInfo(err)
	}
	tableName := prefix + "_" + p.NewTable.Name
	var cols [][]string
	json.Unmarshal([]byte(p.NewTable.Columns), &cols)

	indexes := make([]string, 0)
	mainCondition := `ContractConditions("MainCondition")`
	updateConditions := map[string]string{}

	colsSQL := ""
	for _, data := range cols {
		colType := ``
		colDef := ``
		switch data[1] {
		case "text":
			colType = `varchar(102400)`
		case "int64":
			colType = `bigint`
			colDef = `NOT NULL DEFAULT '0'`
		case "time":
			colType = `timestamp`
		case "hash":
			colType = `bytea`
		case "double":
			colType = `double precision`
		case "money":
			colType = `decimal (30, 0)`
			colDef = `NOT NULL DEFAULT '0'`
		}
		colsSQL += `"` + data[0] + `" ` + colType + " " + colDef + " ,\n"
		updateConditions[data[0]] = mainCondition
		if data[2] == "1" {
			indexes = append(indexes, data[0])
		}
	}
	err = model.CreateTable(p.DbTransaction, tableName, colsSQL)
	if err != nil {
		return p.ErrInfo(err)
	}

	for _, index := range indexes {
		err := model.CreateIndex(p.DbTransaction, tableName+"_"+index, tableName, index)
		if err != nil {
			return p.ErrInfo(err)
		}
	}
	perm := Permissions{
		GeneralUpdate: mainCondition,
		Update:        updateConditions,
		Insert:        mainCondition,
		NewColumn:     mainCondition,
	}
	jsonColumnsAndPerm, err := json.Marshal(perm)
	if err != nil {
		return p.ErrInfo(err)
	}
	t := &model.Table{
		Name: tableName,
		ColumnsAndPermissions: string(jsonColumnsAndPerm),
	}
	t.SetTablePrefix(prefix)
	err = t.Create(p.DbTransaction)
	if err != nil {
		return p.ErrInfo(err)
	}

	return nil
}

func (p *NewTableParser) Rollback() error {
	err := p.autoRollback()
	if err != nil {
		return p.ErrInfo(err)
	}
	prefix, err := GetTablePrefix(p.NewTable.Global, p.NewTable.Header.StateID)
	if err != nil {
		return p.ErrInfo(err)
	}
	tableName := prefix + "_" + p.NewTable.Name
	err = model.DropTable(tableName)
	if err != nil {
		return p.ErrInfo(err)
	}
	t := &model.Table{Name: tableName}
	t.SetTablePrefix(prefix)
	err = t.Delete()
	if err != nil {
		return p.ErrInfo(err)
	}
	return nil
}

func (p NewTableParser) Header() *tx.Header {
	return &p.NewTable.Header
}
