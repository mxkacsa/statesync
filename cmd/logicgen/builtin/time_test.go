package builtin

import (
	"testing"
	"time"
)

func TestTimeNow(t *testing.T) {
	before := time.Now().UnixMilli()

	result, err := Call("Time.Now", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	after := time.Now().UnixMilli()

	ts := result.(int64)
	if ts < before || ts > after {
		t.Errorf("Time.Now() = %d, expected between %d and %d", ts, before, after)
	}
}

func TestTimeSince(t *testing.T) {
	past := time.Now().Add(-100 * time.Millisecond).UnixMilli()

	result, err := Call("Time.Since", map[string]interface{}{
		"timestamp": past,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	elapsed := result.(int64)
	// Should be at least 100ms
	if elapsed < 100 {
		t.Errorf("Time.Since() = %d, expected >= 100", elapsed)
	}
	// But not more than 1 second (test shouldn't take that long)
	if elapsed > 1000 {
		t.Errorf("Time.Since() = %d, expected < 1000", elapsed)
	}
}

func TestTimeSince_InvalidInput(t *testing.T) {
	_, err := Call("Time.Since", map[string]interface{}{
		"timestamp": "not a number",
	})
	if err == nil {
		t.Error("expected error for invalid timestamp")
	}
}

func TestTimeAdd(t *testing.T) {
	tests := []struct {
		name      string
		timestamp int64
		duration  int64
		want      int64
	}{
		{"add positive", 1000, 500, 1500},
		{"add zero", 1000, 0, 1000},
		{"add negative", 1000, -200, 800},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Call("Time.Add", map[string]interface{}{
				"timestamp": tt.timestamp,
				"duration":  tt.duration,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.want {
				t.Errorf("Time.Add(%d, %d) = %v, want %d", tt.timestamp, tt.duration, result, tt.want)
			}
		})
	}
}

func TestTimeAdd_InvalidInput(t *testing.T) {
	_, err := Call("Time.Add", map[string]interface{}{
		"timestamp": "invalid",
		"duration":  100,
	})
	if err == nil {
		t.Error("expected error for invalid timestamp")
	}

	_, err = Call("Time.Add", map[string]interface{}{
		"timestamp": 1000,
		"duration":  "invalid",
	})
	if err == nil {
		t.Error("expected error for invalid duration")
	}
}

func TestTimeSubtract(t *testing.T) {
	tests := []struct {
		timestamp int64
		duration  int64
		want      int64
	}{
		{1000, 500, 500},
		{1000, 0, 1000},
		{1000, -200, 1200},
	}

	for _, tt := range tests {
		result, err := Call("Time.Subtract", map[string]interface{}{
			"timestamp": tt.timestamp,
			"duration":  tt.duration,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Time.Subtract(%d, %d) = %v, want %d", tt.timestamp, tt.duration, result, tt.want)
		}
	}
}

func TestTimeDiff(t *testing.T) {
	tests := []struct {
		a, b, want int64
	}{
		{1000, 500, 500},
		{500, 1000, -500},
		{1000, 1000, 0},
	}

	for _, tt := range tests {
		result, err := Call("Time.Diff", map[string]interface{}{
			"a": tt.a,
			"b": tt.b,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Time.Diff(%d, %d) = %v, want %d", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestTimeFormat(t *testing.T) {
	// Use a known timestamp in local time to avoid timezone issues
	localTime := time.Date(2024, 6, 15, 14, 30, 45, 0, time.Local)
	ts := localTime.UnixMilli()

	tests := []struct {
		name   string
		format string
		want   string
	}{
		{"default format", "", "2024-06-15 14:30:45"},
		{"date only", "2006-01-02", "2024-06-15"},
		{"time only", "15:04:05", "14:30:45"},
		{"custom", "Jan 2, 2006", "Jun 15, 2024"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"timestamp": ts,
			}
			if tt.format != "" {
				args["format"] = tt.format
			}

			result, err := Call("Time.Format", args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.want {
				t.Errorf("Time.Format() = %v, want %s", result, tt.want)
			}
		})
	}
}

func TestTimeFormat_InvalidTimestamp(t *testing.T) {
	_, err := Call("Time.Format", map[string]interface{}{
		"timestamp": "not a number",
	})
	if err == nil {
		t.Error("expected error for invalid timestamp")
	}
}

func TestTimeToSeconds(t *testing.T) {
	tests := []struct {
		ms   int64
		want float64
	}{
		{1000, 1.0},
		{500, 0.5},
		{1500, 1.5},
		{0, 0},
		{100, 0.1},
	}

	for _, tt := range tests {
		result, err := Call("Time.ToSeconds", map[string]interface{}{
			"ms": tt.ms,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Time.ToSeconds(%d) = %v, want %f", tt.ms, result, tt.want)
		}
	}
}

func TestTimeToSeconds_InvalidInput(t *testing.T) {
	_, err := Call("Time.ToSeconds", map[string]interface{}{
		"ms": "not a number",
	})
	if err == nil {
		t.Error("expected error for invalid ms")
	}
}

func TestTimeFromSeconds(t *testing.T) {
	tests := []struct {
		seconds float64
		want    int64
	}{
		{1.0, 1000},
		{0.5, 500},
		{1.5, 1500},
		{0, 0},
		{0.1, 100},
	}

	for _, tt := range tests {
		result, err := Call("Time.FromSeconds", map[string]interface{}{
			"seconds": tt.seconds,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Time.FromSeconds(%f) = %v, want %d", tt.seconds, result, tt.want)
		}
	}
}

func TestTimeFromSeconds_InvalidInput(t *testing.T) {
	_, err := Call("Time.FromSeconds", map[string]interface{}{
		"seconds": "not a number",
	})
	if err == nil {
		t.Error("expected error for invalid seconds")
	}
}

func TestTimeIsBefore(t *testing.T) {
	tests := []struct {
		a, b int64
		want bool
	}{
		{100, 200, true},
		{200, 100, false},
		{100, 100, false},
	}

	for _, tt := range tests {
		result, err := Call("Time.IsBefore", map[string]interface{}{
			"a": tt.a,
			"b": tt.b,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Time.IsBefore(%d, %d) = %v, want %v", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestTimeIsAfter(t *testing.T) {
	tests := []struct {
		a, b int64
		want bool
	}{
		{200, 100, true},
		{100, 200, false},
		{100, 100, false},
	}

	for _, tt := range tests {
		result, err := Call("Time.IsAfter", map[string]interface{}{
			"a": tt.a,
			"b": tt.b,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != tt.want {
			t.Errorf("Time.IsAfter(%d, %d) = %v, want %v", tt.a, tt.b, result, tt.want)
		}
	}
}

func TestTimeIsBefore_InvalidInput(t *testing.T) {
	_, err := Call("Time.IsBefore", map[string]interface{}{
		"a": "invalid",
		"b": 100,
	})
	if err == nil {
		t.Error("expected error for invalid input a")
	}

	_, err = Call("Time.IsBefore", map[string]interface{}{
		"a": 100,
		"b": "invalid",
	})
	if err == nil {
		t.Error("expected error for invalid input b")
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int64
		ok    bool
	}{
		{"int64", int64(42), 42, true},
		{"int", 42, 42, true},
		{"int32", int32(42), 42, true},
		{"float64", 42.9, 42, true}, // Truncates
		{"float32", float32(42.9), 42, true},
		{"uint", uint(42), 42, true},
		{"uint32", uint32(42), 42, true},
		{"uint64", uint64(42), 42, true},
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toInt64(tt.input)
			if ok != tt.ok {
				t.Errorf("toInt64(%v) ok = %v, want %v", tt.input, ok, tt.ok)
				return
			}
			if ok && result != tt.want {
				t.Errorf("toInt64(%v) = %d, want %d", tt.input, result, tt.want)
			}
		})
	}
}
