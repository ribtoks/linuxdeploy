package main

import (
  "log"
  "os/exec"
  "strings"
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
  go func() { ad.libsChannel <- exePath }()
  ad.processLibs()
}

func (ad *AppDeployer) processLibs() {
  for filepath := range ad.libsChannel {
    if _, ok := ad.processedLibs[filepath]; !ok {
      dependencies, err := findLddDependencies(filepath)
      if (err != nil) {
        log.Println(err)
        continue
      }

      ad.processedLibs[filepath] = true
      //go func() { ad.copyChannel <- &{DependencyItem{originalPath: filepath} }()

      for _, dependPath := range dependencies {
        if _, ok := ad.processedLibs[dependPath]; !ok {
          go func() { ad.libsChannel <- dependPath }()
        }
      }
    }
  }
}

func findLddDependencies(filepath string) ([]string, error) {
  log.Printf("Inspecting %v", filepath)

  out, err := exec.Command("ldd", filepath).Output()
  if err != nil { return nil, err }

  dependencies := make([]string, 10)

  output := string(out)
  lines := strings.Split(output, "\n")
  for _, line := range lines {
    if len(line) == 0 { continue }

    var libpath string

    if strings.Contains(line, " => ") {
      parts := strings.Split(line, " => ")

      if len(parts) != 2 {
        log.Printf("Unexpected ldd output: %v", line)
        continue
      }

      shortpath := strings.TrimSpace(parts[0])

      lastUseful := strings.LastIndex(parts[1], "(0x")
      if lastUseful != -1 {
        libpath = strings.TrimSpace(parts[1][:lastUseful])
      } else {
        log.Printf("Cannot find libpath in line %v", line)
        libpath = shortpath
      }
    } else if strings.Contains(line, "not found") {
      trimmed := strings.TrimSpace(line)
      parts := strings.Split(trimmed, " ")

      if len(parts) > 0 {
        libpath = parts[0]
      } else {
        log.Printf("Unexpected ldd output: %v", line)
        continue
      }
    } else {
      log.Printf("Skipping ldd line: %v", line)
      continue
    }

    log.Printf("Found dependency %v", libpath)
    dependencies = append(dependencies, libpath)
  }

  return dependencies, nil
}
