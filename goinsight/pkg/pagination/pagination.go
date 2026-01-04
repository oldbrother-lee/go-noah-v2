package pagination

import (
	"gorm.io/gorm"
)

type Pagination struct {
	PageSize int  `form:"page_size" json:"page_size"`
	Page     int  `form:"page" json:"page"`
	IsPage   bool `form:"is_page" json:"is_page"`
	Current  int  `form:"current" json:"current"`
	Size     int  `form:"size" json:"size"`
}

func Pager(p *Pagination, queryTx *gorm.DB, list interface{}) (total int64) {
	// 兼容 current/size
	if p.Page == 0 && p.Current > 0 {
		p.Page = p.Current
	}
	if p.PageSize == 0 && p.Size > 0 {
		p.PageSize = p.Size
	}

	// 如果没有显式设置 IsPage，但传递了分页参数，则默认为分页
	if !p.IsPage && (p.Page > 0 || p.PageSize > 0) {
		p.IsPage = true
	}

	if !p.IsPage {
		queryTx.Find(list)
		return
	}
	if p.PageSize < 1 {
		p.PageSize = 10
	}
	if p.Page < 1 {
		p.Page = 1
	}
	offset := p.PageSize * (p.Page - 1)
	// count
	queryTx.Count(&total)
	// 获取分页数据
	queryTx.Limit(p.PageSize).Offset(offset).Find(list)
	return
}
