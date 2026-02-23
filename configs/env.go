package configs

type Env string

func (e Env) IsProd() bool {
	return e == "production"
}

func (e Env) IsStaging() bool {
	return e == "staging"
}

func (e Env) IsDev() bool {
	return e == "development"
}

func (e Env) IsLocal() bool {
	return e == "local" || e == ""
}

func (e Env) String() string {
	return string(e)
}
