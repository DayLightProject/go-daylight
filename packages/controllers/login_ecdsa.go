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
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/EGaaS/go-egaas-mvp/packages/config"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

type loginECDSAPage struct {
	Lang        map[string]string
	Title       string
	Key         string
	States      string
	State       int64
	OneCountry  int64
	PrivCountry bool
	Import      bool
	Local       bool
	Private     string
}

// LoginECDSA is a control for the login page
func (c *Controller) LoginECDSA() (string, error) {
	var err error
	var private []byte

	local := strings.HasPrefix(c.r.Host, `localhost`)
	if config.ConfigIni["public_node"] != "1" || local {
		private, _ = ioutil.ReadFile(filepath.Join(*utils.Dir, `PrivateKey`))
	}

	states, _ := c.AjaxStatesList()
	key := c.r.FormValue("key")
	pkey := c.r.FormValue("pkey")
	state := c.r.FormValue("state")
	if len(key) > 0 || len(pkey) > 0 {
		c.Logout()
	}
	if len(pkey) > 0 {
		private = []byte(pkey)
	}
	var stateID int64
	if len(state) > 0 {
		stateID, err = strconv.ParseInt(state, 10, 64)
	}
	TemplateStr, err := makeTemplate("login", "loginECDSA", &loginECDSAPage{
		Lang:        c.Lang,
		Title:       "Login",
		States:      states,
		State:       stateID,
		Key:         key,
		Local:       local,
		Import:      len(pkey) > 0,
		OneCountry:  utils.OneCountry,
		PrivCountry: utils.PrivCountry,
		Private:     string(private),
	})
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	return TemplateStr, nil
}
