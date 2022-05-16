package repository

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"
	"time"

	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/config"

	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/consts"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/domain/entity"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/infrastructure/persistence"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/utils"

	log "github.com/sirupsen/logrus"
)

type WXRepository struct {
	wx   *persistence.WxRepo
	user *persistence.UserRepo
	msg  *persistence.MessageRepo
}

var defaultWXRepository = &WXRepository{}

func NewWXRepository(wx *persistence.WxRepo, user *persistence.UserRepo, msg *persistence.MessageRepo) {
	if defaultWXRepository.wx == nil {
		defaultWXRepository.wx = wx
	}
	if defaultWXRepository.user == nil {
		defaultWXRepository.user = user
	}
	if defaultWXRepository.msg == nil {
		defaultWXRepository.msg = msg
	}
}

func DefaultWXRepository() *WXRepository {
	return defaultWXRepository
}

func (a *WXRepository) GetWXCheckSign(signature, timestamp, nonce, token string) bool {
	// 本地计算signature
	si := []string{token, timestamp, nonce}
	// 字典序排序
	sort.Strings(si)
	n := len(timestamp) + len(nonce) + len(token)
	var b strings.Builder
	b.Grow(n)
	for _, v := range si {
		b.WriteString(v)
	}
	return utils.Sha1(b.String()) == signature
}

func (a *WXRepository) GetEventXML(ctx context.Context, reqBody *entity.TextRequestBody) (respBody []byte, err error) {
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("GetEventXml traceID:%s", traceID)
	if reqBody == nil {
		return nil, fmt.Errorf("xml request body is empty")
	}
	responseTextBody, err := a.handlerEvent(ctx, reqBody)
	if err != nil {
		log.Errorf("GetEventXml handlerEvent failed traceID:%s,err:%v", traceID, err)
		return nil, err
	}
	return responseTextBody, nil
}

func (a *WXRepository) handlerEvent(ctx context.Context, reqBody *entity.TextRequestBody) ([]byte, error) {
	var respContent string
	var err error
	// 事件类型
	switch reqBody.Event {
	// 关注订阅
	case consts.SubscribeEvent:
		if respContent, err = a.handlerSubscribeEvent(ctx, reqBody); err != nil {
			return nil, err
		}
	case consts.UnsubscribeEvent:
		if respContent, err = a.handlerUnSubscribeEvent(ctx, reqBody); err != nil {
			return nil, err
		}
	case consts.TEMPLATESENDJOBFINISHEvent:
		// 事件回调内部系统错误重发
		if respContent, err = a.handlerTEMPLATESENDJOBFINISHEvent(ctx, reqBody); err != nil {
			return nil, err
		}
	}
	return a.makeTextResponseBody(reqBody.ToUserName, reqBody.FromUserName, respContent)
}

func (a *WXRepository) handlerSubscribeEvent(ctx context.Context, reqBody *entity.TextRequestBody) (string, error) {
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("handlerSubscribeEvent traceID:%s", traceID)
	// 判断是否存在该消息id,用 FromUserName+CreateTime 去重
	msgID := fmt.Sprintf("%s%d", reqBody.FromUserName, reqBody.CreateTime)
	exist, err := a.isExistUserMsgID(ctx, msgID, reqBody.FromUserName, reqBody.CreateTime)
	if err != nil {
		log.Errorf("handlerSubscribeEvent WXRepository wx repo isExistUserMsgID traceID:%s,err:%v", traceID, err)
		return "", err
	}
	if exist {
		return "", nil
	}
	// 持久化保存
	u := entity.User{
		OpenID:     reqBody.FromUserName,
		CreateTime: reqBody.CreateTime,
		DeleteTime: 0,
	}
	err = a.user.SaveUser(ctx, u)
	if err != nil {
		log.Errorf("handlerSubscribeEvent WXRepository wx repo SaveUser traceID:%s,err:%v", traceID, err)
		return "", err
	}
	err = a.wx.SetMsgIDToRedis(ctx, msgID)
	if err != nil {
		log.Errorf("handlerSubscribeEvent WXRepository wx repo set msg id to redis failed,traceID:%s,err:%v", traceID, err)
	}
	return fmt.Sprintf("%s%s", consts.SubscribeRespContent, config.VerifyProfileURL), nil
}

func (a *WXRepository) handlerUnSubscribeEvent(ctx context.Context, reqBody *entity.TextRequestBody) (string, error) {
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("handlerUnSubscribeEvent traceID:%s", traceID)
	// 判断是否存在该消息id,用 FromUserName+CreateTime 去重
	msgID := fmt.Sprintf("%s%d", reqBody.FromUserName, reqBody.CreateTime)
	exist, err := a.isExistUserMsgID(ctx, msgID, reqBody.FromUserName, reqBody.CreateTime)
	if err != nil {
		log.Errorf("handlerUnSubscribeEvent WXRepository wx repo isExistUserMsgID traceID:%s,err:%v", traceID, err)
		return "", err
	}
	if exist {
		return "", nil
	}
	// 删除用户信息
	u := entity.User{
		OpenID: reqBody.FromUserName,
	}
	err = a.user.DelUser(ctx, u)
	if err != nil {
		log.Errorf("handlerUnSubscribeEvent WXRepository wx repo SaveUser traceID:%s,err:%v", traceID, err)
		return "", err
	}
	err = a.wx.SetMsgIDToRedis(ctx, msgID)
	if err != nil {
		log.Errorf("handlerUnSubscribeEvent WXRepository wx repo set msg id to redis failed,traceID:%s,err:%v", traceID, err)
	}
	return consts.UnSubscribeRespContent, nil
}

func (a *WXRepository) handlerTEMPLATESENDJOBFINISHEvent(ctx context.Context, reqBody *entity.TextRequestBody) (string, error) {
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("handlerTEMPLATESENDJOBFINISHEvent traceID:%s", traceID)
	// 判断是否存在该消息id,用 FromUserName+CreateTime 去重
	msgID := fmt.Sprintf("%s%d", reqBody.FromUserName, reqBody.CreateTime)
	exist, err := a.isExistTemplateSendJobMsgID(ctx, msgID, reqBody.FromUserName, reqBody.CreateTime)
	if err != nil {
		log.Errorf("handlerTEMPLATESENDJOBFINISHEvent WXRepository wx repo isExistTemplateSendJobMsgID traceID:%s,err:%v", traceID, err)
		return "", err
	}
	if exist {
		return "", nil
	}
	// 对事件推送由于其他原因发送失败的消息进行重发
	// 判断当前发送次数是否小于最大重发次数，若小于则重发
	msg, err := a.msg.GetMaxCountFailureMsgByMsgID(ctx, reqBody.MsgID)
	if err != nil {
		log.Errorf("handlerTEMPLATESENDJOBFINISHEvent GetMaxCountFailureMsgByMsgID failed,traceID:%s,err:%v", traceID, err)
		return consts.TEMPLATESENDJOBFINISHRespContent, err
	}
	// 发送成功，改状态
	if reqBody.Status == consts.TemplateSendSuccessStatus {
		updateItem := entity.FailureMsgLog{
			ID:         msg.ID,
			Status:     consts.SendSuccess,
			Cause:      consts.TemplateSendSuccessStatus,
			UpdateTime: reqBody.CreateTime,
		}
		err = a.msg.UpdateFailureMsg(ctx, updateItem)
		if err != nil {
			log.Errorf("handlerTEMPLATESENDJOBFINISHEvent UpdateFailureMsg TemplateSendSuccessStatus failed,traceID:%s,err:%v", traceID, err)
			return consts.TEMPLATESENDJOBFINISHRespContent, err
		}
	}
	// 发送失败，用户拒接
	if reqBody.Status == consts.TemplateSendUserBlockStatus {
		updateItem := entity.FailureMsgLog{
			ID:         msg.ID,
			Status:     consts.SendFailure,
			Cause:      consts.TemplateSendUserBlockStatus,
			UpdateTime: reqBody.CreateTime,
		}
		err = a.msg.UpdateFailureMsg(ctx, updateItem)
		if err != nil {
			log.Errorf("handlerTEMPLATESENDJOBFINISHEvent UpdateFailureMsg TemplateSendUserBlockStatus failed,traceID:%s,err:%v", traceID, err)
			return consts.TEMPLATESENDJOBFINISHRespContent, err
		}
	}
	// 发送失败，内部错误，重发
	if reqBody.Status == consts.TemplateSendFailedStatus {
		if msg.Count < consts.MaxRetryCount {
			var bs []byte
			bs, err = json.Marshal(msg.TransferSendTmplMsgRemoteReq())
			if err != nil {
				log.Errorf("handlerTEMPLATESENDJOBFINISHEvent json marshal tmpl msg failed,traceID:%s,err:%v", traceID, err)
				return consts.TEMPLATESENDJOBFINISHRespContent, err
			}
			err = a.msg.SendTmplMsgToMQ(ctx, a.msg.GetTopic(), string(bs))
			if err != nil {
				log.Errorf("handlerTEMPLATESENDJOBFINISHEvent send tmpl msg to MQ failed,traceID:%s,err:%v", traceID, err)
				return consts.TEMPLATESENDJOBFINISHRespContent, err
			}
			// 更新上一条消息失败原因
			updateItem := entity.FailureMsgLog{
				ID:         msg.ID,
				Cause:      consts.TemplateSendFailedStatus,
				UpdateTime: reqBody.CreateTime,
			}
			err = a.msg.UpdateFailureMsg(ctx, updateItem)
			if err != nil {
				log.Errorf("handlerTEMPLATESENDJOBFINISHEvent UpdateFailureMsg TemplateSendFailedStatus failed,traceID:%s,err:%v", traceID, err)
				return consts.TEMPLATESENDJOBFINISHRespContent, err
			}
			// 增加重发记录条目
			item := entity.FailureMsgLog{
				MsgID:      msg.MsgID,
				ToUser:     msg.ToUser,
				TemplateID: msg.TemplateID,
				Content:    msg.Content,
				Cause:      msg.Cause,
				Status:     consts.SendRetry,
				Count:      msg.Count + 1,
			}
			err = a.msg.SaveFailureMsgLog(ctx, item)
			if err != nil {
				log.Errorf("handlerTEMPLATESENDJOBFINISHEvent SaveFailureMsgLog TemplateSendFailedStatus failed,traceID:%s,err:%v", traceID, err)
				return consts.TEMPLATESENDJOBFINISHRespContent, err
			}
		} else {
			// 改变发送状态为失败
			updateItem := entity.FailureMsgLog{
				ID:         msg.ID,
				Status:     consts.SendFailure,
				UpdateTime: reqBody.CreateTime,
			}
			err = a.msg.UpdateFailureMsg(ctx, updateItem)
			if err != nil {
				log.Errorf("handlerTEMPLATESENDJOBFINISHEvent UpdateFailureMsg send status TemplateSendFailedStatus failed,traceID:%s,err:%v", traceID, err)
				return consts.TEMPLATESENDJOBFINISHRespContent, err
			}
		}
	}
	return consts.TEMPLATESENDJOBFINISHRespContent, nil
}

func (a *WXRepository) isExistUserMsgID(ctx context.Context, msgID string, fromUserName string, createTime int64) (bool, error) {
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("IsExistUserMsgID traceID:%s", traceID)
	exist, err := a.wx.IsExistMsgIDFromRedis(ctx, msgID)
	if err != nil {
		log.Errorf("isExistUserMsgID IsExistMsgIDFromRedis failed,traceID:%s,err:%v", traceID, err)
		return false, err
	}
	// 若存在返回空串,不存在则持久化存储,并保存msg id 到 redis
	if exist {
		return true, nil
	}
	// 从db上找，存在则返回空串
	exist, err = a.user.IsExistUserMsgFromDB(ctx, fromUserName, createTime)
	if err != nil {
		log.Errorf("isExistUserMsgID IsExistUserFromDB failed,traceID:%s,err:%v", traceID, err)
		return false, err
	}
	if exist {
		// 回写到redis中
		err = a.wx.SetMsgIDToRedis(ctx, msgID)
		if err != nil {
			log.Errorf("isExistUserMsgID WXRepository wx repo set msg id to redis failed,traceID:%s,err:%v", traceID, err)
		}
		return true, nil
	}
	return false, nil
}

func (a *WXRepository) isExistTemplateSendJobMsgID(ctx context.Context, msgID string, fromUserName string, createTime int64) (bool, error) {
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("isExistTemplateSendJobMsgID traceID:%s", traceID)
	exist, err := a.wx.IsExistMsgIDFromRedis(ctx, msgID)
	if err != nil {
		log.Errorf("isExistTemplateSendJobMsgID IsExistMsgIDFromRedis exist msg id from redis,traceID:%s,err:%v", traceID, err)
		return false, err
	}
	if exist {
		return true, nil
	}
	// db查当前消息是否存在
	exist, err = a.msg.IsExistFailureMsgFromDB(ctx, fromUserName, createTime)
	if err != nil {
		log.Errorf("isExistTemplateSendJobMsgID IsExistFailureMsgFromDB failed,traceID:%s,err:%v", traceID, err)
		return false, err
	}
	if exist {
		// 回写到redis
		err = a.wx.SetMsgIDToRedis(ctx, msgID)
		if err != nil {
			log.Errorf("isExistUserMsgID WXRepository wx repo set msg id to redis failed,traceID:%s,err:%v", traceID, err)
		}
		return true, nil
	}
	return false, nil
}

func (a *WXRepository) makeTextResponseBody(fromUserName, toUserName, content string) ([]byte, error) {
	textResponseBody := &entity.TextResponseBody{}
	textResponseBody.FromUserName = a.value2CDATA(fromUserName)
	textResponseBody.ToUserName = a.value2CDATA(toUserName)
	textResponseBody.MsgType = a.value2CDATA("text")
	textResponseBody.Content = a.value2CDATA(content)
	textResponseBody.CreateTime = time.Now().Unix()
	return xml.MarshalIndent(textResponseBody, " ", "  ")
}

func (a *WXRepository) value2CDATA(v string) entity.CDATAText {
	return entity.CDATAText{Text: "<![CDATA[" + v + "]]>"}
}
