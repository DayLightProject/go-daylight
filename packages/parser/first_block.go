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

	"github.com/EGaaS/go-egaas-mvp/packages/consts"
	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/crypto"
	"github.com/EGaaS/go-egaas-mvp/packages/utils/tx"
	"gopkg.in/vmihailenco/msgpack.v2"
)

type FirstBlockParser struct {
	*Parser
	FirstBlock *tx.FirstBlock
}

func (p *FirstBlockParser) Init() error {
	firstBlock := &tx.FirstBlock{}
	if err := msgpack.Unmarshal(p.TxBinaryData, firstBlock); err != nil {
		return p.ErrInfo(err)
	}
	p.FirstBlock = firstBlock
	return nil
}

func (p *FirstBlockParser) Validate() error {
	return nil
}

func (p *FirstBlockParser) Action() error {
	//data := p.TxPtr.(*consts.FirstBlock)
	//	myAddress := b58.Encode(lib.Address(data.PublicKey)) //utils.HashSha1Hex(p.TxMaps.Bytes["public_key"]);
	publicKey := p.FirstBlock.PublicKey
	myAddress := crypto.Address(publicKey)
	log.Debug("data.PublicKey %s", publicKey)
	log.Debug("data.PublicKey %x", publicKey)
	err := p.ExecSQL(`INSERT INTO dlt_wallets (wallet_id, host, address_vote, public_key_0, node_public_key, amount) VALUES (?, ?, ?, [hex], [hex], ?)`,
		myAddress, p.FirstBlock.Host, converter.AddressToString(myAddress), hex.EncodeToString(publicKey), hex.EncodeToString(p.FirstBlock.NodePublicKey), consts.FIRST_QDLT)
	//p.TxMaps.String["host"], myAddress, p.TxMaps.Bytes["public_key"], p.TxMaps.Bytes["node_public_key"], consts.FIRST_DLT)
	if err != nil {
		return p.ErrInfo(err)
	}
	err = p.ExecSQL(`INSERT INTO full_nodes (wallet_id, host) VALUES (?,?)`, myAddress, p.FirstBlock.Host) //p.TxMaps.String["host"])
	if err != nil {
		return p.ErrInfo(err)
	}

	return nil
}

func (p *FirstBlockParser) Rollback() error {
	return nil
}

func (p FirstBlockParser) Header() *tx.Header {
	return &p.FirstBlock.Header
}
