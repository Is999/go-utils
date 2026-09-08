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
	// 无栈错误的输出固定，完整比较业务码与原因，避免空对象也能通过。
	const wantText = "code=10001; cause=base"
	const wantJSON = `{"code":10001,"err":{"msg":"base"}}`

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
		if got := stringer.String(); got != wantText {
			t.Fatalf("String() = %q, want %q", got, wantText)
		}
	} else {
		t.Fatal("WithCode() should implement fmt.Stringer")
	}
	if goStringer, ok := err.(fmt.GoStringer); ok {
		if got := goStringer.GoString(); got != wantJSON {
			t.Fatalf("GoString() = %q, want %q", got, wantJSON)
		}
	} else {
		t.Fatal("WithCode() should implement fmt.GoStringer")
	}

	for _, tc := range []struct {
		format string
		want   string
	}{
		{"%s", wantText},
		{"%q", `"code=10001; cause=base"`},
		{"%+v", wantJSON},
		{"%#v", wantJSON},
	} {
		if got := fmt.Sprintf(tc.format, err); got != tc.want {
			t.Errorf("format %s = %q, want %q", tc.format, got, tc.want)
		}
	}

	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("MarshalJSON() error = %v", marshalErr)
	}
	if string(data) != wantJSON {
		t.Fatalf("MarshalJSON() = %q, want %q", data, wantJSON)
	}
	if textErr, ok := err.(encoding.TextMarshaler); ok {
		text, textMarshalErr := textErr.MarshalText()
		if textMarshalErr != nil {
			t.Fatalf("MarshalText() error = %v", textMarshalErr)
		}
		if string(text) != wantText {
			t.Fatalf("MarshalText() = %q, want %q", text, wantText)
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
