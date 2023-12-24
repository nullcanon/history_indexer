package handlers

import (
	"encoding/hex"
	"encoding/json"
	"open-indexer/db"
	"open-indexer/model"
	"open-indexer/utils"
	"strings"
	"sync"
	"time"
)

var Inscriptions []*model.Inscription

var Tokens = make(map[string]*model.Token)

// address -> ticker -> balnce
var UserBalances = make(map[string]map[string]*model.DDecimal)

// ticker -> address -> number
var TokenHolders = make(map[string]map[string]*model.DDecimal)

// 数据发送变化的用户,刷新数据库
var UpdateUsers = make(map[string]bool)

var BlockNumber uint64 = 0

var InscriptionNumber uint64 = 0

var Gas int = 0

var DBLock sync.RWMutex

func ProcessUpdateARC20(trxs []*model.Transaction) error {
	for _, inscription := range trxs {
		err := Inscribe(inscription)
		if err != nil {
			// return err
		}
	}
	return nil
}

func appendTradeCache(inscription *model.Inscription, tick string, amount string) {
	var tardeinfo db.TradeHistoryTmp
	tardeinfo.Ticks = tick
	tardeinfo.Status = "1"
	tardeinfo.From = inscription.From
	tardeinfo.To = inscription.To
	tardeinfo.Hash = inscription.Id
	tardeinfo.Amount = amount
	tardeinfo.Time = inscription.Timestamp
	tardeinfo.Number = inscription.Number
	tardeinfo.CreateTradeHistory(&tardeinfo)
}

func Inscribe(trx *model.Transaction) error {
	// data:,
	if !strings.HasPrefix(trx.Input, "0x646174613a") {
		return nil
	}

	bytes, err := hex.DecodeString(trx.Input[2:])
	if err != nil {
		logger.Warn("inscribe err", err, " at block ", trx.Block, ":", trx.Idx)
		return nil
	}
	input := string(bytes)

	sepIdx := strings.Index(input, ",")
	if sepIdx == -1 || sepIdx == len(input)-1 {
		return nil
	}
	contentType := "text/plain"
	if sepIdx > 5 {
		contentType = input[5:sepIdx]
	}
	content := input[sepIdx+1:]

	InscriptionNumber++
	var inscription model.Inscription
	inscription.Number = InscriptionNumber
	inscription.Id = trx.Id
	inscription.From = trx.From
	inscription.To = trx.To
	inscription.Block = trx.Block
	inscription.Idx = trx.Idx
	inscription.Timestamp = trx.Timestamp
	inscription.ContentType = contentType
	inscription.Content = content
	BlockNumber = trx.Block

	if err := handleProtocols(&inscription); err != nil {
		logger.Info("error at ", inscription.Number)

		return err
	}

	Inscriptions = append(Inscriptions, &inscription)

	return nil
}

func handleProtocols(inscription *model.Inscription) error {
	content := strings.TrimSpace(inscription.Content)
	if content[0] == '{' {
		var protoData map[string]string
		err := json.Unmarshal([]byte(content), &protoData)
		if err != nil {
			logger.Info("json parse error: ", err, ", at ", inscription.Number)
		} else {
			value, ok := protoData["p"]
			if ok && strings.TrimSpace(value) != "" {
				protocol := strings.ToLower(value)
				// if protocol == "asc-20" {
				if protocol == "aias-20" {
					var asc20 model.Asc20
					asc20.Number = inscription.Number
					if value, ok = protoData["tick"]; ok {
						asc20.Tick = value
					}
					if value, ok = protoData["op"]; ok {
						asc20.Operation = value
					}

					var err error
					if strings.TrimSpace(asc20.Tick) == "" {
						asc20.Valid = -1 // empty tick
					} else if len(asc20.Tick) > 18 {
						asc20.Valid = -2 // too long tick
					} else if asc20.Operation == "deploy" {
						asc20.Valid, err = deployToken(&asc20, inscription, protoData)
					} else if asc20.Operation == "mint" {
						asc20.Valid, err = mintToken(&asc20, inscription, protoData)
					} else if asc20.Operation == "transfer" {
						asc20.Valid, err = transferToken(&asc20, inscription, protoData)
					} else {
						asc20.Valid = -3 // wrong operation
					}
					if err != nil {
						return err
					}
					return nil
				}
			}
		}
	}
	return nil
}

func deployToken(asc20 *model.Asc20, inscription *model.Inscription, params map[string]string) (int8, error) {
	logger.Info("deployToken ", inscription.Id, " tick: ", asc20.Tick)

	// 暂时只索引aias
	if asc20.Tick != "aias" {
		logger.Info("token ", asc20.Tick, " not supply")
		return -10, nil
	}

	value, ok := params["max"]
	if !ok {
		return -11, nil
	}
	max, precision, err1 := model.NewDecimalFromString(value)
	if err1 != nil {
		return -12, nil
	}
	value, ok = params["lim"]
	if !ok {
		return -13, nil
	}
	limit, _, err2 := model.NewDecimalFromString(value)
	if err2 != nil {
		return -14, nil
	}
	if max.Sign() <= 0 || limit.Sign() <= 0 {
		return -15, nil
	}
	if max.Cmp(limit) < 0 {
		return -16, nil
	}

	asc20.Max = max
	asc20.Precision = precision
	asc20.Limit = limit

	// 已经 deploy
	asc20.Tick = strings.TrimSpace(asc20.Tick) // trim tick
	_, exists := Tokens[strings.ToLower(asc20.Tick)]
	if exists {
		logger.Info("token ", asc20.Tick, " has deployed at ", inscription.Number)
		return -17, nil
	}

	token := &model.Token{
		Tick:        asc20.Tick,
		Number:      asc20.Number,
		Precision:   precision,
		Max:         max,
		Limit:       limit,
		Minted:      model.NewDecimal(),
		Progress:    0,
		CreatedAt:   inscription.Timestamp,
		CompletedAt: int64(0),
	}

	// save
	Tokens[strings.ToLower(token.Tick)] = token
	TokenHolders[strings.ToLower(token.Tick)] = make(map[string]*model.DDecimal)

	return 1, nil
}

func mintToken(asc20 *model.Asc20, inscription *model.Inscription, params map[string]string) (int8, error) {
	// logger.Info("mintToken ", inscription.Id, " tick: ", asc20.Tick)

	value, ok := params["amt"]
	if !ok {
		return -21, nil
	}
	amt, precision, err := model.NewDecimalFromString(value)
	if err != nil {
		return -22, nil
	}

	asc20.Amount = amt

	// check token
	tick := strings.ToLower(asc20.Tick)
	token, exists := Tokens[tick]
	if !exists {
		return -23, nil
	}
	// logger.Info("mintToken eists tick: ", asc20.Tick)

	// check precision
	if precision > token.Precision {
		return -24, nil
	}

	if amt.Sign() <= 0 {
		return -25, nil
	}

	if amt.Cmp(token.Limit) == 1 {
		return -26, nil
	}

	var left = token.Max.Sub(token.Minted)

	if left.Cmp(amt) == -1 {
		if left.Sign() > 0 {
			amt = left
		} else {
			// exceed max
			return -27, nil
		}
	}
	// update amount
	asc20.Amount = amt

	var newHolder = false

	tokenHolders, _ := TokenHolders[tick]
	userBalances, ok := UserBalances[inscription.To]
	if !ok {
		userBalances = make(map[string]*model.DDecimal)
		UserBalances[inscription.To] = userBalances
	}

	balance, ok := userBalances[strings.ToLower(asc20.Tick)]
	if !ok {
		balance = model.NewDecimal()
		newHolder = true
	}
	if balance.Sign() == 0 {
		newHolder = true
	}

	balance = balance.Add(amt)

	// update
	userBalances[tick] = balance
	tokenHolders[inscription.To] = balance
	UpdateUsers[inscription.To] = true

	if err != nil {
		return 0, err
	}

	// update token
	token.Minted = token.Minted.Add(amt)
	token.Trxs++

	if token.Minted.Cmp(token.Max) >= 0 {
		token.Progress = 1000000
	} else {
		token.Progress = int32(utils.ParseInt64(token.Minted.String()) * 1000000 / utils.ParseInt64(token.Max.String()))
	}

	if token.Minted.Cmp(token.Max) == 0 {
		token.CompletedAt = time.Now().Unix()
	}

	if newHolder {
		token.Holders++
	}

	appendTradeCache(inscription, asc20.Tick, asc20.Amount.String())
	// logger.Info("mintToken ", inscription.Id, " tick: ", asc20.Tick)

	return 1, err
}

func transferToken(asc20 *model.Asc20, inscription *model.Inscription, params map[string]string) (int8, error) {
	// logger.Info("transferToken ", inscription.Id, " tick: ", asc20.Tick)
	value, ok := params["amt"]
	if !ok {
		return -31, nil
	}
	amt, precision, err := model.NewDecimalFromString(value)
	if err != nil {
		return -32, nil
	}

	// check token
	tick := strings.ToLower(asc20.Tick)
	token, exists := Tokens[tick]
	if !exists {
		return -33, nil
	}

	// check precision
	if precision > token.Precision {
		return -34, nil
	}

	if amt.Sign() <= 0 {
		return -35, nil
	}

	if inscription.From == inscription.To {
		// send to self
		return -36, nil
	}

	asc20.Amount = amt

	tokenHolders, _ := TokenHolders[tick]
	fromBalances, ok := UserBalances[inscription.From]
	if !ok {
		return -37, nil
	}
	toBalances, ok := UserBalances[inscription.To]
	if !ok {
		toBalances = make(map[string]*model.DDecimal)
		UserBalances[inscription.To] = toBalances
	}

	var newHolder = false

	fromBalance, ok := fromBalances[tick]
	if !ok {
		return -37, nil
	}

	if amt.Cmp(fromBalance) == 1 {
		return -37, nil
	}

	fromBalance = fromBalance.Sub(amt)

	if fromBalance.Sign() == 0 {
		token.Holders--
	}

	// To
	toBalance, ok := toBalances[tick]
	if !ok {
		toBalance = model.NewDecimal()
		newHolder = true
	}
	if toBalance.Sign() == 0 {
		newHolder = true
	}
	toBalance = toBalance.Add(amt)

	// update
	fromBalances[tick] = fromBalance
	toBalances[tick] = toBalance
	tokenHolders[inscription.From] = fromBalance
	tokenHolders[inscription.To] = toBalance

	UpdateUsers[inscription.From] = true
	UpdateUsers[inscription.To] = true

	if newHolder {
		token.Holders++
	}

	appendTradeCache(inscription, asc20.Tick, asc20.Amount.String())

	return 1, err
}
