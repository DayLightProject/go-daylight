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
	//	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/template"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

const aCitizenInfo = `ajax_citizen_info`

/*
type FieldInfo struct {
	Name     string `json:"name"`
	HTMLType string `json:"htmlType"`
	TxType   string `json:"txType"`
	Title    string `json:"title"`
	Value    string `json:"value"`
}*/

// CitizenInfoJSON is a structure for the answer of ajax_citizen_info ajax request
type CitizenInfoJSON struct {
	Result bool   `json:"result"`
	Error  string `json:"error"`
}

func init() {
	newPage(aCitizenInfo, `json`)
}

// AjaxCitizenInfo is a controller of ajax_citizen_info request
func (c *Controller) AjaxCitizenInfo() interface{} {
	var (
		result CitizenInfoJSON
		err    error
		data   map[string]string
	)
	c.w.Header().Add("Access-Control-Allow-Origin", "*")
	stateCode := converter.StrToInt64(c.r.FormValue(`stateId`))
	_, err = c.CheckStateName(stateCode)
	c.r.ParseMultipartForm(16 << 20) // Max memory 16 MiB
	formdata := c.r.MultipartForm
	defer formdata.RemoveAll()

	//	fmt.Println(`FORM Start`, formdata)
	//field, err := c.Single(`SELECT value FROM ` + utils.Int64ToStr(stateCode) + `_state_parameters where name='citizen_fields'`).String()
	field, err := `[{"name":"name", "htmlType":"textinput", "txType":"string", "title":"First Name"},
{"name":"lastname", "htmlType":"textinput", "txType":"string", "title":"Last Name"},
{"name":"birthday", "htmlType":"calendar", "txType":"string", "title":"Birthday"},
{"name":"photo", "htmlType":"file", "txType":"binary", "title":"Photo"}
]`, nil
	vals := make(map[string]string)
	time := c.r.FormValue(`time`)
	walletID := c.r.FormValue(`walletId`)

	if err == nil {
		var (
			fields    []template.FieldInfo
			sign      []byte
			checkSign bool
		)
		if err = json.Unmarshal([]byte(field), &fields); err == nil {
			for _, ifield := range fields {
				if ifield.HTMLType != `file` {
					vals[ifield.Name] = c.r.FormValue(ifield.Name)
				}
			}

			data, err = c.GetWalletPublickKeyFromString(walletID)
			if err == nil {
				var PublicKeys [][]byte
				PublicKeys = append(PublicKeys, []byte(data["public_key_0"]))
				forSign := fmt.Sprintf("CitizenInfo,%s,%s", time, walletID)
				sign, err = hex.DecodeString(c.r.FormValue(`signature1`))

				if err == nil {
					checkSign, err = utils.CheckSign(PublicKeys, forSign, sign, true)
					if err == nil && !checkSign {
						err = fmt.Errorf(`incorrect signature`)
					}
				}
			}
		}
	}
	if err == nil {
		data, err = c.GetCitizenshipRequestsFull(converter.Int64ToStr(stateCode), converter.StrToInt64(walletID))
		if err != nil || data == nil || len(data) == 0 {
			err = fmt.Errorf(`unknown request for wallet %s`, walletID)
		}
	}
	if err != nil {
		result.Error = err.Error()
	} else {
		result.Result = true
	}

	return result
}
