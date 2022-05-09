package controller

import (
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/application"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/domain/entity"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/interfaces/errors"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/interfaces/httputil"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/interfaces/middleware"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Template struct {
	template application.TemplateInterface
}

func NewTemplateController(template application.TemplateInterface) *Template {
	return &Template{
		template: template,
	}
}

func (a *Template) ListTemplate(c *gin.Context) {
	ctx := middleware.DefaultTodoNovaContext(c)
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("%s", traceID)

	resp := httputil.DefaultResponse()
	defer httputil.HTTPJSONResponse(ctx, c, &resp)

	var param entity.ListTemplateReq
	if err := c.ShouldBindQuery(&param); err != nil {
		log.Errorf("validate template ShouldBindQuery failed, traceID:%s, err:%v", traceID, err)
		httputil.SetErrorResponse(&resp, errors.CodeInvalidParams, "Invalid query provided")
		return
	}
	errMsg := param.Validate()
	if len(errMsg) > 0 {
		log.Errorf("validate template param failed, traceID:%s, errMsg:%s", traceID, errMsg)
		httputil.SetErrorResponse(&resp, errors.CodeInvalidParams, errMsg)
		return
	}
	templates, err := a.template.ListTemplate(ctx, param)
	if err != nil {
		log.Errorf("ListTemplate TemplateInterface get template list failed,traceID:%s,err:%v", traceID, err)
		httputil.SetErrorResponseWithError(&resp, err)
		return
	}
	httputil.SetSuccessfulResponse(&resp, errors.CodeOK, templates)
}
