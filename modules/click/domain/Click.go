package domain

import (
	common "UnpakSiamida/common/domain"
	"time"
)

type Click struct {
	common.Entity

	ID        uint      `gorm:"primaryKey;autoIncrement"`
	LinkId    uint      `gorm:"column:link_id;"`
	IP        string    `gorm:"type:longtext;column:ip;"`
	IpClient  string    `gorm:"type:longtext;column:ip_client;"`
	ISO       string    `gorm:"type:longtext;column:iso_code;"`
	Country   string    `gorm:"type:longtext;column:country;"`
	Referer   string    `gorm:"type:longtext;column:referer_host;"`
	UserAgent string    `gorm:"type:longtext;column:user_agent;"`
	Created   time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Click) TableName() string {
	return "clicks"
}

func NewClick(
	linkId uint,
	ip string,
	ipClient string,
	iso string,
	country string,
	referer string,
	userAgent string,
) common.ResultValue[*Click] {
	if len(iso) > 2 || iso == "null" {
		iso = ""
	}
	data := Click{
		LinkId:    linkId,
		IP:        ip,
		IpClient:  ipClient,
		ISO:       iso,
		Country:   country,
		Referer:   referer,
		UserAgent: userAgent,
		Created:   time.Now().UTC(),
	}

	return common.SuccessValue(&data)
}
