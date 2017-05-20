package main

import (
  "log"
  "errors"
  "os/exec"
  "strings"
  "sync"
)

type DependencyItem struct {
  targetRelativePath string
  originalPath string
}

type AppDeployer struct {
  processedLibs map[string]bool

  libsChannel chan string
  copyChannel chan DependencyItem
  stripChannel chan string
  rpathChannel chan string
  qtChannel chan string

  additionalLibPaths []string
  destinationPath string
}

func (ad *AppDeployer) DeployApp(exePath string) {
  ad.processLibs()
}

func (ad *AppDeployer) processLibs() {
  var wg sync.WaitGroup

  wg.Add(1)
  go func() { ad.libsChannel <- exePath }()

  go func() {
    for filepath := range ad.libsChannel {
      if _, ok := ad.processedLibs[filepath]; !ok {
        dependencies, err := findLddDependencies(filepath)
        if (err == nil) {
          ad.processedLibs[filepath] = true
          //go func() { ad.copyChannel <- &{DependencyItem{originalPath: filepath} }()

          for _, dependPath := range dependencies {
            if _, ok := ad.processedLibs[dependPath]; !ok {
              wg.Add(1)
              go func() { ad.libsChannel <- dependPath }()
            }
          }
        } else {
          log.Println(err)
        }

        wg.Done()
      }
    }
  }()

  wg.Wait()
  close(ad.libsChannel)
}

func findLddDependencies(filepath string) ([]string, error) {
  log.Printf("Inspecting %v", filepath)

  out, err := exec.Command("ldd", filepath).Output()
  if err != nil { return nil, err }

  dependencies := make([]string, 10)

  output := string(out)
  lines := strings.Split(output, "\n")
  for _, line := range lines {
    line = strings.TrimSpace(line)
    libpath, err := parseLddOutputLine(line)

    if err == nil {
      log.Printf("Found dependency %v", libpath)
      dependencies = append(dependencies, libpath)
    } else {
      log.Printf("Cannot parse ldd line: %v", line)
    }
  }

  return dependencies, nil
}

func parseLddOutputLine(line string) (string, error) {
  if len(line) == 0 { return "", errors.New("Empty") }

  var libpath string

  if strings.Contains(line, " => ") {
    parts := strings.Split(line, " => ")

    if len(parts) != 2 {
      return "", errors.New("Wrong format")
    }

    shortpath := strings.TrimSpace(parts[0])

    if parts[1] == "not found" { return parts[0], nil }

    lastUseful := strings.LastIndex(parts[1], "(0x")
    if lastUseful != -1 {
      libpath = strings.TrimSpace(parts[1][:lastUseful])
      if len(libpath) == 0 { return "", errors.New("vdso") }
    } else {
      libpath = shortpath
    }
  } else {
    log.Printf("Skipping ldd line: %v", line)
  }

  return libpath, nil
}
