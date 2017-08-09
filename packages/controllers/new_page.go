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

	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

// NewPage creates a new page
func (c *Controller) NewPage() (string, error) {

	txType := "NewPage"
	timeNow := time.Now().Unix()

	global := c.r.FormValue("global")
	prefix := c.StateIDStr
	if global == "1" {
		prefix = "global"
	} else {
		global = "0"
	}

	menu := &model.Menu{}
	menus, err := menu.GetAll(prefix)
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	allMenu := make([]map[string]string, 0)
	for _, m := range menus {
		allMenu = append(allMenu, m.ToMap())
	}

	TemplateStr, err := makeTemplate("edit_page", "editPage", &editPagePage{
		Alert:     c.Alert,
		Lang:      c.Lang,
		Global:    global,
		WalletID:  c.SessWalletID,
		CitizenID: c.SessCitizenID,
		TimeNow:   timeNow,
		TxType:    txType,
		Block:     c.r.FormValue("block") == `1`,
		TxTypeID:  utils.TypeInt(txType),
		StateID:   c.SessStateID,
		AllMenu:   allMenu,
		Name:      c.r.FormValue("name"),
		DataMenu:  map[string]string{},
		DataPage:  map[string]string{`conditions`: "ContractConditions(`MainCondition`)"}})
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	return TemplateStr, nil
}
