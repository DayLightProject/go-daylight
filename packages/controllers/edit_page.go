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
	"encoding/json"
	"fmt"
	"time"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

type editPagePage struct {
	Alert           string
	Lang            map[string]string
	WalletID        int64
	CitizenID       int64
	TxType          string
	TxTypeID        int64
	TimeNow         int64
	Name            string
	Block           bool
	DataMenu        map[string]string
	DataPage        map[string]string
	DataPageHistory []map[string]string
	AllMenu         []map[string]string
	StateID         int64
	Global          string
}

// EditPage is a controller for editing pages
func (c *Controller) EditPage() (string, error) {

	txType := "EditPage"
	timeNow := time.Now().Unix()

	var err error

	global := c.r.FormValue("global")
	prefix := c.StateIDStr
	if global == "1" {
		prefix = "global"
	} else {
		global = "0"
	}

	var name string
	if utils.CheckInputData(c.r.FormValue("name"), "string") {
		name = c.r.FormValue("name")
	}

	var dataPageMain map[string]string
	var dataPageHistory []map[string]string
	var rbID int64
	var block bool
	for i := 0; i < 30; i++ {
		if i == 0 {
			page := &model.Page{}
			page.SetTablePrefix(prefix)
			err = page.Get(name)
			if err != nil {
				return "", utils.ErrInfo(err)
			}
			dataPage := page.ToMap()
			if len(dataPage[`conditions`]) == 0 {
				dataPage[`conditions`] = "ContractConditions(`MainCondition`)"
			}

			rbID = page.RbID
			dataPageMain = dataPage
			block = dataPage[`menu`] == `0`
		} else {
			rollback := &model.Rollback{}
			err := rollback.Get(rbID)
			if err != nil {
				return "", utils.ErrInfo(err)
			}
			data := rollback.ToMap()
			var messageMap map[string]string
			json.Unmarshal([]byte(data["data"]), &messageMap)
			fmt.Printf("%s", messageMap)
			rbID = converter.StrToInt64(messageMap["rb_id"])
			messageMap["block_id"] = data["block_id"]
			dataPageHistory = append(dataPageHistory, messageMap)
		}
		if rbID == 0 {
			break
		}
	}

	menu := &model.Menu{}
	menu.SetTablePrefix(prefix)
	err = menu.Get(dataPageMain["menu"])
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	dataMenu := menu.ToMap()

	menus, err := menu.GetAll(prefix)
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	allMenu := make([]map[string]string, 0)
	for _, m := range menus {
		allMenu = append(allMenu, m.ToMap())
	}

	TemplateStr, err := makeTemplate("edit_page", "editPage", &editPagePage{
		Alert:           c.Alert,
		Lang:            c.Lang,
		Global:          global,
		WalletID:        c.SessWalletID,
		CitizenID:       c.SessCitizenID,
		TimeNow:         timeNow,
		TxType:          txType,
		TxTypeID:        utils.TypeInt(txType),
		StateID:         c.SessStateID,
		AllMenu:         allMenu,
		DataMenu:        dataMenu,
		DataPage:        dataPageMain,
		Block:           block,
		DataPageHistory: dataPageHistory})
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	return TemplateStr, nil
}
