package fig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultFilename is the default filename of the config file that fig looks for.
	DefaultFilename = "config.yaml"
	// DefaultDir is the default directory that fig searches in for the config file.
	DefaultDir = "."
	// DefaultTag is the default struct tag name that fig uses for field metadata.
	DefaultTag = "fig"
	// DefaultTimeLayout is the default time layout that fig uses to parse times.
	DefaultTimeLayout = time.RFC3339
)

// ErrFileNotFound is returned as a wrapped error by `Load` when the config file is
// not found in the given search dirs.
var ErrFileNotFound = fmt.Errorf("file not found")

// Load reads a configuration file and loads it into the given struct. The
// parameter `cfg` must be a pointer to a struct.
//
// By default fig looks for a file `config.yaml` in the current directory and
// uses the struct field tag `fig` for matching field names and validation.
// To alter this behaviour pass additional parameters as options.
//
// A field can be marked as required by adding a `required` key in the field's struct tag.
// If a required field is not set by the configuration file an error is returned.
//
//   type config struct {
//     Env string `fig:"env,required"` // or `fig:",required"`
//   }
//
// A field can be configured with a default value by adding a `default=value` in the field's
// struct tag.
// If a field is not set by the configuration file then the default value is set.
//
//  type config struct {
//    Level string `fig:"level,default=info"` // or `fig:",default=info"`
//  }
//
// A single field may not be marked as both `required` and `default`.
func Load(cfg interface{}, options ...Option) error {
	fig := newDefaultFig()

	for _, opt := range options {
		opt(fig)
	}

	return fig.Load(cfg)
}

func newDefaultFig() *fig {
	return &fig{
		filename:    DefaultFilename,
		dirs:        []string{DefaultDir},
		tag:         DefaultTag,
		requiredKey: "required",
		defaultKey:  "default=",
		sliceStart:  '[',
		sliceEnd:    ']',
		timeLayout:  DefaultTimeLayout,
	}
}

type fig struct {
	filename             string
	dirs                 []string
	tag                  string
	requiredKey          string
	defaultKey           string
	sliceStart, sliceEnd byte
	timeLayout           string
}

type fieldErrors map[string]error

func (fe fieldErrors) Error() string {
	keys := make([]string, 0, len(fe))
	for key := range fe {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	var sb strings.Builder
	sb.Grow(len(keys) * 10)

	for _, key := range keys {
		sb.WriteString(key)
		sb.WriteString(": ")
		sb.WriteString(fe[key].Error())
		sb.WriteString(", ")
	}

	return strings.TrimSuffix(sb.String(), ", ")
}

func (f *fig) Load(cfg interface{}) error {
	file, err := f.findFile()
	if err != nil {
		return err
	}

	vals, err := f.decodeFile(file)
	if err != nil {
		return err
	}

	if err := f.decodeMap(vals, cfg); err != nil {
		return err
	}

	if err := f.validate(cfg); err != nil {
		return err
	}

	return nil
}

func (f *fig) findFile() (string, error) {
	var filePath string

	for _, dir := range f.dirs {
		path := filepath.Join(dir, f.filename)
		if fileExists(path) {
			filePath = path
			break
		}
	}

	if filePath == "" {
		return "", fmt.Errorf("%s: %w", f.filename, ErrFileNotFound)
	}

	return filePath, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

// decodeFile reads the file and unmarshalls it using a decoder based on the file extension.
func (f *fig) decodeFile(filename string) (map[string]interface{}, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	vals := make(map[string]interface{})

	switch filepath.Ext(f.filename) {
	case ".yaml", ".yml":
		if err := yaml.NewDecoder(fd).Decode(&vals); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.NewDecoder(fd).Decode(&vals); err != nil {
			return nil, err
		}
	case ".toml":
		tree, err := toml.LoadReader(fd)
		if err != nil {
			return nil, err
		}

		for field, val := range tree.ToMap() {
			vals[field] = val
		}
	default:
		return nil, fmt.Errorf("unsupported file extension %s", filepath.Ext(f.filename))
	}

	return vals, nil
}

// decodeMap decodes a map of values into a result struct using the mapstructure library.
func (f *fig) decodeMap(m map[string]interface{}, result interface{}) error {
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           result,
		TagName:          f.tag,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToTimeHookFunc(f.timeLayout),
		),
	})
	if err != nil {
		return err
	}

	return dec.Decode(m)
}

// validate validates cfg using rules found in its struct tags.
// all fields are validated prior to returning and any errors encountered are returned as fieldErrors.
func (f *fig) validate(cfg interface{}) error {
	v := reflect.ValueOf(cfg)

	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("cfg must be a pointer to a struct")
	}

	errs := make(fieldErrors)
	f.validateStruct(v.Elem(), errs, "")

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// validateStruct validates all of the struct's fields using field tags and adds any errors to errs.
func (f *fig) validateStruct(fieldVal reflect.Value, errs fieldErrors, parentName string) {
	kind := fieldVal.Kind()
	if (kind == reflect.Ptr || kind == reflect.Interface) && !fieldVal.IsNil() {
		f.validateStruct(fieldVal.Elem(), errs, parentName)
		return
	}

	fieldType := fieldVal.Type()
	for i := 0; i < fieldType.NumField(); i++ {
		f.validateField(fieldVal.Field(i), fieldType.Field(i), errs, parentName)
	}
}

// validateField validates the field using the tag in the field struct definition and appends
// any errors into errs.
func (f *fig) validateField(fv reflect.Value, fd reflect.StructField, errs fieldErrors, parentName string) {
	if fd.PkgPath != "" && !fd.Anonymous {
		return // ignore non-embedded unexported fields
	}

	for (fv.Kind() == reflect.Ptr || fv.Kind() == reflect.Interface) && !fv.IsNil() {
		fv = fv.Elem()
	}

	name := strings.TrimPrefix(fmt.Sprintf("%s.%s", parentName, fd.Name), ".")
	f.validateCollection(fv, errs, name)

	tag := fd.Tag.Get(f.tag)
	if err := f.validateFieldWithTag(fv, tag); err != nil {
		errs[name] = err
	}
}

// validateCollection recursively validates colletions (slice, array, struct, ptr, interface).
// ptr and interface fields are dereferenced and recursively validated.
// any other field kinds are safely ignored.
func (f *fig) validateCollection(fv reflect.Value, errs fieldErrors, fieldName string) {
	switch fv.Kind() {
	case reflect.Ptr, reflect.Interface:
		f.validateCollection(fv.Elem(), errs, fieldName)
	case reflect.Struct:
		f.validateStruct(fv, errs, fieldName)
	case reflect.Slice, reflect.Array:
		switch fv.Type().Elem().Kind() {
		case reflect.Struct, reflect.Slice, reflect.Array, reflect.Ptr, reflect.Interface:
			for i := 0; i < fv.Len(); i++ {
				f.validateCollection(fv.Index(i), errs, fmt.Sprintf("%s[%d]", fieldName, i))
			}
		}
	}
}

// validateFieldWithTag validates the field using the tag.
// the tag must be a fig formatted tag with an optional required or default key after the optional field name.
// i.e. s string `fig:",required"`
// an empty tag or a tag with only a field name returns nil.
// if the tag is invalid or contains unexpected keys an errors is returned.
func (f *fig) validateFieldWithTag(fv reflect.Value, tag string) error {
	pairs := f.splitTagCommas(tag)
	if len(pairs) <= 1 {
		return nil
	}

	// fig only accepts max 2 keys in the tag: the field name + required/default
	if len(pairs) > 2 {
		return fmt.Errorf("too many keys in tag")
	}

	pair := pairs[1]

	switch {
	case pair == f.requiredKey:
		if isZero(fv) {
			return fmt.Errorf("required")
		}
	case strings.HasPrefix(pair, f.defaultKey):
		if isZero(fv) { // set a default only if value is zero
			defaultVal := pair[len(f.defaultKey):]
			if err := f.setFieldValue(fv, defaultVal); err != nil {
				return fmt.Errorf("unable to set default: %v", err)
			}
		}
	default:
		return fmt.Errorf("unexpected tag key: %s", pair)
	}

	return nil
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Array:
		return v.Len() == 0
	case reflect.Struct:
		if t, ok := v.Interface().(time.Time); ok {
			return t.IsZero()
		}
		return false
	case reflect.Invalid:
		return true
	default:
		return v.IsZero()
	}
}

// splitTagCommas is like strings.Split with a comma separator but it does not
// split values inside square brackets (f.sliceStart & f.sliceEnd).
// "ports,default=[80,443]" ---> []string{"port", "default=[80,443]"}
// if you pass a string with brackets within brackets then god help you.
func (f *fig) splitTagCommas(s string) []string {
	var (
		res      []string
		start    = 0
		inString = false
	)

	for i := 0; i < len(s); i++ {
		if s[i] == ',' && !inString { // nolint: gocritic
			res = append(res, s[start:i])
			start = i + 1
		} else if s[i] == f.sliceStart {
			inString = true
		} else if s[i] == f.sliceEnd {
			inString = false
		}
	}

	if start < len(s) {
		res = append(res, s[start:])
	}

	return res
}

// setFieldValue populates a field with a value using reflection.
// it attempts to convert val to the correct type based on the field's kind.
func (f *fig) setFieldValue(fv reflect.Value, val string) error {
	switch fv.Kind() {
	case reflect.Ptr:
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		return f.setFieldValue(fv.Elem(), val)
	case reflect.Slice:
		if err := f.setSliceValue(fv, val); err != nil {
			return err
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if _, ok := fv.Interface().(time.Duration); ok {
			d, err := time.ParseDuration(val)
			if err != nil {
				return err
			}
			fv.Set(reflect.ValueOf(d))
		} else {
			i, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				return err
			}
			fv.SetInt(i)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(f)
	case reflect.String:
		fv.SetString(val)
	case reflect.Struct: // struct is only allowed a default in the special case where it's a time.Time
		if _, ok := fv.Interface().(time.Time); ok {
			t, err := time.Parse(f.timeLayout, val)
			if err != nil {
				return err
			}
			fv.Set(reflect.ValueOf(t))
		} else {
			return fmt.Errorf("unsupported type %s", fv.Kind())
		}
	default:
		return fmt.Errorf("unsupported type %s", fv.Kind())
	}

	return nil
}

// setSliceValue populates a slice with val using reflection.
// sv must be a settable slice value.
// val must be a slice in string format (i.e. "[1,2,3]").
func (f *fig) setSliceValue(sv reflect.Value, val string) error {
	ss := f.stringSlice(val)
	slice := reflect.MakeSlice(sv.Type(), len(ss), cap(ss))

	for i, s := range ss {
		if err := f.setFieldValue(slice.Index(i), s); err != nil {
			return err
		}
	}

	sv.Set(slice)

	return nil
}

// stringSlice converts a slice in string format to an actual slice.
// fields should be separated by a comma.
//   "[1,2,3]"     --->   []string{"1", "2", "3"}
//   " foo , bar"  --->   []string{" foo ", " bar"}
func (f *fig) stringSlice(s string) []string {
	s = strings.TrimSuffix(strings.TrimPrefix(s, string(f.sliceStart)), string(f.sliceEnd))
	return strings.Split(s, ",")
}
