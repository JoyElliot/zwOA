package api

import (
	"github.com/JoyElliot/zwOA/internal/modules/organization"
	"github.com/gogf/gf/v2/net/ghttp"
)

type avatarImage struct {
	Image string `json:"image" v:"required" format:"byte" dc:"Base64 编码的 JPEG 图片"`
}

func (api authentication) saveAvatar(r *ghttp.Request, account organization.Account) {
	if account.MustChangePassword {
		writeError(r, organization.ErrForbidden)
		return
	}
	var input avatarImage
	err := readJSON(r, &input)
	if err == nil {
		err = api.store.SaveAvatar(r.Context(), account.ID, input.Image)
	}
	writeResult(r, nil, err)
}

func (api authentication) avatar(r *ghttp.Request, account organization.Account) {
	if account.MustChangePassword {
		writeError(r, organization.ErrForbidden)
		return
	}
	image, err := api.store.Avatar(r.Context(), r.GetRouter("version").String())
	writeResult(r, avatarImage{Image: image}, err)
}
