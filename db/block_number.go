package db

import (
	"errors"

	"github.com/jinzhu/gorm"
)

// 扫链配置表
// blockNumber

type HistoryBlockScan struct {
	Id          int64 `gorm:"type:int(11) UNSIGNED AUTO_INCREMENT;primary_key" json:"id"`
	BlockNumber int64 `gorm:"type:int(64) UNSIGNED not null COMMENT '同步的区块高度'" json:"block_number"`
	Number      int64 `gorm:"type:int(64) UNSIGNED not null COMMENT '同步的区块高度对应的编号'" json:"number"`
	Gas         int   `gorm:"column:gas"`
}

func (b HistoryBlockScan) Create(blockScan HistoryBlockScan) error {
	return db.Create(&blockScan).Error
}
func (b *HistoryBlockScan) GetNumber() (int64, int64, int) {
	var bscScan HistoryBlockScan
	err := db.Order("id desc").First(&bscScan).Error
	if err != nil {
		return 0, 0, 0
	}
	return bscScan.BlockNumber, bscScan.Number, bscScan.Gas
}

func (b *HistoryBlockScan) Edit(data map[string]interface{}) error {
	return db.Model(&b).Updates(data).Error
}

func (b *HistoryBlockScan) UptadeBlockNumber(blockNumber uint64, number uint64, gas int) error {
	var blockscan HistoryBlockScan
	result := db.First(&blockscan, "id = ?", 1)

	if result.Error == nil {
		db.Model(&HistoryBlockScan{}).Where("id = ?", 1).Update(map[string]interface{}{"block_number": blockNumber, "number": number, "gas": gas})
	}

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return db.Create(b).Error
	} else {
		return result.Error
	}
	return nil
}
