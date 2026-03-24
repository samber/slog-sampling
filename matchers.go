package slogsampling

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"hash/fnv"
	"log/slog"
	"runtime"
	"strconv"

	slogcommon "github.com/samber/slog-common"
)

var DefaultMatcher = MatchByLevelAndMessage()

// Matcher is a function that returns a string hash for a given record.
// Returning []byte would have been much much better, but go's hashmap doesn't support it. 🤬
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
		lvl := r.Level.String()
		b := make([]byte, 0, len(lvl)+1+len(r.Message))
		b = append(b, lvl...)
		b = append(b, '@')
		b = append(b, r.Message...)
		return string(b)
	}
}

func MatchBySource() func(context.Context, *slog.Record) string {
	return func(ctx context.Context, r *slog.Record) string {
		pcs := [1]uintptr{r.PC}
		fs := runtime.CallersFrames(pcs[:])
		f, _ := fs.Next()

		// separator is used to avoid collisions
		line := strconv.Itoa(f.Line)
		b := make([]byte, 0, len(f.File)+1+len(line)+1+len(f.Function))
		b = append(b, f.File...)
		b = append(b, '@')
		b = append(b, line...)
		b = append(b, '@')
		b = append(b, f.Function...)
		return string(b)
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
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case complex64:
		return strconv.FormatComplex(complex128(v), 'f', -1, 64)
	case complex128:
		return strconv.FormatComplex(v, 'f', -1, 128)
	}

	// gob-encodable types
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	// bearer:disable go_lang_deserialization_of_user_input
	err := enc.Encode(value)
	if err != nil {
		return fmt.Sprintf("%#v", value)
	}

	return buf.String()
}

func CompactionFNV32a(input string) string {
	hash := fnv.New32a()
	hash.Write([]byte(input))
	return strconv.FormatInt(int64(hash.Sum32()), 10)
}

func CompactionFNV64a(input string) string {
	hash := fnv.New64a()
	hash.Write([]byte(input))
	return strconv.FormatInt(int64(hash.Sum64()), 10)
}

func CompactionFNV128a(input string) string {
	return string(fnv.New128a().Sum([]byte(input)))
}
