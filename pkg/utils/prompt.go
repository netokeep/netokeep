package utils

import (
	"github.com/AlecAivazis/survey/v2"
)

/*
PromptConfirm displays a yes/no confirmation prompt to the user.

Returns true if the user selects 'yes', and false otherwise.
*/
func PromptConfirm(promptMsg string, defaultVal bool) bool {
	var confirm bool
	prompt := &survey.Confirm{
		Message: promptMsg,
		Default: defaultVal,
	}
	err := survey.AskOne(prompt, &confirm)
	if err != nil {
		return false
	}
	return confirm
}
