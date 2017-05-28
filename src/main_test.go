package main

import (
  "testing"
  "bytes"
)

func TestBytesReplace(t *testing.T) {
  buffer := []byte("somename=somevalue\x00")
  varname := []byte("somename=")
  replacement := []byte("test")

  expectedResult := []byte("somename=test\x00\x00\x00\x00\x00\x00")

  replaceQtPathVariable(buffer, varname, replacement)

  if bytes.Compare(buffer, expectedResult) != 0 {
    t.Fatalf("Expected %v but got %v", expectedResult, buffer)
  }
}
