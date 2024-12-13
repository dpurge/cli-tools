package ebook

import (
	"bufio"
	"os"
	"strings"
)

type VocabularyRecord struct {
	Phrase        string
	Grammar       string
	Transcription string
	Translation   string
	Notes         string
}

func buildVocabulary(projectfile string) (string, error) {
	project, err := readProject(projectfile)
	if err != nil {
		return "", err
	}

	outfile := project.Filename
	if strings.HasSuffix(outfile, ".epub") {
		outfile = strings.TrimSuffix(outfile, ".epub") + ".csv"
	}

	f, err := os.Create(outfile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	w.WriteString("Phrase\tGrammar\tTranscription\tTranslation\tNotes\n")

	for _, items := range project.Text {
		for _, filename := range items {
			lines, err := getVocabulary(filename)
			if err != nil {
				return "", err
			}

			w.WriteString("\n# " + filename + "\n")

			for _, line := range lines {
				record := parseRecord(line)
				w.WriteString(
					record.Phrase + "\t" +
						record.Grammar + "\t" +
						record.Transcription + "\t" +
						record.Translation + "\t" +
						record.Notes + "\n")
			}
		}
	}

	return outfile, nil
}

func getVocabulary(filename string) ([]string, error) {
	readFile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	var lines []string
	inVocabulary := false
	for fileScanner.Scan() {
		line := strings.TrimSpace(fileScanner.Text())
		if line == "" {
			continue
		}

		if line == "{end-vocabulary}" {
			inVocabulary = false
		}

		if inVocabulary {
			lines = append(lines, line)
		}

		if line == "{start-vocabulary}" {
			inVocabulary = true
		}
	}

	readFile.Close()

	return lines, nil
}

func parseRecord(line string) VocabularyRecord {
	var record VocabularyRecord

	i := strings.LastIndex(line, "=")
	if i != -1 {
		record.Translation = strings.TrimSpace(line[i+1:])
		if record.Translation[len(record.Translation)-1:] == ")" {
			j := strings.LastIndex(record.Translation, "(")
			record.Notes = strings.TrimSpace(record.Translation[j+1 : len(record.Translation)-1])
			record.Translation = strings.TrimSpace(record.Translation[:j])
		}

		line = strings.TrimSpace(line[:i])
	}

	if line[len(line)-1:] == "]" {
		i = strings.LastIndex(line, "[")
		record.Transcription = strings.TrimSpace(line[i+1 : len(line)-1])
		line = strings.TrimSpace(line[:i])
	}

	if line[len(line)-1:] == "}" {
		i = strings.LastIndex(line, "{")
		record.Grammar = strings.TrimSpace(line[i+1 : len(line)-1])
		line = strings.TrimSpace(line[:i])
	}

	record.Phrase = line
	return record
}
