package fixmore

import (
	"bytes"
	"fmt"
	"github.com/quickfixgo/quickfix"
	"github.com/tsingsun/woocoo/pkg/conf"
	"io/ioutil"
	"os"
)

type FixService struct {
	Settings            *quickfix.Settings
	MessageStoreFactory quickfix.MessageStoreFactory
	LogFactory          quickfix.LogFactory
	*Fix
}

func NewFixService(cfg *conf.Configuration) (*FixService, error) {
	fs := &FixService{
		Fix: NewFix(),
	}
	fs.Apply(cfg)
	return fs, nil
}

func (f *FixService) Apply(cfg *conf.Configuration) {
	err := f.initQuickFix(cfg)
	if err != nil {
		panic(err)
	}
}

func (f *FixService) initQuickFix(cfg *conf.Configuration) (err error) {
	cPath := cfg.Abs(cfg.String("quickfix.configFilePath"))
	f.Settings, err = parseFileSettings(cPath)
	if err != nil {
		return fmt.Errorf("Error reading cfg: %s,", err)
	}
	f.LogFactory = quickfix.NewScreenLogFactory()

	f.MessageStoreFactory = quickfix.NewSQLStoreFactory(f.Settings)
	f.MessageStoreFactory = quickfix.NewMemoryStoreFactory()
	return nil
}

// parseDbSettings 从数据库解析QuickFix配置
func parseDbSettings(dns string) (*quickfix.Settings, error) {
	return nil, nil
}

// parseFileSettings 从配置文件解析QuickFix配置
func parseFileSettings(path string) (*quickfix.Settings, error) {
	cFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Error opening %v, %v\n", path, err)
	}
	defer cFile.Close()
	stringData, readErr := ioutil.ReadAll(cFile)
	if readErr != nil {
		return nil, fmt.Errorf("Error reading cfg: %s,", readErr)
	}
	return quickfix.ParseSettings(bytes.NewReader(stringData))
}
