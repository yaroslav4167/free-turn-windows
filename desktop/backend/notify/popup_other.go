//go:build !windows && !linux

package notify

func showCustomCaptchaPopup(captchaURL string) error {
	return errUnsupported
}
