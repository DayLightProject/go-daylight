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

package daylight

import (
	"fmt"
	"time"

	"github.com/EGaaS/go-egaas-mvp/packages/utils"
	"github.com/EGaaS/go-egaas-mvp/packages/utils/sql"
)

// Stop stops the program
func Stop() {
	log.Debug("Stop()")
	IosLog("Stop()")
	var err error
	sql.DB, err = sql.NewDbConnect(configIni)
	log.Debug("DayLight Stop : %v", sql.DB)
	IosLog("utils.DB:" + fmt.Sprintf("%v", sql.DB))
	if err != nil {
		IosLog("err:" + fmt.Sprintf("%s", utils.ErrInfo(err)))
		log.Error("%v", utils.ErrInfo(err))
		//panic(err)
		//os.Exit(1)
	}
	err = sql.DB.CreateStopDaemons(time.Now().Unix())
	if err != nil {
		IosLog("err:" + fmt.Sprintf("%s", utils.ErrInfo(err)))
		log.Error("%v", utils.ErrInfo(err))
	}
	log.Debug("DayLight Stop")
	IosLog("DayLight Stop")
}
