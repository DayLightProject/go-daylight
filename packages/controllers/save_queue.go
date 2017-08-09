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
	"encoding/hex"
	"fmt"
	"time"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/crypto"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
	"github.com/EGaaS/go-egaas-mvp/packages/utils/tx"
	"gopkg.in/vmihailenco/msgpack.v2"
)

// SaveQueue write a transaction in the queue of transactions
func (c *Controller) SaveQueue() (string, error) {
	var err error
	c.r.ParseForm()

	citizenID := c.SessCitizenID
	walletID := c.SessWalletID

	log.Debug("citizenID %d / walletID %d ", citizenID, walletID)

	if citizenID <= 0 && walletID == 0 {
		return `{"result":"incorrect citizenID || walletID"}`, nil
	}

	txTime := converter.StrToInt64(c.r.FormValue("time"))
	if !utils.CheckInputData(txTime, "int") {
		return `{"result":"incorrect time"}`, nil
	}
	itxType := c.r.FormValue("type")
	if !utils.CheckInputData(itxType, "type") {
		return `{"result":"incorrect type"}`, nil
	}

	publicKey, _ := hex.DecodeString(c.r.FormValue("pubkey"))
	lenpub := len(publicKey)
	if lenpub > 64 {
		publicKey = publicKey[lenpub-64:]
	} else if lenpub == 0 {
		publicKey = []byte("null")
	}
	//	fmt.Printf("PublicKey %d %x\r\n", lenpub, publicKey)
	txType := utils.TypeInt(itxType)
	sign := make([]byte, 0)
	signature, err := crypto.JSSignToBytes(c.r.FormValue("signature1"))
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	if len(signature) > 0 {
		sign = append(sign, converter.EncodeLengthPlusData(signature)...)
	}
	if len(sign) == 0 {
		return `{"result":"signature is empty"}`, nil
	}
	fmt.Printf("Len sign %d\r\n", len(sign))
	log.Debug("txType_", itxType)
	log.Debug("txType", itxType)

	userID := walletID
	stateID := converter.StrToInt64(c.r.FormValue("stateId"))
	if stateID > 0 {
		userID = citizenID
	}

	var (
		data []byte
		//		key  []byte
	)
	var toSerialize interface{}
	header := tx.Header{Type: int(txType), Time: txTime, UserID: userID, StateID: stateID, PublicKey: publicKey, BinSignatures: sign}
	switch itxType {
	case "NewColumn", "AppendPage", "AppendMenu", "EditPage", "NewPage", "EditTable",
		"EditStateParameters", "NewStateParameters", "NewContract", "EditContract", "NewMenu",
		"EditMenu", "NewTable":
		userID := walletID
		stateID := converter.StrToInt64(c.r.FormValue("stateId"))
		if stateID > 0 {
			userID = citizenID
		}
		if stateID == 0 {
			return "", utils.ErrInfo(fmt.Errorf(`StateId is not defined`))
		}
		header.UserID = userID
		header.StateID = stateID
	case "DLTTransfer", "DLTChangeHostVote", "NewState", "ChangeNodeKeyDLT":
		header.StateID = 0
	case "EditWallet", "ActivateContract":
		userID := walletID
		stateID := converter.StrToInt64(c.r.FormValue("stateId"))
		if userID == 0 {
			userID = citizenID
		}
		header.UserID = userID
		header.StateID = stateID
	default:
	}

	switch itxType {
	case "DLTTransfer":
		comment_ := c.r.FormValue("comment")
		if len(comment_) == 0 {
			comment_ = "null"
		}
		toSerialize = tx.DLTTransfer{
			Header:        header,
			WalletAddress: c.r.FormValue("walletAddress"),
			Amount:        c.r.FormValue("amount"),
			Commission:    c.r.FormValue("commission"),
			Comment:       comment_,
		}
	case "DLTChangeHostVote":
		toSerialize = tx.DLTChangeHostVote{
			Header:      header,
			Host:        c.r.FormValue("host"),
			AddressVote: c.r.FormValue("addressVote"),
			FuelRate:    c.r.FormValue("fuelRate"),
		}
	case "NewState":
		toSerialize = tx.NewState{
			Header:       header,
			StateName:    c.r.FormValue("state_name"),
			CurrencyName: c.r.FormValue("currency_name"),
		}
	case "NewColumn":
		toSerialize = tx.NewColumn{Header: header,
			TableName:   c.r.FormValue("table_name"),
			ColumnName:  c.r.FormValue("column_name"),
			ColumnType:  c.r.FormValue("column_type"),
			Permissions: c.r.FormValue("permissions"),
			Index:       c.r.FormValue("index"),
		}
	case "EditColumn":
		toSerialize = tx.EditColumn{Header: header,
			TableName:   c.r.FormValue("table_name"),
			ColumnName:  c.r.FormValue("column_name"),
			Permissions: c.r.FormValue("permissions"),
		}
	case "AppendPage":
		toSerialize = tx.AppendPage{
			Header: header,
			Global: c.r.FormValue("global"),
			Name:   c.r.FormValue("name"),
			Value:  c.r.FormValue("value"),
		}
	case "AppendMenu":
		toSerialize = tx.AppendMenu{
			Header: header,
			Global: c.r.FormValue("global"),
			Name:   c.r.FormValue("name"),
			Value:  c.r.FormValue("value"),
		}
	case "EditPage":
		toSerialize = tx.EditPage{
			Header:     header,
			Global:     c.r.FormValue("global"),
			Name:       c.r.FormValue("name"),
			Value:      c.r.FormValue("value"),
			Menu:       c.r.FormValue("menu"),
			Conditions: c.r.FormValue("conditions"),
		}
	case "NewPage":
		toSerialize = tx.NewPage{Header: header,
			Global:     c.r.FormValue("global"),
			Name:       c.r.FormValue("name"),
			Value:      c.r.FormValue("value"),
			Menu:       c.r.FormValue("menu"),
			Conditions: c.r.FormValue("conditions"),
		}
	case "EditTable":
		toSerialize = tx.EditTable{
			Header:        header,
			Name:          c.r.FormValue("table_name"),
			GeneralUpdate: c.r.FormValue("general_update"),
			Insert:        c.r.FormValue("insert"),
			NewColumn:     c.r.FormValue("new_column"),
		}
	case "EditStateParameters":
		toSerialize = tx.EditStateParameters{
			Header:     header,
			Name:       c.r.FormValue("name"),
			Value:      c.r.FormValue("value"),
			Conditions: c.r.FormValue("conditions"),
		}
	case "NewStateParameters":
		toSerialize = tx.NewStateParameters{
			Header:     header,
			Name:       c.r.FormValue("name"),
			Value:      c.r.FormValue("value"),
			Conditions: c.r.FormValue("conditions"),
		}
	case "NewContract":
		toSerialize = tx.NewContract{
			Header:     header,
			Global:     c.r.FormValue("global"),
			Name:       c.r.FormValue("name"),
			Value:      c.r.FormValue("value"),
			Wallet:     c.r.FormValue("wallet"),
			Conditions: c.r.FormValue("conditions"),
		}
	case "EditContract":
		toSerialize = tx.EditContract{
			Header:     header,
			Global:     c.r.FormValue("global"),
			Id:         c.r.FormValue("id"),
			Value:      c.r.FormValue("value"),
			Conditions: c.r.FormValue("conditions"),
		}
	case "ActivateContract":
		toSerialize = tx.ActivateContract{
			Header: header,
			Global: c.r.FormValue("global"),
			Id:     c.r.FormValue("id"),
		}
	case "NewMenu":
		toSerialize = tx.NewMenu{
			Header:     header,
			Global:     c.r.FormValue("global"),
			Name:       c.r.FormValue("name"),
			Value:      c.r.FormValue("value"),
			Conditions: c.r.FormValue("conditions"),
		}
	case "EditMenu":
		toSerialize = tx.EditMenu{
			Header:     header,
			Global:     c.r.FormValue("global"),
			Name:       c.r.FormValue("name"),
			Value:      c.r.FormValue("value"),
			Conditions: c.r.FormValue("conditions"),
		}
	case "EditWallet":
		toSerialize = tx.EditWallet{
			Header:           header,
			WalletID:         c.r.FormValue("id"),
			SpendingContract: c.r.FormValue("spending_contract"),
			Conditions:       c.r.FormValue("conditions_change"),
		}
	case "NewTable":
		toSerialize = tx.NewTable{
			Header:  header,
			Global:  c.r.FormValue("global"),
			Name:    c.r.FormValue("table_name"),
			Columns: c.r.FormValue("columns"),
		}
	case "EditLang", "NewLang":
		toSerialize = tx.EditNewLang{
			Header: header,
			Name:   c.r.FormValue("name"),
			Trans:  c.r.FormValue("trans"),
		}
	case "EditSign", "NewSign":
		toSerialize = tx.EditNewSign{
			Header:     header,
			Global:     c.r.FormValue("global"),
			Name:       c.r.FormValue("name"),
			Value:      c.r.FormValue("value"),
			Conditions: c.r.FormValue("conditions"),
		}
	case "NewAccount":
		accountID := converter.StrToInt64(c.r.FormValue("accountId"))
		pubKey, err := hex.DecodeString(c.r.FormValue("pubkey"))
		if accountID == 0 || stateID == 0 || userID == 0 || err != nil {
			return ``, fmt.Errorf(`incorrect NewAccount parameters`)
		}
		encKey := c.r.FormValue("prvkey")
		if len(encKey) == 0 {
			return ``, fmt.Errorf(`incorrect encrypted key`)
		}
		anonym := &model.Anonym{IDCitizen: userID, IDAnonym: accountID, Encrypted: []byte(encKey)}
		anonym.SetTableName(stateID)
		err = anonym.Create()
		if err != nil {
			return "", utils.ErrInfo(err)
		}
		header.UserID = accountID
		toSerialize = tx.NewAccount{header, pubKey}
	case "ChangeNodeKey", "ChangeNodeKeyDLT":
		publicKey := []byte(c.r.FormValue("publicKey"))
		privateKey := []byte(c.r.FormValue("privateKey"))
		verifyData := map[string]string{c.r.FormValue("publicKey"): "public_key", c.r.FormValue("privateKey"): "private_key"}
		err := CheckInputData(verifyData)
		if err != nil {
			return "", utils.ErrInfo(err)
		}
		myWalletID, err := model.GetMyWalletID()
		if err != nil {
			return "", utils.ErrInfo(err)
		}
		if myWalletID == walletID {
			nodeKeys := &model.MyNodeKey{PublicKey: publicKey, PrivateKey: privateKey}
			err = nodeKeys.Create()
			if err != nil {
				return "", utils.ErrInfo(err)
			}
		}
		header.PublicKey = publicKey
		if itxType == "ChangeNodeKeyDLT" {
			toSerialize = tx.DLTChangeNodeKey{header, converter.HexToBin(publicKey)}
		} else {
			toSerialize = tx.ChangeNodeKey{header, converter.HexToBin(publicKey)}
		}
	}
	transactionTypeBin := converter.DecToBin(txType, 1)
	serializedData, err := msgpack.Marshal(toSerialize)
	if err != nil {
		return "", utils.ErrInfo(err)
	}
	data = append(transactionTypeBin, serializedData...)

	if err != nil {
		return "", utils.ErrInfo(err)
	}

	hash, err := crypto.Hash(data)
	if err != nil {
		log.Fatal(err)
	}
	hash = converter.BinToHex(hash)
	txStatus := &model.TransactionStatus{Hash: hash, Time: time.Now().Unix(), Type: txType, WalletID: walletID, CitizenID: citizenID}
	err = txStatus.Create()
	if err != nil {
		return "", utils.ErrInfo(err)
	}

	log.Debug("INSERT INTO queue_tx (hash, data) VALUES (%s, %s)", hash, converter.BinToHex(data))
	queueTx := &model.QueueTx{Hash: hash, Data: data}
	err = queueTx.Create()
	if err != nil {
		return "", utils.ErrInfo(err)
	}

	return `{"hash":"` + string(hash) + `"}`, nil
}

// CheckInputData calls utils.CheckInputData for the each item of the map
func CheckInputData(data map[string]string) error {
	for k, v := range data {
		if !utils.CheckInputData(k, v) {
			return utils.ErrInfo(fmt.Errorf("incorrect " + v))
		}
	}
	return nil
}
