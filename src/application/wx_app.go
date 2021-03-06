package application

import (
	"context"

	"github.com/hololee2cn/wxpub/v1/src/domain/entity"
	"github.com/hololee2cn/wxpub/v1/src/domain/repository"
)

type wxApp struct {
	wx repository.WXRepository
}

// wxApp implements the WXInterface
var _ WXInterface = &wxApp{}

type WXInterface interface {
	GetWXCheckSign(signature, timestamp, nonce, token string) bool
	HandleXML(ctx context.Context, reqBody *entity.TextRequestBody) (respBody []byte, err error)
}

func (w *wxApp) GetWXCheckSign(signature, timestamp, nonce, token string) bool {
	return w.wx.GetWXCheckSign(signature, timestamp, nonce, token)
}

func (w *wxApp) HandleXML(ctx context.Context, reqBody *entity.TextRequestBody) (respBody []byte, err error) {
	return w.wx.HandleXML(ctx, reqBody)
}
