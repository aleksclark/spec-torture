package server

import (
	"context"
	"time"

	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// CreateToken issues a new bearer token. Requires admin permission. The issued
// token's scope must be no wider, and permission no higher, than the caller's.
func (s *Server) CreateToken(ctx context.Context, req *arpv1.CreateTokenRequest) (*arpv1.CreateTokenResponse, error) {
	p, err := s.authenticate(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetSubject() == "" {
		return nil, errInvalidArgument("subject is required")
	}
	if req.GetScope() == nil {
		return nil, errInvalidArgument("scope is required")
	}
	if req.GetPermission() == arpv1.Permission_PERMISSION_UNSPECIFIED {
		return nil, errInvalidArgument("permission is required")
	}
	if p.perm != arpv1.Permission_PERMISSION_ADMIN {
		return nil, errPermissionDenied("CreateToken requires admin permission")
	}
	ttl := time.Duration(req.GetExpiresInSeconds()) * time.Second

	s.mu.Lock()
	tok, err := s.issueToken(p, req.GetScope(), req.GetPermission(), req.GetSubject(), "", ttl)
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &arpv1.CreateTokenResponse{
		Token:       tok.toProto(),
		BearerToken: tok.bearer,
	}, nil
}
