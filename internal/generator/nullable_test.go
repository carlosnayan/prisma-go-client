package generator

import (
	"encoding/json"
	"testing"
	"time"
)

// Test NullableString

func TestNullableString_Set(t *testing.T) {
	result := NullableString{}.Set("test value")

	if result.value == nil || *result.value != "test value" {
		t.Error("Set() should store the value")
	}
	if !result.isSet {
		t.Error("Set() should mark as set")
	}
	if result.isNull {
		t.Error("Set() should not mark as null")
	}
}

func TestNullableString_SetNull(t *testing.T) {
	result := NullableString{}.SetNull()

	if result.value != nil {
		t.Error("SetNull() should not store a value")
	}
	if !result.isSet {
		t.Error("SetNull() should mark as set")
	}
	if !result.isNull {
		t.Error("SetNull() should mark as null")
	}
}

func TestNullableString_MarshalJSON_WithValue(t *testing.T) {
	nullable := NullableString{}.Set("test")

	data, err := json.Marshal(nullable)
	if err != nil {
		t.Fatalf("MarshalJSON() failed: %v", err)
	}

	expected := `"test"`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestNullableString_MarshalJSON_Null(t *testing.T) {
	nullable := NullableString{}.SetNull()

	data, err := json.Marshal(nullable)
	if err != nil {
		t.Fatalf("MarshalJSON() failed: %v", err)
	}

	expected := `null`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// Test NullableInt

func TestNullableInt_Set(t *testing.T) {
	result := NullableInt{}.Set(42)

	if result.value == nil || *result.value != 42 {
		t.Error("Set() should store the value")
	}
	if !result.isSet {
		t.Error("Set() should mark as set")
	}
	if result.isNull {
		t.Error("Set() should not mark as null")
	}
}

func TestNullableInt_SetNull(t *testing.T) {
	result := NullableInt{}.SetNull()

	if result.value != nil {
		t.Error("SetNull() should not store a value")
	}
	if !result.isSet {
		t.Error("SetNull() should mark as set")
	}
	if !result.isNull {
		t.Error("SetNull() should mark as null")
	}
}

func TestNullableInt_MarshalJSON_WithValue(t *testing.T) {
	nullable := NullableInt{}.Set(100)

	data, err := json.Marshal(nullable)
	if err != nil {
		t.Fatalf("MarshalJSON() failed: %v", err)
	}

	expected := `100`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestNullableInt_MarshalJSON_Null(t *testing.T) {
	nullable := NullableInt{}.SetNull()

	data, err := json.Marshal(nullable)
	if err != nil {
		t.Fatalf("MarshalJSON() failed: %v", err)
	}

	expected := `null`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// Test NullableBool

func TestNullableBool_Set(t *testing.T) {
	result := NullableBool{}.Set(true)

	if result.value == nil || *result.value != true {
		t.Error("Set() should store the value")
	}
	if !result.isSet {
		t.Error("Set() should mark as set")
	}
	if result.isNull {
		t.Error("Set() should not mark as null")
	}
}

func TestNullableBool_SetNull(t *testing.T) {
	result := NullableBool{}.SetNull()

	if result.value != nil {
		t.Error("SetNull() should not store a value")
	}
	if !result.isSet {
		t.Error("SetNull() should mark as set")
	}
	if !result.isNull {
		t.Error("SetNull() should mark as null")
	}
}

func TestNullableBool_MarshalJSON(t *testing.T) {
	nullable := NullableBool{}.Set(false)

	data, err := json.Marshal(nullable)
	if err != nil {
		t.Fatalf("MarshalJSON() failed: %v", err)
	}

	expected := `false`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// Test NullableDateTime

func TestNullableDateTime_Set(t *testing.T) {
	now := time.Now()
	result := NullableDateTime{}.Set(now)

	if result.value == nil {
		t.Fatal("Set() should store the value")
	}
	if !result.value.Equal(now) {
		t.Error("Set() should store the correct time value")
	}
	if !result.isSet {
		t.Error("Set() should mark as set")
	}
	if result.isNull {
		t.Error("Set() should not mark as null")
	}
}

func TestNullableDateTime_SetNull(t *testing.T) {
	result := NullableDateTime{}.SetNull()

	if result.value != nil {
		t.Error("SetNull() should not store a value")
	}
	if !result.isSet {
		t.Error("SetNull() should mark as set")
	}
	if !result.isNull {
		t.Error("SetNull() should mark as null")
	}
}

func TestNullableDateTime_MarshalJSON_Null(t *testing.T) {
	nullable := NullableDateTime{}.SetNull()

	data, err := json.Marshal(nullable)
	if err != nil {
		t.Fatalf("MarshalJSON() failed: %v", err)
	}

	expected := `null`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// Test with struct containing nullable fields

type TestStruct struct {
	Name        NullableString `json:"name,omitempty"`
	Age         NullableInt    `json:"age,omitempty"`
	Active      NullableBool   `json:"active,omitempty"`
	Description NullableString `json:"description,omitempty"`
}

func TestNullableInStruct_AllSet(t *testing.T) {
	s := TestStruct{
		Name:        NullableString{}.Set("John"),
		Age:         NullableInt{}.Set(30),
		Active:      NullableBool{}.Set(true),
		Description: NullableString{}.Set("A description"),
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := `{"name":"John","age":30,"active":true,"description":"A description"}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestNullableInStruct_SomeNull(t *testing.T) {
	s := TestStruct{
		Name:        NullableString{}.Set("John"),
		Age:         NullableInt{}.SetNull(),
		Active:      NullableBool{}.Set(false),
		Description: NullableString{}.SetNull(),
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := `{"name":"John","age":null,"active":false,"description":null}`
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

// Define the types that match the generated code
type NullableString struct {
	value  *string
	isSet  bool
	isNull bool
}

type NullableInt struct {
	value  *int
	isSet  bool
	isNull bool
}

type NullableBool struct {
	value  *bool
	isSet  bool
	isNull bool
}

type NullableDateTime struct {
	value  *time.Time
	isSet  bool
	isNull bool
}

// Helper functions that match the generated code

func (n NullableString) Set(v string) NullableString {
	return NullableString{value: &v, isSet: true, isNull: false}
}

func (n NullableString) SetNull() NullableString {
	return NullableString{value: nil, isSet: true, isNull: true}
}

func (n NullableString) MarshalJSON() ([]byte, error) {
	if !n.isSet {
		return []byte{}, nil
	}
	if n.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(n.value)
}

func (n NullableInt) Set(v int) NullableInt {
	return NullableInt{value: &v, isSet: true, isNull: false}
}

func (n NullableInt) SetNull() NullableInt {
	return NullableInt{value: nil, isSet: true, isNull: true}
}

func (n NullableInt) MarshalJSON() ([]byte, error) {
	if !n.isSet {
		return []byte{}, nil
	}
	if n.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(n.value)
}

func (n NullableBool) Set(v bool) NullableBool {
	return NullableBool{value: &v, isSet: true, isNull: false}
}

func (n NullableBool) SetNull() NullableBool {
	return NullableBool{value: nil, isSet: true, isNull: true}
}

func (n NullableBool) MarshalJSON() ([]byte, error) {
	if !n.isSet {
		return []byte{}, nil
	}
	if n.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(n.value)
}

func (n NullableDateTime) Set(v time.Time) NullableDateTime {
	return NullableDateTime{value: &v, isSet: true, isNull: false}
}

func (n NullableDateTime) SetNull() NullableDateTime {
	return NullableDateTime{value: nil, isSet: true, isNull: true}
}

func (n NullableDateTime) MarshalJSON() ([]byte, error) {
	if !n.isSet {
		return []byte{}, nil
	}
	if n.isNull {
		return []byte("null"), nil
	}
	return json.Marshal(n.value)
}
