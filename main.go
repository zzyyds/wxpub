package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/domain/repository"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/g"

	extra "git.nova.net.cn/nova/go-common/logrus-extra"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/config"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/consts"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/infrastructure/persistence"
	"git.nova.net.cn/nova/misc/wx-public/proxy/internal/tasks"
	log "github.com/sirupsen/logrus"
)

var (
	globalCtx    context.Context
	globalCancel context.CancelFunc
)

func main() {
	config.Init()
	extra.Default(config.LogLevel)
	globalCtx, globalCancel = context.WithCancel(context.Background())
	// init
	InitService()
	tasks.ConsumerTask(globalCtx)

	engine := internal.Run()
	srv := &http.Server{
		Addr:    config.ListenAddr,
		Handler: engine,
	}
	startServer(srv)
	gracefulShutdown(srv)
	globalCancel()
	g.Wait()
}

func startServer(srv *http.Server) {
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("http server listen err: %+v", err)
		}
	}()
}

func gracefulShutdown(srv *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infoln("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Infoln("Server exiting")
}

func InitService() {
	debugMode := config.SMode == consts.ServerModeDebug
	dbConf := persistence.DBConfig{
		DBUser:      config.DBUser,
		DBPassword:  config.DBPassword,
		DBHost:      config.DBHost,
		DBName:      config.DBName,
		MaxIdleConn: config.DBMaxIdleConn,
		MaxOpenConn: config.DBMaxOpenConn,
	}
	kafkaConf := persistence.KafkaConfig{
		Config:          nil,
		Brokers:         config.KafkaBrokers,
		ConsumerGroupID: config.KafkaGroup,
		Topics:          config.KafkaTopics,
		KafkaVersion:    config.KafkaVersion,
	}
	err := persistence.NewRepositories(kafkaConf, dbConf, config.RedisAddresses, config.SmsRPCAddr, debugMode)
	if err != nil {
		panic(err)
	}
	// repository init
	repository.NewWXRepository(
		persistence.DefaultWxRepo(), persistence.DefaultUserRepo(), persistence.DefaultMessageRepo())
	repository.NewAccessTokenRepository(
		persistence.DefaultAkRepo())
	repository.NewUserRepository(
		persistence.DefaultUserRepo(), persistence.DefaultPhoneVerifyRepo())
	repository.NewMessageRepository(
		persistence.DefaultMessageRepo(), persistence.DefaultUserRepo())
	repository.NewPassportRepository(
		persistence.DefaultPassportRepo())
}
