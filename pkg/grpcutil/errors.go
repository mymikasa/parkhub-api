package grpcutil

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorMapping maps a domain error to a gRPC status code.
type ErrorMapping struct {
	Target error
	Code   codes.Code
}

// ToGRPCError converts a domain error to a gRPC status error using the provided mappings.
// Falls back to codes.Internal if no mapping matches.
func ToGRPCError(err error, mappings []ErrorMapping) error {
	if err == nil {
		return nil
	}
	for _, m := range mappings {
		if errors.Is(err, m.Target) {
			return status.Error(m.Code, err.Error())
		}
	}
	return status.Error(codes.Internal, err.Error())
}
