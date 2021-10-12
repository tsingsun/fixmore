package fixmore

import (
	"bytes"
	"fmt"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/config"
	"github.com/tsingsun/fixmore/store"
	"github.com/tsingsun/woocoo/pkg/conf"
	"io/ioutil"
	"os"
)

type FixService struct {
	Settings            *quickfix.Settings
	MessageStoreFactory quickfix.MessageStoreFactory
	LogFactory          quickfix.LogFactory
	quickfix.Application
}

func NewFixService(cfg *conf.Configuration, fix quickfix.Application) (*FixService, error) {
	fs := &FixService{
		Application: fix,
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
		return fmt.Errorf("error reading cfg: %s,", err)
	}
	// 初始化存储
	if f.Settings.GlobalSettings().HasSetting(config.FileStorePath) {
		fsp, _ := f.Settings.GlobalSettings().Setting(config.FileStorePath)
		f.Settings.GlobalSettings().Set(config.FileStorePath, cfg.Abs(fsp))
		f.MessageStoreFactory = quickfix.NewFileStoreFactory(f.Settings)
	} else if f.Settings.GlobalSettings().HasSetting(config.SQLStoreDriver) {
		f.MessageStoreFactory = store.NewSQLStoreFactory(f.Settings)
	} else {
		f.MessageStoreFactory = quickfix.NewMemoryStoreFactory()
	}
	// 日志
	if f.Settings.GlobalSettings().HasSetting(config.FileLogPath) {
		fsp, _ := f.Settings.GlobalSettings().Setting(config.FileLogPath)
		f.Settings.GlobalSettings().Set(config.FileLogPath, cfg.Abs(fsp))
		f.LogFactory, err = quickfix.NewFileLogFactory(f.Settings)
		if err != nil {
			return fmt.Errorf("error file log cfg: %s", err)
		}
	} else {
		f.LogFactory = quickfix.NewScreenLogFactory()
	}
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
		return nil, fmt.Errorf("error opening %v, %v\n", path, err)
	}
	defer cFile.Close()
	stringData, readErr := ioutil.ReadAll(cFile)
	if readErr != nil {
		return nil, fmt.Errorf("error reading cfg: %s", readErr)
	}
	return quickfix.ParseSettings(bytes.NewReader(stringData))
}
