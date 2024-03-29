package tool

import (
	"archive/zip"
	"database/sql"
	"io"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const schema string = `
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

CREATE TABLE graves (
    usn             integer not null,
    oid             integer not null,
    type            integer not null
);

CREATE INDEX ix_notes_usn on notes (usn);
CREATE INDEX ix_cards_usn on cards (usn);
CREATE INDEX ix_revlog_usn on revlog (usn);
CREATE INDEX ix_cards_nid on cards (nid);
CREATE INDEX ix_cards_sched on cards (did, queue, due);
CREATE INDEX ix_revlog_cid on revlog (cid);
CREATE INDEX ix_notes_csum on notes (csum);
`

type AnkiPackage struct {
	// decks       []string
	// media_files []string
}

type AnkiModel struct{}

type AnkiDeck struct{}

type AnkiNote struct{}

type AnkiCard struct{}

// var db *sql.DB

type PackageWriter struct {
	filename  string
	tempdir   string
	dbfile    string
	database  *sql.DB
	archive   *os.File
	zipWriter *zip.Writer
}

func (apkg *AnkiPackage) Save(filename string) (string, error) {

	writer := new(PackageWriter)
	if err := writer.Open(filename); err != nil {
		return "", err
	}
	defer writer.Close()

	return filename, nil
}

func (writer *PackageWriter) Open(filename string) error {

	var err error

	writer.filename = filename

	writer.tempdir, err = os.MkdirTemp("", "flashcard-*")
	if err != nil {
		return err
	}

	writer.dbfile = filepath.Join(writer.tempdir, "collection.anki2")

	writer.database, err = sql.Open("sqlite3", writer.dbfile)
	if err != nil {
		return err
	}

	if _, err = writer.database.Exec(schema); err != nil {
		return err
	}

	writer.archive, err = os.Create(filename)
	if err != nil {
		return err
	}

	writer.zipWriter = zip.NewWriter(writer.archive)

	return nil
}

func (writer *PackageWriter) Close() error {
	var err error

	writer.database.Close()

	_, err = writer.addFile(writer.dbfile, "collection.anki2")
	if err != nil {
		return err
	}

	writer.zipWriter.Close()
	writer.archive.Close()

	os.RemoveAll(writer.tempdir)

	return nil
}

func (writer *PackageWriter) addFile(filepath string, pkgpath string) (string, error) {

	filedata, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer filedata.Close()

	pkgdata, err := writer.zipWriter.Create(pkgpath)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(pkgdata, filedata); err != nil {
		return "", err
	}

	return pkgpath, nil
}
