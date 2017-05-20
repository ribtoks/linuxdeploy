package main

import (
  "os"
  "os/exec"
  "log"
  "io"
  "path"
  "errors"
  "strings"
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
  log.Printf(fullpath)
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
