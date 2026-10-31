package domain

type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Name         string
	Scopes       []string
	// DisablePassword turns off password login and registration
	// while OIDC is active
	DisablePassword bool
}

func (c OIDCConfig) Complete() bool {
	return c.Issuer != "" && c.ClientID != "" && c.ClientSecret != "" && c.RedirectURL != ""
}
