package eval

import (
	"fmt"
	"reflect"
	"strings"
)

// WritePermission specifies who can modify a field
type WritePermission string

const (
	WriteAnyone WritePermission = ""       // Anyone can write (default)
	WriteServer WritePermission = "server" // Only server/rules can write, no player
	WriteOwner  WritePermission = "owner"  // Only the entity owner can write
)

// FieldPermission defines the permission for a specific field path
type FieldPermission struct {
	Path       string          // e.g., "Players.*.Score" or "Players.*.Hand"
	Write      WritePermission // Who can write
	OwnerField string          // For WriteOwner: which field contains the owner ID (e.g., "ID")
}

// TypePermission defines permissions for a type
type TypePermission struct {
	TypeName   string                     // e.g., "Player", "Drone"
	OwnerField string                     // Field that holds owner ID (e.g., "ID", "OwnerID")
	Fields     map[string]WritePermission // Field name -> permission
}

// PermissionSchema holds all permission definitions
type PermissionSchema struct {
	Types map[string]*TypePermission // Type name -> permissions
}

// NewPermissionSchema creates a new empty permission schema
func NewPermissionSchema() *PermissionSchema {
	return &PermissionSchema{
		Types: make(map[string]*TypePermission),
	}
}

// RegisterType registers a type with its owner field
func (ps *PermissionSchema) RegisterType(typeName, ownerField string) *TypePermission {
	tp := &TypePermission{
		TypeName:   typeName,
		OwnerField: ownerField,
		Fields:     make(map[string]WritePermission),
	}
	ps.Types[typeName] = tp
	return tp
}

// SetFieldPermission sets the write permission for a field
func (tp *TypePermission) SetFieldPermission(fieldName string, perm WritePermission) {
	tp.Fields[fieldName] = perm
}

// PermissionChecker checks write permissions
type PermissionChecker struct {
	schema   *PermissionSchema
	senderID string // ID of the player making the change (empty = server)
	isServer bool   // True if this is a server-side rule (not player-initiated)
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(schema *PermissionSchema) *PermissionChecker {
	return &PermissionChecker{
		schema:   schema,
		isServer: true, // Default to server mode
	}
}

// WithSender returns a checker configured for a specific player
func (pc *PermissionChecker) WithSender(senderID string) *PermissionChecker {
	return &PermissionChecker{
		schema:   pc.schema,
		senderID: senderID,
		isServer: senderID == "", // Empty sender = server
	}
}

// CanWrite checks if the current sender can write to the given path on the entity
func (pc *PermissionChecker) CanWrite(entity interface{}, fieldName string) error {
	if pc.schema == nil {
		return nil // No schema = no restrictions
	}

	// Get the type name
	typeName := getTypeName(entity)
	if typeName == "" {
		return nil // Unknown type = no restrictions
	}

	// Look up type permissions
	typePerm, ok := pc.schema.Types[typeName]
	if !ok {
		return nil // Type not registered = no restrictions
	}

	// Look up field permission
	perm, ok := typePerm.Fields[fieldName]
	if !ok {
		return nil // Field not registered = no restrictions (WriteAnyone)
	}

	// Check permission
	switch perm {
	case WriteAnyone:
		return nil // Anyone can write

	case WriteServer:
		if pc.isServer {
			return nil // Server can always write
		}
		return &PermissionError{
			Field:    fieldName,
			Required: WriteServer,
			SenderID: pc.senderID,
			Message:  fmt.Sprintf("field %q is server-only, player %q cannot modify", fieldName, pc.senderID),
		}

	case WriteOwner:
		if pc.isServer {
			return nil // Server can always write
		}
		// Check if sender is the owner
		ownerID, err := getOwnerID(entity, typePerm.OwnerField)
		if err != nil {
			return fmt.Errorf("cannot determine owner: %w", err)
		}
		if ownerID != pc.senderID {
			return &PermissionError{
				Field:    fieldName,
				Required: WriteOwner,
				SenderID: pc.senderID,
				OwnerID:  ownerID,
				Message:  fmt.Sprintf("field %q is owner-only, player %q is not owner %q", fieldName, pc.senderID, ownerID),
			}
		}
		return nil

	default:
		return nil
	}
}

// PermissionError is returned when a permission check fails
type PermissionError struct {
	Field    string
	Required WritePermission
	SenderID string
	OwnerID  string
	Message  string
}

func (e *PermissionError) Error() string {
	return e.Message
}

// IsPermissionError checks if an error is a permission error
func IsPermissionError(err error) bool {
	_, ok := err.(*PermissionError)
	return ok
}

// getTypeName returns the type name of an entity
func getTypeName(entity interface{}) string {
	if entity == nil {
		return ""
	}
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	return t.Name()
}

// getOwnerID extracts the owner ID from an entity
func getOwnerID(entity interface{}, ownerField string) (string, error) {
	if ownerField == "" {
		return "", fmt.Errorf("no owner field defined")
	}

	val := reflect.ValueOf(entity)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return "", fmt.Errorf("entity is not a struct")
	}

	field := val.FieldByName(ownerField)
	if !field.IsValid() {
		return "", fmt.Errorf("owner field %q not found", ownerField)
	}

	// Owner ID should be a string
	if field.Kind() == reflect.String {
		return field.String(), nil
	}

	// Try to convert to string
	if field.CanInterface() {
		if s, ok := field.Interface().(string); ok {
			return s, nil
		}
		if stringer, ok := field.Interface().(fmt.Stringer); ok {
			return stringer.String(), nil
		}
	}

	return "", fmt.Errorf("owner field %q is not a string", ownerField)
}

// parseFieldFromPath extracts the field name from a path segment
// e.g., "$.Players[0].Score" -> "Score" (for the last segment)
func parseFieldFromPath(path string) string {
	// Remove array indices and get last segment
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	// Remove any bracket notation
	if idx := strings.Index(last, "["); idx > 0 {
		last = last[:idx]
	}
	return last
}
