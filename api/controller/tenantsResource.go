package controller

import (
	"fmt"
	"github.com/goodrain/rainbond/api/handler"
	"github.com/goodrain/rainbond/api/middleware"
	api_model "github.com/goodrain/rainbond/api/model"
	"net/http"
)

func (t *TenantStruct) CheckTenantResource(w http.ResponseWriter, r *http.Request) {
	var tr api_model.CheckTenantResourcesReq
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &tr, nil)
	if !ok {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	tenant := r.Context().Value(middleware.ContextKey("tenant")).(*dbmodel.Tenants)
	err := handler.CheckTenantResource(tenant, tr.NeedMemory)
	if err != nil {
		httputil.ReturnResNotEnough(r, w, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, "success!")
}
