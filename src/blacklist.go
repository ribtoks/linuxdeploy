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
  "log"
  "bufio"
  "os"
  "strings"
  "path/filepath"
)

func DefaultBlacklist() []string {
  blacklist := []string {
    "libcom_err.so",
    "libcrypt.so",
    "libdl.so",
    "libexpat.so",
    "libfontconfig.so",
    "libgcc_s.so",
    "libglib-2.0.so",
    "libgpg-error.so",
    "libgssapi_krb5.so",
    "libgssapi.so",
    "libhcrypto.so",
    "libheimbase.so",
    "libheimntlm.so",
    "libhx509.so",
    "libICE.so",
    "libidn.so",
    "libk5crypto.so",
    "libkeyutils.so",
    "libkrb5.so",
    "libkrb5.so",
    "libkrb5support.so",
    // "liblber-2.4.so.2",  # needed for debian wheezy
    // "libldap_r-2.4.so.2",  # needed for debian wheezy
    "libm.so",
    "libp11-kit.so",
    "libpcre.so",
    "libpthread.so",
    "libresolv.so",
    "libroken.so",
    "librt.so",
    "libsasl2.so",
    "libSM.so",
    "libusb-1.0.so",
    "libuuid.so",
    "libwind.so",
    "libz.so",

    //Delete potentially dangerous libraries
    "libstdc",
    "libgobject",
    "libc.so",

    "libdbus-1.so",

    // Fix the "libGL error" messages
    "libGL.so",
    "libdrm.so",
  }

  return blacklist
}

func generateLibsBlacklist() []string {
  blacklist, err := parseBlacklistFile(*blacklistFileFlag)
  if err != nil { log.Printf("Error while parsing blacklist: %v", err) }

  if *defaultBlackListFlag {
    defaultBlacklist := DefaultBlacklist()
    blacklist = append(blacklist, defaultBlacklist...)
  }

  return blacklist
}

func parseBlacklistFile(filepath string) ([]string, error) {
  log.Printf("Parsing blacklist file %v", filepath)

  file, err := os.Open(filepath)
  if err != nil { return nil, err }

  defer file.Close()

  blacklist := make([]string, 0, 10)

  scanner := bufio.NewScanner(file)
  for scanner.Scan() {
    item := strings.TrimSpace(scanner.Text())

    if strings.HasPrefix(item, "#") { continue }
    blacklist = append(blacklist, strings.ToLower(item))
  }

  log.Printf("Parsed %v blacklisted libraries", len(blacklist))

  // check for errors
  if err = scanner.Err(); err != nil {
    return blacklist, err
  }

  return blacklist, nil
}

func cleanupBlacklistedLibs(libdirpath string, blacklist []string) error {
  if len(blacklist) == 0 {
    log.Printf("No libraries blacklisted")
    return nil
  }

  log.Println("Removing blacklisted libraries...")

  err := filepath.Walk(libdirpath, func(path string, info os.FileInfo, err error) error {
    if err != nil {
      return err
    }

    if !info.Mode().IsRegular() {
      return nil
    }

    basename := strings.ToLower(filepath.Base(path))

    for _, blackLib := range blacklist {
      if strings.HasPrefix(basename, blackLib) {
        log.Printf("Removing blacklisted library [%v] with match on [%v]", path, blackLib)
        os.Remove(path)
        break
      }
    }

    return nil
  })

  return err
}
