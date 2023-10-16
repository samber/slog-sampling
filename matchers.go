package slogsampling

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"log/slog"
	"runtime"
	"strconv"

	slogcommon "github.com/samber/slog-common"
)

var DefaultMatcher = MatchByLevelAndMessage()

type Matcher func(context.Context, *slog.Record) string

func MatchAll() func(context.Context, *slog.Record) string {
	return func(ctx context.Context, r *slog.Record) string {
		return ""
	}
}

func MatchByLevel() func(context.Context, *slog.Record) string {
	return func(ctx context.Context, r *slog.Record) string {
		return r.Level.String()
	}
}

func MatchByMessage() func(context.Context, *slog.Record) string {
	return func(ctx context.Context, r *slog.Record) string {
		return r.Message
	}
}

func MatchByLevelAndMessage() func(context.Context, *slog.Record) string {
	return func(ctx context.Context, r *slog.Record) string {
		// separator is used to avoid collisions
		return r.Level.String() + "@" + r.Message
	}
}

func MatchBySource() func(context.Context, *slog.Record) string {
	return func(ctx context.Context, r *slog.Record) string {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()

		// separator is used to avoid collisions
		return fmt.Sprintf("%s@%d@%s", f.File, f.Line, f.Function)
	}
}

func MatchByAttribute(groups []string, key string) func(context.Context, *slog.Record) string {
	return func(ctx context.Context, r *slog.Record) string {
		var output string

		r.Attrs(func(attr slog.Attr) bool {
			attr, found := slogcommon.FindAttribute([]slog.Attr{attr}, groups, key)
			if found {
				value := attr.Value.Resolve().Any()
				output = anyToString(value)

				// if value is nil or empty, we keep looking for a matching attribute
				return len(output) > 0
			}

			return true
		})

		return output
	}
}

func MatchByContextValue(key any) func(context.Context, *slog.Record) string {
	return func(ctx context.Context, r *slog.Record) string {
		return anyToString(ctx.Value(key))
	}
}

func anyToString(value any) string {
	// no value
	if value == nil {
		return ""
	}

	// primitive types
	switch v := value.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	case int, int64, int32, int16, int8:
		return strconv.FormatInt(value.(int64), 10)
	case uint, uint64, uint32, uint16, uint8:
		return strconv.FormatUint(value.(uint64), 10)
	case float64, float32:
		return strconv.FormatFloat(value.(float64), 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value.(bool))
	case complex128, complex64:
		return strconv.FormatComplex(value.(complex128), 'f', -1, 64)
	}

	// gob-encodable types
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	err := enc.Encode(value)
	if err != nil {
		return fmt.Sprintf("%#v", value)
	}

	return buf.String()
}
