package handler

import (
	"context"

	"github.com/jinzhu/gorm"
	"github.com/micro/go-micro"
	"github.com/micro/go-micro/errors"
	log "github.com/sirupsen/logrus"
	"github.com/thoas/go-funk"

	myErrors "github.com/xmlking/micro-starter-kit/shared/errors"

	pb "github.com/xmlking/micro-starter-kit/srv/account/proto/account"
	"github.com/xmlking/micro-starter-kit/srv/account/repository"
	emailerPB "github.com/xmlking/micro-starter-kit/srv/emailer/proto/emailer"
	greeterPB "github.com/xmlking/micro-starter-kit/srv/greeter/proto/greeter"
)

// UserHandler struct
type userHandler struct {
	userRepository   repository.UserRepository
	Publisher        micro.Publisher
	greeterSrvClient greeterPB.GreeterService
}

// NewUserHandler returns an instance of `UserServiceHandler`.
func NewUserHandler(repo repository.UserRepository, pub micro.Publisher, greeterClient greeterPB.GreeterService) pb.UserServiceHandler {
	return &userHandler{
		userRepository:   repo,
		Publisher:        pub,
		greeterSrvClient: greeterClient,
	}
}

func (h *userHandler) Exist(ctx context.Context, req *pb.UserRequest, rsp *pb.UserExistResponse) error {
	log.Info("Received UserHandler.Exist request")
	model := pb.UserORM{}
	model.Id = req.Id.GetValue()
	model.Username = req.Username.GetValue()
	model.Email = req.Email.GetValue()

	exists := h.userRepository.Exist(&model)
	log.Infof("user exists? %t", exists)
	rsp.Result = exists
	return nil
}

func (h *userHandler) List(ctx context.Context, req *pb.UserListQuery, rsp *pb.UserListResponse) error {
	log.Info("Received UserHandler.List request")
	model := pb.UserORM{}
	model.Username = req.Username.GetValue()
	model.FirstName = req.FirstName.GetValue()
	model.Email = req.Email.GetValue()

	total, users, err := h.userRepository.List(req.Limit.GetValue(), req.Page.GetValue(), req.Sort.GetValue(), &model)
	if err != nil {
		return errors.NotFound("account-srv.user.list", "Error %v", err.Error())
	}
	rsp.Total = total

	// newUsers := make([]*pb.User, len(users))
	// for index, user := range users {
	// 	tmpUser, _ := user.ToPB(ctx)
	// 	newUsers[index] = &tmpUser
	// 	// *newUsers[index], _ = user.ToPB(ctx) ???
	// }
	newUsers := funk.Map(users, func(user *pb.UserORM) *pb.User {
		tmpUser, _ := user.ToPB(ctx)
		return &tmpUser
	}).([]*pb.User)

	rsp.Results = newUsers
	return nil
}

func (h *userHandler) Get(ctx context.Context, req *pb.UserRequest, rsp *pb.UserResponse) error {
	log.Info("Received UserHandler.Get request")
	id := req.Id.GetValue()
	if id == "" {
		return myErrors.ValidationError("account-srv.user.get", "validation error: Missing Id")
	}
	user, err := h.userRepository.Get(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			rsp.Result = nil
			return nil
		}
		return myErrors.AppError(myErrors.DBE, err)
	}

	tempUser, _ := user.ToPB(ctx)
	rsp.Result = &tempUser

	return nil
}

func (h *userHandler) Create(ctx context.Context, req *pb.UserRequest, rsp *pb.UserResponse) error {
	log.Info("Received UserHandler.Create request")

	model := pb.UserORM{}
	model.Username = req.Username.GetValue()
	model.FirstName = req.FirstName.GetValue()
	model.LastName = req.LastName.GetValue()
	model.Email = req.Email.GetValue()

	if err := h.userRepository.Create(&model); err != nil {
		return myErrors.AppError(myErrors.DBE, err)
	}

	// send email
	if err := h.Publisher.Publish(ctx, &emailerPB.Message{To: req.Email.GetValue()}); err != nil {
		log.WithError(err).Error("Received Publisher.Publish request error")
		return myErrors.AppError(myErrors.PSE, err)
	}

	// call greeter
	// if res, err := h.greeterSrvClient.Hello(ctx, &greeterPB.Request{Name: req.GetFirstName().GetValue()}); err != nil {
	if res, err := h.greeterSrvClient.Hello(ctx, &greeterPB.HelloRequest{Name: req.GetFirstName().GetValue()}); err != nil {
		log.WithError(err).Error("Received greeterService.Hello request error")
		return myErrors.AppError(myErrors.PSE, err)
	} else {
		log.Infof("Got greeterService responce %s", res.Msg)
	}

	return nil
}

func (h *userHandler) Update(ctx context.Context, req *pb.UserRequest, rsp *pb.UserResponse) error {
	log.Info("Received UserHandler.Update request")

	id := req.Id.GetValue()
	if id == "" {
		return myErrors.ValidationError("account-srv.user.update", "validation error: Missing Id")
	}

	model := pb.UserORM{}
	model.Username = req.Username.GetValue()
	model.FirstName = req.FirstName.GetValue()
	model.LastName = req.LastName.GetValue()
	model.Email = req.Email.GetValue()

	if err := h.userRepository.Update(id, &model); err != nil {
		return myErrors.AppError(myErrors.DBE, err)
	}

	return nil
}

func (h *userHandler) Delete(ctx context.Context, req *pb.UserRequest, rsp *pb.UserResponse) error {
	log.Info("Received UserHandler.Delete request")

	id := req.Id.GetValue()
	if id == "" {
		return myErrors.ValidationError("account-srv.user.update", "validation error: Missing Id")
	}

	model := pb.UserORM{}
	model.Id = id

	if err := h.userRepository.Delete(&model); err != nil {
		return myErrors.AppError(myErrors.DBE, err)
	}

	return nil
}
