package grpc

import (
	"context"

	"github.com/parkhub/api/internal/domains/identity/domain"
	"github.com/parkhub/api/internal/domains/identity/errs"
	"github.com/parkhub/api/internal/domains/identity/service"
	identityv1 "github.com/parkhub/api/internal/gen/api/proto/identity/v1"
	"github.com/parkhub/api/pkg/grpcutil"
	"google.golang.org/grpc/codes"
)

type AuthGRPCServer struct {
	identityv1.UnimplementedAuthServiceServer
	authSvc service.AuthService
	signer  domain.TokenSigner
}

func NewAuthGRPCServer(svc service.AuthService, signer domain.TokenSigner) *AuthGRPCServer {
	return &AuthGRPCServer{authSvc: svc, signer: signer}
}

var authErrorMappings = []grpcutil.ErrorMapping{
	{Target: errs.ErrInvalidCredentials, Code: codes.Unauthenticated},
	{Target: errs.ErrUserFrozen, Code: codes.Unauthenticated},
	{Target: errs.ErrRefreshTokenInvalid, Code: codes.Unauthenticated},
	{Target: errs.ErrRefreshTokenRevoked, Code: codes.Unauthenticated},
}

func toAuthGRPCError(err error) error {
	return grpcutil.ToGRPCError(err, authErrorMappings)
}

func (s *AuthGRPCServer) Login(ctx context.Context, req *identityv1.LoginRequest) (*identityv1.LoginResponse, error) {
	resp, err := s.authSvc.Login(ctx, &service.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return nil, toAuthGRPCError(err)
	}
	return &identityv1.LoginResponse{
		AccessToken:     resp.TokenPair.AccessToken,
		RefreshToken:    resp.TokenPair.RefreshToken,
		AccessExpiresIn: resp.AccessExpiresIn,
		TokenType:       "Bearer",
		User:            toProtoUser(resp.User),
	}, nil
}

func (s *AuthGRPCServer) RefreshToken(ctx context.Context, req *identityv1.RefreshTokenRequest) (*identityv1.RefreshTokenResponse, error) {
	resp, err := s.authSvc.RefreshToken(ctx, &service.RefreshTokenRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		return nil, toAuthGRPCError(err)
	}
	return &identityv1.RefreshTokenResponse{
		AccessToken:     resp.TokenPair.AccessToken,
		RefreshToken:    resp.TokenPair.RefreshToken,
		AccessExpiresIn: resp.AccessExpiresIn,
		TokenType:       "Bearer",
	}, nil
}

func (s *AuthGRPCServer) Logout(ctx context.Context, req *identityv1.LogoutRequest) (*identityv1.LogoutResponse, error) {
	err := s.authSvc.Logout(ctx, &service.LogoutRequest{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		return nil, toAuthGRPCError(err)
	}
	return &identityv1.LogoutResponse{}, nil
}

func (s *AuthGRPCServer) GetJWKS(ctx context.Context, req *identityv1.GetJWKSRequest) (*identityv1.GetJWKSResponse, error) {
	if s.signer == nil {
		return &identityv1.GetJWKSResponse{}, nil
	}
	jwks, err := s.signer.JWKS()
	if err != nil {
		return nil, grpcutil.ToGRPCError(err, nil)
	}
	return &identityv1.GetJWKSResponse{JwksJson: jwks}, nil
}
