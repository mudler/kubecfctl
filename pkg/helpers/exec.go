package helpers

import (
	"bytes"
	"io"
	"os"

	"github.com/codeskyblue/kexec"
)

func RunProc(cmd, dir string) (string, error) {

	p := kexec.CommandString(cmd)

	var b bytes.Buffer
	p.Stdout = io.MultiWriter(os.Stdout, &b)
	p.Stderr = io.MultiWriter(os.Stderr, &b)
	p.Dir = dir

	if err := p.Run(); err != nil {
		return b.String(), err
	}

	p.Wait()

	return b.String(), nil
}
