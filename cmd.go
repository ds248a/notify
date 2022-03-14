package notify

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"strconv"
	"strings"
)

var defaultDelayToKill = 1000

type Cmd struct {
	Terms       []string
	DelayToKill int
}

//
func Run(ctx context.Context, cmd Cmd, shouldLog bool) error {
	if len(cmd.Terms) == 0 {
		return fmt.Errorf("Empty Cmd params")
	}

	cmdCtx, killCmd := context.WithCancel(ctx)
	defer killCmd()

	cmdDone := make(chan error)

	// запуск комманды
	cmdExec := exec.CommandContext(cmdCtx, cmd.Terms[0], cmd.Terms[1:]...)

	// out
	outPipe, err := cmdExec.StdoutPipe()
	if err != nil {
		return err
	}
	go logCmdStd(cmdCtx, CmdOut, outPipe)

	// err
	errPipe, err := cmdExec.StderrPipe()
	if err != nil {
		return err
	}
	go logCmdStd(cmdCtx, CmdErr, errPipe)

	// start
	err = cmdExec.Start()
	if err != nil {
		return err
	}

	// wait
	go func() {
		cmdDone <- cmdExec.Wait()
		close(cmdDone)
	}()

	timer := time.NewTimer(time.Duration(int(time.Millisecond) * cmd.DelayToKill))

	select {
	case <-cmdCtx.Done():
		return cmdCtx.Err()
	case <-timer.C:
		killCmd()
		if err, ok := <-cmdDone; ok && err != nil {
			return err
		}
	case err := <-cmdDone:
		return err
	}

	return nil
}

//
func Run2(ctx context.Context, cmd Cmd, shouldLog bool) error {
	if len(cmd.Terms) == 0 {
		return fmt.Errorf("Empty Cmd params")
	}

	if cmd.DelayToKill == 0 {
		cmd.DelayToKill = defaultDelayToKill
	}

	cmdCtx, killCmd := context.WithCancel(context.Background())
	defer killCmd()
	cmdDone := make(chan error)

	// пуск комманды
	cmdExec := exec.CommandContext(cmdCtx, cmd.Terms[0], cmd.Terms[1:]...)

	if shouldLog {
		outPipe, err := cmdExec.StdoutPipe()
		if err != nil {
			return err
		}
		go logCmdStd(cmdCtx, CmdOut, outPipe)

		errPipe, err := cmdExec.StderrPipe()
		if err != nil {
			return err
		}
		go logCmdStd(cmdCtx, CmdErr, errPipe)
	}

	err := cmdExec.Start()
	if err != nil {
		return err
	}

	go func() {
		cmdDone <- cmdExec.Wait()
		close(cmdDone)
	}()

	select {
	case <-ctx.Done():
		cmdExec.Process.Signal(os.Interrupt)

		timer := time.NewTimer(time.Duration(int(time.Millisecond) * cmd.DelayToKill))

		select {
		case <-timer.C:
			killCmd()

			if err = <-cmdDone; err != nil {
				return err
			}
		case err = <-cmdDone:
			return err
		}
	case err := <-cmdDone:
		return err
	}

	return nil
}

//
func logCmdStd(ctx context.Context, l *log.Logger, std io.Reader) {
	bs := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := std.Read(bs)
		if err != nil {
			return
		}

		nBs := bs[0:n]

		if nBs[len(nBs)-1] == '\n' {
			l.Print(string(nBs))
		} else {
			l.Println(string(bs))
		}
	}
}

// ------------------------
//   Log
// ------------------------

// CmdOut is the logger to print the data from the command's Stdout.
var CmdOut = log.New(os.Stdout, Format("CMD: ", AttrBold, AttrFgColor4BitsGreen), 0)

// CmdErr is the logger used to print errors from the command's Stderr.
var CmdErr = log.New(os.Stdout, Format("CMD: ", AttrBold, AttrFgColor4BitsRed), 0)

// Err is the logger used to print errors.
var Err = log.New(os.Stderr, Format("ERR: ", AttrBold, AttrFgColor4BitsRed), 0)

// Evt is the logger used to print events.
var Evt = log.New(os.Stdout, Format("EVT: ", AttrBold, AttrFgColor4BitsCyan), 0)

// ------------------------
//   Color
// ------------------------

type Attr interface {
	ANSI() string
}

type SimpleAttr int

func (s SimpleAttr) ANSI() string {
	return strconv.Itoa(int(s))
}

const (
	AttrBold                      = SimpleAttr(1)
	AttrItalic                    = SimpleAttr(3)
	AttrUnderline                 = SimpleAttr(4)
	AttrBlink                     = SimpleAttr(5)
	AttrStrikethrough             = SimpleAttr(9)
	AttrFgColor4BitsBlack         = SimpleAttr(30)
	AttrFgColor4BitsRed           = SimpleAttr(31)
	AttrFgColor4BitsGreen         = SimpleAttr(32)
	AttrFgColor4BitsYellow        = SimpleAttr(33)
	AttrFgColor4BitsBlue          = SimpleAttr(34)
	AttrFgColor4BitsMagenta       = SimpleAttr(35)
	AttrFgColor4BitsCyan          = SimpleAttr(36)
	AttrFgColor4BitsWhite         = SimpleAttr(37)
	AttrBgColor4BitsBlack         = SimpleAttr(40)
	AttrBgColor4BitsRed           = SimpleAttr(41)
	AttrBgColor4BitsGreen         = SimpleAttr(42)
	AttrBgColor4BitsYellow        = SimpleAttr(43)
	AttrBgColor4BitsBlue          = SimpleAttr(44)
	AttrBgColor4BitsMagenta       = SimpleAttr(45)
	AttrBgColor4BitsCyan          = SimpleAttr(46)
	AttrBgColor4BitsWhite         = SimpleAttr(47)
	AttrOverline                  = SimpleAttr(53)
	AttrFgColor4BitsBrightBlack   = SimpleAttr(90)
	AttrFgColor4BitsBrightRed     = SimpleAttr(91)
	AttrFgColor4BitsBrightGreen   = SimpleAttr(92)
	AttrFgColor4BitsBrightYellow  = SimpleAttr(93)
	AttrFgColor4BitsBrightBlue    = SimpleAttr(94)
	AttrFgColor4BitsBrightMagenta = SimpleAttr(95)
	AttrFgColor4BitsBrightCyan    = SimpleAttr(96)
	AttrFgColor4BitsBrightWhite   = SimpleAttr(97)
	AttrBgColor4BitsBrightBlack   = SimpleAttr(100)
	AttrBgColor4BitsBrightRed     = SimpleAttr(101)
	AttrBgColor4BitsBrightGreen   = SimpleAttr(102)
	AttrBgColor4BitsBrightYellow  = SimpleAttr(103)
	AttrBgColor4BitsBrightBlue    = SimpleAttr(104)
	AttrBgColor4BitsBrightMagenta = SimpleAttr(105)
	AttrBgColor4BitsBrightCyan    = SimpleAttr(106)
	AttrBgColor4BitsBrightWhite   = SimpleAttr(107)
)

//
func Format(str string, attrs ...Attr) string {
	var b strings.Builder

	if len(attrs) == 0 {
		return str
	}

	writeAttrs(&b, attrs)
	b.WriteString(str)
	b.WriteString("\x1b[0m")

	return b.String()
}

//
func writeAttrs(b *strings.Builder, attrs []Attr) {
	b.WriteString("\x1b[")

	for i, attr := range attrs {
		b.WriteString(attr.ANSI())

		if i != len(attrs)-1 {
			b.WriteString(";")
		}
	}

	b.WriteString("m")
}

// for tests
func escapeStr(str string) string {
	return strings.ReplaceAll(str, "\x1b", "\\x1b")
}
