package gserror

// todo: think if we want to keep here the stack trace or anything
// GSError has an error if the code returned an error, and the message to display to the user, for that error
// if the error reaches the end user; if Err == nil it means that there are no code error, but if HumanErrorMessage
// is populated it means that there is a logical error (e.g. the user who wanted to do something doesn;t exist)
type GSError struct {
	Err                         error
	HumanErrorMessage           string
	TranslatedHumanErrorMessage string
}

var (
	NoError = GSError{nil, "", ""}
)

// Error returns the golang error if exists, otherwise return the translated message
func (err *GSError) Error() string {
	if err.Err != nil {
		return err.Err.Error()
	} else {
		return err.TranslatedHumanErrorMessage
	}
}

func NewGSError(err error, humanErrorMessage string, translatedHumanErrorMessage string) GSError {
	return GSError{
		Err:                         err,
		HumanErrorMessage:           humanErrorMessage,
		TranslatedHumanErrorMessage: translatedHumanErrorMessage,
	}
}

// NewInternalGSError creates an error that doesn't end up being shown to the user, and translation is not needed
func NewInternalGSError(err error) GSError {
	return GSError{
		Err:                         err,
		HumanErrorMessage:           err.Error(),
		TranslatedHumanErrorMessage: "",
	}
}

// IsFunctional returns true if there is an Error wrapped in the GSError instance
// or false if the error is only a logical one
func(err *GSError) IsFunctional() bool {
	return err.Err != nil
}
