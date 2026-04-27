package schema

import "time"

// Transport defines the communication mechanism for a spec.
type Transport string

const (
	TransportJSONRPCStdio Transport = "jsonrpc-stdio"
	TransportJSONRPCHTTP  Transport = "jsonrpc-http"
	TransportHTTPREST     Transport = "http-rest"
	TransportGRPC         Transport = "grpc"
)

// Spec defines a protocol or API contract to test against.
type Spec struct {
	ID          string     `yaml:"id" json:"id"`
	Name        string     `yaml:"name" json:"name"`
	Version     string     `yaml:"version" json:"version"`
	Description string     `yaml:"description" json:"description"`
	SourceURL   string     `yaml:"source_url" json:"source_url"`
	Transport   Transport  `yaml:"transport" json:"transport"`
	TestCases   []TestCase `yaml:"test_cases" json:"test_cases"`
}

// Validate checks that a Spec has all required fields and valid values.
func (s *Spec) Validate() []error {
	var errs []error

	if s.ID == "" {
		errs = append(errs, &ValidationError{Field: "id", Message: "must not be empty"})
	}
	if s.Name == "" {
		errs = append(errs, &ValidationError{Field: "name", Message: "must not be empty"})
	}
	if s.Version == "" {
		errs = append(errs, &ValidationError{Field: "version", Message: "must not be empty"})
	}
	if s.Transport == "" {
		errs = append(errs, &ValidationError{Field: "transport", Message: "must not be empty"})
	}

	validTransports := map[Transport]bool{
		TransportJSONRPCStdio: true,
		TransportJSONRPCHTTP:  true,
		TransportHTTPREST:     true,
		TransportGRPC:         true,
	}
	if s.Transport != "" && !validTransports[s.Transport] {
		errs = append(errs, &ValidationError{
			Field:   "transport",
			Message: "must be one of: jsonrpc-stdio, jsonrpc-http, http-rest, grpc",
		})
	}

	if len(s.TestCases) == 0 {
		errs = append(errs, &ValidationError{Field: "test_cases", Message: "must have at least one test case"})
	}

	seen := make(map[string]bool)
	for i := range s.TestCases {
		tc := &s.TestCases[i]
		if seen[tc.ID] {
			errs = append(errs, &ValidationError{
				Field:   "test_cases",
				Message: "duplicate test case id: " + tc.ID,
			})
		}
		seen[tc.ID] = true
		for _, e := range tc.Validate() {
			errs = append(errs, e)
		}
	}

	return errs
}

// LoadedAt records when a spec was loaded into the store.
type SpecRecord struct {
	Spec
	LoadedAt time.Time `json:"loaded_at"`
}
