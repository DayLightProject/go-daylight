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
	"encoding/hex"
	"fmt"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/crypto"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
	"github.com/shopspring/decimal"
)

// DLTTransferInit initializes DLTTransfer transaction
func (p *Parser) DLTTransferInit() error {

	fields := []map[string]string{{"walletAddress": "string"}, {"amount": "decimal"}, {"commission": "decimal"}, {"comment": "bytes"}, {"public_key": "bytes"}, {"sign": "bytes"}}
	err := p.GetTxMaps(fields)
	if err != nil {
		return p.ErrInfo(err)
	}
	p.TxMaps.Bytes["public_key"] = converter.BinToHex(p.TxMaps.Bytes["public_key"])
	p.TxMap["public_key"] = converter.BinToHex(p.TxMap["public_key"])
	p.TxMaps.Bytes["sign"] = converter.BinToHex(p.TxMaps.Bytes["sign"])
	return nil
}

// DLTTransferFront checks conditions DLTTransfer transaction
func (p *Parser) DLTTransferFront() error {
	err := p.generalCheck(`dlt_transfer`)
	if err != nil {
		return p.ErrInfo(err)
	}

	verifyData := map[string]string{"walletAddress": "walletAddress", "amount": "decimal", "commission": "decimal", "comment": "comment"}
	err = p.CheckInputData(verifyData)
	if err != nil {
		return p.ErrInfo(err)
	}

	// public key need only when we don't have public_key in the dlt_wallets table
	PublicKey, err := p.GetSingleWalletPublicKey(p.TxWalletID)
	if err != nil {
		return p.ErrInfo(err)
	}
	if len(PublicKey) == 0 {
		bkey, err := hex.DecodeString(string(p.TxMaps.Bytes["public_key"]))
		if err != nil {
			return p.ErrInfo(err)
		}
		if crypto.KeyToAddress(bkey) != converter.AddressToString(p.TxWalletID) {
			return p.ErrInfo("incorrect public_key")
		}
	}

	zero, _ := decimal.NewFromString("0")
	if p.TxMaps.Decimal["amount"].Cmp(zero) <= 0 {
		return p.ErrInfo("amount<=0")
	}

	fPrice, err := p.GetDltTransferPriceString()
	if err != nil {
		return p.ErrInfo(err)
	}

	fuelRate := p.GetFuel()
	if fuelRate.Cmp(decimal.New(0, 0)) <= 0 {
		return fmt.Errorf(`fuel rate must be greater than 0`)
	}
	// 1 000 000 000 000 000 000 qDLT = 1 DLT * 100 000 000
	// fuelRate = 1 000 000 000 000 000
	//
	fPriceDecemal, err := decimal.NewFromString(fPrice)
	if err != nil {
		return p.ErrInfo(err)
	}
	commission := fPriceDecemal.Mul(fuelRate)

	// проверим, удовлетворяет ли нас комиссия, которую предлагает юзер
	// check if commission proposed by the user satisfies us
	if p.TxMaps.Decimal["commission"].Cmp(commission) < 0 {
		return p.ErrInfo(fmt.Sprintf("commission %s < dltPrice %d", p.TxMaps.Decimal["commission"].String(), commission))
	}

	if string(p.TxMap["comment"]) == "null" {
		p.TxMap["comment"] = []byte("")
		p.TxMaps.Bytes["comment"] = []byte("")
	}

	forSign := fmt.Sprintf("%s,%s,%d,%s,%s,%s,%s", p.TxMap["type"], p.TxMap["time"], p.TxWalletID, p.TxMap["walletAddress"], p.TxMap["amount"], p.TxMap["commission"], p.TxMap["comment"])
	CheckSignResult, err := utils.CheckSign(p.PublicKeys, forSign, p.TxMap["sign"], false)
	if err != nil {
		return p.ErrInfo(err)
	}
	if !CheckSignResult {
		return p.ErrInfo("incorrect sign OOPS")
	}

	totalAmount, err := p.GetWalletAmountString(p.TxWalletID)
	if err != nil {
		return p.ErrInfo(err)
	}
	totalAmountDecimal, err := decimal.NewFromString(totalAmount)
	if err != nil {
		return p.ErrInfo(err)
	}
	if totalAmountDecimal.Cmp(p.TxMaps.Decimal["amount"].Add(p.TxMaps.Decimal["commission"])) < 0 {
		return p.ErrInfo(fmt.Sprintf("%s + %s < %s)", p.TxMaps.Decimal["amount"], p.TxMaps.Decimal["commission"], totalAmount))
	}
	return nil
}

// DLTTransfer proceeds DLTTransfer transaction
func (p *Parser) DLTTransfer() error {
	log.Debug("wallet address %s", p.TxMaps.String["walletAddress"])
	address := converter.StringToAddress(p.TxMaps.String["walletAddress"])
	walletID, err := p.IsWalletExistFromInt64(address)
	if err != nil {
		return p.ErrInfo(err)
	}
	log.Debug("walletID %d", walletID)
	//if walletID > 0 {
	pkey, err := p.GetSingleStringWalletPublickKey(p.TxWalletID)
	if err != nil {
		return p.ErrInfo(err)
	}
	log.Debug("pkey %x", pkey)

	log.Debug("amount %s", p.TxMaps.Decimal["amount"])
	log.Debug("commission %s", p.TxMaps.Decimal["commission"])
	amountAndCommission := p.TxMaps.Decimal["amount"].Add(p.TxMaps.Decimal["commission"])
	if err != nil {
		return p.ErrInfo(err)
	}
	log.Debug("amountAndCommission %s", amountAndCommission)
	log.Debug("amountAndCommission %s", amountAndCommission.String())
	if len(p.TxMaps.Bytes["public_key"]) > 30 && len(pkey) == 0 {
		_, err = p.selectiveLoggingAndUpd([]string{"-amount", "public_key_0"}, []interface{}{amountAndCommission.String(), converter.HexToBin(p.TxMaps.Bytes["public_key"])}, "dlt_wallets", []string{"wallet_id"}, []string{converter.Int64ToStr(p.TxWalletID)}, true)
	} else {
		_, err = p.selectiveLoggingAndUpd([]string{"-amount"}, []interface{}{amountAndCommission.String()}, "dlt_wallets", []string{"wallet_id"}, []string{converter.Int64ToStr(p.TxWalletID)}, true)
	}
	if err != nil {
		return p.ErrInfo(err)
	}

	if walletID == 0 {
		log.Debug("walletID == 0")
		log.Debug("%s", string(p.TxMaps.String["walletAddress"]))
		walletID = converter.StringToAddress(p.TxMaps.String["walletAddress"])
		_, err = p.selectiveLoggingAndUpd([]string{"+amount"}, []interface{}{p.TxMaps.Decimal["amount"].String()}, "dlt_wallets",
			[]string{"wallet_id"}, []string{converter.Int64ToStr(walletID)}, true)
	} else {
		_, err = p.selectiveLoggingAndUpd([]string{"+amount"}, []interface{}{p.TxMaps.Decimal["amount"].String()}, "dlt_wallets",
			[]string{"wallet_id"}, []string{converter.Int64ToStr(walletID)}, true)
	}
	if err != nil {
		return p.ErrInfo(err)
	}

	// node commission
	_, err = p.selectiveLoggingAndUpd([]string{"+amount"}, []interface{}{p.TxMaps.Decimal["commission"].String()}, "dlt_wallets", []string{"wallet_id"}, []string{converter.Int64ToStr(p.BlockData.WalletId)}, true)
	if err != nil {
		return p.ErrInfo(err)
	}

	// пишем в общую историю тр-ий
	// record into the general transaction history
	dltTransactionsID, err := p.CreateDltTransaction(p.TxWalletID, walletID, converter.AddressToString(int64(converter.StrToUint64(p.TxMaps.String["walletAddress"]))), p.TxMaps.Decimal["amount"].String(), p.TxMaps.Decimal["commission"].String(), p.TxMaps.Bytes["comment"], p.BlockData.Time, p.BlockData.BlockId)
	if err != nil {
		return p.ErrInfo(err)
	}
	err = p.CreateRollbackTX(p.BlockData.BlockId, p.TxHash, "dlt_transactions", dltTransactionsID)
	if err != nil {
		return err
	}

	dltTransactionsID, err = p.CreateAnotherDltTransaction(p.TxWalletID, p.BlockData.WalletId, converter.AddressToString(p.BlockData.WalletId), p.TxMaps.Decimal["commission"].String(), 0, "Commission", p.BlockData.Time, p.BlockData.BlockId)
	if err != nil {
		return p.ErrInfo(err)
	}
	err = p.CreateRollbackTX(p.BlockData.BlockId, p.TxHash, "dlt_transactions", dltTransactionsID)
	if err != nil {
		return err
	}

	return nil
}

// DLTTransferRollback rollbacks DLTTransfer transaction
func (p *Parser) DLTTransferRollback() error {
	return p.autoRollback()
}
