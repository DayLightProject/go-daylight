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
	"encoding/json"
	"github.com/EGaaS/go-egaas-mvp/packages/lib"
	"fmt"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
	"github.com/EGaaS/go-egaas-mvp/packages/consts"
)

func (p *Parser) UpdFullNodesInit() error {
	err := p.GetTxMaps([]map[string]string{{"sign": "bytes"}})
	if err != nil {
		return p.ErrInfo(err)
	}
	return nil
}

func (p *Parser) UpdFullNodesFront() error {
	err := p.generalCheck()
	if err != nil {
		return p.ErrInfo(err)
	}

	// We check to see if the time elapsed since the last update
	upd_full_nodes, err := p.Single("SELECT time FROM upd_full_nodes").Int64()
	if err != nil {
		return p.ErrInfo(err)
	}
	txTime := p.TxTime
	if p.BlockData!= nil {
		txTime = p.BlockData.Time
	}
	if p.BlockData!=nil && p.BlockData.BlockId < 16500 {
		if txTime - upd_full_nodes <= 120 {
			return utils.ErrInfoFmt("txTime - upd_full_nodes <= 120 (%d - %d <= 120)", txTime, upd_full_nodes)
		}
	} else {
		if txTime - upd_full_nodes <= consts.UPD_FULL_NODES_PERIOD {
			return utils.ErrInfoFmt("txTime - upd_full_nodes <= consts.UPD_FULL_NODES_PERIOD (%d - %d <= %d)", txTime, upd_full_nodes, consts.UPD_FULL_NODES_PERIOD)
		}
	}

	p.nodePublicKey, err = p.GetNodePublicKey(p.TxWalletID)
	if len(p.nodePublicKey) == 0 {
		return utils.ErrInfoFmt("len(nodePublicKey) = 0")
	}
	forSign := fmt.Sprintf("%s,%s,%d,%d", p.TxMap["type"], p.TxMap["time"], p.TxWalletID, 0)
	CheckSignResult, err := utils.CheckSign([][]byte{p.nodePublicKey}, forSign, p.TxMap["sign"], true)
	if err != nil {
		return p.ErrInfo(err)
	}
	if !CheckSignResult {
		return p.ErrInfo("incorrect sign")
	}

	return nil
}

func (p *Parser) UpdFullNodes() error {

	_, err := p.selectiveLoggingAndUpd([]string{"time"}, []interface{}{p.BlockData.Time}, "upd_full_nodes", []string{`update`}, nil, false)
	if err != nil {
		return p.ErrInfo(err)
	}

	// выбирем ноды, где wallet_id
	data, err := p.GetAll(`SELECT * FROM full_nodes WHERE wallet_id != 0`, -1)
	if err != nil {
		return p.ErrInfo(err)
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return p.ErrInfo(err)
	}

	log.Debug("data %v", data)
	log.Debug("data %v", data[0])
	log.Debug("data %v", data[0]["rb_id"])
	// логируем их в одну запись JSON
	rbId, err := p.ExecSqlGetLastInsertId(`INSERT INTO rb_full_nodes (full_nodes_wallet_json, block_id, prev_rb_id) VALUES (?, ?, ?)`, "rb_full_nodes", string(jsonData), p.BlockData.BlockId, data[0]["rb_id"])
	if err != nil {
		return p.ErrInfo(err)
	}

	// удаляем где wallet_id
	err = p.ExecSql(`DELETE FROM full_nodes WHERE wallet_id != 0`)
	if err != nil {
		return p.ErrInfo(err)
	}
	maxId, err := p.Single(`SELECT max(id) FROM full_nodes`).Int64()
	if err != nil {
		return p.ErrInfo(err)
	}

	// обновляем AI
	if p.ConfigIni["db_type"] == "sqlite" {
		err = p.SetAI("full_nodes", maxId)
	} else {
		err = p.SetAI("full_nodes", maxId+1)
	}
	if err != nil {
		return p.ErrInfo(err)
	}

	where := ``
	if p.BlockData.BlockId > 1604000 {
		// min 10000 EGS ~ 5000$
		where = ` AND amount > 10000000000000000000000`
	} else if p.BlockData.BlockId > 23900 {
		// min 100 EGS ~ 10$
		where = ` AND amount > 100000000000000000000`
	}
	// получаем новые данные по wallet-нодам
	all, err := p.GetList(`SELECT address_vote FROM dlt_wallets WHERE address_vote !='' `+where+` GROUP BY address_vote ORDER BY sum(amount) DESC LIMIT 101`).String()
	if err != nil {
		return p.ErrInfo(err)
	}
	for _, address_vote := range all {
		dlt_wallets, err := p.OneRow(`SELECT host, wallet_id FROM dlt_wallets WHERE wallet_id = ?`, int64(lib.StringToAddress(address_vote))).String()
		if err != nil {
			return p.ErrInfo(err)
		}
		// вставляем новые данные по wallet-нодам с указанием общего rb_id
		err = p.ExecSql(`INSERT INTO full_nodes (wallet_id, host, rb_id) VALUES (?, ?, ?)`, dlt_wallets["wallet_id"], dlt_wallets["host"], rbId)
		if err != nil {
			return p.ErrInfo(err)
		}
	}

	return nil
}

func (p *Parser) UpdFullNodesRollback() error {
	err := p.selectiveRollback("upd_full_nodes", "", false)
	if err != nil {
		return p.ErrInfo(err)
	}

	// получим rb_id чтобы восстановить оттуда данные
	rbId, err := p.Single(`SELECT rb_id FROM full_nodes WHERE wallet_id != 0`).Int64()
	if err != nil {
		return p.ErrInfo(err)
	}
	log.Debug("rb_id %v", rbId)

	full_nodes_wallet_json, err := p.Single(`SELECT full_nodes_wallet_json FROM rb_full_nodes WHERE rb_id = ?`, rbId).Bytes()
	if err != nil {
		return p.ErrInfo(err)
	}
	log.Debug("full_nodes_wallet_json %v", full_nodes_wallet_json)
	full_nodes_wallet := []map[string]string{{}}
	err = json.Unmarshal(full_nodes_wallet_json, &full_nodes_wallet)
	if err != nil {
		return p.ErrInfo(fmt.Sprintf("%v : (%s)", err, full_nodes_wallet_json))
	}

	// удаляем новые данные
	err = p.ExecSql(`DELETE FROM full_nodes WHERE wallet_id != 0`)
	if err != nil {
		return p.ErrInfo(err)
	}

	maxId, err := p.Single(`SELECT max(id) FROM full_nodes`).Int64()
	if err != nil {
		return p.ErrInfo(err)
	}

	// обновляем AI
	if p.ConfigIni["db_type"] == "sqlite" {
		err = p.SetAI("full_nodes", maxId)
	} else {
		err = p.SetAI("full_nodes", maxId+1)
	}
	if err != nil {
		return p.ErrInfo(err)
	}

	// удаляем новые данные
	err = p.ExecSql(`DELETE FROM rb_full_nodes WHERE rb_id = ?`, rbId)
	if err != nil {
		return p.ErrInfo(err)
	}
	p.rollbackAI("rb_full_nodes", 1)

	for _, data := range full_nodes_wallet {
		// вставляем новые данные по wallet-нодам с указанием общего rb_id
		err = p.ExecSql(`INSERT INTO full_nodes (id, host, wallet_id, state_id, final_delegate_wallet_id, final_delegate_state_id, rb_id) VALUES (?, ?, ?, ?, ?, ?, ?)`, data["id"], data["host"], data["wallet_id"], data["state_id"], data["final_delegate_wallet_id"], data["final_delegate_state_id"], data["rb_id"])
		if err != nil {
			return p.ErrInfo(err)
		}
	}

	return nil
}
