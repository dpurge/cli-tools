# CLI tools

Command line tools for private projects

- [https://github.com/ankitects/anki]
- [https://github.com/kerrickstaley/genanki]
- [https://gist.github.com/sartak/3921255]
- [https://github.com/ankidroid/Anki-Android/wiki/Database-Structure]

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

protoc --proto_path=./pkg/tool/proto/anki --go_out=./pkg/tool/proto/anki ./pkg/tool/proto/anki/generic.proto
protoc --proto_path=./pkg/tool/proto/anki --go_out=./pkg/tool/proto/anki ./pkg/tool/proto/anki/notetypes.proto
protoc --proto_path=./pkg/tool/proto/anki --go_out=./pkg/tool/proto/anki ./pkg/tool/proto/anki/notes.proto
```
