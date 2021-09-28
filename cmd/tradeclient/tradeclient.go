// Copyright (c) quickfixengine.org  All rights reserved.
//
// This file may be distributed under the terms of the quickfixengine.org
// license as defined by quickfixengine.org and appearing in the file
// LICENSE included in the packaging of this file.
//
// This file is provided AS IS with NO WARRANTY OF ANY KIND, INCLUDING
// THE WARRANTY OF DESIGN, MERCHANTABILITY AND FITNESS FOR A
// PARTICULAR PURPOSE.
//
// See http://www.quickfixengine.org/LICENSE for licensing information.
//
// Contact ask@quickfixengine.org if any conditions of this licensing
// are not clear to you.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/quickfixgo/quickfix"
	"github.com/tsingsun/fixmore/cmd/tradeclient/internal/basic"
	"github.com/tsingsun/fixmore/cmd/tradeclient/internal/oms"
	"github.com/tsingsun/fixmore/cmd/tradeclient/internal/secmaster"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web"
	"github.com/urfave/cli/v2"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

var (
	// TradeClientCmd is the quote command.
	TradeClientCmd = &cli.Command{
		Name:    "fix-client",
		Aliases: []string{"tc"},
		Usage:   "fix tc -c ./etc/tradeclient.cfg",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "./etc/app.yaml",
			},
		},
		Action: execute,
	}
)

func execute(c *cli.Context) error {
	var srv = web.Default()
	cfgFileName := conf.Abs(conf.String("quickfix.configFilePath"))
	cfg, err := os.Open(cfgFileName)
	if err != nil {
		return fmt.Errorf("Error opening %v, %v\n", cfgFileName, err)
	}
	defer cfg.Close()

	stringData, readErr := ioutil.ReadAll(cfg)
	if readErr != nil {
		return fmt.Errorf("Error reading cfg: %s,", readErr)
	}

	appSettings, err := quickfix.ParseSettings(bytes.NewReader(stringData))
	if err != nil {
		return fmt.Errorf("Error reading cfg: %s,", err)
	}

	var fixApp quickfix.Application
	app := newTradeClient(basic.FIXFactory{}, new(basic.ClOrdIDGenerator))
	fixApp = &basic.FIXApplication{
		SessionIDs:   app.SessionIDs,
		OrderManager: app.OrderManager,
	}

	fileLogFactory, err := quickfix.NewFileLogFactory(appSettings)

	if err != nil {
		return fmt.Errorf("Error creating file log factory: %s,", err)
	}

	initiator, err := quickfix.NewInitiator(fixApp, quickfix.NewMemoryStoreFactory(), appSettings, fileLogFactory)
	if err != nil {
		return fmt.Errorf("Unable to create Initiator: %s\n", err)
	}

	err = initiator.Start()
	if err != nil {
		return fmt.Errorf("Unable to start Initiator: %s\n", err)
	}
	defer initiator.Stop()

	router := srv.Router().Engine
	//router.LoadHTMLGlob("tmpl/*")
	router.POST("/orders", app.newOrder)
	router.GET("/orders", app.getOrders)
	router.GET("/orders/{id:[0-9]+}", app.getOrder)
	router.DELETE("/orders/{id:[0-9]+}", app.deleteOrder)

	router.GET("/executions", app.getExecutions)
	router.GET("/executions/{id:[0-9]+}", app.getExecution)

	router.POST("/securitydefinitionrequest", app.newSecurityDefintionRequest)

	router.Static("/assets/", conf.Abs("./assets/"))

	router.GET("/", gin.WrapF(app.traderView))
	return srv.Run(true)
}

type fixFactory interface {
	NewOrderSingle(ord oms.Order) (msg quickfix.Messagable, err error)
	OrderCancelRequest(ord oms.Order, clOrdID string) (msg quickfix.Messagable, err error)
	SecurityDefinitionRequest(req secmaster.SecurityDefinitionRequest) (msg quickfix.Messagable, err error)
}

//TradeClient implements the quickfix.Application interface
type tradeClient struct {
	SessionIDs map[string]quickfix.SessionID
	fixFactory
	*oms.OrderManager
}

func newTradeClient(factory fixFactory, idGen oms.ClOrdIDGenerator) *tradeClient {
	tc := &tradeClient{
		SessionIDs:   make(map[string]quickfix.SessionID),
		fixFactory:   factory,
		OrderManager: oms.NewOrderManager(idGen),
	}

	return tc
}

func (c tradeClient) SessionsAsJSON() (string, error) {
	sessionIDs := make([]string, 0, len(c.SessionIDs))

	for s := range c.SessionIDs {
		sessionIDs = append(sessionIDs, s)
	}

	b, err := json.Marshal(sessionIDs)
	return string(b), err
}

func (c tradeClient) OrdersAsJSON() (string, error) {
	c.RLock()
	defer c.RUnlock()

	b, err := json.Marshal(c.GetAll())
	return string(b), err
}

func (c tradeClient) ExecutionsAsJSON() (string, error) {
	c.RLock()
	defer c.RUnlock()

	b, err := json.Marshal(c.GetAllExecutions())
	return string(b), err
}

func (c tradeClient) traderView(w http.ResponseWriter, r *http.Request) {
	var templates = template.Must(template.New("tradeclient").ParseFiles("tmpl/index.html"))
	if err := templates.ExecuteTemplate(w, "index.html", c); err != nil {
		log.Printf("[ERROR] err = %+v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (c tradeClient) fetchRequestedOrder(ctx *gin.Context) (*oms.Order, error) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		return nil, err
	}

	return c.Get(id)
}

func (c tradeClient) fetchRequestedExecution(ctx *gin.Context) (*oms.Execution, error) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		panic(err)
	}

	return c.GetExecution(id)
}

func (c tradeClient) getOrder(ctx *gin.Context) {
	c.RLock()
	defer c.RUnlock()

	order, err := c.fetchRequestedOrder(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusNotFound, err)
		return
	}
	ctx.JSON(http.StatusOK, order)
}

func (c tradeClient) getExecution(ctx *gin.Context) {
	c.RLock()
	defer c.RUnlock()

	exec, err := c.fetchRequestedExecution(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusNotFound, err)
		return
	}
	ctx.JSON(200, exec)
}

func (c tradeClient) deleteOrder(ctx *gin.Context) {
	order, err := c.fetchRequestedOrder(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusNotFound, err)
		return
	}

	clOrdID := c.AssignNextClOrdID(order)
	msg, err := c.OrderCancelRequest(*order, clOrdID)
	if err != nil {
		log.Printf("[ERROR] err = %+v\n", err)
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	err = quickfix.SendToTarget(msg, order.SessionID)
	if err != nil {
		log.Printf("[ERROR] err = %+v\n", err)
		ctx.AbortWithError(http.StatusInternalServerError, err)
	}
	ctx.JSON(http.StatusOK, order)
}

func (c tradeClient) getOrders(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.GetAll())
}

func (c tradeClient) getExecutions(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, c.GetAllExecutions())
}

func (c tradeClient) newSecurityDefintionRequest(ctx *gin.Context) {
	var secDefRequest secmaster.SecurityDefinitionRequest
	err := ctx.ShouldBindJSON(&secDefRequest)
	if err != nil {
		log.Printf("[ERROR] %v\n", err)
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	log.Printf("secDefRequest = %+v\n", secDefRequest)

	if sessionID, ok := c.SessionIDs[secDefRequest.Session]; ok {
		secDefRequest.SessionID = sessionID
	} else {
		log.Println("[ERROR] Invalid SessionID")
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("Invalid SessionID"))
		return
	}

	msg, err := c.fixFactory.SecurityDefinitionRequest(secDefRequest)
	if err != nil {
		log.Printf("[ERROR] %v\n", err)
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	err = quickfix.SendToTarget(msg, secDefRequest.SessionID)

	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
	}
}

func (c tradeClient) newOrder(ctx *gin.Context) {
	var order oms.Order
	err := ctx.ShouldBindJSON(&order)
	if err != nil {
		log.Printf("[ERROR] %v\n", err)
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if sessionID, ok := c.SessionIDs[order.Session]; ok {
		order.SessionID = sessionID
	} else {
		log.Println("[ERROR] Invalid SessionID")
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if err = order.Init(); err != nil {
		log.Printf("[ERROR] %v\n", err)
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.Lock()
	_ = c.OrderManager.Save(&order)
	c.Unlock()

	msg, err := c.NewOrderSingle(order)
	if err != nil {
		log.Printf("[ERROR] %v\n", err)
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	err = quickfix.SendToTarget(msg, order.SessionID)

	if err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
	}
}
