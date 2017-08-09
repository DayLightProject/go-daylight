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

package parser

import (
	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
	"github.com/EGaaS/go-egaas-mvp/packages/utils/tx"

	"gopkg.in/vmihailenco/msgpack.v2"
)

type ChangeNodeKeyDLTParser struct {
	*Parser
	DLTChangeNodeKey *tx.DLTChangeNodeKey
}

func (p *ChangeNodeKeyDLTParser) Init() error {
	changeNodeKey := &tx.DLTChangeNodeKey{}
	if err := msgpack.Unmarshal(p.TxBinaryData, changeNodeKey); err != nil {
		return p.ErrInfo(err)
	}
	p.DLTChangeNodeKey = changeNodeKey
	p.DLTChangeNodeKey.NewNodePublicKey = converter.BinToHex(p.DLTChangeNodeKey.NewNodePublicKey)
	return nil
}

func (p *ChangeNodeKeyDLTParser) Validate() error {
	err := p.generalCheck(`change_node`, &p.DLTChangeNodeKey.Header, map[string]string{})
	if err != nil {
		return p.ErrInfo(err)
	}

	verifyData := map[string][]interface{}{"public_key": []interface{}{p.DLTChangeNodeKey.PublicKey}}
	err = p.CheckInputData(verifyData)
	if err != nil {
		return p.ErrInfo(err)
	}

	txTime := p.DLTChangeNodeKey.Header.Time
	if p.BlockData != nil {
		txTime = p.BlockData.Time
	}
	dltW := &model.DltWallet{}
	err = dltW.GetWallet(p.TxWalletID)
	if err != nil || txTime-dltW.LastForgingDataUpd < 600 {
		return p.ErrInfo("txTime - last_forging_data_upd < 600 sec")
	}

	forSign := p.DLTChangeNodeKey.ForSign()
	CheckSignResult, err := utils.CheckSign(p.PublicKeys, forSign, p.DLTChangeNodeKey.BinSignatures, false)
	if err != nil || !CheckSignResult {
		return p.ErrInfo("incorrect sign " + forSign)
	}
	return nil
}

func (p *ChangeNodeKeyDLTParser) Action() error {
	_, _, err := p.selectiveLoggingAndUpd([]string{"node_public_key", "last_forging_data_upd"}, []interface{}{converter.HexToBin(p.DLTChangeNodeKey.NewNodePublicKey), p.BlockData.Time}, "dlt_wallets", []string{"wallet_id"}, []string{converter.Int64ToStr(p.TxWalletID)}, true)
	if err != nil {
		return p.ErrInfo(err)
	}
	mnk := &model.MyNodeKey{}
	myKey := mnk.ID
	err = mnk.GetZeroBlock(p.DLTChangeNodeKey.NewNodePublicKey)
	if err != nil {
		return p.ErrInfo(err)
	}
	log.Debug("myKey %d", myKey)
	if myKey > 0 {
		_, _, err := p.selectiveLoggingAndUpd([]string{"block_id"}, []interface{}{p.BlockData.BlockID}, "my_node_keys", []string{"id"}, []string{converter.Int64ToStr(int64(myKey))}, true)
		if err != nil {
			return p.ErrInfo(err)
		}
	}
	return nil
}

func (p *ChangeNodeKeyDLTParser) Rollback() error {
	return p.autoRollback()
}

func (p ChangeNodeKeyDLTParser) Header() *tx.Header {
	return &p.DLTChangeNodeKey.Header
}
