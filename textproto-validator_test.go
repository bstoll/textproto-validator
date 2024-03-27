package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

type fakeFileReader struct {
	data map[string][]byte
}

func (ffr fakeFileReader) ReadFile(name string) ([]byte, error) {
	if ffr.data[name] != nil {
		return ffr.data[name], nil
	}
	return nil, fmt.Errorf("unknown file %s", name)
}

func (ffr fakeFileReader) Open(name string) (io.ReadCloser, error) {
	if ffr.data[name] != nil {
		reader := io.NopCloser(bytes.NewReader(ffr.data[name]))
		return reader, nil
	}
	return nil, fmt.Errorf("unknown file %s", name)
}

func TestValidateProto(t *testing.T) {
	cases := []struct {
		desc    string
		name    string
		files   map[string][]byte
		wantErr string
	}{
		{
			desc: "Valid textproto",
			files: map[string][]byte{
				"input.textproto": []byte("# proto-file: valid.proto\n# proto-message: Example\n\nname: \"example\""),
				"valid.proto":     []byte("syntax = \"proto3\";\npackage foo.bar.baz;\nmessage Example {\nstring name = 1;\n}"),
			},
		},
		{
			desc: "Valid fully qualified proto-message name",
			files: map[string][]byte{
				"input.textproto": []byte("# proto-file: valid.proto\n# proto-message: foo.bar.baz.Example\n\nname: \"example\""),
				"valid.proto":     []byte("syntax = \"proto3\";\npackage foo.bar.baz;\nmessage Example {\nstring name = 1;\n}"),
			},
		},
		{
			desc: "Syntax error in textproto",
			files: map[string][]byte{
				"input.textproto": []byte("# proto-file: valid.proto\n# proto-message: Example\n\nbadfield: \"example\""),
				"valid.proto":     []byte("syntax = \"proto3\";\npackage foo.bar.baz;\nmessage Example {\nstring name = 1;\n}"),
			},
			wantErr: "unknown field: badfield",
		},
		{
			desc: "Syntax error in proto-file",
			files: map[string][]byte{
				"input.textproto": []byte("# proto-file: valid.proto\n# proto-message: Example\n\n"),
				"valid.proto":     []byte("syntax = \"proto3\";\npackage foo.bar.baz;\nmessage Example {\nstring name = 1;\n}badtext"),
			},
			wantErr: "valid.proto:5:2: syntax error: unexpected identifier",
		},
		{
			desc: "proto-message not in proto-file",
			files: map[string][]byte{
				"input.textproto": []byte("# proto-file: valid.proto\n# proto-message: Missing\n\n"),
				"valid.proto":     []byte("syntax = \"proto3\";\npackage foo.bar.baz;\nmessage Example {\nstring name = 1;\n}"),
			},
			wantErr: "failed to find message Missing",
		},
		{
			desc: "Missing proto-file field",
			files: map[string][]byte{
				"input.textproto": []byte("# proto-message: Example\n\n"),
			},
			wantErr: "could not find proto-file comment",
		},
		{
			desc: "Missing proto-message field",
			files: map[string][]byte{
				"input.textproto": []byte("# proto-file: valid.proto\n\n"),
			},
			wantErr: "could not find proto-message comment",
		},
		{
			desc: "Missing proto-file due to newline comment break",
			files: map[string][]byte{
				"input.textproto": []byte("\n\n# proto-file: valid.proto\n# proto-message: Example\n\n"),
			},
			wantErr: "could not find proto-file comment",
		},
		{
			desc: "Missing proto-file due to extra invalid characters",
			files: map[string][]byte{
				"input.textproto": []byte("# proto-file: valid.proto junk\n# proto-message: Example\n\n"),
			},
			wantErr: "could not find proto-file comment",
		},
		{
			desc: "Duplicate proto-message field",
			files: map[string][]byte{
				"input.textproto": []byte("# proto-file: valid.proto\n# proto-message: Example\n# proto-message: Example\n\n"),
			},
			wantErr: "duplicate proto-message",
		},
		{
			desc: "Duplicate proto-file field",
			files: map[string][]byte{
				"input.textproto": []byte("# proto-file: valid.proto\n# proto-file: valid.proto\n# proto-message: Example\n\n"),
			},
			wantErr: "duplicate proto-file",
		},
		{
			desc:    "Missing textproto",
			wantErr: "unknown file input.textproto",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ffr := fakeFileReader{
				data: tc.files,
			}
			err := validateTextproto(ffr, "input.textproto")
			if (err == nil) != (tc.wantErr == "") || (err != nil && !strings.Contains(err.Error(), tc.wantErr)) {
				t.Errorf("validateTextproto() got error %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}
