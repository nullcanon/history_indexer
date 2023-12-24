package db

import (
	"errors"

	"github.com/jinzhu/gorm"
)

// 资产信息表
// avas trxs: 21002, total: 21000000, minted: 21000000, holders: 443

type InscriptionInfoHistory struct {
	Trxs        int32  `gorm:"column:trxs"`
	Total       string `gorm:"column:total; default:'0'"`
	Minted      string `gorm:"column:minted"`
	Holders     int32  `gorm:"column:holders"`
	Limit       string `gorm:"column:mint_limit"`
	Ticks       string `gorm:"column:ticks;primary_key"`
	CreatedAt   uint64 `gorm:"column:created_at"`
	CompletedAt uint64 `gorm:"column:completed_at"`
	Number      uint64 `gorm:"column:number"`
}

func (u InscriptionInfoHistory) CreateInscriptionInfo(inscriptionInfo InscriptionInfoHistory) error {
	return db.Create(&inscriptionInfo).Error
}

func (u InscriptionInfoHistory) Update(args map[string]interface{}) error {
	var inscriptionInfo InscriptionInfoHistory
	result := db.First(&inscriptionInfo, "ticks = ?", u.Ticks)

	if result.Error == nil {
		result = db.Model(&InscriptionInfoHistory{}).Where("ticks = ?", u.Ticks).Update(args)
	}

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return u.CreateInscriptionInfo(u)
	} else {
		return result.Error
	}
}

func (u InscriptionInfoHistory) FetchInscriptionInfo(inscriptionInfo *[]InscriptionInfoHistory) {
	db.Find(&inscriptionInfo)
}
