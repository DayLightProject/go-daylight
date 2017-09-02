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
	//	"github.com/EGaaS/go-egaas-mvp/packages/static"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
	//	"html/template"
	//	"fmt"
)

type loginECDSAPage struct {
	Lang    map[string]string
	Title   string
	States  map[string]string
	Private string
	Import  bool
	/*	MyModalIdName string
		UserID        int64
		PoolTechWorks int
		Community     bool
		Mobile        bool
		SignUp        bool
		Desktop bool*/
}

func (c *Controller) LoginECDSA() (string, error) {

	/*	var pool_tech_works int

		funcMap := template.FuncMap{
			"noescape": func(s string) template.HTML {
				return template.HTML(s)
			},
		}

		data, err := static.Asset("static/templates/login.html")
		if err != nil {
			return "", err
		}
		modal, err := static.Asset("static/templates/modal.html")
		if err != nil {
			return "", err
		}

		t := template.Must(template.New("template").Funcs(funcMap).Parse(string(data)))
		t = template.Must(t.Parse(string(modal)))

		b := new(bytes.Buffer)
		signUp := true
		// есть ли установочный пароль и был ли начально записан ключ
		if !c.Community {
			// Нельзя зарегистрироваться если в my_table уже есть статус
			if status, err := c.Single("SELECT status FROM my_table").String(); err == nil && status != "my_pending" {
				signUp = false
			}

			myKey, err := c.GetMyPublicKey(c.MyPrefix)
			if err != nil {
				return "", err
			}
		}
		//fmt.Println(c.Lang)
		// проверим, не идут ли тех. работы на пуле
		if len(c.NodeConfig["pool_admin_user_id"]) > 0 && c.NodeConfig["pool_admin_user_id"] != utils.Int64ToStr(c.UserId) && c.NodeConfig["pool_tech_works"] == "1" && c.Community {
			pool_tech_works = 1
		} else {
			pool_tech_works = 0
		}
		err = t.ExecuteTemplate(b, "login", &loginStruct{
			Lang:          c.Lang,
			MyModalIdName: "myModalLogin",
			UserID:        c.UserId,
			PoolTechWorks: pool_tech_works,
			Community:     c.Community,
			SignUp:        signUp,
			Desktop: utils.Desktop(),
			Mobile:        utils.Mobile()})
		if err != nil {
			return "", err
		}
		return b.String(), nil*/

	states := make(map[string]string)
	data, err := c.GetList(`SELECT id FROM system_states`).String()
	if err != nil {
		return ``, err
	}
	for _, id := range data {
		state_name, err := c.Single(`SELECT value FROM "` + id + `_state_parameters" WHERE name = 'state_name'`).String()
		if err != nil {
			return ``, err
		}
		states[id] = state_name
	}
	pkey := c.r.FormValue("pkey")

	TemplateStr, err := makeTemplate("login", "loginECDSA", &loginECDSAPage{
		Lang:    c.Lang,
		Title:   "Login",
		States:  states,
		Import:  len(pkey) > 0,
		Private: pkey,
		/*		MyWalletData:          MyWalletData,
				Title:                 "modalAnonym",
		*/})
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	return TemplateStr, nil
}
