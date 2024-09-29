package types

type FlashcardFont struct {
	Name string `yaml:"name"`
	Size uint32 `yaml:"size"`
}

type FlashcardLatex struct {
	Prefix  string `yaml:"prefix"`
	Postfix string `yaml:"postfix"`
}

type FlashcardStyle struct {
	CSS   string         `yaml:"css"`
	Latex FlashcardLatex `yaml:"latex"`
}

type FlashcardTemplate struct {
	Name string `yaml:"name"`
	QFmt string `yaml:"qfmt"`
	AFmt string `yaml:"afmt"`
}

type FlashcardField struct {
	Name        string        `yaml:"name"`
	Template    string        `yaml:"template"`
	Format      string        `yaml:"format"`
	Index       bool          `yaml:"index"`
	RTL         bool          `yaml:"rtl"`
	Font        FlashcardFont `yaml:"font"`
	Description string        `yaml:"description"`
}

type FlashcardData struct {
	Filename string   `yaml:"filename"`
	Tags     []string `yaml:"tags"`
}

type FlashcardDeck struct {
	Identifier int64  `yaml:"identifier"`
	Name       string `yaml:"name"`
}

type FlashcardModel struct {
	Identifier int64               `yaml:"identifier"`
	Name       string              `yaml:"name"`
	Kind       string              `yaml:"kind"`
	Style      FlashcardStyle      `yaml:"style"`
	Templates  []FlashcardTemplate `yaml:"templates"`
	Fields     []FlashcardField    `yaml:"fields"`
}

type FlashcardProject struct {
	Filename string          `yaml:"filename"`
	Deck     FlashcardDeck   `yaml:"deck"`
	Model    FlashcardModel  `yaml:"model"`
	Data     []FlashcardData `yaml:"data"`
}
