package main

import (
	"github.com/micro/go-micro"
	log "github.com/sirupsen/logrus"
	"github.com/xmlking/micro-starter-kit/api/account/handler"

	accountPB "github.com/xmlking/micro-starter-kit/api/account/proto/account"
	userPB "github.com/xmlking/micro-starter-kit/srv/account/proto/account"
)

func main() {

	// New Service
	service := micro.NewService(
		micro.Name("go.micro.api.account"),
		micro.Version("latest"),
	)

	userSrvClient := userPB.NewUserService("go.micro.srv.account", service.Client())
	profSrvClient := userPB.NewProfileService("go.micro.srv.account", service.Client()) // service.Client() or client.DefaultClient???
	accountHandler := handler.NewAccountHandler(userSrvClient, profSrvClient)

	// Initialise service
	service.Init()

	// Register Handler
	accountPB.RegisterAccountServiceHandler(service.Server(), accountHandler)
	// service.Server().Handle(service.Server().NewHandler(accountHandler))

	// Run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}

}
