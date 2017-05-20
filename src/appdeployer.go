package main

import (
  "log"
  "errors"
  "os"
  "os/exec"
  "strings"
  "sync"
  "path/filepath"
)

type CopyRequest struct {
  relativeTarget string
  originPath string
}

type AppDeployer struct {
  waitGroup sync.WaitGroup
  processedLibs map[string]bool

  libsChannel chan string
  copyChannel chan string
  stripChannel chan string
  rpathChannel chan string
  qtChannel chan string

  additionalLibPaths []string
  destinationPath string
  targetExePath string
}

func (ad *AppDeployer) DeployApp(exePath string) {
  ad.waitGroup.Add(1)
  go func() { ad.libsChannel <- exePath }()

  ensureDirExists(filepath.Join(ad.destinationPath, "lib"))
  go ad.processCopyRequests()
  go ad.processLibs()

  ad.waitGroup.Wait()
  close(ad.libsChannel)
  close(ad.copyChannel)
}

func (ad *AppDeployer) processLibs() {
  for filepath := range ad.libsChannel {
    if _, ok := ad.processedLibs[filepath]; !ok {
      dependencies, err := ad.findLddDependencies(filepath)
      if (err == nil) {
        ad.processedLibs[filepath] = true

        ad.waitGroup.Add(1)
        go func() { ad.copyChannel <- filepath }()

        for _, dependPath := range dependencies {
          if _, ok := ad.processedLibs[dependPath]; !ok {
            ad.waitGroup.Add(1)
            go func() { ad.libsChannel <- dependPath }()
          }
        }
      } else {
        log.Println(err)
      }
    }

    ad.waitGroup.Done()
  }
}

func (ad *AppDeployer) processCopyRequests() {
  for fileToCopy := range ad.copyChannel {
    destination := filepath.Join(ad.destinationPath, filepath.Base(fileToCopy))
    log.Printf("Copying %v to %v", fileToCopy, destination)
    copyFile(fileToCopy, destination)

    ad.waitGroup.Done()
  }
}

func (ad *AppDeployer) findLddDependencies(filepath string) ([]string, error) {
  log.Printf("Inspecting %v", filepath)

  out, err := exec.Command("ldd", filepath).Output()
  if err != nil { return nil, err }

  dependencies := make([]string, 0, 10)

  output := string(out)
  lines := strings.Split(output, "\n")
  for _, line := range lines {
    line = strings.TrimSpace(line)
    libname, libpath, err := parseLddOutputLine(line)

    if err == nil {
      if len(libpath) == 0 {
        libpath = ad.resolveLibrary(libname)
      }

      log.Printf("Found dependency %v for line [%v]", libpath, line)
      dependencies = append(dependencies, libpath)
    } else {
      log.Printf("Cannot parse ldd line: %v", line)
    }
  }

  return dependencies, nil
}

func (ad *AppDeployer) addAdditionalLibPath(libpath string) {
  log.Printf("Adding addition libpath: %v", libpath)
  foundPath := libpath
  var err error

  if !filepath.IsAbs(foundPath) {
    if foundPath, err = filepath.Abs(foundPath); err == nil {
      log.Printf("Trying to resolve libpath to: %v", foundPath)

      if _, err = os.Stat(foundPath); os.IsNotExist(err) {
        exeDir := filepath.Dir(ad.targetExePath)
        foundPath = filepath.Join(exeDir, libpath)
        log.Printf("Trying to resolve libpath to: %v", foundPath)
      }
    }
  }

  if _, err := os.Stat(foundPath); os.IsNotExist(err) {
    log.Printf("Cannot find library path: %v", foundPath)
    return
  }

  log.Printf("Resolved additional libpath to: %v", foundPath)
  ad.additionalLibPaths = append(ad.additionalLibPaths, foundPath)
}

func (ad *AppDeployer) resolveLibrary(libname string) (foundPath string) {
  foundPath = libname

  for _, extraLibPath := range ad.additionalLibPaths {
    possiblePath := filepath.Join(extraLibPath, libname)

    if _, err := os.Stat(possiblePath); err == nil {
      foundPath = possiblePath
      break
    }
  }

  log.Printf("Resolving library %v to %v", libname, foundPath)
  return foundPath
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
