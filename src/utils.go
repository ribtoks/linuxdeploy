/*
 * This file is a part of linuxdeploy - tool for
 * creating standalone applications for Linux
 *
 * Copyright (C) 2017 Taras Kushnir <kushnirTV@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the MIT License.

 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 */

package main

import (
  "os"
  "os/exec"
  "log"
  "io"
  "path"
  "errors"
  "strings"
  "bytes"
)

func executablePath() string {
  fullpath, _ := exec.LookPath(os.Args[0])
  return fullpath
}

func copyFile(src, dst string) (err error) {
  log.Printf("About to copy file %v to %v", src, dst)
  
  fi, err := os.Stat(src)
  if err != nil { return err }
  sourceMode := fi.Mode()

  in, err := os.Open(src)
  if err != nil {
    log.Printf("Failed to open source: %v", err)
    return err
  }

  defer in.Close()

  out, err := os.OpenFile(dst, os.O_RDWR | os.O_TRUNC | os.O_CREATE, sourceMode)
  if err != nil {
    log.Printf("Failed to create destination: %v", err)
    return
  }

  defer func() {
    cerr := out.Close()
    if err == nil {
      err = cerr
    }
  }()

  if _, err = io.Copy(out, in); err != nil {
    return
  }

  err = out.Sync()
  return
}

func ensureDirExists(fullpath string) (err error) {
  log.Printf("Ensure directory exists for file %v", fullpath)
  dirpath := path.Dir(fullpath)
  err = os.MkdirAll(dirpath, os.ModePerm)
  if err != nil {
    log.Printf("Failed to create directory %v", dirpath)
  }

  return err
}

func parseLddOutputLine(line string) (string, string, error) {
  if len(line) == 0 { return "", "", errors.New("Empty") }

  var libpath, libname string

  if strings.Contains(line, " => ") {
    parts := strings.Split(line, " => ")

    if len(parts) != 2 {
      return "", "", errors.New("Wrong format")
    }

    libname = strings.TrimSpace(parts[0])

    if parts[1] == "not found" { return parts[0], "", nil }

    lastUseful := strings.LastIndex(parts[1], "(0x")
    if lastUseful != -1 {
      libpath = strings.TrimSpace(parts[1][:lastUseful])
    }
  } else {
    log.Printf("Skipping ldd line: %v", line)
    return "", "", errors.New("Not with =>")
  }

  return libname, libpath, nil
}

func replaceInBuffer(buffer, key, replacement []byte) {
  index := bytes.Index(buffer, key)
  if index == -1 {
    log.Printf("Not found \"%s\" %v when replacing", key, key)
    return
  }

  nextIndex := len(key) + index
  log.Printf("Start of %s found at %v", key, nextIndex)

  endIndex := bytes.IndexByte(buffer[nextIndex:], byte(0))
  if endIndex == -1 {
    log.Printf("End not found for %s (%v) when replacing", key, key)
    return
  }

  log.Printf("Replacement End found at %v", endIndex + nextIndex)

  if endIndex < len(replacement) {
    log.Printf("Cannot exceed length when replacing %s", key)
    return
  }

  i := nextIndex
  j := 0
  replacementSize := len(replacement)
  endIndex += nextIndex

  log.Printf("Replacement previous value is %s", buffer[nextIndex:endIndex])

  for (i < endIndex) && (j < replacementSize) {
    buffer[i] = replacement[j]
    j++
    i++
  }

  // pad with zeroes
  for i < endIndex {
    buffer[i] = byte(0)
    i++
  }

  log.Printf("Replaced \"%s\" %v to \"%s\" %v", key, key, replacement, replacement)
}

func replaceVariable(buffer []byte, varname, varvalue string) {
  replaceInBuffer(buffer, []byte(varname), []byte(varvalue))
}
