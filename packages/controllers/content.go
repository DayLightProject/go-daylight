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
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/EGaaS/go-egaas-mvp/packages/config"
	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/static"
	tpl "github.com/EGaaS/go-egaas-mvp/packages/template"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

var (
	passMutex = sync.Mutex{}
	passUpd   = time.Now()
	passwords = make(map[string]bool)
	alphabet  = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
)

func genPass(length int) string {
	ret := make([]byte, length)
	for i := range ret {
		ret[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(ret)
}

// IsPassValid checks password and update passwords list
func IsPassValid(pass, psw string) bool {
	passMutex.Lock()
	defer passMutex.Unlock()

	if len(passwords) == 0 || passUpd.Add(5*time.Minute).Before(time.Now()) {

		filename := *utils.Dir + `/passlist.txt`
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			out := make([]string, 1000)
			out[0] = pass
			for i := 1; i < 1000; i++ {
				out[i] = genPass(6)
			}
			ioutil.WriteFile(filename, []byte(strings.Join(out, "\r\n")), 0644)
		}
		if list, err := ioutil.ReadFile(filename); err == nil && len(list) > 0 {
			for key := range passwords {
				passwords[key] = false
			}
			out := strings.Split(string(list), "\r\n")
			for i := range out {
				plist := strings.SplitN(out[i], `,`, 2)
				if len(plist[0]) > 0 {
					passwords[plist[0]] = true
				}
			}
			passUpd = time.Now()
		}
	}
	return passwords[psw]
}

// Content is the main controller
func Content(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if e := recover(); e != nil {
			log.Error("Content Recovered", fmt.Sprintf("%s: %s", e, debug.Stack()))
			fmt.Println("Content Recovered", fmt.Sprintf("%s: %s", e, debug.Stack()))
		}
	}()
	var err error

	w.Header().Set("Content-type", "text/html")

	log.Debug("Content")
	sess, err := globalSessions.SessionStart(w, r)
	if err != nil {
		log.Error("%v", err)
	}
	defer sess.SessionRelease(w)
	sessWalletID := GetSessWalletID(sess)
	sessCitizenID := GetSessCitizenID(sess)
	sessStateID := GetSessInt64("state_id", sess)
	sessAddress := GetSessString(sess, "address")
	//	sessAccountId := GetSessInt64("account_id", sess)
	log.Debug("sessWalletID %v / sessCitizenID %v / sessStateID %v", sessWalletID, sessCitizenID, sessStateID)

	c := new(Controller)
	c.r = r
	c.w = w
	c.sess = sess
	c.SessWalletID = sessWalletID
	c.SessCitizenID = sessCitizenID
	c.SessStateID = sessStateID
	c.SessAddress = sessAddress

	c.ContentInc = true

	var configExists string
	var lastBlockTime int64
	install := &model.Install{}
	dbInit := false
	if len(config.ConfigIni["db_user"]) > 0 || (config.ConfigIni["db_type"] == "sqlite") {
		dbInit = true
	}

	if dbInit {
		var err error
		if model.DBConn == nil {
			dbInit = false
		}
		if dbInit {
			install := &model.Install{}
			// the absence of table will show the mistake, this means that the process of installation is not finished and zero-step should be shown
			err = install.Get()
			if err != nil {
				log.Error("%v", err)
				dbInit = false
			}
		}
	}
	stateName := ""
	if sessStateID > 0 {
		stateParameter := &model.StateParameter{}
		stateParameter.SetTablePrefix(converter.Int64ToStr(sessStateID))
		err = stateParameter.GetByName("state_name")
		if err != nil {
			log.Error("%v", err)
		}
		c.StateName = stateParameter.Value
		c.StateID = sessStateID
		c.StateIDStr = converter.Int64ToStr(sessStateID)
	}

	c.dbInit = dbInit

	if dbInit {
		var err error
		err = install.Get()
		if err != nil {
			log.Error("%v", err)
		}
		config := &model.Config{}
		err = config.GetConfig()
		if err != nil {
			log.Error("%v", err)
		}
		configExists = config.FirstLoadBlockchainURL

		// Information about the last block
		block := &model.Block{}
		blockData, err := block.GetLastBlockData()
		if err != nil {
			log.Error("%v", err)
		}
		// time of the last block
		lastBlockTime = blockData["lastBlockTime"]
		log.Debug("installProgress", install.Progress, "configExists", configExists, "lastBlockTime", lastBlockTime)

		confirmation := &model.Confirmation{}
		err = confirmation.GetMaxGoodBlock()
		if err != nil {
			log.Error("%v", err)
		}
		c.ConfirmedBlockID = confirmation.BlockID

	}
	r.ParseForm()
	pageName := r.FormValue("page")

	tplName := r.FormValue("tpl_name")
	if len(tplName) == 0 {
		tplName = r.FormValue("controllerHTML")
		if len(tplName) == 0 {
			if len(pageName) == 0 {
				if len(r.FormValue("key")) > 0 || len(r.FormValue("pkey")) > 0 {
					c.Logout()
					tplName = `loginECDSA`
				}
			} else {
				tplName = pageName
				if len(tplName) == 0 {
					tplName = "dashboardAnonym"
				}
			}
		}
	}
	c.Parameters, err = c.GetParameters()
	log.Debug("parameters=", c.Parameters)

	log.Debug("tpl_name=", tplName)
	// if the language has come in parameters, install it
	newLang := converter.StrToInt(c.Parameters["lang"])
	if newLang > 0 {
		log.Debug("newLang", newLang)
		SetLang(w, r, newLang)
	}
	// notifications
	c.Alert = c.Parameters["alert"]

	lang := GetLang(w, r, c.Parameters)
	log.Debug("lang", lang)

	c.Lang = globalLangReadOnly[lang]
	c.LangInt = int64(lang)
	if lang == 42 {
		c.TimeFormat = "2006-01-02 15:04:05"
	} else {
		c.TimeFormat = "2006-02-01 15:04:05"
	}

	c.Periods = map[int64]string{86400: "1 " + c.Lang["day"], 604800: "1 " + c.Lang["week"], 31536000: "1 " + c.Lang["year"], 2592000: "1 " + c.Lang["month"], 1209600: "2 " + c.Lang["weeks"]}

	match, _ := regexp.MatchString("^(installStep[0-9_]+)|(blockExplorer)$", tplName)
	// CheckInputData - ensures that tplName is clean
	if tplName != "" && utils.CheckInputData(tplName, "tpl_name") && (sessWalletID != 0 || sessCitizenID > 0 || len(sessAddress) > 0 || match) {
	} else if dbInit && install.Progress == "complete" && len(configExists) == 0 {
		// the first running, blockchain is not uploaded yet
		tplName = "updatingBlockchain"
	} else if dbInit && install.Progress == "complete" && (sessWalletID != 0 || sessCitizenID > 0 || len(sessAddress) > 0) {
		tplName = "dashboardAnonym"
	} else if dbInit && install.Progress == "complete" {
		if tplName != "loginECDSA" {
			tplName = "login"
		}
	} else {
		tplName = "installStep0" // the very first launch
	}
	log.Debug("dbInit", dbInit, "installProgress", install.Progress, "configExists", configExists)
	log.Debug("tplName>>>>>>>>>>>>>>>>>>>>>>", tplName)

	// blockchain is loading
	wTime := int64(2)
	if config.ConfigIni != nil && config.ConfigIni["test_mode"] == "1" {
		wTime = 2 * 365 * 86400
		log.Debug("%v", wTime)
		log.Debug("%v", lastBlockTime)
	}
	now := time.Now().Unix()
	if dbInit && tplName != "installStep0" && (now-lastBlockTime > 3600*wTime) && len(configExists) > 0 {
		tplName = "updatingBlockchain"
	}
	log.Debug("lastBlockTime %v / utils.Time() %v / wTime %v", lastBlockTime, now, wTime)

	if tplName == "installStep0" {
		log.Debug("ConfigInit monitor")
		if err := config.Read(); err == nil {
			if len(config.ConfigIni["db_type"]) > 0 {
				tplName = "updatingBlockchain"
			}
		}
	}

	log.Debug("tplName2=", tplName)

	if tplName == "" {
		tplName = "login"
	}

	log.Debug("tplName::", tplName, sessCitizenID, sessWalletID, install.Progress)

	fmt.Println("tplName::", tplName, sessCitizenID, sessWalletID, sessAddress)
	controller := r.FormValue("controllerHTML")
	if val, ok := config.ConfigIni[`psw`]; ok && ((tplName != `login` && tplName != `loginECDSA`) || len(controller) > 0) {
		if psw, err := r.Cookie(`psw`); err != nil || !IsPassValid(val, psw.Value) {
			if err == nil {
				cookie := http.Cookie{Name: "psw", Value: ``, Expires: time.Now().AddDate(0, 0, -1)}
				http.SetCookie(w, &cookie)
			}
			if controller == `menu` || tplName == `menu` || tplName == `ModalAnonym` {
				w.Write([]byte{})
				return
			}
			c.Logout()
			controller = `psw`
			pageName = ``
			tplName = ``
		}
	}

	if len(controller) > 0 {
		fmt.Println(`Controller HTML`, controller)
		log.Debug("controller:", controller)

		funcMap := template.FuncMap{
			"noescape": func(s string) template.HTML {
				return template.HTML(s)
			},
		}
		data, err := static.Asset("static/" + controller + ".html")
		t := template.New("template").Funcs(funcMap)
		t, err = t.Parse(string(data))
		if err != nil {
			w.Write([]byte(fmt.Sprintf("Error: %v", err)))
		}

		b := new(bytes.Buffer)
		err = t.Execute(b, c)
		if err != nil {
			w.Write([]byte(fmt.Sprintf("Error: %v", err)))
		}
		w.Write(b.Bytes())
		return
	}
	if len(pageName) > 0 && isPage(pageName, TPage) {
		c.Data = &CommonPage{
			Address:   c.SessAddress,
			WalletId:  c.SessWalletID,
			CitizenId: c.SessCitizenID,
			StateId:   c.SessStateID,
			StateName: stateName,
		}
		w.Write([]byte(CallPage(c, pageName)))
		return
	}
	if len(tplName) > 0 && (sessCitizenID > 0 || sessWalletID != 0 || len(sessAddress) > 0) && install.Progress == "complete" {

		if tplName == "login" {
			tplName = "dashboard_anonym"
		}

		c.TplName = tplName

		if dbInit && tplName != "updatingBlockchain" {
			html, err := CallController(c, "AlertMessage")
			if err != nil {
				log.Error("%v", err)
			}
			w.Write([]byte(html))
		}
		w.Write([]byte("<input type='hidden' id='tpl_name' value='" + tplName + "'>"))

		log.Debug("tplName==", tplName)

		// We highlight the block number in red if the update process is in progress
		var blockJs string
		ib := &model.InfoBlock{}
		err = ib.GetInfoBlock()
		if err != nil {
			log.Error("%v", err)
		}
		blockID := ib.BlockID
		blockJs = "$('#block_id').html(" + converter.Int64ToStr(blockID) + ");$('#block_id').css('color', '#428BCA');"

		w.Write([]byte(`<script>
								$( document ).ready(function() {
								` + blockJs + `
								});
								</script>`))
		skipRestrictedUsers := []string{"cashRequestIn", "cashRequestOut", "upgrade", "notifications"}

		if c.StateID > 0 && (tplName == "dashboard_anonym" || tplName == "home") {
			tpl, err := tpl.CreateHTMLFromTemplate("dashboard_default", sessCitizenID, sessStateID, &map[string]string{})
			if err != nil {
				log.Error("%v", err)
				return
			}
			w.Write([]byte(tpl))
			return
		}

		// We don't give some pages for ones who are not registered in the pool
		if !converter.InSliceString(tplName, skipRestrictedUsers) {
			// We call controller depending on template
			html, err := CallController(c, tplName)
			if err != nil {
				log.Error("%v", err)
			}
			w.Write([]byte(html))
		}
	} else if len(tplName) > 0 {
		if tplName == "login" {
			tplName = "LoginECDSA"
		}

		log.Debug("tplName", tplName)
		html := ""
		// if session has been resetted during the navigation of the admin area, instead of login we'll send to / to clear the menu
		if len(r.FormValue("tpl_name")) > 0 && tplName == "login" {
			log.Debug("window.location.href = /")
			w.Write([]byte("<script language=\"javascript\">window.location.href = \"/\"</script>If you are not redirected automatically, follow the <a href=\"/\">/</a>"))
			return
		}
		// We call controller depending on template
		html, err = CallController(c, tplName)
		if err != nil {
			log.Error("%v", err)
		}
		w.Write([]byte(html))
	} else {
		html, err := CallController(c, "LoginECDSA")
		if err != nil {
			log.Error("%v", err)
		}
		w.Write([]byte(html))
	}
}
