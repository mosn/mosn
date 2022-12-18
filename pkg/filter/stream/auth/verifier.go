package auth

import (
	"errors"
	"fmt"

	"mosn.io/api"
	pb "mosn.io/mosn/pkg/filter/stream/auth/matchpb"
)

// Verifier supports verification of JWTs with configured requirements.
type Verifier interface {
	Verify(headers api.HeaderMap, requestArg string) bool
}

// NewVerifier creates a npb.
func NewVerifier(require *pb.Requirement, authenticators map[string]Authenticator) (Verifier, error) {

	switch require.RequiresType.(type) {
	case *pb.Requirement_AuthenticatorName:
		return newBaseVerifier(require.GetAuthenticatorName(), authenticators)
	case *pb.Requirement_RequiresAny:
		return newAnyVerifier(require.GetRequiresAny().GetRequirements(), authenticators)
	case *pb.Requirement_RequiresAll:
		return newAllVerifier(require.GetRequiresAll().GetRequirements(), authenticators)
	}

	return nil, errors.New("Not supported Verifier type")
}

type BaseVerifier struct {
	authenticator Authenticator
}

func newBaseVerifier(authenticatorName string, authenticators map[string]Authenticator) (*BaseVerifier, error) {

	if _, exist := authenticators[authenticatorName]; !exist {
		return nil, fmt.Errorf("Not found %s authenticator", authenticatorName)
	}

	return &BaseVerifier{
		authenticator: authenticators[authenticatorName],
	}, nil
}

func (p *BaseVerifier) Verify(headers api.HeaderMap, requestArg string) bool {
	return p.authenticator.Authenticate(headers, requestArg)
}

// Base verifier for requires all or any.
type baseGroupVerifier struct {
	verifiers []Verifier
}

func newBaseGroupVerifier(requires []*pb.Requirement, authenticators map[string]Authenticator) (baseGroupVerifier, error) {
	var verifiers []Verifier
	for _, require := range requires {

		verifier, err := NewVerifier(require, authenticators)
		if err != nil {
			return baseGroupVerifier{}, err
		}
		verifiers = append(verifiers, verifier)
	}

	return baseGroupVerifier{
		verifiers: verifiers,
	}, nil
}

// Requires any verifier.
type anyVerifier struct {
	baseGroupVerifier
}

func newAnyVerifier(requires []*pb.Requirement, authenticators map[string]Authenticator) (anyVerifier, error) {

	baseGroupVerifier, err := newBaseGroupVerifier(requires, authenticators)

	if err != nil {
		return anyVerifier{}, err
	}

	return anyVerifier{
		baseGroupVerifier: baseGroupVerifier,
	}, nil
}

func (a anyVerifier) Verify(headers api.HeaderMap, requestArg string) bool {

	for _, verifier := range a.verifiers {
		accept := verifier.Verify(headers, requestArg)
		if accept {
			return true
		}
	}

	return false
}

// Requires all verifier.
type allVerifier struct {
	baseGroupVerifier
}

func newAllVerifier(requires []*pb.Requirement, authenticators map[string]Authenticator) (allVerifier, error) {
	baseGroupVerifier, err := newBaseGroupVerifier(requires, authenticators)

	if err != nil {
		return allVerifier{}, err
	}

	return allVerifier{
		baseGroupVerifier: baseGroupVerifier,
	}, nil
}

func (a allVerifier) Verify(headers api.HeaderMap, requestArg string) bool {

	for _, verifier := range a.verifiers {
		accept := verifier.Verify(headers, requestArg)
		if !accept {
			return false
		}
	}

	return true
}
