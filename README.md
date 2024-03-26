# textproto-validator

Textproto Validator is a tool to verify the [Text Format](https://protobuf.dev/reference/protobuf/textformat-spec/) of protobuf data.

A file is expected to contain a `proto-message` and a `proto-file` comment near the top section of the file.  The validator will attempt to compile `proto-file` and parse the textproto into the `proto-message`.

## Compiling

```
go test ./...
go build ./...
```

## Validating a file
Example `example.textproto` file:

```
# proto-file: example.proto
# proto-message: Example

name: "foo"
```

An `example.proto` could contain:

```
syntax = "proto3";
package example;
message Example {
    string name = 1;
}
```

```
$ ./textproto-validator example.textproto 
2024/03/26 05:41:58 Successfully validated example.textproto
```
