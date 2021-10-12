package symbol

//Symbol 提供与标的有关的接口,包含标的代码,交易所代码的转换功能
//Sender对应FIX SenderCompID
type Symbol interface {
	ToSender(symbol, counterparty string) (string, error)
	FromSender(symbol, counterparty string) (string, error)
	ExchangeToSender(exchange, counterparty string) (string, error)
	ExchangeFromSender(exchange, counterparty string) (string, error)
}
