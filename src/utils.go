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
  in, err := os.Open(src)
  if err != nil {
    log.Printf("Failed to open source: %v", err)
    return
  }

  defer in.Close()

  out, err := os.Create(dst)
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

func replaceQtPathVariable(buffer []byte, varname []byte, replacement []byte) {
  index := bytes.Index(buffer, varname)
  if index == -1 {
    log.Printf("Not found %v when replacing Qt Path", varname)
    return
  }

  nextIndex := len(varname) + index
  endIndex := bytes.IndexByte(buffer[nextIndex:], byte(0))
  if endIndex == -1 {
    log.Printf("End not found for %v when replacing Qt Path", varname)
    return
  }

  if (endIndex - nextIndex) < len(replacement) {
    log.Printf("Cannot exceed length when replacing %v in Qt Path", varname)
    return
  }

  i := nextIndex
  j := 0
  replacementSize := len(replacement)

  for (i < endIndex) && (j < replacementSize) {
    buffer[i] = replacement[j]
    j++
    i++
  }

  // pad with zeroes
  for (i < endIndex) {
    buffer[i] = byte(0)
  }

  log.Printf("Replaced %v to %v for Qt Path", varname, replacement)
}
