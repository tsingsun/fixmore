package hsfile

const ()

//委托状态
type WTZT string

const (
	//未报
	WTZT_IN WTZT = "0"
	//待报
	WTZT_WAITING WTZT = "1"
	//正报
	WTZT_PENDING_NEW WTZT = "3"
	//已报
	WTZT_NEW WTZT = "4"
	//废单
	WTZT_EXPIRED WTZT = "5"
	//部成
	WTZT_PARTIALLY_FILLED WTZT = "6"
	//已成
	WTZT_FILLED WTZT = "7"
	//部撤
	WTZT_PARTIALLY_CANCEL WTZT = "8"
	//已撤
	WTZT_CANCELED WTZT = "9"
	//待撤
	WTZT_PENDING_CANCEL WTZT = "a"
	//未审批
	WTZT_NO_APPROVE WTZT = "b"
	//审批拒绝
	WTZT_APPROVE_REJECTED WTZT = "c"
	//未审批即撤销
	WTZT_CANCELED_BEFORE_APPROVE WTZT = "d"
)

//WTFX 委托方向
type WTFX string

const (
	WTFX_BUY  WTFX = "1"
	WTFX_SELL WTFX = "2"
)

//交易市场
type JYSC string

const (
	//上交所
	JYSC_XSHG JYSC = "1"
	//深交所
	JYSC_XSHE JYSC = "2"
	//港股通(沪)
	JYSC_XSSC JYSC = "n"
	//港股通(深)
	JYSC_XSEC JYSC = "o"
)

//WTJGLX 价格类型/申报类型
type WTJGLX rune

const (
	//限价;限价盘(零股)(港股通)
	WTJGLX_LIMIT_ODD WTJGLX = '0'
	//竞价限价盘(港股通);市价剩余撤消（上交所股票期权）
	WTJGLX_LIMIT WTJGLX = '2'
	//增强限价盘(港股通);FOK市价（上交所股票期权）
	WTJGLX_ENHANCED_LIMIT WTJGLX = '4'
	//五档即成剩撤（上交所市价）
	WTJGLX_MARKET_CANCEL WTJGLX = 'a'
	//五档即成剩转（上交所市价）
	WTJGLX_market_limit WTJGLX = 'b'
	//五档即成剩撤（深交所市价）
	WTJGLX_MARKET_CANCEL_SZ WTJGLX = 'A'
	//即成剩撤（深交所市价）
	WTJGLX_MARKET_IM_CANCEL_SZ WTJGLX = 'C'
	//对手方最优（深交所市价，上交所科创板市价）
	WTJGLX_MARKET_PARTY_BETTER WTJGLX = 'D'
	//本方最优（深交所市价，上交所科创板市价）
	WTJGLX_MARKET_BETTER WTJGLX = 'E'
	//全额成或撤（深交所市价）
	WTJGLX_MARKET_FULL_OR_Cancel_SZ WTJGLX = 'F'
)

//WTSBDM 委托失败代码
type WTSBDM int32

const (
	//成功
	WTSBDM_OK WTSBDM = 0
	//风控禁止
	WTSBDM_RISK WTSBDM = 1
	//可用不足
	WTSBDM_AVAILABLE WTSBDM = 2
	//其他
	WTSBDM_OTHER WTSBDM = 3
	//因为其他委托导致此笔委托失败
	WTSBDM_OTHER_WT WTSBDM = 99
)
