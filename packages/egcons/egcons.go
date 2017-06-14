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

package main

import (
	"crypto/aes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/crypto"

	"github.com/go-yaml/yaml"
)

// Settings contains options of the program
type Settings struct {
	NodeURL string // URL of EGAAS node
	Log     bool   // if true then the program writes log data
	Cookie  string //*http.Cookie
	Key     string // decrypted private key
	Address int64  // wallet id
}

var (
	gSettings    Settings
	gPrivate     []byte // private key
	gPublic      []byte
	ellipticSize = crypto.Elliptic256
	hashProv     = crypto.SHA256
	cryptoProv   = crypto.AESCBC
	signProv     = crypto.ECDSA
)

func logOut(format string, params ...interface{}) {
	if !gSettings.Log {
		return
	}
	log.Printf(format, params...)
}

func saveSetting() {
	out, err := json.Marshal(gSettings)
	if err != nil {
		logOut(`saveSetting`, err)
	}
	ioutil.WriteFile(`settings.json`, out, 0600)
}

func checkKey() bool {
	var privKey, pass []byte
	var err error
	// Reads the hex private key from the file
	for len(gSettings.Key) == 0 {
		var (
			filename string
		)
		fmt.Println(`Enter the filename with the private key:`)
		n, err := fmt.Scanln(&filename)
		if err != nil || n == 0 {
			fmt.Println(err)
			continue
		}
		if privKey, err = ioutil.ReadFile(filename); err != nil {
			fmt.Println(err)
			continue
		}
		privKey, err = hex.DecodeString(strings.TrimSpace(string(privKey)))
		if err != nil {
			fmt.Println(err)
			continue
		}
		if len(privKey) != 32 {
			fmt.Println(`Wrong the length of private key`, len(privKey))
			continue
		}
		fmt.Println(`Enter a new password:`)
		n, err = fmt.Scanln(&pass)
		if err != nil || n == 0 {
			fmt.Println(err)
			continue
		}
		pubKey, err := crypto.PrivateToPublic(privKey, ellipticSize)
		if err != nil {
			log.Fatal(err)
		}
		gSettings.Address = crypto.Address(pubKey)
		hash, err := crypto.Hash(pass, hashProv)
		if err != nil {
			log.Fatal(err)
		}
		privKey, _, err = crypto.Encrypt(hash, privKey, make([]byte, aes.BlockSize), cryptoProv)
		if err != nil {
			fmt.Println(err)
			continue
		}
		gSettings.Key = hex.EncodeToString(privKey[aes.BlockSize:])
	}
	if privKey, err = hex.DecodeString(gSettings.Key); err != nil {
		fmt.Println(err)
		return false
	}
	for {
		if len(pass) == 0 {
			fmt.Println(`Enter the password:`)
			n, err := fmt.Scanln(&pass)
			if err != nil || n == 0 {
				fmt.Println(err)
				continue
			}
		}
		hash, err := crypto.Hash(pass, hashProv)
		if err != nil {
			log.Fatal(err)
		}
		pass = pass[:0]
		gPrivate, err = crypto.Decrypt(hash, privKey, make([]byte, aes.BlockSize), cryptoProv)
		if err != nil {
			fmt.Println(err)
			continue
		}
		gPublic, err = crypto.PrivateToPublic(gPrivate, ellipticSize)
		if err != nil {
			log.Fatal(err)
		}
		if gSettings.Address != crypto.Address(gPublic) {
			fmt.Println(`Wrong password`)
			continue
		}
		break
	}
	return true
}

func login() error {
	ret, err := sendGet(`ajax_get_uid`, nil)
	if err != nil {
		return err
	}
	if len(ret[`uid`].(string)) == 0 {
		return fmt.Errorf(`Unknown uid`)
	}
	sign, err := crypto.Sign(hex.EncodeToString(gPrivate), ret[`uid`].(string), hashProv, signProv, ellipticSize)
	if err != nil {
		return err
	}
	form := url.Values{"key": {hex.EncodeToString(gPublic)}, "sign": {hex.EncodeToString(sign)},
		`state_id`: {`0`}, `citizen_id`: {converter.Int64ToStr(gSettings.Address)}}

	ret, err = sendPost(`ajax_sign_in`, &form)
	if err != nil {
		return err
	}
	if ret[`result`].(bool) != true {
		return fmt.Errorf(`Login is incorrect`)
	}
	fmt.Println(`Address: `, ret[`address`])
	saveSetting()
	return nil
}

func map2yaml(in map[string]string, filename string) error {
	data, err := yaml.Marshal(in)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0600)
}

func yaml2map(filename string, out *map[string]string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(`Dir`, err)
	}
	params, err := ioutil.ReadFile(filepath.Join(dir, `settings.json`))
	if err != nil {
		log.Fatalln(dir, `Settings.json`, err)
	}
	if err = json.Unmarshal(params, &gSettings); err != nil {
		log.Fatalln(`Unmarshall`, err)
	}
	if gSettings.Log {
		logfile, err := os.OpenFile(filepath.Join(dir, "egcons.log"),
			os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalln(`Egcons log`, err)
		}
		defer logfile.Close()
		log.SetOutput(logfile)
	}
	os.Chdir(dir)
	/*	tmp := make(map[string]string)
			tmp[`test`] = `Test string`
			tmp[`param`] = `Test string
		edededed
		edededed
		1111
		 222
		  3333`
			tmp[`ok`] = `76436734`
			err = map2yaml(tmp, `test.yaml`)
			if err != nil {
				fmt.Println(`YAML`, err)
			}
			var tmp2 map[string]string
			err = yaml2map(`test.yaml`, &tmp2)
			fmt.Println(`YAML`, tmp2)
	*/
	if !checkKey() {
		return
	}
	if err = login(); err != nil {
		fmt.Println(`ERROR:`, err)
		return
	}

cmd:
	for {
		var cmd string
		fmt.Printf(`>`)
		_, err := fmt.Scanln(&cmd)
		if err != nil {
			fmt.Println(err)
			continue
		}
		switch {
		case cmd == `quit`:
			break cmd
		default:
			fmt.Println(`Unknown command`)
		}
	}
}
