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

package apiv2

import (
	"net/http"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/templatev2"
)

type contentResult struct {
	Menu  string `json:"menu,omitempty"`
	Title string `json:"title,omitempty"`
	Tree  string `json:"tree"`
}

func initVars(r *http.Request, data *apiData) *map[string]string {
	vars := make(map[string]string)
	for name := range r.Form {
		vars[name] = r.FormValue(name)
	}
	vars[`state`] = converter.Int64ToStr(data.state)
	vars[`wallet`] = converter.Int64ToStr(data.wallet)
	return &vars
}

func getPage(w http.ResponseWriter, r *http.Request, data *apiData) error {
	/*	var result contentResult

		result = contentResult{
			Tree: `[{"type":"fn","name":"Title","data":["State info"]},{"type":"fn","name":"Navigation","data":[[{"type":"fn","name":"LiTemplate","data":["government","Government"]}],"State info"]},{"type":"block","name":"Divs","data":["md-4","panel panel-default elastic center"],"children":[{"type":"block","name":"Divs","data":["panel-body"],"children":[{"type":"fn","name":"IfParams","data":["#flag#==\"\"",[{"type":"fn","name":"Image","data":["static/img/noflag.svg","No flag","img-responsive"]}],[{"type":"fn","name":"Image","data":["#flag#","Flag","img-responsive"]}]]},{"type":"fn","name":"DivsEnd"}]},{"type":"fn","name":"DivsEnd"}]}]`,
		}*/
	var query string
	params := make(map[string]string)
	/*	for name, val := range data.params {
		params[name] = val
	}*/
	page := data.params[`name`].(string)
	/*	if page == `body` {
		params[`autobody`] = r.FormValue("body")
	}*/
	params[`accept_lang`] = r.Header.Get(`Accept-Language`)
	query = `SELECT value,menu FROM "` + converter.Int64ToStr(data.state) + `_pages" WHERE name = ?`
	pattern, err := model.GetOneRow(query, page).String()
	if err != nil {
		return err
	}
	if len(pattern) == 0 {
		return errorAPI(w, `E_NOTFOUND`, http.StatusNotFound)
	}
	ret := templatev2.Template2JSON(pattern[`value`], false, initVars(r, data))
	data.result = &contentResult{Tree: string(ret), Menu: pattern[`menu`]}
	return nil
}

func getMenu(w http.ResponseWriter, r *http.Request, data *apiData) error {
	var query string
	params := make(map[string]string)
	params[`accept_lang`] = r.Header.Get(`Accept-Language`)
	query = `SELECT value, title FROM "` + converter.Int64ToStr(data.state) + `_menu" WHERE name = ?`
	pattern, err := model.GetOneRow(query, data.params[`name`].(string)).String()
	if err != nil {
		return errorAPI(w, err, http.StatusBadRequest)
	}
	if len(pattern) == 0 {
		return errorAPI(w, `E_NOTFOUND`, http.StatusNotFound)
	}

	ret := templatev2.Template2JSON(pattern[`value`], false, initVars(r, data))
	data.result = &contentResult{Tree: string(ret), Title: pattern[`title`]}
	return nil
}

func jsonContent(w http.ResponseWriter, r *http.Request, data *apiData) error {
	ret := templatev2.Template2JSON(data.params[`template`].(string), false, initVars(r, data))
	data.result = &contentResult{Tree: string(ret)}
	return nil
}
