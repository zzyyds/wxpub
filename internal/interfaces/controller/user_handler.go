package controller

import (
	"strconv"

	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/application"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/domain/entity"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/domain/repository"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/interfaces/errors"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/interfaces/httputil"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/interfaces/middleware"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/utils"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type User struct {
	user application.UserInterface
}

func NewUserController(user application.UserInterface) *User {
	return &User{
		user: user,
	}
}

func (u *User) ListUser(c *gin.Context) {
	ctx := middleware.DefaultTodoNovaContext(c)
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("%s", traceID)

	resp := httputil.DefaultResponse()
	defer httputil.HTTPJSONResponse(ctx, c, &resp)

	users, err := u.user.ListUser(ctx)
	if err != nil {
		log.Errorf("ListUser UserInterface get list user by id failed,traceID:%s,err:%v", traceID, err)
		httputil.SetErrorResponseWithError(&resp, err)
		return
	}
	httputil.SetSuccessfulResponse(&resp, errors.CodeOK, users)
}

func (u *User) GetUser(c *gin.Context) {
	ctx := middleware.DefaultTodoNovaContext(c)
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("%s", traceID)

	resp := httputil.DefaultResponse()
	defer httputil.HTTPJSONResponse(ctx, c, &resp)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		log.Errorf("validate param id failed, traceID:%s, err:%v", traceID, err)
		httputil.SetErrorResponse(&resp, errors.CodeInvalidParams, "Invalid id provided")
		return
	}
	user, err := u.user.GetUserByID(ctx, id)
	if err != nil {
		log.Errorf("GetUser UserInterface get user by id failed,traceID:%s,err:%v", traceID, err)
		httputil.SetErrorResponseWithError(&resp, err)
		return
	}
	httputil.SetSuccessfulResponse(&resp, errors.CodeOK, user)
}

func (u *User) SendSms(c *gin.Context) {
	ctx := middleware.DefaultTodoNovaContext(c)
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("%s", traceID)

	resp := httputil.DefaultResponse()
	defer httputil.HTTPJSONResponse(ctx, c, &resp)

	var req entity.SendSmsReq
	err := c.ShouldBindJSON(&req)
	if err != nil {
		log.Errorf("SendSms ShouldBindJSON error: %+v, traceID:%s", err, traceID)
		httputil.SetErrorResponse(&resp, errors.CodeInvalidParams, errors.GetErrorMessage(errors.CodeInvalidParams))
		return
	}

	if utils.VerifyMobilePhoneFormat(req.Phone) {
		log.Errorf("invaild phone number: %s, traceID:%s", req.Phone, traceID)
		httputil.SetErrorResponse(&resp, errors.CodeInvalidParams, errors.GetErrorMessage(errors.CodeInvalidParams))
		return
	}

	err = repository.DefaultPhoneVerifyRepository().SendSms(ctx, req)
	if err != nil {
		log.Errorf("validate SendSms ShouldBindJSON failed, traceID:%s, err:%v", traceID, err)
		httputil.SetErrorResponse(&resp, errors.CodeInternalServerError, errors.GetErrorMessage(errors.CodeInternalServerError))
		return
	}

	httputil.SetSuccessfulResponse(&resp, errors.CodeOK, nil)
}
