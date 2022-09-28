package enums

import "strings"

type AuthType string

const (
	AuthNone             = "none"
	AuthBasic            = "basic"
	AuthXApiKey          = "x-api-key"
	AuthSecretManagerKey = "secret_manager_key"
	AuthBearer           = "bearer"
	AuthBearerSecret     = "bearer_secret"
)

func AuthTypeList() []string {
	return []string{AuthNone, AuthBasic, AuthXApiKey, AuthSecretManagerKey, AuthBearer, AuthBearerSecret}
}

func (a AuthType) String() string {
	l := AuthTypeList()
	x := strings.ToLower(string(a))
	for _, v := range l {
		if v == x {
			return x
		}
	}
	return ""
}
