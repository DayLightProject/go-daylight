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

package tcpserver

import (
	"bytes"
	"errors"
	"io"

	"github.com/EGaaS/go-egaas-mvp/packages/consts"
	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/crypto"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

// get the list of transactions which belong to the sender from 'disseminator' daemon
// do not load the blocks here because here could be the chain of blocks that are loaded for a long time
// download the transactions here, because they are small and definitely will be downloaded in 60 sec
func Type1(r *DisRequest, rw io.ReadWriter) error {

	buf := bytes.NewBuffer(r.Data)

	/*
	 *  data structure
	 *  type - 1 byte. 0 - block, 1 - list of transactions
	 *  {if type==1}:
	 *  <any number of the next sets>
	 *   tx_hash - 16 bytes
	 * </>
	 * {if type==0}:
	 *  block_id - 3 bytes
	 *  hash - 32 bytes
	 * <any number of the next sets>
	 *   tx_hash - 16 bytes
	 * </>
	 * */

	// full_node_id of the sender to know where to take a data when it will be downloaded by another daemon
	fullNodeID := converter.BinToDec(buf.Next(2))

	// get data type (0 - block and transactions, 1 - only transactions)
	newDataType := converter.BinToDec(buf.Next(1))

	if newDataType == 0 {
		err := processBlock(buf, fullNodeID)
		if err != nil {
			return err
		}
	}

	// get unknown transactions from received packet
	needTx, err := getUnknownTransactions(buf)
	if err != nil {
		return err
	}

	// send the list of transactions which we want to get
	err = SendRequest(&DisHashResponse{Data: needTx}, rw)
	if err != nil {
		return err
	}

	if len(needTx) == 0 {
		return nil
	}

	// get this new transactions
	trs := &DisRequest{}
	err = ReadRequest(trs, rw)
	if err != nil {
		return err
	}

	// and save them
	return saveNewTransactions(trs)
}

func processBlock(buf *bytes.Buffer, fullNodeID int64) error {

	blockID, err := model.GetCurBlockID()
	if err != nil {
		return utils.ErrInfo(err)
	}

	// get block ID
	newBlockID := converter.BinToDec(buf.Next(3))
	log.Debug("newDataBlockID: %d / blockID: %d", newBlockID, blockID)

	// get block hash
	blockHash := buf.Next(32)

	// we accept only new blocks
	if newBlockID >= blockID {
		newDataHash := converter.BinToHex(blockHash)
		queueBlock := &model.QueueBlock{Hash: newDataHash, FullNodeID: fullNodeID, BlockID: newBlockID}
		err = queueBlock.Create()
		if err != nil {
			return utils.ErrInfo(err)
		}
	}

	return nil
}

func getUnknownTransactions(buf *bytes.Buffer) ([]byte, error) {

	var needTx []byte
	for buf.Len() > 0 {
		newDataTxHash := converter.BinToHex(buf.Next(16))
		if len(newDataTxHash) == 0 {
			return nil, errors.New("wrong transactions hash size")
		}
		log.Debug("newDataTxHash %s", newDataTxHash)

		// check if we have such a transaction
		// check log_transaction
		exists, err := model.GetLogTransactionsCount(newDataTxHash)
		if err != nil {
			return nil, utils.ErrInfo(err)
		}
		if exists > 0 {
			log.Debug("exists")
			continue
		}

		exists, err = model.GetTransactionsCount(newDataTxHash)
		if err != nil {
			return nil, utils.ErrInfo(err)
		}
		if exists > 0 {
			log.Debug("exists")
			continue
		}

		// check transaction queue
		exists, err = model.GetQueuedTransactionsCount(newDataTxHash)
		if err != nil {
			return nil, utils.ErrInfo(err)
		}
		if exists > 0 {
			log.Debug("exists")
			continue
		}
		needTx = append(needTx, converter.HexToBin(newDataTxHash)...)
	}

	return needTx, nil
}

func saveNewTransactions(r *DisRequest) error {

	binaryTxs := r.Data
	log.Debug("binaryTxs %x", binaryTxs)

	for len(binaryTxs) > 0 {
		txSize, err := converter.DecodeLength(&binaryTxs)
		if err != nil {
			return err
		}
		if int64(len(binaryTxs)) < txSize {
			return utils.ErrInfo(errors.New("bad transactions packet"))
		}

		txBinData := converter.BytesShift(&binaryTxs, txSize)
		if len(txBinData) == 0 {
			return utils.ErrInfo(errors.New("len(txBinData) == 0"))
		}

		if int64(len(txBinData)) > consts.MAX_TX_SIZE {
			return utils.ErrInfo("len(txBinData) > max_tx_size")
		}

		hash, err := crypto.Hash(txBinData)
		if err != nil {
			log.Fatal(err)
		}

		//txHex := converter.BinToHex(txBinData)
		//hash = converter.BinToHex(hash)

		queueTx := &model.QueueTx{Hash: hash, Data: txBinData, FromGate: 1}
		err = queueTx.Create()
		if err != nil {
			return err
		}
	}
	return nil
}
