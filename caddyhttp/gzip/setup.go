package gzip

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

// setup configures a new gzip middleware instance.
func setup(c *caddy.Controller) error {
	configs, err := gzipParse(c)
	if err != nil {
		return err
	}

	httpserver.GetConfig(c).AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return Gzip{Next: next, Configs: configs}
	})

	return nil
}

func gzipParse(c *caddy.Controller) ([]Config, error) {
	var configs []Config

	for c.Next() {
		config := Config{}

		// Request Filters
		pathFilter := PathFilter{IgnoredPaths: make(Set)}
		extFilter := ExtFilter{Exts: make(Set)}

		// Response Filters
		lengthFilter := LengthFilter(0)

		// No extra args expected
		if len(c.RemainingArgs()) > 0 {
			return configs, c.ArgErr()
		}

		for c.NextBlock() {
			switch c.Val() {
			case "ext":
				exts := c.RemainingArgs()
				if len(exts) == 0 {
					return configs, c.ArgErr()
				}
				for _, e := range exts {
					if !strings.HasPrefix(e, ".") && e != ExtWildCard && e != "" {
						return configs, fmt.Errorf(`gzip: invalid extension "%v" (must start with dot)`, e)
					}
					extFilter.Exts.Add(e)
				}
			case "not":
				paths := c.RemainingArgs()
				if len(paths) == 0 {
					return configs, c.ArgErr()
				}
				for _, p := range paths {
					if p == "/" {
						return configs, fmt.Errorf(`gzip: cannot exclude path "/" - remove directive entirely instead`)
					}
					if !strings.HasPrefix(p, "/") {
						return configs, fmt.Errorf(`gzip: invalid path "%v" (must start with /)`, p)
					}
					pathFilter.IgnoredPaths.Add(p)
				}
			case "level":
				if !c.NextArg() {
					return configs, c.ArgErr()
				}
				level, _ := strconv.Atoi(c.Val())
				config.Level = level
			case "min_length":
				if !c.NextArg() {
					return configs, c.ArgErr()
				}
				length, err := strconv.ParseInt(c.Val(), 10, 64)
				if err != nil {
					return configs, err
				} else if length == 0 {
					return configs, fmt.Errorf(`gzip: min_length must be greater than 0`)
				}
				lengthFilter = LengthFilter(length)
			default:
				return configs, c.ArgErr()
			}
		}

		// Request Filters
		config.RequestFilters = []RequestFilter{}

		// If ignored paths are specified, put in front to filter with path first
		if len(pathFilter.IgnoredPaths) > 0 {
			config.RequestFilters = []RequestFilter{pathFilter}
		}

		// Then, if extensions are specified, use those to filter.
		// Otherwise, use default extensions filter.
		if len(extFilter.Exts) > 0 {
			config.RequestFilters = append(config.RequestFilters, extFilter)
		} else {
			config.RequestFilters = append(config.RequestFilters, DefaultExtFilter())
		}

		config.ResponseFilters = append(config.ResponseFilters, SkipCompressedFilter{})

		// Response Filters
		// If min_length is specified, use it.
		if int64(lengthFilter) != 0 {
			config.ResponseFilters = append(config.ResponseFilters, lengthFilter)
		}

		configs = append(configs, config)
	}

	return configs, nil
}
