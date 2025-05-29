package grpc

import (
	"context"
	"errors"
	"fmt"

	"cs2-marketplace-microservices/user-service/internal/models"
	"cs2-marketplace-microservices/user-service/internal/usecase"
	"cs2-marketplace-microservices/user-service/proto/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserHandler struct {
	user.UnimplementedUserServiceServer
	userUC usecase.UserUseCase
}

func NewUserHandler(userUC usecase.UserUseCase) *UserHandler {
	return &UserHandler{
		userUC: userUC,
	}
}

func (h *UserHandler) RegisterUser(ctx context.Context, req *user.RegisterRequest) (*user.RegisterResponse, error) {
	userModel, token, err := h.userUC.Register(ctx, req.GetUsername(), req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &user.RegisterResponse{
		User:         userModel.ToProto(),
		SessionToken: token,
	}, nil
}

func (h *UserHandler) LoginUser(ctx context.Context, req *user.LoginRequest) (*user.LoginResponse, error) {
	userModel, token, err := h.userUC.Login(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	return &user.LoginResponse{
		User:         userModel.ToProto(),
		SessionToken: token,
	}, nil
}

func (h *UserHandler) LogoutUser(ctx context.Context, req *user.LogoutRequest) (*user.LogoutResponse, error) {
	err := h.userUC.Logout(ctx, req.GetSessionToken())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to logout")
	}

	return &user.LogoutResponse{
		Success: true,
	}, nil
}

func (h *UserHandler) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.GetUserResponse, error) {
	userModel, err := h.userUC.GetUserProfile(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &user.GetUserResponse{
		User: userModel.ToProto(),
	}, nil
}

func (h *UserHandler) UpdateUser(ctx context.Context, req *user.UpdateUserRequest) (*user.UpdateUserResponse, error) {
	userModel, err := h.userUC.UpdateUserProfile(ctx, req.GetUserId(), req.GetUsername(), req.GetEmail())
	if err != nil {
		if errors.Is(err, usecase.ErrEmailExists) || errors.Is(err, usecase.ErrUsernameExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &user.UpdateUserResponse{
		User: userModel.ToProto(),
	}, nil
}

func (h *UserHandler) DeleteUser(ctx context.Context, req *user.DeleteUserRequest) (*user.DeleteUserResponse, error) {
	err := h.userUC.DeleteUser(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete user")
	}

	return &user.DeleteUserResponse{
		Success: true,
	}, nil
}

func (h *UserHandler) ForgotPassword(ctx context.Context, req *user.ForgotPasswordRequest) (*user.ForgotPasswordResponse, error) {
	err := h.userUC.ForgotPassword(ctx, req.GetEmail())
	if err != nil {
		return nil, status.Error(codes.Internal, "password reset failed")
	}

	return &user.ForgotPasswordResponse{
		Success: true,
	}, nil
}

func (h *UserHandler) ResetPassword(ctx context.Context, req *user.ResetPasswordRequest) (*user.ResetPasswordResponse, error) {
	err := h.userUC.ResetPassword(ctx, req.GetToken(), req.GetNewPassword())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid or expired token")
	}

	return &user.ResetPasswordResponse{
		Success: true,
	}, nil
}

func (h *UserHandler) ChangePassword(ctx context.Context, req *user.ChangePasswordRequest) (*user.ChangePasswordResponse, error) {
	err := h.userUC.ChangePassword(ctx, req.GetUserId(), req.GetCurrentPassword(), req.GetNewPassword())
	if err != nil {
		if errors.Is(err, usecase.ErrWrongPassword) {
			return nil, status.Error(codes.PermissionDenied, "current password is incorrect")
		}
		return nil, status.Error(codes.Internal, "failed to change password")
	}

	return &user.ChangePasswordResponse{
		Success: true,
	}, nil
}

func (h *UserHandler) GetBalance(ctx context.Context, req *user.GetBalanceRequest) (*user.GetBalanceResponse, error) {
	balance, err := h.userUC.GetBalance(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &user.GetBalanceResponse{
		Balance: balance,
	}, nil
}

func (h *UserHandler) UpdateBalance(ctx context.Context, req *user.UpdateBalanceRequest) (*user.UpdateBalanceResponse, error) {
	var amount float64
	switch req.GetOperation() {
	case "add":
		amount = req.GetAmount()
	case "subtract":
		amount = -req.GetAmount()
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid operation")
	}

	err := h.userUC.UpdateBalance(ctx, req.GetUserId(), amount)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update balance")
	}

	// Get updated balance
	balance, err := h.userUC.GetBalance(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get updated balance")
	}

	return &user.UpdateBalanceResponse{
		NewBalance: balance,
	}, nil
}

func (h *UserHandler) TransferBalance(ctx context.Context, req *user.TransferBalanceRequest) (*user.TransferBalanceResponse, error) {
	err := h.userUC.TransferBalance(ctx, req.GetFromUserId(), req.GetToUserId(), req.GetAmount())
	if err != nil {
		if errors.Is(err, usecase.ErrInsufficientBalance) {
			return nil, status.Error(codes.FailedPrecondition, "insufficient balance")
		}
		return nil, status.Error(codes.Internal, "transfer failed")
	}

	return &user.TransferBalanceResponse{
		Success: true,
		Message: "transfer completed successfully",
	}, nil
}

func (h *UserHandler) AdminGetAllUsers(ctx context.Context, req *user.AdminGetAllUsersRequest) (*user.AdminGetAllUsersResponse, error) {
	// First validate admin privileges
	adminUser, err := h.userUC.ValidateSession(ctx, req.GetAdminToken())
	if err != nil || !adminUser.IsAdmin {
		return nil, status.Error(codes.PermissionDenied, "admin privileges required")
	}

	users, err := h.userUC.AdminGetAllUsers(ctx, int64(req.GetPage()), int64(req.GetLimit()))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get users")
	}

	var protoUsers []*user.User
	for _, u := range users {
		protoUsers = append(protoUsers, u.ToProto())
	}

	return &user.AdminGetAllUsersResponse{
		Users: protoUsers,
		Total: int32(len(protoUsers)),
	}, nil
}

func (h *UserHandler) AdminUpdateUser(ctx context.Context, req *user.AdminUpdateUserRequest) (*user.AdminUpdateUserResponse, error) {
	// First validate admin privileges
	adminUser, err := h.userUC.ValidateSession(ctx, req.GetAdminToken())
	if err != nil || !adminUser.IsAdmin {
		return nil, status.Error(codes.PermissionDenied, "admin privileges required")
	}

	// Convert proto updates to model
	updates, err := models.FromProto(req.GetUpdates())
	fmt.Println("updates: ", updates)
	if err != nil {
		fmt.Println("h1")
		return nil, status.Error(codes.InvalidArgument, "invalid user data")
	}

	updatedUser, err := h.userUC.AdminUpdateUser(ctx, adminUser.ID.Hex(), req.GetUserId(), updates)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	return &user.AdminUpdateUserResponse{
		User: updatedUser.ToProto(),
	}, nil
}
