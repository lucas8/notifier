package fifo

import (
    "syscall"
    "os"
    "bufio"

    "github.com/lucas8/notifier/lib/config"
    "github.com/lucas8/notifier/lib/types"
)

const defaultPath = "/tmp/xcbnotif.fifo"

type Command interface {
    Validate(string) bool
    Get() types.Order
}

type fdReader int
func (fdr fdReader) Read(p []byte) (int, error) {
    n, e := syscall.Read(int(fdr), p)
    if n < 0 {
        n = 0
    }
    return n, e
}

type Fifo struct {
    path string
    fd   fdReader
    rd   *bufio.Reader
    cmds []Command
}

func Open() (*Fifo, error) {
    var pipe Fifo
    pipe.path = defaultPath
    if config.Has("global.fifo") {
        pipe.path, _ = config.String("config.fifo")
    }

    if _, err := os.Stat(pipe.path); err == nil {
        err = os.Remove(pipe.path)
        if err != nil {
            return nil, err
        }
    }

    if err := syscall.Mkfifo(pipe.path, 0777); err != nil {
        return nil, err
    }

    fd, err := syscall.Open(pipe.path, syscall.O_RDONLY | syscall.O_NONBLOCK, 0777)
    if err != nil {
        return nil, err
    }

    pipe.fd   = fdReader(fd)
    pipe.rd   = bufio.NewReader(pipe.fd)
    pipe.cmds = make([]Command, 0, 5)
    return &pipe, nil
}

func (pipe *Fifo) Close() {
    syscall.Close(int(pipe.fd))
    os.Remove(pipe.path)
}

func (pipe *Fifo) AddCmd(cmd Command) {
    pipe.cmds = append(pipe.cmds, cmd)
}

func (pipe *Fifo) ReadOrders(c chan<- types.Order) {
    for {
        line, err := pipe.rd.ReadString('\n')
        if err != nil || len(line) == 0 {
            continue
        }
        line = line[:len(line) - 1]
        for _, cmd := range pipe.cmds {
            if cmd.Validate(line) {
                c <- cmd.Get()
                break
            }
        }
    }
}

