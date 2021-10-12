package hsfile

import (
	"github.com/quickfixgo/enum"
	fix42nos "github.com/quickfixgo/fix42/newordersingle"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
	"github.com/tsingsun/fixmore/service/symbol"
)

func wtNewFromFIXNewOrderSingle(msg fix42nos.NewOrderSingle, symbology symbol.Symbol, counterparty string, acMap map[string]accountSet) (wt *WT, err quickfix.MessageRejectError) {
	var symno, exno, acno string
	wt = newWT()

	//订单类型
	ordType, err := msg.GetOrdType()
	if err != nil {
		return nil, err
	}

	orderQty, err := msg.GetOrderQty()
	if err != nil {
		return
	}
	wt.WTSL = orderQty.IntPart()
	//价格
	if ordType != enum.OrdType_MARKET {
		price, err := msg.GetPrice()
		if err != nil {
			return nil, err
		}
		wt.WTJG = price
	}
	// 标的
	symno, err = msg.GetSymbol()
	if err != nil {
		return nil, err
	}
	zqdm, ierr := symbology.ToSender(symno, counterparty)
	if ierr != nil {
		err = quickfix.ValueIsIncorrect(tag.Symbol)
		return nil, err
	}
	wt.ZQDM = zqdm
	// 交易所
	exno, err = msg.GetSecurityExchange()
	if err != nil {
		return nil, err
	}
	jysc, ierr := symbology.ExchangeToSender(exno, counterparty)
	if err != nil {
		err = quickfix.ValueIsIncorrect(tag.SecurityExchange)
		return nil, err
	}
	wt.JYSC = JYSC(jysc)

	//委托方向
	side, err := msg.GetSide()
	if err != nil {
		return nil, err
	}
	if err = wt.ParseSide(side); err != nil {
		return nil, err
	}

	tif, err := msg.GetTimeInForce()
	if err != nil {
		return nil, err
	}
	if err = wt.ParseOrdType(ordType, tif); err != nil {
		return nil, err
	}

	// 账号相关
	acno, err = msg.GetString(tag.Account)
	if err != nil {
		return nil, err
	}

	if a, ok := acMap[acno]; ok {
		if wt.SetAccount(a) != nil {
			err = quickfix.ValueIsIncorrect(tag.Account)
			return nil, err
		}
	} else {
		err = quickfix.ValueIsIncorrect(tag.Account)
		return nil, err
	}
	return
}
