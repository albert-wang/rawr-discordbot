package handlers

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func AttachmentIsImage(attachment *discordgo.MessageAttachment) bool {
	return strings.HasPrefix(attachment.ContentType, "image/")
}

func AttachmentExtension(attachment *discordgo.MessageAttachment) string {
	name := attachment.Filename
	return path.Ext(name)
}

func DownloadAttachment(attach *discordgo.MessageAttachment) ([]byte, error) {
	resp, err := http.Get(attach.URL)
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

// Some Image utilities
func ConvertAttachmentsToDataURL(attachments []*discordgo.MessageAttachment, maxWidth int, maxHeight int) []string {
	result := []string{}
	for _, attachment := range attachments {
		if !AttachmentIsImage(attachment) {
			continue
		}

		b, err := DownloadAttachment(attachment)
		if err != nil {
			log.Print(err)
			continue
		}

		out, err := ConvertImage(b, AttachmentExtension(attachment),
			"-resize",
			fmt.Sprintf("%dx%d>", maxWidth, maxHeight),
		)

		if err != nil {
			log.Print(err)
			continue
		}

		defer os.Remove(out)
		output, err := os.Open(out)
		if err != nil {
			log.Print(err)
			continue
		}

		defer output.Close()
		newBytes, err := io.ReadAll(output)
		if err != nil {
			log.Print(err)
			continue
		}

		bs := base64.StdEncoding.EncodeToString(newBytes)
		log.Print(attachment.ContentType)
		result = append(result, fmt.Sprintf("data:%s;base64,%s", attachment.ContentType, bs))
	}

	return result
}
