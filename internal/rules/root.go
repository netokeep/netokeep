package rules

import "netokeep/internal/local"

func LoadServerMatcher() (*RuleMatcher, error) {
	nksConfig, err := local.LoadNksConfig()
	if err != nil {
		return nil, err
	}
	rules := nksConfig.Rules
	denyList := rules.DenyList
	allowList := rules.AllowList
	defaultAllow := rules.Default != "deny"

	return NewRuleMatcher(defaultAllow, allowList, denyList), nil
}

func LoadClientRules() ([]string, string, string, uint16, error) {
	nkConfig, err := local.LoadNkConfig()
	if err != nil {
		return nil, "", "", 0, err
	}
	proxyType := nkConfig.Proxy.Type
	proxyAddr := nkConfig.Proxy.Addr
	proxyPort := nkConfig.Proxy.Port
	allowList := nkConfig.Proxy.AllowList

	return allowList, proxyType, proxyAddr, uint16(proxyPort), nil
}
