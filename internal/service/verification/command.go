package verification

type ResendVerificationCmd struct {
	Email string
}

type VerifyEmailCmd struct {
	Token string
}
