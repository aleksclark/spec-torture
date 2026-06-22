package server

import (
	"time"

	arpv1 "github.com/aleksclark/spec-torture/gen/arp/v1"
)

// scopeSubset reports whether child is a subset of (i.e. no wider than) parent.
func scopeSubset(child, parent *arpv1.Scope) bool {
	if parent == nil || parent.GetGlobal() {
		return true
	}
	if child == nil {
		// Inheriting parent scope is always allowed.
		return true
	}
	if child.GetGlobal() {
		return false // widening to global
	}
	parentSet := map[string]bool{}
	for _, p := range parent.GetProjects() {
		parentSet[p] = true
	}
	for _, c := range child.GetProjects() {
		if !parentSet[c] {
			return false
		}
	}
	return true
}

func cloneScope(s *arpv1.Scope) *arpv1.Scope {
	if s == nil {
		return nil
	}
	return &arpv1.Scope{Global: s.GetGlobal(), Projects: append([]string(nil), s.GetProjects()...)}
}

// issueToken creates and stores a token derived from a parent principal,
// enforcing scope-narrowing and permission-lowering. Caller must hold s.mu.
// A nil parent means a root/admin issuer (no constraints).
func (s *Server) issueToken(parent *principal, reqScope *arpv1.Scope, reqPerm arpv1.Permission, subject, sessionID string, ttl time.Duration) (*tokenEntry, error) {
	var parentScope *arpv1.Scope
	parentPerm := arpv1.Permission_PERMISSION_ADMIN
	parentID := ""
	if parent != nil {
		parentScope = parent.scope
		parentPerm = parent.perm
		if parent.token != nil {
			parentID = parent.token.id
		}
	}

	scope := reqScope
	if scope == nil {
		scope = cloneScope(parentScope)
		if scope == nil {
			scope = &arpv1.Scope{Global: true}
		}
	} else if !scopeSubset(scope, parentScope) {
		return nil, errPermissionDenied("requested scope is wider than caller scope")
	} else {
		scope = cloneScope(scope)
	}

	perm := reqPerm
	if perm == arpv1.Permission_PERMISSION_UNSPECIFIED {
		perm = parentPerm
	} else if perm > parentPerm {
		return nil, errPermissionDenied("requested permission exceeds caller permission")
	}

	tok := &tokenEntry{
		id:            newID("tok"),
		subject:       subject,
		scope:         scope,
		perm:          perm,
		sessionID:     sessionID,
		parentTokenID: parentID,
		bearer:        randomToken(),
		issuedAt:      time.Now().UTC(),
	}
	if ttl > 0 {
		tok.expiresAt = tok.issuedAt.Add(ttl)
	}
	s.tokens[tok.id] = tok
	s.byBearer[tok.bearer] = tok
	return tok, nil
}

func (t *tokenEntry) toProto() *arpv1.Token {
	pb := &arpv1.Token{
		Id:            t.id,
		Subject:       t.subject,
		Scope:         cloneScope(t.scope),
		Permission:    t.perm,
		SessionId:     t.sessionID,
		ParentTokenId: t.parentTokenID,
	}
	if !t.issuedAt.IsZero() {
		pb.IssuedAt = tsFrom(t.issuedAt)
	}
	if !t.expiresAt.IsZero() {
		pb.ExpiresAt = tsFrom(t.expiresAt)
	}
	return pb
}
