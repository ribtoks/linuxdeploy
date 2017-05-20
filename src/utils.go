package main

import (
  "os"
  "os/exec"
  "log"
  "io"
  "path"
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
  dirpath := path.Dir(fullpath)
  err = os.MkdirAll(dirpath, os.ModeDir)
  if err != nil {
    log.Printf("Failed to create directory %v", dirpath)
  }

  return err
}
