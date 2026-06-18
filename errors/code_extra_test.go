package errors_test

import (
	"encoding"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"testing"

	errutils "github.com/Is999/go-utils/errors"
)

func TestCodeFormattingAndEncoding(t *testing.T) {
	base := stderrors.New("base")
	err := errutils.WithCode(base, 10001)

	if !errutils.HasCode(err, 10001) {
		t.Fatal("HasCode() should match wrapped code")
	}
	if errutils.HasCode(err, 10002) {
		t.Fatal("HasCode() should reject different code")
	}
	if got, ok := errutils.Code(err); !ok || got != 10001 {
		t.Fatalf("Code() = %d,%v", got, ok)
	}
	if got := err.Error(); got != "base" {
		t.Fatalf("Error() = %q", got)
	}
	if !stderrors.Is(err, base) {
		t.Fatal("WithCode() should preserve unwrap chain")
	}

	if stringer, ok := err.(fmt.Stringer); ok {
		if got := stringer.String(); got == "" {
			t.Fatal("String() should return trace text")
		}
	} else {
		t.Fatal("WithCode() should implement fmt.Stringer")
	}
	if goStringer, ok := err.(fmt.GoStringer); ok {
		if got := goStringer.GoString(); !json.Valid([]byte(got)) {
			t.Fatalf("GoString() invalid JSON: %s", got)
		}
	} else {
		t.Fatal("WithCode() should implement fmt.GoStringer")
	}

	_ = fmt.Sprintf("%s", err)
	_ = fmt.Sprintf("%q", err)
	_ = fmt.Sprintf("%+v", err)
	_ = fmt.Sprintf("%#v", err)

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("MarshalJSON() error = %v", marshalErr)
	}
	if !json.Valid(data) {
		t.Fatalf("MarshalJSON() invalid JSON: %s", data)
	}
	if textErr, ok := err.(encoding.TextMarshaler); ok {
		text, textMarshalErr := textErr.MarshalText()
		if textMarshalErr != nil {
			t.Fatalf("MarshalText() error = %v", textMarshalErr)
		}
		if len(text) == 0 {
			t.Fatal("MarshalText() should return trace text")
		}
	} else {
		t.Fatal("WithCode() should implement encoding.TextMarshaler")
	}
}

func TestCodeNilInputs(t *testing.T) {
	if errutils.WithCode(nil, 1) != nil {
		t.Fatal("WithCode(nil) should return nil")
	}
	if errutils.HasCode(nil, 1) {
		t.Fatal("HasCode(nil) should be false")
	}
	if got, ok := errutils.Code(nil); ok || got != 0 {
		t.Fatalf("Code(nil) = %d,%v", got, ok)
	}
}
