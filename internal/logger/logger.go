// Package logger builds the application's *slog.Logger from configuration.
//
// One logger is created at startup and passed to every component through its
// constructor. Development defaults to human-readable text on stderr;
// production can fan records out to several outputs (stdout, stderr, files),
// each with its own format.
package logger

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Environments.
const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

// Formats.
const (
	FormatText = "text"
	FormatJSON = "json"
)

const (
	filePrefix   = "file:"
	logFileFlags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
)

// ErrInvalidOption is returned for an unknown environment, level, format or output.
var ErrInvalidOption = errors.New("invalid logger option")

// Options configures New. Empty Level, Format and Outputs take the defaults
// for Env (see Defaults).
type Options struct {
	// Env is EnvDevelopment or EnvProduction.
	Env string
	// Level is debug, info, warn or error.
	Level string
	// Format is the default format for outputs without an @format suffix.
	Format string
	// Outputs lists destinations: "stdout", "stderr" or "file:<path>", each
	// optionally suffixed with "@text" or "@json", e.g. "file:/var/log/app.log@json".
	Outputs []string
	// Service is added to every record as the "service" attribute.
	Service string
}

// Defaults returns the level, format and outputs used for env when Options
// leaves them empty.
func Defaults(env string) (level, format string, outputs []string) {
	if env == EnvProduction {
		return "info", FormatJSON, []string{"stdout"}
	}

	return "debug", FormatText, []string{"stderr"}
}

// New builds a logger from opts. The returned close function closes any files
// it opened and must be called once logging is finished.
func New(opts Options) (*slog.Logger, func() error, error) {
	if opts.Env != EnvDevelopment && opts.Env != EnvProduction {
		return nil, nil, fmt.Errorf("%w: environment %q (want %s or %s)",
			ErrInvalidOption, opts.Env, EnvDevelopment, EnvProduction,
		)
	}

	defLevel, defFormat, defOutputs := Defaults(opts.Env)
	opts.Level = cmp.Or(opts.Level, defLevel)
	opts.Format = cmp.Or(opts.Format, defFormat)

	if len(opts.Outputs) == 0 {
		opts.Outputs = defOutputs
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(opts.Level)); err != nil {
		return nil, nil, fmt.Errorf("%w: log level %q", ErrInvalidOption, opts.Level)
	}

	if !validFormat(opts.Format) {
		return nil, nil, fmt.Errorf("%w: log format %q (want %s or %s)",
			ErrInvalidOption, opts.Format, FormatText, FormatJSON,
		)
	}

	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: opts.Env == EnvDevelopment,
	}

	var (
		handlers []slog.Handler
		files    []*os.File
	)

	closeFiles := func() error {
		var errs []error
		for _, f := range files {
			errs = append(errs, f.Close())
		}

		return errors.Join(errs...)
	}

	for _, spec := range opts.Outputs {
		w, format, file, err := openOutput(spec, opts.Format)
		if err != nil {
			return nil, nil, errors.Join(err, closeFiles())
		}

		if file != nil {
			files = append(files, file)
		}

		handlers = append(handlers, newHandler(w, format, handlerOpts))
	}

	handler := handlers[0]
	if len(handlers) > 1 {
		handler = slog.NewMultiHandler(handlers...)
	}

	log := slog.New(handler)
	if opts.Service != "" {
		log = log.With("service", opts.Service)
	}

	return log, closeFiles, nil
}

// openOutput resolves one output spec to a writer and format. file is non-nil
// when the caller must close it.
func openOutput(spec, defaultFormat string) (w io.Writer, format string, file *os.File, err error) {
	dest, format, hasFormat := strings.Cut(strings.TrimSpace(spec), "@")
	if !hasFormat {
		format = defaultFormat
	}

	if !validFormat(format) {
		return nil, "", nil, fmt.Errorf("%w: output %q has format %q (want %s or %s)",
			ErrInvalidOption, spec, format, FormatText, FormatJSON,
		)
	}

	switch {
	case dest == "stdout":
		return os.Stdout, format, nil, nil
	case dest == "stderr":
		return os.Stderr, format, nil, nil
	case strings.HasPrefix(dest, filePrefix) && len(dest) > len(filePrefix):
		path := strings.TrimPrefix(dest, filePrefix)

		f, err := os.OpenFile(path, logFileFlags, 0o640) //nolint:gosec // operator-set path; group-readable logs
		if err != nil {
			return nil, "", nil, fmt.Errorf("open log file %q: %w", path, err)
		}

		return f, format, f, nil
	default:
		return nil, "", nil, fmt.Errorf("%w: output %q (want stdout, stderr or file:<path>)", ErrInvalidOption, spec)
	}
}

func newHandler(w io.Writer, format string, opts *slog.HandlerOptions) slog.Handler {
	if format == FormatJSON {
		return slog.NewJSONHandler(w, opts)
	}

	return slog.NewTextHandler(w, opts)
}

func validFormat(f string) bool {
	return f == FormatText || f == FormatJSON
}
