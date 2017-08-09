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
	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

type contractsPage struct {
	Lang               map[string]string
	WalletID           int64
	CitizenID          int64
	AllStateParameters []string
	StateSmartLaws     []map[string]string
	Global             string
}

// Contracts is a handle function for showing the list of contracts
func (c *Controller) Contracts() (string, error) {

	var err error

	global := c.r.FormValue("global")
	prefix := "global"
	if global == "" || global == "0" {
		prefix = c.StateIDStr
		global = "0"
	}

	contracts, err := model.GetAllSmartContracts(prefix)
	stateSmartLaws := make([]map[string]string, 0)
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	for ind, contract := range contracts {
		stateSmartLaws = append(stateSmartLaws, contract.ToMap())
		stateSmartLaws[ind][`wallet`] = converter.AddressToString(contract.WalletID)
	}
	var allStateParameters []string
	if global == "0" {
		stateParameter := &model.StateParameter{}
		list, err := stateParameter.GetAllStateParameters(prefix)
		if err != nil {
			return "", utils.ErrInfo(err)
		}
		allStateParameters := make([]string, 0)
		for _, param := range list {
			allStateParameters = append(allStateParameters, param.Name)
		}
	}

	TemplateStr, err := makeTemplate("contracts", "contracts", &contractsPage{
		Lang:               c.Lang,
		WalletID:           c.SessWalletID,
		CitizenID:          c.SessCitizenID,
		StateSmartLaws:     stateSmartLaws,
		Global:             global,
		AllStateParameters: allStateParameters})
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	return TemplateStr, nil
}
