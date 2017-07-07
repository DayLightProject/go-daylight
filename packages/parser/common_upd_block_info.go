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
	"fmt"

	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/crypto"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

// UpdBlockInfo updates info_block table
func (p *Parser) UpdBlockInfo() {

	blockID := p.BlockData.BlockId
	// для локальных тестов
	// for the local tests
	if p.BlockData.BlockId == 1 {
		if *utils.StartBlockID != 0 {
			blockID = *utils.StartBlockID
		}
	}
	forSha := fmt.Sprintf("%d,%s,%s,%d,%d,%d", blockID, p.PrevBlock.Hash, p.MrklRoot, p.BlockData.Time, p.BlockData.WalletId, p.BlockData.StateID)
	log.Debug("forSha", forSha)
	hash, err := crypto.DoubleHash([]byte(forSha))
	if err != nil {
		log.Fatal(err)
	}
	hash = converter.BinToHex(hash)
	p.BlockData.Hash = hash
	log.Debug("%v", p.BlockData.Hash)
	log.Debug("%v", blockID)
	log.Debug("%v", p.BlockData.Time)
	log.Debug("%v", p.CurrentVersion)

	if p.BlockData.BlockId == 1 {
		err := p.CreateInfoBlock(p.BlockData.Hash, blockID, p.BlockData.Time, p.BlockData.StateID, p.BlockData.WalletId, p.CurrentVersion)
		if err != nil {
			log.Error("%v", err)
		}
	} else {
		err := p.UpdateInfoBlock(p.BlockData.Hash, blockID, p.BlockData.Time, p.BlockData.StateID, p.BlockData.WalletId)
		if err != nil {
			log.Error("%v", err)
		}
		err = p.AnotherUpdateBlockID(blockID, blockID)
		if err != nil {
			log.Error("%v", err)
		}
	}
}
