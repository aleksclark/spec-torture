package server

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"

	a2av1 "github.com/aleksclark/spec-torture/gen/lf/a2a/v1"
)

// a2aUnmarshal converts an A2A v1.0 HTTP+JSON AgentCard (received as a decoded
// map from the agent) into a typed protobuf message. DiscardUnknown tolerates
// fields the reference proto doesn't model; field names round-trip because
// protojson accepts both the JSON (camelCase) and proto (snake_case) names.
var a2aUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}

func mapToAgentCard(m map[string]any) (*a2av1.AgentCard, error) {
	card := &a2av1.AgentCard{}
	if len(m) == 0 {
		return card, nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	if err := a2aUnmarshal.Unmarshal(raw, card); err != nil {
		return nil, err
	}
	return card, nil
}
