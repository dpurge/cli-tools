package tool

import (
	"archive/zip"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dpurge/cli-tools/pkg/tool/proto/anki"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/protobuf/proto"
)

//  I N T E R F A C E S

type AnkiDatabase interface {
	Open(string) error
	Close() error
	CollectionExec(string) error
	MediaExec(string) error
}

type AnkiPackage interface {
	Open(string) error
	Close() error
	ImportProject(*FlashcardProject) error
}

type AnkiCollection interface {
	Init(db AnkiDatabase) error
	Add(db AnkiDatabase) (int64, error)
}

type AnkiNotes interface {
	Init(db AnkiDatabase) error
	Add(db AnkiDatabase) error
}

type AnkiCards interface {
	Init(AnkiDatabase) error
}

type AnkiRevlog interface {
	Init(AnkiDatabase) error
}

type AnkiDeckConfig interface {
	Init(AnkiDatabase) error
}

type AnkiConfig interface {
	Init(AnkiDatabase) error
}

type AnkiFields interface {
	Init(db AnkiDatabase) error
	Add(db AnkiDatabase, ntid int64, ord int, name string, config []byte) error
}

type AnkiTemplates interface {
	Init(db AnkiDatabase) error
	Add(db AnkiDatabase, ntid int64, ord int, name string, config []byte) error
}

type AnkiNoteTypes interface {
	Init(db AnkiDatabase) error
	Add(db AnkiDatabase, id int64, name string, config []byte) (int64, error)
}

type AnkiDecks interface {
	Init(AnkiDatabase) error
}

type AnkiTags interface {
	Init(AnkiDatabase) error
}

type AnkiGraves interface {
	Init(AnkiDatabase) error
}

type AnkiAndroidMetadata interface {
	Init(AnkiDatabase) error
}

type AnkiMedia interface {
	Init(AnkiDatabase) error
}

type AnkiMeta interface {
	Init(AnkiDatabase) error
}

// ==================================================

//  T Y P E S

type ankiCollection struct {
	schema string
	add    string
}

func (col *ankiCollection) Init(db AnkiDatabase) error {
	var err error

	err = db.CollectionExec(col.schema)
	if err != nil {
		return err
	}

	_, err = col.Add(db)
	if err != nil {
		return err
	}

	return nil
}

func (col *ankiCollection) Add(db AnkiDatabase) (int64, error) {
	var err error
	var id int64 = 1

	now := time.Now()

	err = db.CollectionExec(fmt.Sprintf(col.add, id, now.Unix(), now.UnixMilli(), now.UnixMilli()))
	if err != nil {
		return 0, err
	}

	return id, nil
}

// ==================================================

type ankiNotes struct {
	schema string
	add    string
}

func (nt *ankiNotes) Init(db AnkiDatabase) error {
	return db.CollectionExec(nt.schema)
}

func (nt *ankiNotes) Add(db AnkiDatabase) error {
	var err error
	return err
}

// ==================================================

type ankiCards struct {
	schema string
}

func (cards *ankiCards) Init(db AnkiDatabase) error {
	return db.CollectionExec(cards.schema)
}

// ==================================================

type ankiRevlog struct {
	schema string
}

func (revlog *ankiRevlog) Init(db AnkiDatabase) error {
	return db.CollectionExec(revlog.schema)
}

// ==================================================

type ankiDeckConfig struct {
	schema string
}

func (deckcfg *ankiDeckConfig) Init(db AnkiDatabase) error {
	return db.CollectionExec(deckcfg.schema)
}

// ==================================================

type ankiConfig struct {
	schema string
}

func (cfg *ankiConfig) Init(db AnkiDatabase) error {
	return db.CollectionExec(cfg.schema)
}

// ==================================================

type ankiFields struct {
	schema string
	add    string
}

func (fields *ankiFields) Init(db AnkiDatabase) error {
	return db.CollectionExec(fields.schema)
}

func (fields *ankiFields) Add(db AnkiDatabase, ntid int64, ord int, name string, config []byte) error {
	return db.CollectionExec(fmt.Sprintf(fields.add, ntid, ord, name, hex.EncodeToString(config)))
}

// ==================================================

type ankiTemplates struct {
	schema string
	add    string
}

func (tpl *ankiTemplates) Init(db AnkiDatabase) error {
	return db.CollectionExec(tpl.schema)
}

func (tpl *ankiTemplates) Add(db AnkiDatabase, ntid int64, ord int, name string, config []byte) error {
	var err error

	now := time.Now()

	err = db.CollectionExec(fmt.Sprintf(tpl.add, ntid, ord, name, now.Unix(), hex.EncodeToString(config)))
	if err != nil {
		return err
	}

	return err
}

// ==================================================

type ankiNoteTypes struct {
	schema string
	add    string
}

func (nt *ankiNoteTypes) Init(db AnkiDatabase) error {
	return db.CollectionExec(nt.schema)
}

func (nt *ankiNoteTypes) Add(db AnkiDatabase, id int64, name string, config []byte) (int64, error) {
	var err error

	now := time.Now()

	err = db.CollectionExec(fmt.Sprintf(nt.add, id, name, now.Unix(), 0, hex.EncodeToString(config)))
	if err != nil {
		return 0, err
	}

	return id, nil
}

// ==================================================

type ankiDecks struct {
	schema string
}

func (decks *ankiDecks) Init(db AnkiDatabase) error {
	return db.CollectionExec(decks.schema)
}

// ==================================================

type ankiTags struct {
	schema string
}

func (tags *ankiTags) Init(db AnkiDatabase) error {
	return db.CollectionExec(tags.schema)
}

// ==================================================

type ankiGraves struct {
	schema string
}

func (graves *ankiGraves) Init(db AnkiDatabase) error {
	return db.CollectionExec(graves.schema)
}

// ==================================================

type ankiAndroidMetadata struct {
	schema string
}

func (meta *ankiAndroidMetadata) Init(db AnkiDatabase) error {
	return db.CollectionExec(meta.schema)
}

// ==================================================

type ankiMedia struct {
	schema string
}

func (media *ankiMedia) Init(db AnkiDatabase) error {
	return db.MediaExec(media.schema)
}

// ==================================================

type ankiMeta struct {
	schema string
}

func (meta *ankiMeta) Init(db AnkiDatabase) error {
	return db.MediaExec(meta.schema)
}

// ==================================================

type ankiDatabase struct {
	isOpen     bool
	collection *sql.DB
	media      *sql.DB
}

func (db *ankiDatabase) Open(directory string) error {
	var err error

	if db.isOpen {
		return fmt.Errorf("database is already open in directory: %s", directory)
	}

	db.collection, err = sql.Open("sqlite3", filepath.Join(directory, "collection.anki2"))
	if err != nil {
		return err
	}

	db.media, err = sql.Open("sqlite3", filepath.Join(directory, "collection.media.db2"))
	if err != nil {
		return err
	}

	db.isOpen = true

	return nil
}

func (db *ankiDatabase) Close() error {
	var err error

	if !db.isOpen {
		return fmt.Errorf("database is not open")
	}

	db.collection.Close()
	db.collection = nil

	db.media.Close()
	db.media = nil

	db.isOpen = false

	return err
}

func dbexec(db *sql.DB, sql string) error {
	var err error
	if _, err = db.Exec(sql); err != nil {
		return err
	}
	return nil
}

func (db *ankiDatabase) CollectionExec(sql string) error { return dbexec(db.collection, sql) }
func (db *ankiDatabase) MediaExec(sql string) error      { return dbexec(db.media, sql) }

// ==================================================

type ankiPackage struct {
	filename        string
	tempdir         string
	isOpen          bool
	database        AnkiDatabase
	collection      AnkiCollection
	notes           AnkiNotes
	cards           AnkiCards
	revlog          AnkiRevlog
	deckConfig      AnkiDeckConfig
	config          AnkiConfig
	fields          AnkiFields
	templates       AnkiTemplates
	noteTypes       AnkiNoteTypes
	decks           AnkiDecks
	tags            AnkiTags
	graves          AnkiGraves
	androidMetadata AnkiAndroidMetadata
	media           AnkiMedia
	meta            AnkiMeta
}

func (apkg *ankiPackage) Open(filename string) error {
	var err error
	var isNew bool

	if apkg.isOpen {
		return fmt.Errorf("package is already open in temporary directory: %s", apkg.tempdir)
	}

	apkg.filename = filename

	if _, err := os.Stat(apkg.filename); errors.Is(err, os.ErrNotExist) {
		isNew = true
	}

	apkg.tempdir, err = os.MkdirTemp("", "flashcard-*")
	if err != nil {
		return err
	}

	if isNew {
		// touch
		file, err := os.OpenFile(apkg.filename, os.O_RDONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		file.Close()
	} else {
		// unzip
		archive, err := zip.OpenReader(apkg.filename)
		if err != nil {
			return err
		}
		defer archive.Close()

		for _, f := range archive.File {
			fpath := filepath.Join(apkg.tempdir, f.Name)
			if !strings.HasPrefix(fpath, filepath.Clean(apkg.tempdir)+string(os.PathSeparator)) {
				return fmt.Errorf("invalid file path in the package: %s", fpath)
			}
			if f.FileInfo().IsDir() {
				os.MkdirAll(fpath, os.ModePerm)
				continue
			}
			if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}
			dstFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			fileInArchive, err := f.Open()
			if err != nil {
				return err
			}
			if _, err := io.Copy(dstFile, fileInArchive); err != nil {
				return err
			}
			dstFile.Close()
			fileInArchive.Close()
		}
	}

	err = apkg.database.Open(apkg.tempdir)
	if err != nil {
		return err
	}

	if isNew {
		// init
		apkg.collection.Init(apkg.database)
		apkg.notes.Init(apkg.database)
		apkg.cards.Init(apkg.database)
		apkg.revlog.Init(apkg.database)
		apkg.deckConfig.Init(apkg.database)
		apkg.config.Init(apkg.database)
		apkg.fields.Init(apkg.database)
		apkg.templates.Init(apkg.database)
		apkg.noteTypes.Init(apkg.database)
		apkg.decks.Init(apkg.database)
		apkg.tags.Init(apkg.database)
		apkg.graves.Init(apkg.database)
		apkg.androidMetadata.Init(apkg.database)
		apkg.media.Init(apkg.database)
		apkg.meta.Init(apkg.database)
	}

	apkg.isOpen = true

	return nil
}

func (apkg *ankiPackage) Close() error {
	var err error

	if !apkg.isOpen {
		return fmt.Errorf("package is not open")
	}

	err = apkg.database.Close()
	if err != nil {
		return err
	}

	archive, err := os.Create(apkg.filename)
	if err != nil {
		return err
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	zipWalker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		filedata, err := os.Open(path)
		if err != nil {
			return err
		}
		defer filedata.Close()

		relpath, err := filepath.Rel(apkg.tempdir, path)
		if err != nil {
			return err
		}

		pkgdata, err := zipWriter.Create(relpath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(pkgdata, filedata); err != nil {
			return err
		}

		return nil
	}

	err = filepath.Walk(apkg.tempdir, zipWalker)
	if err != nil {
		return err
	}

	os.RemoveAll(apkg.tempdir)
	apkg.tempdir = ""

	apkg.isOpen = false

	return nil
}

func (apkg *ankiPackage) ImportProject(project *FlashcardProject) error {
	var err error

	kinds := map[string]anki.Notetype_Config_Kind{
		"normal": anki.Notetype_Config_KIND_NORMAL,
		"cloze":  anki.Notetype_Config_KIND_CLOZE,
	}

	style, err := os.ReadFile(project.Model.Style.CSS)
	if err != nil {
		return err
	}

	latexPrefix, err := os.ReadFile(project.Model.Style.Latex.Prefix)
	if err != nil {
		return err
	}

	latexPostfix, err := os.ReadFile(project.Model.Style.Latex.Postfix)
	if err != nil {
		return err
	}

	ntconfig := &anki.Notetype_Config{
		Kind:         kinds[project.Model.Kind],
		SortFieldIdx: 0,
		Css:          string(style),
		LatexPre:     string(latexPrefix),
		LatexPost:    string(latexPostfix),
		LatexSvg:     false,
		Reqs:         nil,
		Other:        []byte("{\"vers\":[],\"tags\":[]}"),
	}

	ntcfg, err := proto.Marshal(ntconfig)
	if err != nil {
		return err
	}

	notetypes := NewAnkiNoteTypes()
	ntid, err := notetypes.Add(apkg.database, project.Model.Identifier, project.Model.Name, ntcfg)
	if err != nil {
		return err
	}

	templates := NewAnkiTemplates()
	for ord, tpl := range project.Model.Templates {
		qfmt, err := os.ReadFile(tpl.QFmt)
		if err != nil {
			return err
		}

		afmt, err := os.ReadFile(tpl.AFmt)
		if err != nil {
			return err
		}

		cfg := &anki.Notetype_Template_Config{
			QFormat: string(qfmt),
			AFormat: string(afmt),
		}

		tplcfg, err := proto.Marshal(cfg)
		if err != nil {
			return err
		}

		err = templates.Add(apkg.database, ntid, ord, tpl.Name, tplcfg)
		if err != nil {
			return err
		}
	}

	fields := NewAnkiFields()
	for ord, fld := range project.Model.Fields {
		cfg := &anki.Notetype_Field_Config{
			Sticky:            false,
			Rtl:               fld.RTL,
			FontName:          fld.Font.Name,
			FontSize:          fld.Font.Size,
			Description:       fld.Description,
			PlainText:         fld.Format != "text",
			Collapsed:         true,
			ExcludeFromSearch: false,
			// Id: 0,
			// Tag: 0,
			PreventDeletion: false,
			Other:           []byte("{\"media\":[]}"),
		}

		fldcfg, err := proto.Marshal(cfg)
		if err != nil {
			return err
		}

		fields.Add(apkg.database, ntid, ord, fld.Name, fldcfg)
	}

	return err
}

// ==================================================

//  C O N S T R U C T O R S

func NewAnkiCollection() AnkiCollection {
	col := new(ankiCollection)

	/**
	a single row with information about the collection
	  id = arbitrary number
	  crt = creation date in seconds
	  mod = last modified in milliseconds
	  scm = schema modification time
	  ver = schema version
	  dty = dirty, unused, set to 0
	  usn = update sequence number
	  ls = last sync time
	  conf = json with synced configuration options
	  models = json representing models
	  decks = json representing decks
	  dconf = json representing deck configuration
	  tags = cache of tags used in the collection
	**/
	col.schema = `
		CREATE TABLE col (
			id              integer primary key,
			crt             integer not null,
			mod             integer not null,
			scm             integer not null,
			ver             integer not null,
			dty             integer not null,
			usn             integer not null,
			ls              integer not null,
			conf            text not null,
			models          text not null,
			decks           text not null,
			dconf           text not null,
			tags            text not null
		);`

	col.add = `
		DELETE FROM col;
		INSERT INTO col VALUES(%d,%d,%d,%d,18,0,0,0,'','','','','');`

	return col
}

func NewAnkiNotes() AnkiNotes {
	notes := new(ankiNotes)

	/**
	raw information
	  id = creation time in milliseconds
	  guid = globally unique ID
	  mid = model ID
	  mod = modification timestamp in seconds
	  usn = update sequence number
	  tags = tags space-separated
	  flds = fields separated by 0x1f (31) character
	  sfld = sort field
	  csum = checksum, integer representation of the first 8 digits of sha1 hash of the first field
	  flags = not used, set to 0
	  data = not used, set to empty string
	**/
	notes.schema = `
    CREATE TABLE notes (
        id              integer primary key,   /* 0 */
        guid            text not null,         /* 1 */
        mid             integer not null,      /* 2 */
        mod             integer not null,      /* 3 */
        usn             integer not null,      /* 4 */
        tags            text not null,         /* 5 */
        flds            text not null,         /* 6 */
        sfld            integer not null,      /* 7 */
        csum            integer not null,      /* 8 */
        flags           integer not null,      /* 9 */
        data            text not null          /* 10 */
    );

    CREATE INDEX ix_notes_usn on notes (usn);
	CREATE INDEX ix_notes_csum on notes (csum);
	CREATE INDEX idx_notes_mid ON notes (mid);`

	notes.add = `
		INSERT INTO notes VALUES(%d,'%s',%d,%d,%d,'%s','%s',%d,%d,0,'');`

	return notes
}

func NewAnkiCards() AnkiCards {
	cards := new(ankiCards)

	cards.schema = `
    CREATE TABLE cards (
      id              integer primary key,   /* 0 */
      nid             integer not null,      /* 1 */
      did             integer not null,      /* 2 */
      ord             integer not null,      /* 3 */
      mod             integer not null,      /* 4 */
      usn             integer not null,      /* 5 */
      type            integer not null,      /* 6 */
      queue           integer not null,      /* 7 */
      due             integer not null,      /* 8 */
      ivl             integer not null,      /* 9 */
      factor          integer not null,      /* 10 */
      reps            integer not null,      /* 11 */
      lapses          integer not null,      /* 12 */
      left            integer not null,      /* 13 */
      odue            integer not null,      /* 14 */
      odid            integer not null,      /* 15 */
      flags           integer not null,      /* 16 */
      data            text not null          /* 17 */
    );

    CREATE INDEX ix_cards_usn on cards (usn);
	CREATE INDEX ix_cards_nid on cards (nid);
	CREATE INDEX ix_cards_sched on cards (did, queue, due);
	CREATE INDEX idx_cards_odid ON cards (odid) WHERE odid != 0;`

	return cards
}

func NewAnkiRevlog() AnkiRevlog {
	revlog := new(ankiRevlog)

	revlog.schema = `
    CREATE TABLE revlog (
      id              integer primary key,
      cid             integer not null,
      usn             integer not null,
      ease            integer not null,
      ivl             integer not null,
      lastIvl         integer not null,
      factor          integer not null,
      time            integer not null,
      type            integer not null
    );
    CREATE INDEX ix_revlog_usn on revlog (usn);
	CREATE INDEX ix_revlog_cid on revlog (cid);`

	return revlog
}

func NewAnkiDeckConfig() AnkiDeckConfig {
	deckcfg := new(ankiDeckConfig)

	deckcfg.schema = `
		CREATE TABLE deck_config (
		id integer PRIMARY KEY NOT NULL,
		name text NOT NULL COLLATE NOCASE,
		mtime_secs integer NOT NULL,
		usn integer NOT NULL,
		config blob NOT NULL
		);`

	return deckcfg
}

func NewAnkiConfig() AnkiConfig {
	cfg := new(ankiConfig)

	cfg.schema = `
		CREATE TABLE config (
		KEY text NOT NULL PRIMARY KEY,
		usn integer NOT NULL,
		mtime_secs integer NOT NULL,
		val blob NOT NULL
		) without rowid;`

	return cfg
}

func NewAnkiFields() AnkiFields {
	fields := new(ankiFields)

	fields.schema = `
		CREATE TABLE fields (
		ntid integer NOT NULL,
		ord integer NOT NULL,
		name text NOT NULL COLLATE NOCASE,
		config blob NOT NULL,
		PRIMARY KEY (ntid, ord)
		) without rowid;

		CREATE UNIQUE INDEX idx_fields_name_ntid ON fields (name, ntid);`

	fields.add = `INSERT INTO fields VALUES(%d,%d,'%s',X'%s');`

	return fields
}

func NewAnkiTemplates() AnkiTemplates {
	tpl := new(ankiTemplates)

	tpl.schema = `
		CREATE TABLE templates (
		ntid integer NOT NULL,
		ord integer NOT NULL,
		name text NOT NULL COLLATE NOCASE,
		mtime_secs integer NOT NULL,
		usn integer NOT NULL,
		config blob NOT NULL,
		PRIMARY KEY (ntid, ord)
		) without rowid;

		CREATE UNIQUE INDEX idx_templates_name_ntid ON templates (name, ntid);
		CREATE INDEX idx_templates_usn ON templates (usn);`
	tpl.add = `
		INSERT INTO templates
			VALUES(%d,%d,'%s',%d,0,X'%s')
		ON CONFLICT (name, ntid) DO UPDATE
		SET
			ord = excluded.ord,
			mtime_secs = excluded.mtime_secs,
			usn = usn + 1,
			config = excluded.config
		WHERE ntid = excluded.ntid AND name = excluded.name;`

	return tpl
}

/**
**/
func NewAnkiNoteTypes() AnkiNoteTypes {
	nt := new(ankiNoteTypes)

	nt.schema = `
		CREATE TABLE notetypes (
		id integer NOT NULL PRIMARY KEY,
		name text NOT NULL COLLATE NOCASE,
		mtime_secs integer NOT NULL,
		usn integer NOT NULL,
		config blob NOT NULL
		);

		CREATE UNIQUE INDEX idx_notetypes_name ON notetypes (name);
		CREATE INDEX idx_notetypes_usn ON notetypes (usn);`
	nt.add = `
		INSERT INTO notetypes
			VALUES(%d,'%s',%d,%d,X'%s')
		ON CONFLICT(id) DO UPDATE
		SET
			name = excluded.name,
			mtime_secs = excluded.mtime_secs,
			usn = usn + 1,
			config = excluded.config
		WHERE id = excluded.id;`

	return nt
}

func NewAnkiDecks() AnkiDecks {
	decks := new(ankiDecks)

	decks.schema = `
		CREATE TABLE decks (
		id integer PRIMARY KEY NOT NULL,
		name text NOT NULL COLLATE NOCASE,
		mtime_secs integer NOT NULL,
		usn integer NOT NULL,
		common blob NOT NULL,
		kind blob NOT NULL
		);

		CREATE UNIQUE INDEX idx_decks_name ON decks (name);`

	return decks
}

func NewAnkiTags() AnkiTags {
	tags := new(ankiTags)

	tags.schema = `
		CREATE TABLE tags (
		tag text NOT NULL PRIMARY KEY COLLATE NOCASE,
		usn integer NOT NULL,
		collapsed boolean NOT NULL,
		config blob NULL
		) without rowid;`

	return tags
}

func NewAnkiGraves() AnkiGraves {
	graves := new(ankiGraves)

	graves.schema = `
		CREATE TABLE graves (
		oid integer NOT NULL,
		type integer NOT NULL,
		usn integer NOT NULL,
		PRIMARY KEY (oid, type)
		) WITHOUT ROWID;

		CREATE INDEX idx_graves_pending ON graves (usn);`

	return graves
}

func NewAnkiAndroidMetadata() AnkiAndroidMetadata {
	meta := new(ankiAndroidMetadata)

	meta.schema = `
	CREATE TABLE android_metadata (
	locale TEXT
	);`

	return meta
}

func NewAnkiMedia() AnkiMedia {
	media := new(ankiMedia)

	media.schema = `
		CREATE TABLE media (
		fname text NOT NULL PRIMARY KEY,
		csum text,
		mtime int NOT NULL,
		dirty int NOT NULL
		) without rowid;

		CREATE INDEX idx_media_dirty ON media (dirty) WHERE dirty = 1;`

	return media
}

func NewAnkiMeta() AnkiMeta {
	meta := new(ankiMeta)

	meta.schema = `
		CREATE TABLE meta (
		dirMod int,
		lastUsn int
		);

		INSERT INTO meta VALUES(1698008361925,156);`

	return meta
}

func NewAnkiDatabase() AnkiDatabase {
	db := new(ankiDatabase)
	return db
}

func NewAnkiPackage() AnkiPackage {
	apkg := new(ankiPackage)
	apkg.database = NewAnkiDatabase()
	apkg.collection = NewAnkiCollection()
	apkg.notes = NewAnkiNotes()
	apkg.cards = NewAnkiCards()
	apkg.revlog = NewAnkiRevlog()
	apkg.deckConfig = NewAnkiDeckConfig()
	apkg.config = NewAnkiConfig()
	apkg.fields = NewAnkiFields()
	apkg.templates = NewAnkiTemplates()
	apkg.noteTypes = NewAnkiNoteTypes()
	apkg.decks = NewAnkiDecks()
	apkg.tags = NewAnkiTags()
	apkg.graves = NewAnkiGraves()
	apkg.androidMetadata = NewAnkiAndroidMetadata()
	apkg.media = NewAnkiMedia()
	apkg.meta = NewAnkiMeta()
	return apkg
}
