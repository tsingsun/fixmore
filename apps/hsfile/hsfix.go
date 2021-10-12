// 恒生批量埋单文件接口
package hsfile

import (
	"fmt"
	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	fix42er "github.com/quickfixgo/fix42/executionreport"
	fix42nos "github.com/quickfixgo/fix42/newordersingle"
	"github.com/quickfixgo/quickfix"
	"github.com/shopspring/decimal"
	"github.com/tsingsun/fixmore/service/symbol"
	"github.com/tsingsun/woocoo/pkg/conf"
	"strconv"
	"sync/atomic"
)

var _ quickfix.Application = (*HSFix)(nil)

type HSFix struct {
	*quickfix.MessageRouter
	orderID    int32
	execID     int
	symbology  symbol.Symbol
	accountMap map[string]accountSet
}

func NewHSFix(cfg *conf.Configuration) *HSFix {
	sym, err := symbol.NewFileSymbology(cfg.Sub("symbol.hundsun"))
	if err != nil {
		panic(err)
	}
	f := &HSFix{MessageRouter: quickfix.NewMessageRouter()}
	f.AddRoute(fix42nos.Route(f.OnFIX42NewOrderSingle))
	f.symbology = sym
	if err := cfg.Sub("hundsun.accountMap").Parser().UnmarshalExact(&f.accountMap); err != nil {
		panic(fmt.Errorf("config err at path: hundsun.accountMap\n %s", err))
	}
	return f
}

//quickfix.Application interface
func (HSFix) OnCreate(sessionID quickfix.SessionID)                           {}
func (HSFix) OnLogon(sessionID quickfix.SessionID)                            {}
func (HSFix) OnLogout(sessionID quickfix.SessionID)                           {}
func (HSFix) ToAdmin(msg *quickfix.Message, sessionID quickfix.SessionID)     {}
func (HSFix) ToApp(msg *quickfix.Message, sessionID quickfix.SessionID) error { return nil }
func (HSFix) FromAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}
func (f *HSFix) FromApp(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return f.Route(msg, sessionID)
}

// 订单ID,需要整数才可符合HS协议
func (f *HSFix) genOrderID() int32 {
	ret := atomic.AddInt32(&f.orderID, 1)
	return ret
}

func (f *HSFix) genExecID() field.ExecIDField {
	f.execID++
	return field.NewExecID(strconv.Itoa(f.execID))
}

func (f *HSFix) OnFIX42NewOrderSingle(msg fix42nos.NewOrderSingle, sessionID quickfix.SessionID) (err quickfix.MessageRejectError) {
	var wt *WT
	orderid := f.genOrderID()
	clOrdID, err := msg.GetClOrdID()
	if err != nil {
		return
	}
	if wt, err = wtNewFromFIXNewOrderSingle(msg, f.symbology, sessionID.TargetCompID, f.accountMap); err != nil {
		return
	}
	wt.WBZDYXH = orderid

	side, _ := msg.GetSide()
	symno, _ := msg.GetSymbol()

	if err = wt.Validate(); err != nil {
		return
	}

	wterr := wt.Save()
	if wterr != nil {
		execReport := fix42er.New(
			field.NewOrderID(string(orderid)),
			f.genExecID(),
			field.NewExecTransType(enum.ExecTransType_STATUS),
			field.NewExecType(enum.ExecType_REJECTED),
			field.NewOrdStatus(enum.OrdStatus_REJECTED),
			field.NewSymbol(symno),
			field.NewSide(side),
			field.NewLeavesQty(decimal.Zero, 2),
			field.NewCumQty(decimal.Zero, 2),
			field.NewAvgPx(wt.WTJG, 2),
		)

		execReport.SetClOrdID(clOrdID)
		execReport.SetOrderQty(decimal.Zero, 2)
		execReport.SetLastShares(decimal.Zero, 2)
		execReport.SetLastPx(decimal.Zero, 2)

		if msg.HasAccount() {
			acct, err := msg.GetAccount()
			if err != nil {
				return err
			}
			execReport.SetAccount(acct)
		}

		sendErr := quickfix.SendToTarget(execReport, sessionID)
		if sendErr != nil {
			fmt.Println(sendErr)
		}
	}

	return
}
