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

package api

import (
	"net/http"
)

func authWallet(w http.ResponseWriter, r *http.Request, data *apiData) error {
	if data.sess.Get("wallet") == nil {
		return errorAPI(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
	return nil
}

func authState(w http.ResponseWriter, r *http.Request, data *apiData) error {
	if data.sess.Get("wallet") == nil || data.sess.Get("state").(int64) == 0 {
		return errorAPI(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
	return nil
}