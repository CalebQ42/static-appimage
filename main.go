package main

import (
	"debug/elf"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"runtime"
	"syscall"

	"github.com/CalebQ42/squashfs"
	"github.com/kardianos/osext"
)

func main() {
	executable, err := osext.Executable()
	if err != nil {
		panic(err)
	}
	fil, err := os.Open(executable)
	if err != nil {
		panic(err)
	}
	stat, _ := fil.Stat()
	elfSize, err := CalculateElfSize(fil)
	if err != nil {
		panic(err)
	}
	rdr, err := squashfs.NewReader(io.NewSectionReader(fil, elfSize, stat.Size()))
	if err != nil {
		panic(err)
	}
	mnt, err := os.MkdirTemp("", ".mount_")
	if err != nil {
		panic(err)
	}
	con, err := rdr.Mount(mnt)
	if err != nil {
		panic(err)
	}

	signals := make(chan os.Signal, 1)
	exitCode := 0
	go func() {
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		<-signals
		con.Close()
		err = os.Remove(mnt)
		if err != nil {
			panic(err)
		}

		os.Exit(exitCode)
	}()

	cmd := exec.Cmd{
		Path:   path.Join(mnt, "AppRun"),
		Args:   os.Args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	err = cmd.Run()
	if cmd.ProcessState != nil {
		if waitStatus, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			exitCode = waitStatus.ExitStatus()
			err = nil
		}
	}
	signals <- syscall.SIGTERM
	runtime.Goexit()
}

// CalculateElfSize returns the size of an ELF binary as an int64 based on the information in the ELF header
// Taken from github.com/probonopd/go-appimage/internal/helpers/elfsize.go
func CalculateElfSize(f *os.File) (len int64, err error) {

	// Open given elf file
	_, err = f.Stat()
	if err != nil {
		return
	}

	e, err := elf.NewFile(f)
	if err != nil {
		return
	}

	// Read identifier
	var ident [16]uint8
	_, err = f.ReadAt(ident[0:], 0)
	if err != nil {
		return
	}

	// Decode identifier
	if ident[0] != '\x7f' ||
		ident[1] != 'E' ||
		ident[2] != 'L' ||
		ident[3] != 'F' {
		log.Printf("Bad magic number at %d\n", ident[0:4])
		return
	}

	// Process by architecture
	sr := io.NewSectionReader(f, 0, 1<<63-1)
	var shoff, shentsize, shnum int64
	switch e.Class.String() {
	case "ELFCLASS64":
		hdr := new(elf.Header64)
		_, err = sr.Seek(0, 0)
		if err != nil {
			return
		}
		err = binary.Read(sr, e.ByteOrder, hdr)
		if err != nil {
			return
		}

		shoff = int64(hdr.Shoff)
		shnum = int64(hdr.Shnum)
		shentsize = int64(hdr.Shentsize)
	case "ELFCLASS32":
		hdr := new(elf.Header32)
		_, err = sr.Seek(0, 0)
		if err != nil {
			return
		}
		err = binary.Read(sr, e.ByteOrder, hdr)
		if err != nil {
			return
		}

		shoff = int64(hdr.Shoff)
		shnum = int64(hdr.Shnum)
		shentsize = int64(hdr.Shentsize)
	default:
		err = errors.New("unsupported elf architecture")
		return
	}
	// Calculate ELF size
	len = shoff + (shentsize * shnum)
	return
}
