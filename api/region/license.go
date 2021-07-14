// RAINBOND, Application Management Platform
// Copyright (C) 2021-2021 Goodrain Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package region

import (
	"fmt"
	"github.com/goodrain/rainbond/api/util"
	licenseutil "github.com/goodrain/rainbond/api/util/license"
	utilhttp "github.com/goodrain/rainbond/util/http"
)

func (r *regionImpl) License() LicenseInterface {
	return &license{regionImpl: *r, prefix: "/license"}
}

type license struct {
	regionImpl
	prefix string
}

type LicenseInterface interface {
	Get() (*licenseutil.LicenseResp, *util.APIHandleError)
}

// Get -
func (l *license) Get() (*licenseutil.LicenseResp, *util.APIHandleError) {
	var res utilhttp.ResponseBody
	var lic licenseutil.LicenseResp
	res.Bean = &lic
	code, err := l.DoRequest(l.prefix, "GET", nil, &res)
	if err != nil {
		return nil, util.CreateAPIHandleError(code, err)
	}
	if code != 200 {
		return nil, util.CreateAPIHandleError(code, fmt.Errorf("Get license code %d", code))
	}
	return &lic, nil
}
