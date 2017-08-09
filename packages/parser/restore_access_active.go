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

type RestoreAccessActiveParser struct {
	*Parser
	RestoreAccessActive *tx.RestoreAccessActive
	SecretHex           string
	Active              int64
}

func (p *RestoreAccessActiveParser) Init() error {
	restoreAccessActive := &tx.RestoreAccessActive{}
	if err := msgpack.Unmarshal(p.TxBinaryData, restoreAccessActive); err != nil {
		return p.ErrInfo(err)
	}
	p.RestoreAccessActive = restoreAccessActive
	p.SecretHex = string(converter.BinToHex(p.RestoreAccessActive.Secret))
	if p.SecretHex == "30" {
		p.Active = 0
	} else {
		p.Active = 1
	}
	return nil
}

func (p *RestoreAccessActiveParser) Validate() error {
	err := p.generalCheck(`system_restore_access_active`, &p.RestoreAccessActive.Header, map[string]string{})
	if err != nil {
		return p.ErrInfo(err)
	}
	if len(p.RestoreAccessActive.Secret) > 2048 {
		return p.ErrInfo("len secret > 2048")
	}

	// check that there is no repeat shift
	restoreAccess := &model.SystemRestoreAccess{}
	err = restoreAccess.Get(p.RestoreAccessActive.StateID)
	if err != nil {
		return p.ErrInfo(err)
	}
	if restoreAccess.Active == p.Active {
		return p.ErrInfo("active")
	}
	CheckSignResult, err := utils.CheckSign(p.PublicKeys, p.RestoreAccessActive.ForSign(), p.RestoreAccessActive.BinSignatures, false)
	if err != nil {
		return p.ErrInfo(err)
	}
	if !CheckSignResult {
		return p.ErrInfo("incorrect sign")
	}

	if err = p.AccessRights(`restore_access_condition`, false); err != nil {
		return p.ErrInfo(err)
	}
	return nil
}

func (p *RestoreAccessActiveParser) Action() error {
	_, _, err := p.selectiveLoggingAndUpd([]string{"active", "secret"}, []interface{}{p.Active, p.RestoreAccessActive.Secret}, "system_restore_access", []string{"state_id"}, []string{converter.Int64ToStr(p.RestoreAccessActive.Header.StateID)}, true)
	if err != nil {
		return p.ErrInfo(err)
	}
	return nil
}

func (p *RestoreAccessActiveParser) Rollback() error {
	return p.autoRollback()
}

func (p RestoreAccessActiveParser) Header() *tx.Header {
	return &p.RestoreAccessActive.Header
}
