package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

func metadataFromContext(ctx context.Context) (metadata.MD, bool) {
	return metadata.FromIncomingContext(ctx)
}

func tsFrom(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

func errUnauthenticated(msg string) error {
	return status.Error(codes.Unauthenticated, msg)
}

func errInvalidArgument(format string, args ...any) error {
	return status.Errorf(codes.InvalidArgument, format, args...)
}

func errNotFound(format string, args ...any) error {
	return status.Errorf(codes.NotFound, format, args...)
}

func errAlreadyExists(format string, args ...any) error {
	return status.Errorf(codes.AlreadyExists, format, args...)
}

func errPermissionDenied(format string, args ...any) error {
	return status.Errorf(codes.PermissionDenied, format, args...)
}

func errFailedPrecondition(format string, args ...any) error {
	return status.Errorf(codes.FailedPrecondition, format, args...)
}

func errInternal(format string, args ...any) error {
	return status.Errorf(codes.Internal, format, args...)
}

func cloneAgent(a *arpv1.AgentInstance) *arpv1.AgentInstance {
	if a == nil {
		return nil
	}
	return proto.Clone(a).(*arpv1.AgentInstance)
}

func cloneWorkspace(w *arpv1.Workspace) *arpv1.Workspace {
	if w == nil {
		return nil
	}
	return proto.Clone(w).(*arpv1.Workspace)
}

func cloneProject(p *arpv1.Project) *arpv1.Project {
	if p == nil {
		return nil
	}
	return proto.Clone(p).(*arpv1.Project)
}

// cloneMap deep-copies a JSON-like map[string]any structure.
func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = cloneValue(v)
	}
	return out
}

func cloneValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return cloneMap(t)
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = cloneValue(e)
		}
		return out
	default:
		return v
	}
}
