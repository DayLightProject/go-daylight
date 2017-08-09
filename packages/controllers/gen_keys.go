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
	"fmt"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/template"
)

const nGenKeys = `gen_keys`

type genKeysPage struct {
	Data      *CommonPage
	Message   string
	Generated int64
	Used      int64
	Available int64
}

func init() {
	newPage(nGenKeys)
}

// GenKeys show information about generated and available keys
func (c *Controller) GenKeys() (string, error) {
	govAccount, _ := template.StateParam(int64(c.SessStateID), `gov_account`)
	if c.SessCitizenID != converter.StrToInt64(govAccount) {
		return ``, fmt.Errorf(`Access denied`)
	}
	testnetKey := &model.TestnetKey{}
	generated, err := testnetKey.GetGeneratedCount(c.SessCitizenID, c.SessStateID)
	if err != nil {
		return ``, err
	}
	available, err := testnetKey.GetAvailableCount(c.SessCitizenID, c.SessStateID)
	if err != nil {
		return ``, err
	}
	pageData := genKeysPage{Data: c.Data, Generated: generated, Available: available, Used: generated - available}
	return proceedTemplate(c, nGenKeys, &pageData)
}
