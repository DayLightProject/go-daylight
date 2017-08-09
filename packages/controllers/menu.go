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
	"strings"

	"github.com/EGaaS/go-egaas-mvp/packages/consts"
	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/language"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/textproc"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

const nMenu = `menu`

// LangInfo is a structure for language name and code
type LangInfo struct {
	Title string
	Code  string
}

type menuPage struct {
	Data          *CommonPage
	Menu          string
	MainMenu      bool
	CanCitizen    bool
	States        string
	StateName     string
	StateFlag     string
	CitizenName   string
	CitizenAvatar string
	UpdVer        string
	Btc           string
	LogoExt       string
	Langs         []LangInfo
	CountLangs    int
	DefLang       string
}

func init() {
	newPage(nMenu)
}

/*
func ReplaceMenu(menu string) string {
	qrx := regexp.MustCompile(`(?is)\[([\w\s]*)\]\(glob.([\w\s]*)\){?([\w\d\s""'',:]*)?}?`)
	menu = qrx.ReplaceAllString(menu, "<li class='citizen_$2'><a href='#' onclick=\"load_template('$2',{global:1, $3});\"><span>$1</span></a></li>")
	qrx = regexp.MustCompile(`(?is)\[([\w\s]*)\]\(([\w\s]*)\){?([\w\d\s"",:]*)?}?`)
	menu = qrx.ReplaceAllString(menu, "<li class='citizen_$2'><a href='#' onclick=\"load_template('$2',{$3});\"><span>$1</span></a></li>")
	qrx = regexp.MustCompile(`(?is)\[([\w\s]*)\]\(sys.([\w\s]*)\){?([\w\d\s"",:]*)?}?`)
	return qrx.ReplaceAllString(menu, "<li class='citizen_$2'><a href='#' onclick=\"load_page('$2', {$3});\"><span>$1</span></a></li>")
}
*/

// Menu is controller for displaying the left menu
func (c *Controller) Menu() (string, error) {
	var (
		err                          error
		updver, stateName, stateFlag string
		isMain                       bool
	)
	citizen := &model.Citizen{}
	menu := &model.Menu{}
	if strings.HasPrefix(c.r.Host, `localhost`) {
		updinfo, err := utils.GetUpdVerAndURL(consts.UPD_AND_VER_URL)
		if err == nil && updinfo != nil {
			updver = updinfo.Version
		}
	}

	systemStates := &model.SystemState{}
	canCitizen, _ := systemStates.GetCount()
	if c.StateIDStr != "" {
		params := make(map[string]string)
		params[`state_id`] = c.StateIDStr
		params[`accept_lang`] = c.r.Header.Get(`Accept-Language`)

		menu.SetTablePrefix(c.StateIDStr)
		err = menu.Get("main_menu")
		if err != nil {
			err = menu.Get("menu_default")
			if err != nil {
				return "", err
			}
		} else {
			isMain = true
		}

		stateParameter := &model.StateParameter{}
		stateParameter.SetTablePrefix(c.StateIDStr)
		err := stateParameter.GetByName("state_name")
		if err != nil {
			return "", err
		}
		stateName = stateParameter.Value

		err = stateParameter.GetByName("state_flag")
		if err != nil {
			return "", err
		}
		stateFlag = stateParameter.Value

		citizen := &model.Citizen{}
		citizen.SetTablePrefix(c.StateIDStr)
		err = citizen.Get(c.SessCitizenID)
		if err != nil {
			log.Error("%v", err)
		}

		if err != nil {
			log.Error("%v", err)
		}
		//		menu = ReplaceMenu(menu)
		menu.Value = language.LangMacro(textproc.Process(menu.Value, &params), converter.StrToInt(c.StateIDStr), params[`accept_lang`])
	}
	var langs []LangInfo
	if len(language.LangList) > 0 {
		for _, val := range language.LangList {
			if val == `en` {
				langs = append(langs, LangInfo{Title: `English (UK)`, Code: `gb`})
			}
			if val == `nl` {
				langs = append(langs, LangInfo{Title: `Nederlands (NL)`, Code: `nl`})
			}
		}
	} else {
		langs = []LangInfo{{Title: `English (UK)`, Code: `gb`},
			{Title: `Nederlands (NL)`, Code: `nl`}}
	}
	states, _ := c.AjaxStatesList()
	return proceedTemplate(c, nMenu, &menuPage{Data: c.Data, Menu: menu.Value, MainMenu: isMain, CanCitizen: canCitizen > 0,
		States: states, StateName: stateName, StateFlag: stateFlag, CitizenName: string(citizen.Name), LogoExt: utils.LogoExt,
		CitizenAvatar: citizen.Avatar, UpdVer: updver, Btc: GetBtc(), Langs: langs, CountLangs: len(langs), DefLang: langs[0].Code})
}
