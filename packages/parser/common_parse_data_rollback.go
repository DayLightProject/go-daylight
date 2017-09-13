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
	"bytes"
	"fmt"

	"github.com/EGaaS/go-egaas-mvp/packages/consts"
	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/smart"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

func BlockRollback(data []byte) error {
	buf := bytes.NewBuffer(data)
	if buf.Len() == 0 {
		return fmt.Errorf("empty buffer")
	}
	dataType := int(converter.BinToDec(buf.Next(1)))
	if dataType != 0 {
		return utils.ErrInfo(fmt.Errorf("incorrect dataType %d", dataType))
	}

	block, err := parseBlock(buf)
	if err != nil {
		return err
	}

	dbTransaction, err := model.StartTransaction()
	if err != nil {
		return err
	}

	err = doBlockRollback(dbTransaction, block)

	if err != nil {
		dbTransaction.Rollback()
		return err
	}

	b := &model.Block{}
	err = b.DeleteById(dbTransaction, block.Header.BlockID)
	if err != nil {
		dbTransaction.Rollback()
		return err
	}

	err = dbTransaction.Commit()
	return err
}

func doBlockRollback(transaction *model.DbTransaction, block *Block) error {
	// rollback transactions in reverse order
	for i := len(block.Parsers) - 1; i >= 0; i-- {
		p := block.Parsers[i]
		p.DbTransaction = transaction

		_, err := model.MarkTransactionUnusedAndUnverified(transaction, p.TxHash)
		if err != nil {
			return utils.ErrInfo(err)
		}
		_, err = model.DeleteLogTransactionsByHash(transaction, p.TxHash)
		if err != nil {
			return utils.ErrInfo(err)
		}

		ts := &model.TransactionStatus{}
		err = ts.UpdateBlockID(transaction, 0, p.TxHash)
		if err != nil {
			return utils.ErrInfo(err)
		}

		_, err = model.DeleteQueueTxByHash(transaction, p.TxHash)
		if err != nil {
			return utils.ErrInfo(err)
		}
		queueTx := &model.QueueTx{Hash: p.TxHash, Data: p.TxFullData}
		err = queueTx.Save(transaction)
		if err != nil {
			return p.ErrInfo(err)
		}

		if p.TxContract != nil {
			if err := p.CallContract(smart.CallInit | smart.CallRollback); err != nil {
				return utils.ErrInfo(err)
			}
			if err = p.autoRollback(); err != nil {
				return p.ErrInfo(err)
			}
		} else {
			MethodName := consts.TxTypes[int(p.TxType)]
			parser, err := GetParser(p, MethodName)
			if err != nil {
				return p.ErrInfo(err)
			}
			result := parser.Init()
			if _, ok := result.(error); ok {
				return p.ErrInfo(result.(error))
			}
			result = parser.Rollback()
			if _, ok := result.(error); ok {
				return p.ErrInfo(result.(error))
			}
		}
	}

	return nil
}
