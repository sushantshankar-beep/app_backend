package validation

import "regexp"

var (
	ifscRegex      = regexp.MustCompile(`^[A-Z]{4}0[A-Z0-9]{6}$`)
	gstRegex       = regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z]{1}[1-9A-Z]{1}Z[0-9A-Z]{1}$`)
	upiRegex       = regexp.MustCompile(`^[a-zA-Z0-9.\-_]{2,256}@[a-zA-Z]{2,64}$`)
	accountRegex  = regexp.MustCompile(`^[0-9]{9,18}$`)
)

func IsValidIFSC(ifsc string) bool {
	return ifscRegex.MatchString(ifsc)
}

func IsValidGST(gst string) bool {
	return gstRegex.MatchString(gst)
}

func IsValidUPI(upi string) bool {
	return upiRegex.MatchString(upi)
}

func IsValidAccountNumber(acc string) bool {
	return accountRegex.MatchString(acc)
}
