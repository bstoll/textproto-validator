// textproto-validator is a tool to verify the text format of protobuf data.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/jhump/protoreflect/desc/protoparse"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type importPathFlag []string

func (f *importPathFlag) String() string {
	return "[" + strings.Join(*f, ", ") + "]"
}

func (f *importPathFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

var importPath importPathFlag = []string{"."}

type fileReader interface {
	ReadFile(name string) ([]byte, error)
	Open(name string) (io.ReadCloser, error)
}

type realFileReader struct{}

func (rfr realFileReader) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (rfr realFileReader) Open(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

func main() {
	flag.Var(&importPath, "I", "Specify the directory in which to search for proto imports. May be specified multiple times.")
	flag.Parse()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s [example.textproto]:\n", os.Args[0])
		flag.PrintDefaults()
	}
	txtpb := flag.Arg(0)
	if txtpb == "" {
		flag.Usage()
		os.Exit(1)
	}
	if err := validateTextproto(realFileReader{}, txtpb); err != nil {
		log.Fatalf("Failed parsing %s: %s", txtpb, err)
	}
	log.Printf("Successfully validated %s", txtpb)
}

func validateTextproto(reader fileReader, txtpb string) error {
	b, err := reader.ReadFile(txtpb)
	if err != nil {
		return err
	}

	protoFile, protoMsg, err := extractHeaders(b)
	if err != nil {
		return err
	}

	protoMessage, err := findMessage(reader, protoFile, protoMsg)
	if err != nil {
		return fmt.Errorf("failed to find message %s in %s: %v", protoMsg, protoFile, err)
	}

	if err := prototext.Unmarshal(b, protoMessage); err != nil {
		return fmt.Errorf("unmarshal failure: %w", err)
	}
	return nil
}

// findMessage parses protoFile, searching for protoMsg and returning it if found.
func findMessage(reader fileReader, protoFile, protoMsg string) (protoreflect.ProtoMessage, error) {
	parser := protoparse.Parser{
		ImportPaths: importPath,
		Accessor:    reader.Open,
	}
	fds, err := parser.ParseFiles(protoFile)
	if err != nil {
		return nil, err
	}
	for _, fd := range fds {
		pfd := fd.UnwrapFile()
		protoMd := pfd.Messages().ByName(protoreflect.Name(protoMsg))
		if protoMd == nil {
			continue
		}
		return dynamicpb.NewMessage(protoMd), nil
	}

	return nil, errors.New("message not found")
}

// extractHeaders returns the proto-file and proto-message values from input.
//
// Input lines are expected to start with hash characters, and eventually find
// the strings proto-file: and proto-text:. Extra whitespace is ignored.
// Processing stops once a line does not begin with a # character.
func extractHeaders(input []byte) (string, string, error) {
	var protoFile, protoMsg string

	scanner := bufio.NewScanner(bytes.NewReader(input))
	for line := 0; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(text, "#") {
			break
		}
		fields := strings.Fields(text)
		if len(fields) != 3 {
			continue
		}
		switch fields[1] {
		case "proto-file:":
			if protoFile != "" {
				return "", "", fmt.Errorf("duplicate proto-file at line %d", line)
			}
			protoFile = fields[2]
		case "proto-message:":
			if protoMsg != "" {
				return "", "", fmt.Errorf("duplicate proto-message at line %d", line)
			}
			protoMsg = fields[2]
		}
	}

	if protoFile == "" {
		return "", "", errors.New("could not find proto-file comment")
	}
	if protoMsg == "" {
		return "", "", errors.New("could not find proto-message comment")
	}
	return protoFile, protoMsg, nil
}
