package tool

import (
	"archive/zip"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
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
}

type AnkiCollection interface {
	Init(AnkiDatabase) error
}

type AnkiNotes interface {
	Init(AnkiDatabase) error
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
	Init(AnkiDatabase) error
}

type AnkiTemplates interface {
	Init(AnkiDatabase) error
}

type AnkiNoteTypes interface {
	Init(AnkiDatabase) error
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
}

func (col *ankiCollection) Init(db AnkiDatabase) error {
	return db.CollectionExec(col.schema)
}

// ==================================================

type ankiNotes struct {
	schema string
}

func (notes *ankiNotes) Init(db AnkiDatabase) error {
	return db.CollectionExec(notes.schema)
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
}

func (fields *ankiFields) Init(db AnkiDatabase) error {
	return db.CollectionExec(fields.schema)
}

// ==================================================

type ankiTemplates struct {
	schema string
}

func (templates *ankiTemplates) Init(db AnkiDatabase) error {
	return db.CollectionExec(templates.schema)
}

// ==================================================

type ankiNoteTypes struct {
	schema string
}

func (nt *ankiNoteTypes) Init(db AnkiDatabase) error {
	return db.CollectionExec(nt.schema)
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

// ==================================================

//  C O N S T R U C T O R S

func NewAnkiCollection() AnkiCollection {
	col := new(ankiCollection)

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
    );

    INSERT INTO col VALUES(1,1570327200,1716314047291,1712183754557,18,0,2108,1716314047291,'','','','','');
  `

	return col
}

func NewAnkiNotes() AnkiNotes {
	notes := new(ankiNotes)

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
	CREATE INDEX idx_notes_mid ON notes (mid);
  `

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
	CREATE INDEX idx_cards_odid ON cards (odid) WHERE odid != 0;
  `

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
	CREATE INDEX ix_revlog_cid on revlog (cid);
  `

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
		);
  	`

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
		) without rowid;
  	`

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

		CREATE UNIQUE INDEX idx_fields_name_ntid ON fields (name, ntid);
  	`

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
		CREATE INDEX idx_templates_usn ON templates (usn);
  	`

	return tpl
}

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
		CREATE INDEX idx_notetypes_usn ON notetypes (usn);
  	`

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

		CREATE UNIQUE INDEX idx_decks_name ON decks (name);
  	`

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
		) without rowid;
  	`

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

		CREATE INDEX idx_graves_pending ON graves (usn);
  	`

	return graves
}

func NewAnkiAndroidMetadata() AnkiAndroidMetadata {
	meta := new(ankiAndroidMetadata)

	meta.schema = `
	CREATE TABLE android_metadata (
	locale TEXT
	);
  	`

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

		CREATE INDEX idx_media_dirty ON media (dirty) WHERE dirty = 1;
  	`

	return media
}

func NewAnkiMeta() AnkiMeta {
	meta := new(ankiMeta)

	meta.schema = `
		CREATE TABLE meta (
		dirMod int,
		lastUsn int
		);

		INSERT INTO meta VALUES(1698008361925,156);
  	`

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
