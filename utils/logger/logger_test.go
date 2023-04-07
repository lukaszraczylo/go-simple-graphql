package logger

import "testing"

type Person struct {
	Name string
	Age  int
}

func TestSerializeStruct(t *testing.T) {
	person := Person{Name: "Alice", Age: 30}
	expected := `{"_":"logger.Person","Name":"Alice","Age":30}`
	actual, err := Serialize(person)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected: %v\nActual: %v", expected, actual)
	}
}

func TestSerializeNilPointer(t *testing.T) {
	var person *Person
	expected := `null`
	actual, err := Serialize(person)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected: %v\nActual: %v", expected, actual)
	}
}

func TestSerializeNonNilPointer(t *testing.T) {
	person := &Person{Name: "Alice", Age: 30}
	expected := `{"_":"logger.Person","Name":"Alice","Age":30}`
	actual, err := Serialize(person)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected: %v\nActual: %v", expected, actual)
	}
}

func TestSerializeSlice(t *testing.T) {
	people := []Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 40},
	}
	expected := `[{"_":"logger.Person","Name":"Alice","Age":30},{"_":"logger.Person","Name":"Bob","Age":40}]`
	actual, err := Serialize(people)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected: %v\nActual: %v", expected, actual)
	}
}

func TestSerializeSliceOfPointers(t *testing.T) {
	people := []*Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 40},
	}
	expected := `[{"_":"logger.Person","Name":"Alice","Age":30},{"_":"logger.Person","Name":"Bob","Age":40}]`

	actual, err := Serialize(people)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if actual != expected {
		t.Errorf("Expected: %v\nActual: %v", expected, actual)
	}
}

func TestSerializeUnsupportedType(t *testing.T) {
	type UnsupportedType chan int
	unsupportedType := make(UnsupportedType)
	_, err := Serialize(unsupportedType)
	if err == nil {
		t.Errorf("Expected an error, but got nil")
	}
}
