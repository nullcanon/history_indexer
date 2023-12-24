package main

import (
	"context"
	"flag"
	"fmt"
	indexer "open-indexer"
	"open-indexer/config"
	"open-indexer/db"
	"open-indexer/handlers"
	"open-indexer/loader"
	"open-indexer/model"
	"open-indexer/plugin"
	"open-indexer/structs"
)

var (
	inputfile  string
	outputfile string
)

func init() {
	flag.StringVar(&inputfile, "input", "./data/asc20.input.txt", "the filename of input data, default(./data/asc20.input.txt)")
	flag.StringVar(&outputfile, "output", "./data/asc20.output.txt", "the filename of output result, default(./data/asc20.output.txt)")

	flag.Parse()
}

func main() {

	config.InitConfig()
	db.Setup()
	blockscan := db.HistoryBlockScan{}
	nblockNumber, number, gas := blockscan.GetNumber()
	blockNumber := uint64(nblockNumber) + 1
	handlers.InscriptionNumber = uint64(number)
	handlers.Gas = gas
	api := config.Global.Api
	w := indexer.NewHttpBasedEthWatcher(context.Background(), api)

	var logger = handlers.GetLogger()
	loader.LoadDataBase()

	logger.Info("start index ", blockNumber)

	// we use BlockPlugin here
	w.RegisterBlockPlugin(plugin.NewSimpleBlockPlugin(func(block *structs.RemovableBlock) {
		if block.IsRemoved {
			fmt.Println("Removed >>", block.Hash(), block.Number())
		} else {
			var trxs []*model.Transaction

			// fmt.Println("Adding >>", block.Hash(), block.Number())

			for i := 0; i < len(block.GetTransactions()); i++ {
				tx := block.GetTransactions()[i]
				// fmt.Println("Adding >>", tx.GetHash(), tx.GetIndex(), tx.GetIndex(), tx.GetInput())

				var data model.Transaction

				data.Id = tx.GetHash()
				data.From = tx.GetFrom()
				data.To = tx.GetTo()
				data.Block = tx.GetBlockNumber()
				data.Idx = uint32(tx.GetIndex())
				data.Timestamp = block.Timestamp()
				data.Input = tx.GetInput()
				data.Gas = tx.GetGas()

				trxs = append(trxs, &data)
			}
			handlers.BlockNumber = block.Number()
			err := handlers.ProcessUpdateARC20(trxs)
			if err != nil {
				// logger.Fatalf("process error, %s", err)
			}
			loader.DumpTickerInfoToDB(handlers.Tokens, handlers.UserBalances, handlers.TokenHolders)
			loader.DumpBlockNumber()
			logger.Infoln("block ", block.Number())
		}
	}))

	err := w.RunTillExitFromBlock(blockNumber)
	logger.Errorln(err.Error())

	// trxs, err := loader.LoadTransactionData(inputfile)
	// if err != nil {
	// 	logger.Fatalf("invalid input, %s", err)
	// }

	// err = handlers.ProcessUpdateARC20(trxs)
	// if err != nil {
	// 	logger.Fatalf("process error, %s", err)
	// }

	// print
	// loader.DumpTickerInfoMap(outputfile, handlers.Tokens, handlers.UserBalances, handlers.TokenHolders)

}
