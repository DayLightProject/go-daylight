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

package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/crypto"
	"github.com/shopspring/decimal"
)

var (
	gCookie           string
	gAddress          string
	gPrivate, gPublic string
)

type global struct {
	url   string
	value string
}

func sendRequest(rtype, url string, form *url.Values) (map[string]interface{}, error) {
	client := &http.Client{}
	var ioform io.Reader
	if form != nil {
		ioform = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(rtype, `http://localhost:7079/api/v1/`+url, ioform)
	if err != nil {
		return nil, err
	}
	if len(gCookie) > 0 {
		req.Header.Set("Cookie", gCookie)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	//	req.Header.Set("Connection", `keep-alive`)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	var cookie []string
	for _, val := range resp.Cookies() {
		cookie = append(cookie, fmt.Sprintf(`%s=%s`, val.Name, val.Value))
	}
	if len(cookie) > 0 {
		gCookie = strings.Join(cookie, `;`)
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	//fmt.Println(`ANSWER`, resp.StatusCode, strings.TrimSpace(string(data)), `<<<`)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`%d %s`, resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var v map[string]interface{}
	if err = json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func sendGet(url string, form *url.Values) (map[string]interface{}, error) {
	return sendRequest("GET", url, form)
}

func sendPost(url string, form *url.Values) (map[string]interface{}, error) {
	return sendRequest("POST", url, form)
}

func sendPut(url string, form *url.Values) (map[string]interface{}, error) {
	return sendRequest("PUT", url, form)
}

func keyLogin(state int64) (err error) {
	var (
		key, sign []byte
		uid       interface{}
		ok        bool
	)

	key, err = ioutil.ReadFile(`key`)
	if err != nil {
		return
	}
	ret, err := sendGet(`getuid`, nil)
	if err != nil {
		return
	}
	if uid, ok = ret[`uid`]; !ok {
		return fmt.Errorf(`getuid has returned empty uid`)
	}

	var pub string

	sign, err = crypto.Sign(string(key), uid.(string))
	if err != nil {
		return
	}
	pub, err = crypto.PrivateToPublicHex(string(key))
	if err != nil {
		return
	}
	form := url.Values{"pubkey": {pub}, "signature": {hex.EncodeToString(sign)},
		`state`: {converter.Int64ToStr(state)}}
	ret, err = sendPost(`login`, &form)
	if err != nil {
		return
	}
	gAddress = ret[`address`].(string)
	gPrivate = string(key)
	gPublic, err = crypto.PrivateToPublicHex(gPrivate)
	if err != nil {
		return
	}
	return
}

func getSign(forSign string) (string, error) {
	sign, err := crypto.Sign(gPrivate, forSign)
	if err != nil {
		return ``, err
	}
	return hex.EncodeToString(sign), nil
}

func appendSign(ret map[string]interface{}, form *url.Values) error {
	sign, err := getSign(ret[`forsign`].(string))
	if err != nil {
		return err
	}
	(*form)[`time`] = []string{ret[`time`].(string)}
	(*form)[`signature`] = []string{sign}
	return nil
}

func waitTx(hash string) (int64, error) {
	for i := 0; i < 15; i++ {
		ret, err := sendGet(`txstatus/`+hash, nil)
		if err != nil {
			return 0, err
		}
		if len(ret[`blockid`].(string)) > 0 {
			return converter.StrToInt64(ret[`blockid`].(string)), nil
		}
		if len(ret[`errmsg`].(string)) > 0 {
			return 0, fmt.Errorf(ret[`errmsg`].(string))
		}
		time.Sleep(time.Second)
	}
	return 0, fmt.Errorf(`TxStatus timeout`)
}

func getBalance(wallet string) (decimal.Decimal, error) {
	ret, err := sendGet(`balance/`+wallet, nil)
	if err != nil {
		return decimal.New(0, 0), err
	}
	if len(ret[`amount`].(string)) == 0 {
		return decimal.New(0, 0), nil
	}
	val, err := decimal.NewFromString(ret[`amount`].(string))
	if err != nil {
		return decimal.New(0, 0), err
	}
	return val, nil
}

func randName(prefix string) string {
	return fmt.Sprintf(`%s%d`, prefix, time.Now().Unix())
}

func postTx(txname string, form *url.Values) error {
	ret, err := sendPost(`prepare/`+txname, form)
	if err != nil {
		return err
	}
	if err = appendSign(ret, form); err != nil {
		return err
	}
	ret, err = sendPost(txname, form)
	if err != nil {
		return err
	}
	if _, err = waitTx(ret[`hash`].(string)); err != nil {
		return err
	}
	return nil
}

func putTx(txname string, form *url.Values) error {
	ret, err := sendPut(`prepare/`+txname, form)
	if err != nil {
		return err
	}
	if err = appendSign(ret, form); err != nil {
		return err
	}
	ret, err = sendPut(txname, form)
	if err != nil {
		return err
	}
	if _, err = waitTx(ret[`hash`].(string)); err != nil {
		return err
	}
	return nil
}

func TestSign(t *testing.T) {
	var (
		err               error
		key, sign, public []byte
		uid               interface{}
		ok                bool
	)
	key, err = ioutil.ReadFile(`key`)
	if err != nil {
		t.Error(err)
		return
	}
	ret, err := sendGet(`getuid`, nil)
	if err != nil {
		t.Error(err)
		return
	}
	if uid, ok = ret[`uid`]; !ok {
		t.Error(fmt.Errorf(`getuid has returned empty uid`))
		return
	}
	form := url.Values{"private": {string(key)}, "forsign": {uid.(string)}}
	ret, err = sendPost(`signtest`, &form)
	if err != nil {
		t.Error(err)
		return
	}
	public, err = hex.DecodeString(ret[`pubkey`].(string))
	if err != nil {
		t.Error(err)
		return
	}
	sign, err = hex.DecodeString(ret[`signature`].(string))
	if err != nil {
		t.Error(err)
		return
	}
	ok, err = crypto.CheckSign(public, uid.(string), sign)
	if err != nil {
		t.Error(err)
		return
	}
	if !ok {
		t.Error(fmt.Errorf(`incorrect signature`))
		return
	}
}
