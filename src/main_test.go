package main

import (
  "testing"
  "bytes"
)

func TestBasicBytesReplace(t *testing.T) {
  buffer := []byte("somename=somevalue\x00")
  varname := []byte("somename=")
  replacement := []byte("test")

  expectedResult := []byte("somename=test\x00\x00\x00\x00\x00\x00")

  replaceInBuffer(buffer, varname, replacement)

  if bytes.Compare(buffer, expectedResult) != 0 {
    t.Fatalf("Expected %v but got %v", expectedResult, buffer)
  }
}

func TestNotFoundKeyBytesReplace(t *testing.T) {
  originalString := "somename=somevalue\x00"
  buffer := []byte(originalString)
  varname := []byte("somename1=")
  replacement := []byte("test")

  expectedResult := []byte(originalString)

  replaceInBuffer(buffer, varname, replacement)

  if bytes.Compare(buffer, expectedResult) != 0 {
    t.Fatalf("Expected %v but got %v", expectedResult, buffer)
  }
}

func TestNotFoundZeroEndReplace(t *testing.T) {
  originalString := "somename=somevaluexs"
  buffer := []byte(originalString)
  varname := []byte("somename=")
  replacement := []byte("test")

  expectedResult := []byte(originalString)

  replaceInBuffer(buffer, varname, replacement)

  if bytes.Compare(buffer, expectedResult) != 0 {
    t.Fatalf("Expected %v but got %v", expectedResult, buffer)
  }
}

func TestBasicPaddedReplace(t *testing.T) {
  buffer := []byte("otherStartsomename=somevalue\x00otherEnd\x00")
  varname := []byte("somename=")
  replacement := []byte("test")

  expectedResult := []byte("otherStartsomename=test\x00\x00\x00\x00\x00\x00otherEnd\x00")

  replaceInBuffer(buffer, varname, replacement)

  if bytes.Compare(buffer, expectedResult) != 0 {
    t.Fatalf("Expected %v but got %v", expectedResult, buffer)
  }
}

func TestZeroLengthValueReplace(t *testing.T) {
  originalString := "otherStartsomename=\x00otherEnd\x00"
  buffer := []byte(originalString)
  varname := []byte("somename=")
  replacement := []byte("test")

  expectedResult := []byte(originalString)

  replaceInBuffer(buffer, varname, replacement)

  if bytes.Compare(buffer, expectedResult) != 0 {
    t.Fatalf("Expected %v but got %v", expectedResult, buffer)
  }
}

func TestSmallerValueThanReplacementReplace(t *testing.T) {
  originalString := "otherStartsomename=tes\x00otherEnd\x00"
  buffer := []byte(originalString)
  varname := []byte("somename=")
  replacement := []byte("test")

  expectedResult := []byte(originalString)

  replaceInBuffer(buffer, varname, replacement)

  if bytes.Compare(buffer, expectedResult) != 0 {
    t.Fatalf("Expected %v but got %v", expectedResult, buffer)
  }
}

func TestReplaceToTheSameValue(t *testing.T) {
  originalString := "otherStart\x00somename=test\x00otherEnd\x00"
  buffer := []byte(originalString)
  varname := []byte("somename=")
  replacement := []byte("test")

  expectedResult := []byte(originalString)

  replaceInBuffer(buffer, varname, replacement)

  if bytes.Compare(buffer, expectedResult) != 0 {
    t.Fatalf("Expected %v but got %v", expectedResult, buffer)
  }
}

func TestReplaceToZeroLength(t *testing.T) {
  buffer := []byte("otherStart\x00somename=test\x00otherEnd\x00")
  varname := []byte("somename=")
  replacement := []byte("")

  expectedResult := []byte("otherStart\x00somename=\x00\x00\x00\x00\x00otherEnd\x00")

  replaceInBuffer(buffer, varname, replacement)

  if bytes.Compare(buffer, expectedResult) != 0 {
    t.Fatalf("Expected %v but got %v", expectedResult, buffer)
  }
}

func TestReplaceOnlyFirstMatch(t *testing.T) {
  originalString := "otherStart\x00somename=somevalue\x00otherEnd\x00somename=another"
  buffer := []byte(originalString)
  varname := []byte("somename=")
  replacement := []byte("test")

  expectedResult := []byte("otherStart\x00somename=test\x00\x00\x00\x00\x00\x00otherEnd\x00somename=another")

  replaceInBuffer(buffer, varname, replacement)

  if bytes.Compare(buffer, expectedResult) != 0 {
    t.Fatalf("Expected %v but got %v", expectedResult, buffer)
  }
}

func TestInSliceOfSlice(t *testing.T) {
  originalString := "otherStart\x00somename=somevalue\x00otherEnd\x00somename=another\x00"
  buffer := []byte(originalString)
  varname := []byte("somename=")
  replacement := []byte("test")

  expectedResult := []byte("otherStart\x00somename=somevalue\x00otherEnd\x00somename=test\x00\x00\x00\x00")

  replaceInBuffer(buffer[20:], varname, replacement)

  if bytes.Compare(buffer, expectedResult) != 0 {
    t.Fatalf("Expected %v but got %v", expectedResult, buffer)
  }
}
