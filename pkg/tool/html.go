package tool

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

func GetHtmlTitle(document string) (string, error) {
	reader := strings.NewReader(document)
	tokenizer := html.NewTokenizer(reader)
	inTitle := false
	for {
		token := tokenizer.Next()
		switch token {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err == io.EOF {
				fmt.Println(document)
				return "", fmt.Errorf("title not found")
			} else {
				return "", err
			}
		case html.StartTagToken:
			tag := tokenizer.Token()
			if tag.Data == "h1" {
				inTitle = true
			}
		case html.TextToken:
			if inTitle {
				tag := tokenizer.Token()
				return tag.Data, nil
			}
		case html.EndTagToken:
			tag := tokenizer.Token()
			if tag.Data == "h1" {
				inTitle = false
			}
		}
	}
}

// func ParseHtml(document string) (*html.Node, error) {
// 	reader := strings.NewReader(document)
// 	node, err := html.Parse(reader)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return node, nil
// }
