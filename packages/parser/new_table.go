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

	"github.com/EGaaS/go-egaas-mvp/packages/consts"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

/*
Adding state tables should be spelled out in state settings
*/

// NewTableInit initializes NewTable transaction
func (p *Parser) NewTableInit() error {

	fields := []map[string]string{{"global": "int64"}, {"table_name": "string"}, {"columns": "string"}, {"sign": "bytes"}}
	err := p.GetTxMaps(fields)
	if err != nil {
		return p.ErrInfo(err)
	}
	return nil
}

// NewTableFront checks conditions of NewTable transaction
func (p *Parser) NewTableFront() error {

	err := p.generalCheck(`add_table`)
	if err != nil {
		return p.ErrInfo(err)
	}

	// Check the system limits. You can not send more than X time a day this TX
	// ...

	// Check InputData
	verifyData := map[string]string{"global": "int64", "table_name": "string"}
	err = p.CheckInputData(verifyData)
	if err != nil {
		return p.ErrInfo(err)
	}

	var cols [][]string
	err = json.Unmarshal([]byte(p.TxMaps.String["columns"]), &cols)
	if err != nil {
		return p.ErrInfo(err)
	}
	if len(cols) == 0 {
		return p.ErrInfo(`len(cols) == 0`)
	}
	if len(cols) > consts.MAX_COLUMNS {
		return fmt.Errorf(`Too many columns. Limit is %d`, consts.MAX_COLUMNS)
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
	if indexes > consts.MAX_INDEXES {
		return fmt.Errorf(`Too many indexes. Limit is %d`, consts.MAX_INDEXES)
	}

	prefix := p.TxStateIDStr
	table := p.TxStateIDStr + `_tables`
	if p.TxMaps.Int64["global"] == 1 {
		table = `global_tables`
		prefix = `global`
	}

	exists, err := p.GetRecordsCountWhereName(`SELECT count(*) FROM "`+table+`" WHERE name = ?`, prefix+`_`+p.TxMaps.String["table_name"])
	log.Debug(`SELECT count(*) FROM "` + table + `" WHERE name = ?`)
	if err != nil {
		return p.ErrInfo(err)
	}
	if exists > 0 {
		return p.ErrInfo(`table exists`)
	}

	// must be supplemented
	forSign := fmt.Sprintf("%s,%s,%d,%d,%s,%s,%s", p.TxMap["type"], p.TxMap["time"], p.TxCitizenID, p.TxStateID, p.TxMap["global"], p.TxMap["table_name"], p.TxMap["columns"])
	CheckSignResult, err := utils.CheckSign(p.PublicKeys, forSign, p.TxMap["sign"], false)
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

// NewTable proceeds NewTable transaction
func (p *Parser) NewTable() error {
	var cols [][]string
	json.Unmarshal([]byte(p.TxMaps.String["columns"]), &cols)

	tableName := `global_` + p.TxMaps.String["table_name"]
	if p.TxMaps.Int64["global"] == 0 {
		tableName = p.TxStateIDStr + `_` + p.TxMaps.String["table_name"]
	}

	err := p.CreateNewTable(cols, tableName, p.TxMaps.Int64["global"], p.TxStateIDStr)
	if err != nil {
		return p.ErrInfo(err)
	}
	return nil
}

// NewTableRollback rollbacks NewTable transaction
func (p *Parser) NewTableRollback() error {
	err := p.autoRollback()
	if err != nil {
		return p.ErrInfo(err)
	}

	tableName := `global_` + p.TxMaps.String["table_name"]
	if p.TxMaps.Int64["global"] == 0 {
		tableName = p.TxStateIDStr + `_` + p.TxMaps.String["table_name"]
	}
	err = p.AnotherDropTable(tableName)

	prefix := `global`
	if p.TxMaps.Int64["global"] == 0 {
		prefix = p.TxStateIDStr
	}
	err = p.DeleteFromTable(prefix, tableName)
	if err != nil {
		return p.ErrInfo(err)
	}
	return nil
}
