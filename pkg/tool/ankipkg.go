package tool

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const collection_schema string = `
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

CREATE TABLE android_metadata (locale TEXT);

CREATE TABLE deck_config (
  id integer PRIMARY KEY NOT NULL,
  name text NOT NULL COLLATE NOCASE,
  mtime_secs integer NOT NULL,
  usn integer NOT NULL,
  config blob NOT NULL
);

CREATE TABLE config (
  KEY text NOT NULL PRIMARY KEY,
  usn integer NOT NULL,
  mtime_secs integer NOT NULL,
  val blob NOT NULL
) without rowid;

CREATE TABLE fields (
  ntid integer NOT NULL,
  ord integer NOT NULL,
  name text NOT NULL COLLATE NOCASE,
  config blob NOT NULL,
  PRIMARY KEY (ntid, ord)
) without rowid;

CREATE TABLE templates (
  ntid integer NOT NULL,
  ord integer NOT NULL,
  name text NOT NULL COLLATE NOCASE,
  mtime_secs integer NOT NULL,
  usn integer NOT NULL,
  config blob NOT NULL,
  PRIMARY KEY (ntid, ord)
) without rowid;

CREATE TABLE notetypes (
  id integer NOT NULL PRIMARY KEY,
  name text NOT NULL COLLATE NOCASE,
  mtime_secs integer NOT NULL,
  usn integer NOT NULL,
  config blob NOT NULL
);

CREATE TABLE decks (
  id integer PRIMARY KEY NOT NULL,
  name text NOT NULL COLLATE NOCASE,
  mtime_secs integer NOT NULL,
  usn integer NOT NULL,
  common blob NOT NULL,
  kind blob NOT NULL
);

CREATE TABLE tags (
  tag text NOT NULL PRIMARY KEY COLLATE NOCASE,
  usn integer NOT NULL,
  collapsed boolean NOT NULL,
  config blob NULL
) without rowid;

CREATE TABLE graves (
  oid integer NOT NULL,
  type integer NOT NULL,
  usn integer NOT NULL,
  PRIMARY KEY (oid, type)
) WITHOUT ROWID;

CREATE INDEX ix_notes_usn on notes (usn);
CREATE INDEX ix_cards_usn on cards (usn);
CREATE INDEX ix_revlog_usn on revlog (usn);
CREATE INDEX ix_cards_nid on cards (nid);
CREATE INDEX ix_cards_sched on cards (did, queue, due);
CREATE INDEX ix_revlog_cid on revlog (cid);
CREATE INDEX ix_notes_csum on notes (csum);
CREATE UNIQUE INDEX idx_fields_name_ntid ON fields (name, ntid);
CREATE UNIQUE INDEX idx_templates_name_ntid ON templates (name, ntid);
CREATE INDEX idx_templates_usn ON templates (usn);
CREATE UNIQUE INDEX idx_notetypes_name ON notetypes (name);
CREATE INDEX idx_notetypes_usn ON notetypes (usn);
CREATE UNIQUE INDEX idx_decks_name ON decks (name);
CREATE INDEX idx_notes_mid ON notes (mid);
CREATE INDEX idx_cards_odid ON cards (odid) WHERE odid != 0;
CREATE INDEX idx_graves_pending ON graves (usn);
`

const media_schema string = `
CREATE TABLE media (
  fname text NOT NULL PRIMARY KEY,
  csum text,
  mtime int NOT NULL,
  dirty int NOT NULL
) without rowid;

CREATE TABLE meta (dirMod int, lastUsn int);
INSERT INTO meta VALUES(1698008361925,156);

CREATE INDEX idx_media_dirty ON media (dirty) WHERE dirty = 1;
`

type AnkiDatabase struct {
	collection *sql.DB
	media      *sql.DB
}

type AnkiPackage struct {
	tempdir  string
	database AnkiDatabase
	// decks       []string
	// media_files []string
}

// type AnkiModel struct{}

// type AnkiDeck struct{}

// type AnkiNote struct{}

// type AnkiCard struct{}

func NewAnkiPackage() (*AnkiPackage, error) {
	var err error
	apkg := new(AnkiPackage)

	apkg.tempdir, err = os.MkdirTemp("", "flashcard-*")
	if err != nil {
		return nil, err
	}

	apkg.database.collection, err = sql.Open("sqlite3", filepath.Join(apkg.tempdir, "collection.anki2"))
	if err != nil {
		return nil, err
	}

	if _, err = apkg.database.collection.Exec(collection_schema); err != nil {
		return nil, err
	}

	apkg.database.media, err = sql.Open("sqlite3", filepath.Join(apkg.tempdir, "collection.media.db2"))
	if err != nil {
		return nil, err
	}

	if _, err = apkg.database.media.Exec(media_schema); err != nil {
		return nil, err
	}

	fmt.Println(apkg.tempdir)

	return apkg, nil
}

// func (apkg *AnkiPackage) Open(filename string) (error) {
// }

func (apkg *AnkiPackage) Save(filename string) (string, error) {

	apkg.database.collection.Close()
	apkg.database.media.Close()

	archive, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	walker := func(path string, info os.FileInfo, err error) error {
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

	err = filepath.Walk(apkg.tempdir, walker)
	if err != nil {
		return "", err
	}

	os.RemoveAll(apkg.tempdir)
	return filename, nil
}
