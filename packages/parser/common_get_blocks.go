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
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/EGaaS/go-egaas-mvp/packages/consts"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

/*
 * $get_block_script_name, $add_node_host используется только при работе в защищенном режиме и только из blocks_collection.php
 * */
func (p *Parser) GetOldBlocks(walletId, CBID, blockId int64, host string, goroutineName string, dataTypeBlockBody int64) error {
	log.Debug("walletId", walletId, "CBID", CBID, "blockId", blockId)
	err := p.GetBlocks(blockId, host, "rollback_blocks_2", goroutineName, dataTypeBlockBody)
	if err != nil {
		log.Error("v", err)
		return err
	}
	return nil
}

func (p *Parser) GetBlocks(blockId int64, host string, rollbackBlocks, goroutineName string, dataTypeBlockBody int64) error {

	log.Debug("blockId", blockId)

	parser := new(Parser)
	parser.DCDB = p.DCDB
	var count int64
	blocks := make(map[int64]string)
	for {
		/*
			// отметимся в БД, что мы живы.
			upd_deamon_time($db);
			// отметимся, чтобы не спровоцировать очистку таблиц
			upd_main_lock($db);
			// проверим, не нужно нам выйти, т.к. обновилась версия скрипта
			if (check_deamon_restart($db)){
				main_unlock();
				exit;
			}*/
		if blockId < 2 {
			return utils.ErrInfo(errors.New("block_id < 2"))
		}
		// если превысили лимит кол-ва полученных от нода блоков
		var rollback = consts.RB_BLOCKS_1
		if rollbackBlocks == "rollback_blocks_2" {
			rollback = consts.RB_BLOCKS_2
		}
		if count > int64(rollback) {
			ClearTmp(blocks)
			return utils.ErrInfo(errors.New("count > variables[rollback_blocks]"))
		}

		// качаем тело блока с хоста host
		dwI := 0
		var binaryBlock []byte
		var err error
		for {
			binaryBlock, err = utils.GetBlockBody(host, blockId, dataTypeBlockBody)
			if err == nil {
				break
			} else {
				utils.Sleep(1)
			}
			if dwI > 5 {
				break
			}
			dwI++
		}
		if err != nil {
			ClearTmp(blocks)
			return utils.ErrInfo(err)
		}
		log.Debug("binaryBlock: %x\n", binaryBlock)
		binaryBlockFull := binaryBlock
		if len(binaryBlock) == 0 {
			log.Debug("len(binaryBlock) == 0")
			ClearTmp(blocks)
			return utils.ErrInfo(errors.New("len(binaryBlock) == 0"))
		}
		utils.BytesShift(&binaryBlock, 1) // уберем 1-й байт - тип (блок/тр-я)
		// распарсим заголовок блока
		blockData := utils.ParseBlockHeader(&binaryBlock)
		log.Debug("blockData", blockData)

		// если существуют глючная цепочка, тот тут мы её проигнорируем
		badBlocks_, err := p.Single("SELECT bad_blocks FROM config").Bytes()
		if err != nil {
			ClearTmp(blocks)
			return utils.ErrInfo(err)
		}
		badBlocks := make(map[int64]string)
		if len(badBlocks_) > 0 {
			err = json.Unmarshal(badBlocks_, &badBlocks)
			if err != nil {
				ClearTmp(blocks)
				return utils.ErrInfo(err)
			}
		}
		if badBlocks[blockData.BlockId] == string(utils.BinToHex(blockData.Sign)) {
			ClearTmp(blocks)
			return utils.ErrInfo(errors.New("bad block"))
		}
		if blockData.BlockId != blockId {
			ClearTmp(blocks)
			return utils.ErrInfo(errors.New("bad block_data['block_id']"))
		}

		// размер блока не может быть более чем max_block_size
		if int64(len(binaryBlock)) > consts.MAX_BLOCK_SIZE {
			ClearTmp(blocks)
			return utils.ErrInfo(errors.New(`len(binaryBlock) > variables.Int64["max_block_size"]`))
		}

		// нам нужен хэш предыдущего блока, чтобы найти, где началась вилка
		prevBlockHash, err := p.Single("SELECT hash FROM block_chain WHERE id  =  ?", blockId-1).String()
		if err != nil {
			ClearTmp(blocks)
			return utils.ErrInfo(err)
		}

		// нам нужен меркель-рут текущего блока
		mrklRoot, err := utils.GetMrklroot(binaryBlock, false)
		if err != nil {
			ClearTmp(blocks)
			return utils.ErrInfo(err)
		}

		// публичный ключ того, кто этот блок сгенерил
		nodePublicKey, err := p.GetNodePublicKeyWalletOrCB(blockData.WalletId, blockData.CBID)
		if err != nil {
			return utils.ErrInfo(err)
		}

		// SIGN от 128 байта до 512 байт. Подпись от TYPE, BLOCK_ID, PREV_BLOCK_HASH, TIME, WALLET_ID, state_id, MRKL_ROOT
		forSign := fmt.Sprintf("0,%v,%x,%v,%v,%v,%s", blockData.BlockId, prevBlockHash, blockData.Time, blockData.WalletId, blockData.CBID, mrklRoot)
		log.Debug("forSign", forSign)

		// проверяем подпись
		_, okSignErr := utils.CheckSign([][]byte{nodePublicKey}, forSign, blockData.Sign, true)
		log.Debug("okSignErr", okSignErr)

		// сам блок сохраняем в файл, чтобы не нагружать память
		file, err := ioutil.TempFile(*utils.Dir, "DC")
		defer os.Remove(file.Name())
		_, err = file.Write(binaryBlockFull)
		if err != nil {
			ClearTmp(blocks)
			return utils.ErrInfo(err)
		}
		blocks[blockId] = file.Name()
		blockId--
		count++

		// качаем предыдущие блоки до тех пор, пока отличается хэш предыдущего.
		// другими словами, пока подпись с prevBlockHash будет неверной, т.е. пока что-то есть в okSignErr
		if okSignErr == nil {
			log.Debug("plug found blockId=%v\n", blockData.BlockId)
			break
		}
	}

	// чтобы брать блоки по порядку
	blocksSorted := utils.SortMap(blocks)
	log.Debug("blocks", blocksSorted)
	log.Debug("blocks len %d", len(blocksSorted))

	// получим наши транзакции в 1 бинарнике, просто для удобства
	var transactions []byte
	utils.WriteSelectiveLog(`SELECT data FROM transactions WHERE verified = 1 AND used = 0`)
	all, err := p.GetAll(`SELECT data FROM transactions WHERE verified = 1 AND used = 0`, -1)
	if err != nil {
		utils.WriteSelectiveLog(err)
		return utils.ErrInfo(err)
	}
	for _, data := range all {
		utils.WriteSelectiveLog(utils.BinToHex(data["data"]))
		log.Debug("data", data)
		transactions = append(transactions, utils.EncodeLengthPlusData([]byte(data["data"]))...)
	}
	if len(transactions) > 0 {
		// отмечаем, что эти тр-ии теперь нужно проверять по новой
		utils.WriteSelectiveLog("UPDATE transactions SET verified = 0 WHERE verified = 1 AND used = 0")
		affect, err := p.ExecSqlGetAffect("UPDATE transactions SET verified = 0 WHERE verified = 1 AND used = 0")
		if err != nil {
			utils.WriteSelectiveLog(err)
			return utils.ErrInfo(err)
		}
		utils.WriteSelectiveLog("affect: " + utils.Int64ToStr(affect))
		// откатываем по фронту все свежие тр-ии
		/*parser.GoroutineName = goroutineName
		parser.BinaryData = transactions
		err = parser.ParseDataRollbackFront(false)
		if err != nil {
			return utils.ErrInfo(err)
		}*/
	}

	// откатываем наши блоки до начала вилки
	rows, err := p.Query(p.FormatQuery(`
			SELECT data
			FROM block_chain
			WHERE id > ?
			ORDER BY id DESC`), blockId)
	if err != nil {
		return p.ErrInfo(err)
	}
	defer rows.Close()
	for rows.Next() {
		var data []byte
		err = rows.Scan(&data)
		if err != nil {
			return p.ErrInfo(err)
		}
		log.Debug("We roll away blocks before plug", blockId)
		parser.GoroutineName = goroutineName
		parser.BinaryData = data
		err = parser.ParseDataRollback()
		if err != nil {
			return utils.ErrInfo(err)
		}
	}
	log.Debug("blocks", blocksSorted)

	prevBlock := make(map[int64]*utils.BlockData)

	// проходимся по новым блокам
	for _, data := range blocksSorted {
		for intBlockId, tmpFileName := range data {
			log.Debug("Go on new blocks", intBlockId, tmpFileName)

			// проверяем и заносим данные
			binaryBlock, err := ioutil.ReadFile(tmpFileName)
			if err != nil {
				return utils.ErrInfo(err)
			}
			log.Debug("binaryBlock: %x\n", binaryBlock)
			parser.GoroutineName = goroutineName
			parser.BinaryData = binaryBlock
			// передаем инфу о предыдущем блоке, т.к. это новые блоки, то инфа о предыдущих блоках в block_chain будет всё еще старая, т.к. обновление block_chain идет ниже
			if prevBlock[intBlockId-1] != nil {
				log.Debug("prevBlock[intBlockId-1] != nil : %v", prevBlock[intBlockId-1])
				parser.PrevBlock.Hash = prevBlock[intBlockId-1].Hash
				parser.PrevBlock.Time = prevBlock[intBlockId-1].Time
				parser.PrevBlock.BlockId = prevBlock[intBlockId-1].BlockId
				parser.PrevBlock.WalletId = prevBlock[intBlockId-1].WalletId
			}

			// если вернулась ошибка, значит переданный блок уже откатился
			// info_block и config.my_block_id обновляются только если ошибки не было
			err = parser.ParseDataFull(false)
			// для последующей обработки получим хэши и time
			if err == nil {
				prevBlock[intBlockId] = parser.GetBlockInfo()
				log.Debug("prevBlock[%d] = %v", intBlockId, prevBlock[intBlockId])
			}
			// если есть ошибка, то откатываем все предыдущие блоки из новой цепочки
			if err != nil {
				parser.BlockError(err)
				log.Debug("there is an error is rolled back all previous blocks of a new chain: %v", err)

				// баним на 1 час хост, который дал нам ложную цепочку
				err = p.NodesBan(fmt.Sprintf("%s", err))
				if err != nil {
					return utils.ErrInfo(err)
				}
				// обязательно проходимся по блокам в обратном порядке
				blocksSorted := utils.RSortMap(blocks)
				for _, data := range blocksSorted {
					for int2BlockId, tmpFileName := range data {
						log.Debug("int2BlockId", int2BlockId)
						if int2BlockId >= intBlockId {
							continue
						}
						binaryBlock, err := ioutil.ReadFile(tmpFileName)
						if err != nil {
							return utils.ErrInfo(err)
						}
						parser.GoroutineName = goroutineName
						parser.BinaryData = binaryBlock
						err = parser.ParseDataRollback()
						if err != nil {
							return utils.ErrInfo(err)
						}
					}
				}
				// заносим наши данные из block_chain, которые были ранее
				log.Debug("We push data from our block_chain, which were previously")
				rows, err := p.Query(p.FormatQuery(`
					SELECT data
					FROM block_chain
					WHERE id > ?
					ORDER BY id ASC`), blockId)
				if err != nil {
					return p.ErrInfo(err)
				}
				defer rows.Close()
				for rows.Next() {
					var data []byte
					err = rows.Scan(&data)
					if err != nil {
						return p.ErrInfo(err)
					}
					log.Debug("blockId", blockId, "intBlockId", intBlockId)
					parser.GoroutineName = goroutineName
					parser.BinaryData = data
					err = parser.ParseDataFull(false)
					if err != nil {
						return utils.ErrInfo(err)
					}
				}
				// т.к. в предыдущем запросе к block_chain могло не быть данных, т.к. $block_id больше чем наш самый большой id в block_chain
				// то значит info_block мог не обновится и остаться от занесения новых блоков, что приведет к пропуску блока в block_chain
				lastMyBlock, err := p.OneRow("SELECT * FROM block_chain ORDER BY id DESC").String()
				if err != nil {
					return utils.ErrInfo(err)
				}
				binary := []byte(lastMyBlock["data"])
				utils.BytesShift(&binary, 1) // уберем 1-й байт - тип (блок/тр-я)
				lastMyBlockData := utils.ParseBlockHeader(&binary)
				err = p.ExecSql(`
					UPDATE info_block
					SET   hash = [hex],
							block_id = ?,
							time = ?,
							sent = 0
					`, utils.BinToHex(lastMyBlock["hash"]), lastMyBlockData.BlockId, lastMyBlockData.Time)
				if err != nil {
					return utils.ErrInfo(err)
				}
				err = p.ExecSql(`UPDATE config SET my_block_id = ?`, lastMyBlockData.BlockId)
				if err != nil {
					return utils.ErrInfo(err)
				}
				ClearTmp(blocks)
				return utils.ErrInfo(err) // переходим к следующему блоку в queue_blocks
			}
		}
	}
	log.Debug("remove the blocks and enter new block_chain")

	// если всё занеслось без ошибок, то удаляем блоки из block_chain и заносим новые
	affect, err := p.ExecSqlGetAffect("DELETE FROM block_chain WHERE id > ?", blockId)
	if err != nil {
		return utils.ErrInfo(err)
	}
	log.Debug("affect", affect)
	log.Debug("prevblock", prevBlock)
	log.Debug("blocks", blocks)

	// для поиска бага
	maxBlockId, err := p.Single("SELECT id FROM block_chain ORDER BY id DESC LIMIT 1").Int64()
	if err != nil {
		return utils.ErrInfo(err)
	}
	log.Debug("maxBlockId", maxBlockId)

	// проходимся по новым блокам
	blocksSorted_ := utils.SortMap(blocks)
	log.Debug("blocksSorted_", blocksSorted_)
	for _, data := range blocksSorted_ {
		for blockId, tmpFileName := range data {

			block, err := ioutil.ReadFile(tmpFileName)
			if err != nil {
				return utils.ErrInfo(err)
			}
			blockHex := utils.BinToHex(block)

			// пишем в цепочку блоков
			err = p.ExecSql("UPDATE info_block SET hash = [hex], block_id = ?, time = ?, wallet_id = ?, state_id = ?, sent = 0", prevBlock[blockId].Hash, prevBlock[blockId].BlockId, prevBlock[blockId].Time, prevBlock[blockId].WalletId, prevBlock[blockId].CBID)
			if err != nil {
				return utils.ErrInfo(err)
			}
			err = p.ExecSql(`UPDATE config SET my_block_id = ?`, prevBlock[blockId].BlockId)
			if err != nil {
				return utils.ErrInfo(err)
			}

			// т.к. эти данные создали мы сами, то пишем их сразу в таблицу проверенных данных, которые будут отправлены другим нодам
			exists, err := p.Single("SELECT id FROM block_chain WHERE id = ?", blockId).Int64()
			if err != nil {
				return utils.ErrInfo(err)
			}
			if exists == 0 {
				affect, err := p.ExecSqlGetAffect("INSERT INTO block_chain (id, hash, state_id, wallet_id, time, data) VALUES (?, [hex], ?, ?, ?, [hex])", blockId, prevBlock[blockId].Hash, prevBlock[blockId].CBID, prevBlock[blockId].WalletId, prevBlock[blockId].Time, blockHex)
				if err != nil {
					return utils.ErrInfo(err)
				}
				log.Debug("affect", affect)
			}
			err = os.Remove(tmpFileName)
			if err != nil {
				return utils.ErrInfo(err)
			}
			log.Debug("tmpFileName %v", tmpFileName)
			// для поиска бага
			maxBlockId, err := p.Single("SELECT id FROM block_chain ORDER BY id DESC LIMIT 1").Int64()
			if err != nil {
				return utils.ErrInfo(err)
			}
			log.Debug("maxBlockId", maxBlockId)
		}
	}
	log.Debug("HAPPY END")

	return nil
}
