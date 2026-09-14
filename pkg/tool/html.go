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
	inSectionTitle := false
	var firstTitle string
	for {
		token := tokenizer.Next()
		switch token {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err == io.EOF {
				if firstTitle != "" {
					return firstTitle, nil
				}
				fmt.Println(document)
				return "", fmt.Errorf("title not found")
			} else {
				return "", err
			}
		case html.StartTagToken:
			tag := tokenizer.Token()
			if tag.Data == "h1" {
				inTitle = true
				inSectionTitle = hasClass(tag, "section-title")
			}
		case html.TextToken:
			if inTitle {
				tag := tokenizer.Token()
				if firstTitle == "" {
					firstTitle = tag.Data
				}
				if inSectionTitle {
					return tag.Data, nil
				}
			}
		case html.EndTagToken:
			tag := tokenizer.Token()
			if tag.Data == "h1" {
				inTitle = false
				inSectionTitle = false
			}
		}
	}
}

func hasClass(tag html.Token, class string) bool {
	for _, attr := range tag.Attr {
		if attr.Key == "class" {
			for _, field := range strings.Fields(attr.Val) {
				if field == class {
					return true
				}
			}
		}
	}
	return false
}

// func ParseHtml(document string) (*html.Node, error) {
// 	reader := strings.NewReader(document)
// 	node, err := html.Parse(reader)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return node, nil
// }
