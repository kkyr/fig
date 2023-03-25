package fig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultFilename is the default filename of the config file that fig looks for.
	DefaultFilename = "config.yaml"
	// DefaultDir is the default directory that fig searches in for the config file.
	DefaultDir = "."
	// DefaultTag is the default struct tag key that fig uses to find the field's alt
	// name.
	DefaultTag = "fig"
	// DefaultTimeLayout is the default time layout that fig uses to parse times.
	DefaultTimeLayout = time.RFC3339
)

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
//	type Config struct {
//	  Env string `fig:"env" validate:"required"` // or just `validate:"required"`
//	}
//
// A field can be configured with a default value by adding a `default` key in the
// field's struct tag.
// If a field is not set by the configuration file then the default value is set.
//
//	type Config struct {
//	  Level string `fig:"level" default:"info"` // or just `default:"info"`
//	}
//
// A single field may not be marked as both `required` and `default`.
func Load(cfg interface{}, options ...Option) error {
	fig := defaultFig()

	for _, opt := range options {
		opt(fig)
	}

	return fig.Load(cfg)
}

func defaultFig() *fig {
	return &fig{
		filename:   DefaultFilename,
		dirs:       []string{DefaultDir},
		tag:        DefaultTag,
		timeLayout: DefaultTimeLayout,
	}
}

type fig struct {
	filename   string
	dirs       []string
	tag        string
	timeLayout string
	useEnv     bool
	useStrict  bool
	ignoreFile bool
	envPrefix  string
}

func (f *fig) Load(cfg interface{}) error {
	if !isStructPtr(cfg) {
		return fmt.Errorf("cfg must be a pointer to a struct")
	}

	vals := make(map[string]interface{})

	if !f.ignoreFile {
		file, err := f.findCfgFile()
		if err != nil {
			return err
		}

		vals, err = f.decodeFile(file)
		if err != nil {
			return err
		}
	}

	if err := f.decodeMap(vals, cfg); err != nil {
		return err
	}

	return f.processCfg(cfg)
}

func (f *fig) findCfgFile() (path string, err error) {
	for _, dir := range f.dirs {
		path = filepath.Join(dir, f.filename)
		if fileExists(path) {
			return
		}
	}
	return "", fmt.Errorf("%s: %w", f.filename, ErrFileNotFound)
}

// decodeFile reads the file and unmarshalls it using a decoder based on the file extension.
func (f *fig) decodeFile(file string) (map[string]interface{}, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	vals := make(map[string]interface{})

	switch filepath.Ext(file) {
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

// decodeMap decodes a map of values into result using the mapstructure library.
func (f *fig) decodeMap(m map[string]interface{}, result interface{}) error {
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           result,
		TagName:          f.tag,
		ErrorUnused:      f.useStrict,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToTimeHookFunc(f.timeLayout),
			stringToRegexpHookFunc(),
		),
	})
	if err != nil {
		return err
	}
	return dec.Decode(m)
}

// stringToRegexpHookFunc returns a DecodeHookFunc that converts strings to regexp.Regexp.
func stringToRegexpHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(&regexp.Regexp{}) {
			return data, nil
		}
		//nolint:forcetypeassert
		return regexp.Compile(data.(string))
	}
}

// processCfg processes a cfg struct after it has been loaded from
// the config file, by validating required fields and setting defaults
// where applicable.
func (f *fig) processCfg(cfg interface{}) error {
	fields := flattenCfg(cfg, f.tag)
	errs := make(fieldErrors)

	for _, field := range fields {
		if err := f.processField(field); err != nil {
			errs[field.path()] = err
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

// processField processes a single field and is called by processCfg
// for each field in cfg.
func (f *fig) processField(field *field) error {
	if field.required && field.setDefault {
		return fmt.Errorf("field cannot have both a required validation and a default value")
	}

	if f.useEnv {
		if err := f.setFromEnv(field.v, field.path()); err != nil {
			return fmt.Errorf("unable to set from env: %w", err)
		}
	}

	if field.required && isZero(field.v) {
		return fmt.Errorf("required validation failed")
	}

	if field.setDefault && isZero(field.v) {
		if err := f.setDefaultValue(field.v, field.defaultVal); err != nil {
			return fmt.Errorf("unable to set default: %w", err)
		}
	}

	return nil
}

func (f *fig) setFromEnv(fv reflect.Value, key string) error {
	key = f.formatEnvKey(key)
	if val, ok := os.LookupEnv(key); ok {
		return f.setValue(fv, val)
	}
	return nil
}

func (f *fig) formatEnvKey(key string) string {
	// loggers[0].level --> loggers_0_level
	key = strings.NewReplacer(".", "_", "[", "_", "]", "").Replace(key)
	if f.envPrefix != "" {
		key = fmt.Sprintf("%s_%s", f.envPrefix, key)
	}
	return strings.ToUpper(key)
}

// setDefaultValue calls setValue but disallows booleans from
// being set.
func (f *fig) setDefaultValue(fv reflect.Value, val string) error {
	if fv.Kind() == reflect.Bool {
		return fmt.Errorf("unsupported type: %v", fv.Kind())
	}
	return f.setValue(fv, val)
}

// setValue sets fv to val. it attempts to convert val to the correct
// type based on the field's kind. if conversion fails an error is
// returned.
// fv must be settable else this panics.
func (f *fig) setValue(fv reflect.Value, val string) error {
	switch fv.Kind() {
	case reflect.Ptr:
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		return f.setValue(fv.Elem(), val)
	case reflect.Slice:
		if err := f.setSlice(fv, val); err != nil {
			return err
		}
	case reflect.Bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		fv.SetBool(b)
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
		} else if _, ok := fv.Interface().(regexp.Regexp); ok {
			re, err := regexp.Compile(val)
			if err != nil {
				return err
			}
			fv.Set(reflect.ValueOf(*re))
		} else {
			return fmt.Errorf("unsupported type %s", fv.Kind())
		}
	default:
		return fmt.Errorf("unsupported type %s", fv.Kind())
	}
	return nil
}

// setSlice val to sv. val should be a Go slice formatted as a string
// (e.g. "[1,2]") and sv must be a slice value. if conversion of val
// to a slice fails then an error is returned.
// sv must be settable else this panics.
func (f *fig) setSlice(sv reflect.Value, val string) error {
	ss := stringSlice(val)
	slice := reflect.MakeSlice(sv.Type(), len(ss), cap(ss))
	for i, s := range ss {
		if err := f.setValue(slice.Index(i), s); err != nil {
			return err
		}
	}
	sv.Set(slice)
	return nil
}
