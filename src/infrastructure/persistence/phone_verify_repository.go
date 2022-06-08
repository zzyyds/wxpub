package persistence

import (
	"context"
	"encoding/json"
	"git.nova.net.cn/nova/misc/wx-public/proxy/src/pkg/uuid"
	"time"

	redis2 "git.nova.net.cn/nova/misc/wx-public/proxy/src/pkg/redis"

	"git.nova.net.cn/nova/misc/wx-public/proxy/src/consts"
	"git.nova.net.cn/nova/misc/wx-public/proxy/src/domain/entity"
	"git.nova.net.cn/nova/misc/wx-public/proxy/src/utils"

	smsPb "git.nova.net.cn/nova/notify/sms-xuanwu/pkg/grpcIFace"
	captchaPb "git.nova.net.cn/nova/shared/captcha/pkg/grpcIFace"
	log "github.com/sirupsen/logrus"
)

type PhoneVerifyRepo struct {
	smsGRPCClient    smsPb.SenderClient
	captchaRPCClient captchaPb.CaptchaServiceClient
}

var defaultPhoneVerifyRepo *PhoneVerifyRepo

func NewPhoneVerifyRepo() {
	if defaultPhoneVerifyRepo == nil {
		defaultPhoneVerifyRepo = &PhoneVerifyRepo{
			smsGRPCClient:    CommonRepositories.SmsGRPCClient,
			captchaRPCClient: CommonRepositories.CaptchaGRPCClient,
		}
	}
}

func DefaultPhoneVerifyRepo() *PhoneVerifyRepo {
	return defaultPhoneVerifyRepo
}

func (r *PhoneVerifyRepo) GenCaptcha(ctx context.Context, width int32, height int32) (captchaID, captchaBase64Value string, err error) {
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("GenCaptcha traceID:%s", traceID)

	c := utils.ToOutGoingContext(ctx)
	rpcResp, err := r.captchaRPCClient.Get(c, &captchaPb.GetCaptchaRequest{
		Width:           width,
		Height:          height,
		NoiseCount:      10,
		ShowLineOptions: 2,
	})
	if err != nil {
		log.Errorf("GenCaptcha get captcha error: %+v, traceID: %s", err, traceID)
		return
	}

	captchaID = rpcResp.GetID()
	captchaBase64Value = rpcResp.GetBase64Value()
	return
}

func (r *PhoneVerifyRepo) VerifyCaptcha(ctx context.Context, captchaID string, captchaAnswer string) (ok bool, err error) {
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("VerifyCaptcha traceID:%s", traceID)

	c := utils.ToOutGoingContext(ctx)
	rpcResp, err := r.captchaRPCClient.Verify(c, &captchaPb.VerifyCaptchaRequest{
		ID:     captchaID,
		Answer: captchaAnswer,
	})
	if err != nil {
		log.Errorf("VerifyCaptcha Verify error: %+v, traceID: %s", err, traceID)
		return
	}

	ok = rpcResp.GetData()
	return
}

func (r *PhoneVerifyRepo) SendSms(ctx context.Context, content string, sender string, phone string) (err error) {
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("SendSms traceID:%s", traceID)

	c := utils.ToOutGoingContext(ctx)
	// 发短信是调用的第三方的服务，计费使用
	_, err = r.smsGRPCClient.SendMessage(c, &smsPb.SendMsgRequest{
		Content: content,
		Sender:  sender,
		Items: []*smsPb.SendMsgRequest_Item{
			{
				To:        phone,
				MessageID: uuid.Get(), // 不需要查询，可以忽略
			},
		},
	})
	if err != nil {
		log.Errorf("send sms message error: %+v, traceID: %s", err, traceID)
	}

	return
}

func (r *PhoneVerifyRepo) SetVerifyCodeSmsStorage(ctx context.Context, challenge string, verifyCodeAnswer string) (err error) {
	var verifyCodeSmsRedisValue entity.VerifyCodeRedisValue

	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("SetVerifyCodeSmsStorage traceID:%s", traceID)

	smsCreateTime := time.Now().UnixNano()
	verifyCodeSmsRedisValue.VerifyCodeCreateTime = smsCreateTime
	verifyCodeSmsRedisValue.VerifyCodeAnswer = verifyCodeAnswer

	smsRedisValue, _ := json.Marshal(verifyCodeSmsRedisValue)
	// redis存放open_id+phone:{verifyCodeAnswer,smsCreateTime}，过期时间为30分钟
	err = redis2.RSet(consts.RedisKeyPrefixChallenge+challenge, smsRedisValue, consts.VerifyCodeSmsChallengeTTL)
	if err != nil {
		log.Errorf("failed to do redis Set, error: %+v, traceID: %s", err, traceID)
		return
	}

	return
}

func (r *PhoneVerifyRepo) VerifySmsCode(ctx context.Context, challenge, verifyCodeAnswer string, ttl int64) (ok, isExpire bool, err error) {
	var value []byte
	var verifyCodeValue entity.VerifyCodeRedisValue
	now := time.Now().UnixNano()
	traceID := utils.ShouldGetTraceID(ctx)
	log.Debugf("VerifySmsCode traceID:%s", traceID)

	value, err = redis2.RGet(consts.RedisKeyPrefixChallenge + challenge)
	if err != nil {
		log.Errorf("failed to do redis HGet, error: %+v, traceID: %s", err, traceID)
		return
	}

	err = json.Unmarshal(value, &verifyCodeValue)
	if err != nil {
		log.Errorf("VerifySmsCode json unmarshal failed, error: %+v, traceID: %s", err, traceID)
		return
	}

	// 是否过期
	if (now-verifyCodeValue.VerifyCodeCreateTime)/1e9 > ttl {
		isExpire = true
	}

	// 检查验证码
	if verifyCodeValue.VerifyCodeAnswer == verifyCodeAnswer {
		ok = true
	}

	return
}
