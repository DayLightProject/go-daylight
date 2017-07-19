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

package daemons

import (
	"fmt"
	"time"

	"github.com/EGaaS/go-egaas-mvp/packages/consts"
	"github.com/EGaaS/go-egaas-mvp/packages/converter"
	"github.com/EGaaS/go-egaas-mvp/packages/model"
	"github.com/EGaaS/go-egaas-mvp/packages/parser"
	"github.com/EGaaS/go-egaas-mvp/packages/utils"
)

/* Берем блок. Если блок имеет лучший хэш, то ищем, в каком блоке у нас пошла вилка // Take the block. If the block has the best hash, then look for the block where the fork started
 * Если вилка пошла менее чем variables->rollback_blocks блоков назад, то // If the fork begins less then variables->rollback_blocks blocks ago, than
 *  - получаем всю цепочку блоков, // get the whole chain of blocks
 *  - откатываем фронтальные данные от наших блоков, // roll back the frontal data from our blocks
 *  - заносим фронт. данные из новой цепочки // insert the frontal data from a new chain
 *  - если нет ошибок, то откатываем наши данные из блоков // if there is no error, then roll back our data from the blocks
 *  - и заносим новые данные // and insert new data
 *  - если где-то есть ошибки, то откатываемся к нашим прежним данным // if there are errors, then roll back to the former data
 * Если вилка была давно, то ничего не трогаем, и оставлеяем скрипту blocks_collection.php // if the fork was long ago then do not touch anything and leave the script blocks_collection.php
 * Ограничение variables->rollback_blocks нужно для защиты от подставных блоков // the limitation variables->rollback_blocks is needed for the protection against the false blocks
 *
 * */

// QueueParserBlocks parses blocks from the queue
func QueueParserBlocks(chBreaker chan bool, chAnswer chan string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("daemon Recovered", r)
			panic(r)
		}
	}()

	const GoroutineName = "QueueParserBlocks"
	d := new(daemon)
	d.DCDB = DbConnect(chBreaker, chAnswer, GoroutineName)
	if d.DCDB == nil {
		return
	}
	d.goRoutineName = GoroutineName
	d.chAnswer = chAnswer
	d.chBreaker = chBreaker
	d.sleepTime = 1

	if !d.CheckInstall(chBreaker, chAnswer, GoroutineName) {
		return
	}
	d.DCDB = DbConnect(chBreaker, chAnswer, GoroutineName)
	if d.DCDB == nil {
		return
	}

BEGIN:
	for {
		logger.Info(GoroutineName)
		MonitorDaemonCh <- []string{GoroutineName, converter.Int64ToStr(time.Now().Unix())}

		// проверим, не нужно ли нам выйти из цикла
		// check if we have to break the cycle
		if CheckDaemonsRestart(chBreaker, chAnswer, GoroutineName) {
			break BEGIN
		}

		restart, err := d.dbLock()
		if restart {
			break BEGIN
		}
		if err != nil {
			if d.dPrintSleep(err, d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}

		infoBlock := &model.InfoBlock{}
		err = infoBlock.GetInfoBlock()
		if err != nil {
			if d.unlockPrintSleep(utils.ErrInfo(err), d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}
		queueBlock := &model.QueueBlocks{}
		err = queueBlock.GetQueueBlock()
		if err != nil {
			if d.unlockPrintSleep(utils.ErrInfo(err), d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}
		if len(queueBlock.Hash) == 0 {
			if d.unlockPrintSleep(utils.ErrInfo(err), d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}
		queueBlock.Hash = converter.BinToHex(queueBlock.Hash)
		infoBlock.Hash = converter.BinToHex(infoBlock.Hash)

		/*
		 * Базовая проверка
		 */
		// basic check

		// проверим, укладывается ли блок в лимит
		// check if the block gets in the rollback_blocks_1 limit
		if queueBlock.BlockID > infoBlock.BlockID+consts.RB_BLOCKS_1 {
			queueBlock.Delete()
			if d.unlockPrintSleep(utils.ErrInfo("rollback_blocks_1"), 1) {
				break BEGIN
			}
			continue BEGIN
		}

		// проверим не старый ли блок в очереди
		// check whether the new block is in the turn
		if queueBlock.BlockID <= infoBlock.BlockID {
			queueBlock.Delete()
			if d.unlockPrintSleepInfo(utils.ErrInfo("old block"), 1) {
				break BEGIN
			}
			continue BEGIN
		}

		/*
		 * Загрузка блоков для детальной проверки
		 */
		// download of the blocks for the detailed check
		fullNode := &model.FullNodes{}

		err = fullNode.FindNodeByID(queueBlock.FullNodeID)
		if err != nil {
			queueBlock.Delete()
			if d.unlockPrintSleep(utils.ErrInfo(err), d.sleepTime) {
				break BEGIN
			}
			continue BEGIN
		}
		blockID := queueBlock.BlockID

		p := new(parser.Parser)
		p.DCDB = d.DCDB
		p.GoroutineName = GoroutineName
		err = p.GetBlocks(blockID, fullNode.Host+":"+consts.TCP_PORT, "rollback_blocks_1", GoroutineName, 7)
		if err != nil {
			logger.Error("v", err)
			queueBlock.Delete()
			d.NodesBan(fmt.Sprintf("%v", err))
			if d.unlockPrintSleep(utils.ErrInfo(err), 1) {
				break BEGIN
			}
			continue BEGIN
		}

		d.dbUnlock()

		if d.dSleep(d.sleepTime) {
			break BEGIN
		}
	}
	logger.Debug("break BEGIN %v", GoroutineName)

}
