package workbench

import (
	"context"

	"github.com/gogf/gf/v2/util/gmeta"
)

type Controller struct{}

type OverviewReq struct {
	gmeta.Meta `path:"/workbench/overview" method:"get" tags:"工作台" summary:"查询工作台服务信息"`
}

type OverviewRes struct {
	Name    string `json:"name" dc:"系统名称"`
	Version string `json:"version" dc:"应用版本"`
}

func (Controller) Overview(context.Context, *OverviewReq) (*OverviewRes, error) {
	return &OverviewRes{Name: "中网联合 OA / CRM", Version: "0.1.0"}, nil
}
