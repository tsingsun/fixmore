package hsfile

import (
	"bytes"
	"fmt"
	"github.com/svolodeev/xbase"
	testdata "github.com/tsingsun/fixmore/test/data"
	"github.com/tsingsun/woocoo/pkg/conf"
	"path/filepath"
	"testing"
)

func TestWtDbf(t *testing.T) {
	db := xbase.New()
	db.SetPanic(false)
	// Open 委托 file
	fileName := filepath.Join(testdata.BaseDir(), "dbf/XHPT_WT20210929.dbf")
	db.OpenFile(fileName, false)
	if db.Error() != nil {
		t.Fatal(db.Error())
	}
	defer db.CloseFile()

	db.First()
	// Print all the fieldnames
	for !db.EOF() {
		name := db.FieldValueAsString(1)
		salary := db.FieldValueAsFloat(2)
		bDate := db.FieldValueAsDate(3)
		fmt.Println(name, salary, bDate)
		db.Next()
	}
	db.Save()
}

func TestNewHSFix(t *testing.T) {
	b := []byte(`
appName: fixmore
development: true
quickfix:
  configFilePath: "etc/executor.cfg"
hundsun: 
  accountMap:
    1000: 
      ZJZH: "02510006" 
      CPBH: "00080003" 
      SHAGDDM: "B881182401" 
      ZZAGDDM: "0800049247" 
      ZCDYBH: "1"
symbol:
  # 恒生
  hundsun:
    counterparty: qeelyn
    symbolSet:
      100001: 100001
      passthrough: true
    exchangeSet:
      XSHG: "1"
      XSHE: "2"
      XSSC: "3"
      XSEC: "4" 
`)
	p, err := conf.NewParserFromBuffer(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	cfg := conf.Operator().CutFromParser(p)
	NewHSFix(cfg)

}
