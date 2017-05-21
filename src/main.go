package main

import (
  "fmt"
  "log"
  "flag"
  "os"
  "os/exec"
  "io"
  "errors"
  "path/filepath"
)

type stringsParam []string

func (s *stringsParam) String() string {
  return fmt.Sprintf("%s", *s)
}

func (s *stringsParam) Set(value string) error {
  *s = append(*s, value)
  return nil
}

var (
  qmlImports stringsParam
  librariesDirs stringsParam
  currentExeFullPath string
)

// flags
var (
  outTypeFlag = flag.String("out", "appimage", "Type of the generated output")
  blacklistFileFlag = flag.String("blacklist", "libs.blacklist", "Path to the libraries blacklist file")
  logPathFlag = flag.String("log", "linuxdeploy.log", "Path to the logfile")
  stdoutFlag = flag.Bool("stdout", false, "Log to stdout and to logfile")
  verboseFlag = flag.Bool("verbose", true, "Verbose logging")
  exePathFlag = flag.String("exe", "", "Path to the executable")
  appDirPathFlag = flag.String("appdir", "", "Path to the AppDir (if 'type' is appimage)")
  overwriteFlag = flag.Bool("overwrite", false, "Overwrite output if preset")
  qmakePathFlag = flag.String("qmake", "", "Path to qmake")
)

const (
  appName = "linuxdeploy"
)

func init() {
  flag.Var(&qmlImports, "qmldir", "QML imports dir")
  flag.Var(&librariesDirs, "libs", "Additional libraries search paths")
}

func main() {
  err := parseFlags()
  if err != nil {
    flag.PrintDefaults()
    log.Fatal(err.Error())
  }

  logfile, err := setupLogging()
  if err != nil {
    defer logfile.Close()
  }

  currentExeFullPath = executablePath()
  log.Println("Current exe path is", currentExeFullPath)

  appDirPath := resolveAppDir()
  os.RemoveAll(appDirPath)
  os.MkdirAll(appDirPath, os.ModePerm)
  log.Printf("Created directory %v", appDirPath)

  appDeployer := &AppDeployer{
    processedLibs: make(map[string]bool),
    libsChannel: make(chan DeployRequest),
    copyChannel: make(chan DeployRequest),
    rpathChannel: make(chan string),
    qtChannel: make(chan DeployRequest),

    qtDeployer: &QtDeployer{
      qmakePath: resolveQMake(),
      qmakeVars: make(map[string]string),
      qtEnv: make(map[QMakeKey]string),
      qmlImportDirs: qmlImports,
    },

    additionalLibPaths: make([]string, 0, 10),
    destinationPath: appDirPath,
    targetExePath: resolveTargetExe(),
  }

  for _, libpath := range librariesDirs {
    appDeployer.addAdditionalLibPath(libpath)
  }

  appDeployer.DeployApp()
}

func parseFlags() error {
  flag.Parse()

  _, err := os.Stat(*exePathFlag)
  if os.IsNotExist(err) { return err }

  if len(*outTypeFlag) > 0 && (*outTypeFlag != "appimage") { return errors.New(appName + " only supports appimage type at this time") }

  log.Printf("AppDir is %v", *appDirPathFlag)

  appDirInfo, err := os.Stat(*appDirPathFlag)
  if err == nil && appDirInfo.IsDir() {
    if !(*overwriteFlag) {
      return errors.New("AppDir already exists. Please set overwrite flag to overwrite it")
    }
  }

  return nil
}

func setupLogging() (f *os.File, err error) {
  f, err = os.OpenFile(*logPathFlag, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
  if err != nil {
    fmt.Println("error opening file: %v", *logPathFlag)
    return nil, err
  }

  if *stdoutFlag {
    mw := io.MultiWriter(os.Stdout, f)
    log.SetOutput(mw)
  } else {
    log.SetOutput(f)
  }

  log.Println("------------------------------")
  log.Println(appName + " log started")

  return f, err
}

func resolveAppDir() string {
  foundPath := *appDirPathFlag
  var err error

  if !filepath.IsAbs(foundPath) {
    if foundPath, err = filepath.Abs(foundPath); err != nil {
      foundPath = *appDirPathFlag
    }
  }

  return foundPath
}

func resolveTargetExe() string {
  foundPath := *exePathFlag
  var err error

  if !filepath.IsAbs(foundPath) {
    if foundPath, err = filepath.Abs(foundPath); err != nil {
      foundPath = *exePathFlag
    }
  }

  return foundPath
}

func resolveQMake() string {
  var err error
  currentPath := *qmakePathFlag
  if len(currentPath) == 0 { currentPath = "qmake" }

  if _, err = os.Stat(currentPath); os.IsNotExist(err) {
    if currentPath, err = exec.LookPath("qmake"); err != nil {
      if currentPath, err = exec.LookPath("qmake-qt5"); err != nil {
        if currentPath, err = exec.LookPath("qmake-qt4"); err != nil {
          return ""
        }
      }
    }
  }

  return currentPath
}
