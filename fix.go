package fixmore

import (
	"fmt"
	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	fix42er "github.com/quickfixgo/fix42/executionreport"
	fix42nos "github.com/quickfixgo/fix42/newordersingle"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
	"github.com/shopspring/decimal"
	"strconv"
)

var _ quickfix.Application = (*Fix)(nil)

type Fix struct {
	*quickfix.MessageRouter
	orderID int
	execID  int
}

func NewFix() *Fix {
	f := &Fix{MessageRouter: quickfix.NewMessageRouter()}
	f.AddRoute(fix42nos.Route(f.OnFIX42NewOrderSingle))
	return f
}

//quickfix.Application interface
func (Fix) OnCreate(sessionID quickfix.SessionID)                           {}
func (Fix) OnLogon(sessionID quickfix.SessionID)                            {}
func (Fix) OnLogout(sessionID quickfix.SessionID)                           {}
func (Fix) ToAdmin(msg *quickfix.Message, sessionID quickfix.SessionID)     {}
func (Fix) ToApp(msg *quickfix.Message, sessionID quickfix.SessionID) error { return nil }
func (Fix) FromAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}
func (f *Fix) FromApp(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return f.Route(msg, sessionID)
}

func (f *Fix) genOrderID() field.OrderIDField {
	f.orderID++
	return field.NewOrderID(strconv.Itoa(f.orderID))
}

func (f *Fix) genExecID() field.ExecIDField {
	f.execID++
	return field.NewExecID(strconv.Itoa(f.execID))
}

func (f *Fix) OnFIX42NewOrderSingle(msg fix42nos.NewOrderSingle, sessionID quickfix.SessionID) (err quickfix.MessageRejectError) {
	ordType, err := msg.GetOrdType()
	if err != nil {
		return err
	}

	if ordType != enum.OrdType_LIMIT {
		return quickfix.ValueIsIncorrect(tag.OrdType)
	}

	symbol, err := msg.GetSymbol()
	if err != nil {
		return
	}

	side, err := msg.GetSide()
	if err != nil {
		return
	}

	orderQty, err := msg.GetOrderQty()
	if err != nil {
		return
	}

	price, err := msg.GetPrice()
	if err != nil {
		return
	}

	clOrdID, err := msg.GetClOrdID()
	if err != nil {
		return
	}

	execReport := fix42er.New(
		f.genOrderID(),
		f.genExecID(),
		field.NewExecTransType(enum.ExecTransType_NEW),
		field.NewExecType(enum.ExecType_FILL),
		field.NewOrdStatus(enum.OrdStatus_FILLED),
		field.NewSymbol(symbol),
		field.NewSide(side),
		field.NewLeavesQty(decimal.Zero, 2),
		field.NewCumQty(orderQty, 2),
		field.NewAvgPx(price, 2),
	)

	execReport.SetClOrdID(clOrdID)
	execReport.SetOrderQty(orderQty, 2)
	execReport.SetLastShares(orderQty, 2)
	execReport.SetLastPx(price, 2)

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

	return
}
