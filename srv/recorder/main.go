package main

import (
	"github.com/micro/cli"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/config"
	"github.com/micro/go-micro/service/grpc"
	log "github.com/sirupsen/logrus"
	myConfig "github.com/xmlking/micro-starter-kit/shared/config"
	logger "github.com/xmlking/micro-starter-kit/shared/log"
	"github.com/xmlking/micro-starter-kit/shared/util"
	logWrapper "github.com/xmlking/micro-starter-kit/shared/wrapper/log"
	recorderPB "github.com/xmlking/micro-starter-kit/srv/recorder/proto/recorder"
	"github.com/xmlking/micro-starter-kit/srv/recorder/registry"
)

const (
	serviceName = "recordersrv"
)

var (
	configDir  string
	configFile string
	cfg        myConfig.ServiceConfiguration
)

func main() {
	// New Service
	service := grpc.NewService(
		// optional cli flag to override config.
		// comment out if you don't need to override any base config via CLI
		micro.Flags(
			cli.StringFlag{
				Name:        "configDir, d",
				Value:       "/config",
				Usage:       "Path to the config directory. Defaults to 'config'",
				EnvVar:      "CONFIG_DIR",
				Destination: &configDir,
			},
			cli.StringFlag{
				Name:        "configFile, f",
				Value:       "config.yaml",
				Usage:       "Config file in configDir. Defaults to 'config.yaml'",
				EnvVar:      "CONFIG_FILE",
				Destination: &configFile,
			}),
		micro.Name(serviceName),
		micro.Version(myConfig.Version),
	)

	// Initialize service
	service.Init(
		// TODO : implement graceful shutdown
		micro.Action(func(c *cli.Context) {
			// load config
			myConfig.InitConfig(configDir, configFile)
			_ = config.Scan(&cfg)
			logger.InitLogger(cfg.Log)
		}),
	)
	// Initialize Features
	var options []micro.Option
	if cfg.Features["mtls"].Enabled {
		if tlsConf, err := util.GetSelfSignedTLSConfig("localhost"); err != nil {
			log.WithError(err).Fatal("unable to load certs")
		} else {
			options = append(options,
				// https://github.com/ykumar-rb/ZTP/blob/master/pnp/server.go
				grpc.WithTLS(tlsConf),
			)
		}
	}
	if cfg.Features["reqlogs"].Enabled {
		options = append(options,
			micro.WrapSubscriber(logWrapper.NewSubscriberWrapper()),
		)
	}

	// Initialize Features
	service.Init(
		options...,
	)

	// Initialize DI Container
	ctn, err := registry.NewContainer(cfg)
	defer ctn.Clean()
	if err != nil {
		log.Fatalf("failed to build container: %v", err)
	}

	transactionSubscriber := ctn.Resolve("transaction-subscriber") //.(subscriber.TransactionSubscriber)
	recorderTopic := config.Get("recorder", "topic").String("recordersrv")

	// Register Struct as Subscriber
	_ = micro.RegisterSubscriber(recorderTopic, service.Server(), transactionSubscriber)

	// register subscriber with queue, each message is delivered to a unique subscriber
	// _ = micro.RegisterSubscriber(recorderTopic, service.Server(), transactionSubscriber, server.SubscriberQueue("queue.pubsub"))

	transactionHandler := ctn.Resolve("transaction-handler").(recorderPB.TransactionHandler)
	recorderPB.RegisterTransactionHandler(service.Server(), transactionHandler)

	myConfig.PrintBuildInfo()
	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}