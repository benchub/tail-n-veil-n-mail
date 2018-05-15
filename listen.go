package main

import (
  "fmt"
  "log"
  "os"
  "time"
  "strings"
  "syscall"
)

import (
  "github.com/lib/pq"
  "database/sql"
)


func watchForConfigChanges(db *sql.DB, conninfo string, args []string) {
    reportProblem := func(ev pq.ListenerEventType, err error) {
        if err != nil {
            log.Println(err.Error())
        }
    }

    listener := pq.NewListener(conninfo, 10 * time.Second, time.Minute, reportProblem)
    err := listener.Listen("configChange")
    if err != nil {
        panic(err)
    }

    for {
        select {
            case <-listener.Notify:
		restartMe(args)
            case <-time.After(90 * time.Second):
                err = listener.Ping()
                if err != nil {
                  log.Println("server seems dead?", err)
                  os.Exit(3)
                }
        }
    }
}

func restartMe(args []string) {
  var newArgs []string
  var binary string

  for _, s := range args {
    if !strings.HasPrefix(s, "-warp=") {
      newArgs = append(newArgs,s)
    }
  }

  newArgs = append(newArgs, fmt.Sprintf("-warp=%d",warpTo.Offset))
  binary = fmt.Sprintf("%s/%s", os.Getenv("PWD"), args[0])

  log.Println("received notification of a config change at ", warpTo.Offset)

  execErr := syscall.Exec(binary, newArgs, os.Environ())
  if execErr != nil {
    log.Println("couldn't exec!", execErr)
    os.Exit(3)    
  }
}
