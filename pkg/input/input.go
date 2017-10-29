package input

import (
	"fmt"
	"strings"

	"github.com/howeyc/gopass"
)

func Readln(prompt string) (line string, err error) {
	fmt.Print(prompt)
	_, err = fmt.Scanln(&line)
	err = filterEmptyError(err)
	return
}

func GetPasswd(prompt string) (passwd string, err error) {
	fmt.Print(prompt)
	password, err := gopass.GetPasswd()
	if err != nil {
		err = filterEmptyError(err)
		return "", err
	}
	return string(password), nil
}

func filterEmptyError(err error) error {
	if err == nil || strings.Contains(err.Error(), "unexpected newline") {
		return nil
	}
	return err
}
