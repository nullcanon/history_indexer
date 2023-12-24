// 转账记录表
// ticks status from to amount time hash
package db

type TradeHistoryTmp struct {
	Id     int64  `gorm:"type:int(64) UNSIGNED AUTO_INCREMENT;primary_key" json:"id"`
	Ticks  string `gorm:"column:ticks"`
	Status string `gorm:"column:status"`
	From   string `gorm:"column:from_address"`
	To     string `gorm:"column:to_address"`
	Hash   string `gorm:"column:hash;unique_index"`
	Amount string `gorm:"column:amount"`
	Time   uint64 `gorm:"column:time"`
	Number uint64 `gorm:"column:number;index"`
}

func (u TradeHistoryTmp) GetInscriptionNumber() uint64 {
	var history TradeHistoryTmp
	err := db.Order("number desc").First(&history).Error
	if err != nil {
		return 0
	}
	return history.Number
}

func (u TradeHistoryTmp) CreateTradeHistory(tradeHistory *TradeHistoryTmp) error {
	return db.Create(tradeHistory).Error
}

// func (u TradeHistoryTmp) Update(args map[string]interface{}) error {
// 	var tradeHistory TradeHistoryTmp
// 	result := db.First(&tradeHistory, "hash = ?", u.Hash)

// 	if result.Error == nil {
// 		db.Model(&TradeHistoryTmp{}).Where("hash = ?", u.Hash).Update(args)
// 	}

// 	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
// 		return u.CreateTradeHistory(u)
// 	} else {
// 		return result.Error
// 	}
// 	return nil
// }
