package builtin

import (
	"fmt"
	"time"
)

func init() {
	registerTimeNodes()
}

func registerTimeNodes() {
	// Time.Now
	Register(&NodeDefinition{
		Name:        "Time.Now",
		Category:    CategoryTime,
		Description: "Returns the current timestamp in milliseconds",
		Inputs:      []PortDefinition{},
		Outputs: []PortDefinition{
			{Name: "timestamp", Type: "int64", Description: "Current timestamp in milliseconds"},
		},
		Func: timeNow,
	})

	// Time.Since
	Register(&NodeDefinition{
		Name:        "Time.Since",
		Category:    CategoryTime,
		Description: "Returns milliseconds elapsed since a timestamp",
		Inputs: []PortDefinition{
			{Name: "timestamp", Type: "int64", Required: true, Description: "Starting timestamp"},
		},
		Outputs: []PortDefinition{
			{Name: "elapsed", Type: "int64", Description: "Milliseconds elapsed"},
		},
		Func: timeSince,
	})

	// Time.Add
	Register(&NodeDefinition{
		Name:        "Time.Add",
		Category:    CategoryTime,
		Description: "Adds duration to a timestamp",
		Inputs: []PortDefinition{
			{Name: "timestamp", Type: "int64", Required: true, Description: "Base timestamp"},
			{Name: "duration", Type: "int64", Required: true, Description: "Duration in milliseconds"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "int64", Description: "New timestamp"},
		},
		Func: timeAdd,
	})

	// Time.Subtract
	Register(&NodeDefinition{
		Name:        "Time.Subtract",
		Category:    CategoryTime,
		Description: "Subtracts duration from a timestamp",
		Inputs: []PortDefinition{
			{Name: "timestamp", Type: "int64", Required: true, Description: "Base timestamp"},
			{Name: "duration", Type: "int64", Required: true, Description: "Duration in milliseconds"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "int64", Description: "New timestamp"},
		},
		Func: timeSubtract,
	})

	// Time.Diff
	Register(&NodeDefinition{
		Name:        "Time.Diff",
		Category:    CategoryTime,
		Description: "Calculates difference between two timestamps",
		Inputs: []PortDefinition{
			{Name: "a", Type: "int64", Required: true, Description: "First timestamp"},
			{Name: "b", Type: "int64", Required: true, Description: "Second timestamp"},
		},
		Outputs: []PortDefinition{
			{Name: "diff", Type: "int64", Description: "Difference in milliseconds (a - b)"},
		},
		Func: timeDiff,
	})

	// Time.Format
	Register(&NodeDefinition{
		Name:        "Time.Format",
		Category:    CategoryTime,
		Description: "Formats a timestamp as a string",
		Inputs: []PortDefinition{
			{Name: "timestamp", Type: "int64", Required: true, Description: "Timestamp to format"},
			{Name: "format", Type: "string", Required: false, Default: "2006-01-02 15:04:05", Description: "Go time format string"},
		},
		Outputs: []PortDefinition{
			{Name: "formatted", Type: "string", Description: "Formatted time string"},
		},
		Func: timeFormat,
	})

	// Time.ToSeconds
	Register(&NodeDefinition{
		Name:        "Time.ToSeconds",
		Category:    CategoryTime,
		Description: "Converts milliseconds to seconds",
		Inputs: []PortDefinition{
			{Name: "ms", Type: "int64", Required: true, Description: "Milliseconds"},
		},
		Outputs: []PortDefinition{
			{Name: "seconds", Type: "float64", Description: "Seconds"},
		},
		Func: timeToSeconds,
	})

	// Time.FromSeconds
	Register(&NodeDefinition{
		Name:        "Time.FromSeconds",
		Category:    CategoryTime,
		Description: "Converts seconds to milliseconds",
		Inputs: []PortDefinition{
			{Name: "seconds", Type: "float64", Required: true, Description: "Seconds"},
		},
		Outputs: []PortDefinition{
			{Name: "ms", Type: "int64", Description: "Milliseconds"},
		},
		Func: timeFromSeconds,
	})

	// Time.IsBefore
	Register(&NodeDefinition{
		Name:        "Time.IsBefore",
		Category:    CategoryTime,
		Description: "Checks if timestamp a is before timestamp b",
		Inputs: []PortDefinition{
			{Name: "a", Type: "int64", Required: true, Description: "First timestamp"},
			{Name: "b", Type: "int64", Required: true, Description: "Second timestamp"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool", Description: "True if a < b"},
		},
		Func: timeIsBefore,
	})

	// Time.IsAfter
	Register(&NodeDefinition{
		Name:        "Time.IsAfter",
		Category:    CategoryTime,
		Description: "Checks if timestamp a is after timestamp b",
		Inputs: []PortDefinition{
			{Name: "a", Type: "int64", Required: true, Description: "First timestamp"},
			{Name: "b", Type: "int64", Required: true, Description: "Second timestamp"},
		},
		Outputs: []PortDefinition{
			{Name: "result", Type: "bool", Description: "True if a > b"},
		},
		Func: timeIsAfter,
	})
}

// Time node implementations

func timeNow(args map[string]interface{}) (interface{}, error) {
	return time.Now().UnixMilli(), nil
}

func timeSince(args map[string]interface{}) (interface{}, error) {
	timestamp, ok := toInt64(args["timestamp"])
	if !ok {
		return nil, fmt.Errorf("timestamp must be an integer")
	}
	return time.Now().UnixMilli() - timestamp, nil
}

func timeAdd(args map[string]interface{}) (interface{}, error) {
	timestamp, ok := toInt64(args["timestamp"])
	if !ok {
		return nil, fmt.Errorf("timestamp must be an integer")
	}
	duration, ok := toInt64(args["duration"])
	if !ok {
		return nil, fmt.Errorf("duration must be an integer")
	}
	return timestamp + duration, nil
}

func timeSubtract(args map[string]interface{}) (interface{}, error) {
	timestamp, ok := toInt64(args["timestamp"])
	if !ok {
		return nil, fmt.Errorf("timestamp must be an integer")
	}
	duration, ok := toInt64(args["duration"])
	if !ok {
		return nil, fmt.Errorf("duration must be an integer")
	}
	return timestamp - duration, nil
}

func timeDiff(args map[string]interface{}) (interface{}, error) {
	a, ok := toInt64(args["a"])
	if !ok {
		return nil, fmt.Errorf("a must be an integer")
	}
	b, ok := toInt64(args["b"])
	if !ok {
		return nil, fmt.Errorf("b must be an integer")
	}
	return a - b, nil
}

func timeFormat(args map[string]interface{}) (interface{}, error) {
	timestamp, ok := toInt64(args["timestamp"])
	if !ok {
		return nil, fmt.Errorf("timestamp must be an integer")
	}
	format, _ := args["format"].(string)
	if format == "" {
		format = "2006-01-02 15:04:05"
	}
	t := time.UnixMilli(timestamp)
	return t.Format(format), nil
}

func timeToSeconds(args map[string]interface{}) (interface{}, error) {
	ms, ok := toInt64(args["ms"])
	if !ok {
		return nil, fmt.Errorf("ms must be an integer")
	}
	return float64(ms) / 1000.0, nil
}

func timeFromSeconds(args map[string]interface{}) (interface{}, error) {
	seconds, ok := toFloat64(args["seconds"])
	if !ok {
		return nil, fmt.Errorf("seconds must be a number")
	}
	return int64(seconds * 1000), nil
}

func timeIsBefore(args map[string]interface{}) (interface{}, error) {
	a, ok := toInt64(args["a"])
	if !ok {
		return nil, fmt.Errorf("a must be an integer")
	}
	b, ok := toInt64(args["b"])
	if !ok {
		return nil, fmt.Errorf("b must be an integer")
	}
	return a < b, nil
}

func timeIsAfter(args map[string]interface{}) (interface{}, error) {
	a, ok := toInt64(args["a"])
	if !ok {
		return nil, fmt.Errorf("a must be an integer")
	}
	b, ok := toInt64(args["b"])
	if !ok {
		return nil, fmt.Errorf("b must be an integer")
	}
	return a > b, nil
}

// Helper function
func toInt64(v interface{}) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case float64:
		return int64(val), true
	case float32:
		return int64(val), true
	case uint:
		return int64(val), true
	case uint32:
		return int64(val), true
	case uint64:
		return int64(val), true
	default:
		return 0, false
	}
}
