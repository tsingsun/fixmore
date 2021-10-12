package symbol

import (
	"fmt"
	"github.com/tsingsun/woocoo/pkg/conf"
	"log"
	"strconv"
	"sync"
)

// symbolSet is the qeelyn symbol
type symbolSet struct {
	symbols     map[string]string
	passthrough bool
}

func newSymbolset() *symbolSet {
	return &symbolSet{
		symbols: make(map[string]string),
	}
}

func (s *symbolSet) set(k, v string) {
	s.symbols[k] = v
}

func (s *symbolSet) get(k string) (string, bool) {
	sym, ok := s.symbols[k]
	return sym, ok
}

// exchangeSet is the qeelyn symbol
type exchangeSet struct {
	exchanges   map[string]string
	passthrough bool
}

func newExchangeSet() *exchangeSet {
	return &exchangeSet{
		exchanges: make(map[string]string),
	}
}

func (s *exchangeSet) set(k, v string) {
	s.exchanges[k] = v
}

func (s *exchangeSet) get(k string) (string, bool) {
	sym, ok := s.exchanges[k]
	return sym, ok
}

var _ Symbol = (*FileSymbology)(nil)

type FileSymbology struct {
	counterparty string
	symbols      map[string]*symbolSet
	exchanges    map[string]*exchangeSet
	lock         sync.Mutex
}

func (f *FileSymbology) Apply(cfg *conf.Configuration) {
	if cfg == nil || len(cfg.AllSettings()) == 0 {
		panic(fmt.Errorf("symbol config is not found"))
	}
	f.counterparty = cfg.String("counterparty")
	symbols := newSymbolset()
	for k, v := range cfg.Sub("symbolSet").AllSettings() {
		if k == "passthrough" {
			symbols.passthrough = true
		} else {
			//TODO need all type convert
			switch v.(type) {
			case string:
				symbols.set(k, v.(string))
			case int:
				symbols.set(k, strconv.Itoa(v.(int)))
			}
		}
	}
	f.symbols[f.counterparty] = symbols
	exchanges := newExchangeSet()
	for k, v := range cfg.Sub("exchangeSet").AllSettings() {
		if k == "passthrough" {
			exchanges.passthrough = true
		} else {
			exchanges.set(k, v.(string))
		}
	}
	f.exchanges[f.counterparty] = exchanges
}

// NewFileSymbology creates a new file symbology object from a given path
func NewFileSymbology(cfg *conf.Configuration) (*FileSymbology, error) {
	s := &FileSymbology{
		symbols:   make(map[string]*symbolSet),
		exchanges: make(map[string]*exchangeSet),
	}
	s.Apply(cfg)
	return s, nil
}

// ToSender converts symbol to Qeelyn form
func (f *FileSymbology) ToSender(symbol, counterparty string) (string, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	symset, ok := f.symbols[counterparty]
	if !ok {
		log.Printf("could not find counterparty: %s", counterparty)
		return "", fmt.Errorf("could not find counterparty: %s", counterparty)
	}

	for bfx, cp := range symset.symbols {
		if cp == symbol {
			return bfx, nil
		}
	}

	if symset.passthrough {
		return symbol, nil
	}

	log.Printf("could not find Qeelyn symbol mapping \"%s\" for counterparty \"%s\"", symbol, counterparty)
	return "", fmt.Errorf(
		"could not find Qeelyn symbol mapping \"%s\" for counterparty \"%s\"", symbol, counterparty)
}

// FromSender converts symbol from Qeelyn form
func (f *FileSymbology) FromSender(symbol, counterparty string) (string, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	symset, ok := f.symbols[counterparty]
	if !ok {
		return "", fmt.Errorf("could not find counterparty: %s", counterparty)
	}
	sym, ok := symset.get(symbol)
	if !ok && !symset.passthrough {
		return "", fmt.Errorf("could not find symbol \"%s\" for counterparty \"%s\"", symbol, counterparty)
	} else if symset.passthrough {
		return symbol, nil
	}

	return sym, nil
}

func (f *FileSymbology) ExchangeToSender(exchange, counterparty string) (string, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	exset, ok := f.exchanges[counterparty]
	if !ok {
		log.Printf("could not find counterparty: %s", counterparty)
		return "", fmt.Errorf("could not find counterparty: %s", counterparty)
	}
	for bfx, cp := range exset.exchanges {
		if cp == exchange {
			return bfx, nil
		}
	}
	if exset.passthrough {
		return exchange, nil
	}

	log.Printf("could not find Qeelyn exchange mapping \"%s\" for counterparty \"%s\"", exchange, counterparty)
	return "", fmt.Errorf(
		"could not find Qeelyn exchange mapping \"%s\" for counterparty \"%s\"", exchange, counterparty)
}

func (f *FileSymbology) ExchangeFromSender(exchange, counterparty string) (string, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	exset, ok := f.exchanges[counterparty]
	if !ok {
		return "", fmt.Errorf("could not find counterparty: %s", counterparty)
	}
	sym, ok := exset.get(exchange)
	if !ok && !exset.passthrough {
		return "", fmt.Errorf("could not find exchange \"%s\" for counterparty \"%s\"", exchange, counterparty)
	} else if exset.passthrough {
		return exchange, nil
	}

	return sym, nil
}
