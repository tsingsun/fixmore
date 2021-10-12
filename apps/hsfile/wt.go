// 恒生PDF
package hsfile

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
	"github.com/shopspring/decimal"
	"github.com/svolodeev/xbase"
	"path/filepath"
	"sync"
	"time"
)

var (
	wtdb *xbase.XBase
)

// WT according version 1.2.5
type WT struct {
	//产品代码/基金代码
	CPBH string `validate:"required,len=32"`
	//单元编号/组合编号
	ZCDYBH string `validate:"required,len=16"`
	//组合编号
	ZHBH string `validate:"len=16"`
	//股东代码
	GDDM string `validate:"len=20"`
	//交易市场
	JYSC JYSC `validate:"len=3"`
	//证券代码
	ZQDM string `validate:"len=16"`
	//委托方向
	WTFX WTFX `validate:"required,len=4"`
	//委托价格类型
	WTJGLX WTJGLX `validate:"required,len=1"`
	//委托价格,TODO 精度验证
	WTJG decimal.Decimal `validate:"required,len=11"`
	//委托数量
	WTSL int64 `validate:"required,len=12"`
	//第三方系统自定义号
	WBZDYXH int32 `validate:"required,len=9"`
	//委托序号
	WTXH int64 `validate:"len=8"`
	//委托失败代码
	WTSBDM int32 `validate:"len=8"`
	//失败原因
	SBYY string
	//处理标志
	CLBZ string
	//备用字段
	BYZD string
	//委托金额 TODO 精度验证
	WTJE decimal.Decimal `validate:"required,len=16"`
	//特殊标识
	TSBS string
	//业务标识
	YWBS string
	mu   sync.RWMutex
}

func newWT() *WT {
	return &WT{}
}

//NewWTDB init WT
func NewWTDB(dir string, test bool) (*xbase.XBase, error) {
	var filename string
	wtdb := xbase.New()
	wtdb.SetPanic(false)
	if test {
		filename = filepath.Join(dir, fmt.Sprintf("XHPT_WT%s.dbf", time.Now().Format("20060102")))
	} else {
		filename = filepath.Join(dir, "XHPT_WT20210929.dbf")
	}
	wtdb.OpenFile(filename, false)
	if wtdb.Error() != nil {
		return nil, wtdb.Error()
	}
	return wtdb, nil
}

//SetAccount must call at last
func (w *WT) SetAccount(set accountSet) error {
	w.CPBH = set.CPBH
	w.ZCDYBH = set.ZCDYBH
	w.ZHBH = set.ZCDYBH
	if w.JYSC == "" {
		return fmt.Errorf("exchange not found,call SetAccount may be wrong")
	}
	switch JYSC(w.JYSC) {
	case JYSC_XSHG, JYSC_XSSC: //上交所
		w.GDDM = set.SHAGDDM
	case JYSC_XSHE, JYSC_XSEC: //深交所
		w.GDDM = set.ZZAGDDM
	}
	return nil
}

func (w *WT) ParseSide(side enum.Side) quickfix.MessageRejectError {
	switch side {
	case enum.Side_BUY:
		w.WTFX = WTFX_BUY
	case enum.Side_SELL:
		w.WTFX = WTFX_SELL
	default:
		return quickfix.ValueIsIncorrect(tag.Side)
	}
	return nil
}

//TODO check qeelyn support
func (w *WT) ParseOrdType(ordType enum.OrdType, tif enum.TimeInForce) quickfix.MessageRejectError {
	var wtjglx WTJGLX
	switch ordType {
	case enum.OrdType_LIMIT:
		if w.JYSC == JYSC_XSHG || w.JYSC == JYSC_XSHE { //A股
			wtjglx = WTJGLX_LIMIT_ODD
		} else if w.JYSC == JYSC_XSEC || w.JYSC == JYSC_XSSC { //港股通
			//TODO odd check
			if w.WTSL < 100 {
				wtjglx = WTJGLX_LIMIT_ODD
			} else {
				wtjglx = WTJGLX_LIMIT
			}
		}
	case enum.OrdType_MARKET:
		if w.JYSC == JYSC_XSHG { //上
			wtjglx = WTJGLX_MARKET_CANCEL
		} else if w.JYSC == JYSC_XSHE { //深
			wtjglx = WTJGLX_MARKET_CANCEL_SZ
		}
	default:
		return quickfix.ValueIsIncorrect(tag.OrdType)
	}
	w.WTJGLX = wtjglx
	return nil
}

func (w *WT) Validate() quickfix.MessageRejectError {
	validate := validator.New()
	err := validate.Struct(w)
	if err != nil {
		return quickfix.NewMessageRejectError(err.Error(), 5, nil)
	}
	return nil
}

func (w *WT) Save() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	wtdb.Add()
	wtdb.SetFieldValue(1, w.CPBH)
	wtdb.SetFieldValue(2, w.ZCDYBH)
	if w.ZHBH != "" {
		wtdb.SetFieldValue(3, w.ZHBH)
	}
	if w.GDDM != "" {
		wtdb.SetFieldValue(4, w.GDDM)
	}
	if w.JYSC != "" {
		wtdb.SetFieldValue(5, w.JYSC)
	}
	if w.ZQDM != "" {
		wtdb.SetFieldValue(6, w.ZQDM)
	}
	wtdb.SetFieldValue(7, w.WTFX)
	wtdb.SetFieldValue(8, w.WTJGLX)
	wtdb.SetFieldValue(9, w.WTJG)
	wtdb.SetFieldValue(10, w.WTSL)
	wtdb.SetFieldValue(11, w.WBZDYXH)
	if w.WTJE != decimal.Zero {
		wtdb.SetFieldValue(17, w.WTJE.String())
	}
	if w.TSBS != "" {
		wtdb.SetFieldValue(18, w.TSBS)
	}
	if w.YWBS != "" {
		wtdb.SetFieldValue(18, w.YWBS)
	}
	wtdb.Save()
	return nil
}
