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

package controllers

import (
	"time"

	"github.com/EGaaS/go-egaas-mvp/packages/config/syspar"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

type editTablePage struct {
	Alert                 string
	Lang                  map[string]string
	WalletID              int64
	CitizenID             int64
	TableName             string
	TxType                string
	TxTypeID              int64
	TimeNow               int64
	CanColumns            bool
	TableData             map[string]string
	ColumnsAndPermissions []map[string]string
	StateID               int64
	TablePermission       map[string]string
	Global                string
}

// EditTable is a controller for editing table
func (c *Controller) EditTable() (string, error) {

	var err error

	txType := "EditTable"
	timeNow := time.Now().Unix()

	var tableName string
	if utils.CheckInputData(c.r.FormValue("name"), "string") {
		tableName = c.r.FormValue("name")
	}

	prefix, err := utils.GetPrefix(tableName, c.StateIDStr)
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	global := ""
	if prefix == "global" {
		global = "1"
	}

	table := &model.Table{}
	table.SetTablePrefix(prefix)
	err = table.Get(tableName)

	if err != nil {
		return "", utils.ErrInfo(err)
	}

	tableData := table.ToMap()

	tablePermission, err := table.GetTablePermissions(prefix, tableName)
	if err != nil {
		return "", utils.ErrInfo(err)
	}

	columnsAndPermissions, err := table.GetColumnsAndPermissions(prefix, tableName)
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	list := make([]map[string]string, 0)
	for key, value := range columnsAndPermissions {
		list = append(list, map[string]string{`name`: key, `perm`: value, `type`: model.GetColumnType(tableName, key)})
	}

	count, err := model.GetColumnsCount(tableName)
	if err != nil {
		return "", utils.ErrInfo(err)
	}

	TemplateStr, err := makeTemplate("edit_table", "editTable", &editTablePage{
		Alert:                 c.Alert,
		Lang:                  c.Lang,
		WalletID:              c.SessWalletID,
		CitizenID:             c.SessCitizenID,
		TableName:             tableName,
		TimeNow:               timeNow,
		TxType:                txType,
		TxTypeID:              utils.TypeInt(txType),
		StateID:               c.SessStateID,
		CanColumns:            count < int64(syspar.GetMaxColumns()+2),
		Global:                global,
		TablePermission:       tablePermission,
		ColumnsAndPermissions: list,
		TableData:             tableData})
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	return TemplateStr, nil
}
