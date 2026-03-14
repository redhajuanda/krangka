package libctx

import (
	"context"
	"errors"
	"strconv"

	"github.com/golang-jwt/jwt"
	"github.com/redhajuanda/komon/fail"
)

type ContextKeys string

type Stringable int

func (s Stringable) String() string {
	return strconv.Itoa(int(s))
}

type JwtClaims struct {
	*jwt.StandardClaims
	Aud         string     `json:"aud"`
	BranchID    Stringable `json:"branchId"`
	BranchName  string     `json:"branchName"`
	Client      string     `json:"client"`
	DeviceID    string     `json:"deviceId"`
	DisplayName string     `json:"displayName"`
	Email       string     `json:"email"`
	EmployeeID  Stringable `json:"employeeId"`
	Exp         int        `json:"exp"`
	Group       string     `json:"group"`
	Groups      []string   `json:"groups"`
	Iat         int        `json:"iat"`
	Iss         string     `json:"iss"`
	Nik         string     `json:"nik"`
	OtpVerified bool       `json:"otpVerified"`
	Policy      string     `json:"policy"`
	RoleID      Stringable `json:"roleId"`
	RoleName    string     `json:"roleName"`
	Roles       []string   `json:"roles"`
	Scopes      string     `json:"scopes"`
	Sub         string     `json:"sub"`
	Type        string     `json:"type"`
	UserID      Stringable `json:"userId"`
	Username    string     `json:"username"`
}

// Valid implements jwt.Claims interface to prevent nil pointer dereference
func (j *JwtClaims) Valid() error {
	// If StandardClaims is nil, initialize it
	if j.StandardClaims == nil {
		j.StandardClaims = &jwt.StandardClaims{}
	}

	// Call the parent Valid method
	return j.StandardClaims.Valid()
}

var (
	BearerKey    = ContextKeys("bearer")
	JwtClaimsKey = ContextKeys("jwt_claims")
	AccountKey   = ContextKeys("account")
)

func SetClaims(ctx context.Context, bearerToken string) (context.Context, error) {

	claims := &JwtClaims{}

	ctx = context.WithValue(ctx, BearerKey, bearerToken)

	token, _ := jwt.ParseWithClaims(bearerToken, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return nil, nil
	})

	if token == nil {
		return nil, fail.New("invalid token").WithFailure(fail.ErrUnauthorized)
	}

	claims, ok := token.Claims.(*JwtClaims)
	if !ok {
		return nil, fail.New("invalid token").WithFailure(fail.ErrUnauthorized)
	}

	ctx = context.WithValue(ctx, JwtClaimsKey, claims)

	return ctx, nil
}

func SetClaimsInternal(ctx context.Context, claims *JwtClaims) context.Context {
	ctx = context.WithValue(ctx, JwtClaimsKey, claims)
	return ctx
}

func SetBearerTokenInternal(ctx context.Context, bearerToken string) context.Context {
	ctx = context.WithValue(ctx, BearerKey, bearerToken)
	return ctx
}

// GetClaims gets the jwt claims from the context
func GetClaims(ctx context.Context) (*JwtClaims, error) {
	claims, ok := ctx.Value(JwtClaimsKey).(*JwtClaims)
	if !ok {
		return nil, errors.New("jwt claims not found in context")
	}
	return claims, nil
}

// GetBearerToken gets the bearer token from the context
func GetBearerToken(ctx context.Context) (string, error) {
	bearerToken, ok := ctx.Value(BearerKey).(string)
	if !ok {
		return "", errors.New("bearer token not found in context")
	}
	return bearerToken, nil
}