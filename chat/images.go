package chat

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func GetURLBytes(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Non-200 return code: %d", resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	return bytes, err
}

// Runs imagemagick convert, in the form `convert <input> [arguments] blah`
func ConvertImage(in []byte, ext string, arguments ...string) (string, error) {
	f, err := os.CreateTemp(os.TempDir(), "file-")
	if err != nil {
		return "", err
	}

	_, err = f.Write(in)
	if err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}

	f.Close()
	defer os.Remove(f.Name())

	output := fmt.Sprintf("%s%s", f.Name(), ext)

	args := []string{f.Name()}
	args = append(args, arguments...)
	args = append(args, output)

	cmd := exec.Command("convert", args...)
	log.Print("convert ", strings.Join(args, " "))

	err = cmd.Run()
	if err != nil {
		errstr, _ := cmd.CombinedOutput()
		log.Print(err)
		log.Print(string(errstr))
		return "", err
	}

	return output, nil
}
