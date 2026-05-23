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
