package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

const (
	peg = "peg"
)

func cmd(ctx context.Context) *cobra.Command {
	rootCmd := &cmdBuilder{
		ctx:      ctx,
		numCPUFn: runtime.NumCPU,
	}
	return rootCmd.cmd()
}

type cmdBuilder struct {
	ctx         context.Context
	force       bool
	format      string
	fps         string
	output      string
	outputIsDir bool
	numWorkers  int
	speed       float64
	stripAudio  bool

	numCPUFn func() int
}

func (c *cmdBuilder) cmd() *cobra.Command {
	rootCmd := cobra.Command{
		Use:          peg,
		Short:        "ffmpeg for the rest of us",
		Example:      "peg --format mp4 $FILE",
		RunE:         c.runE,
		Args:         cobra.MinimumNArgs(1),
		SilenceUsage: true,
	}

	rootCmd.Flags().BoolVar(&c.force, "force", false, "force files to be override existing files")
	rootCmd.Flags().StringVar(&c.format, "format", "", "convert input to desired format")
	rootCmd.Flags().StringVar(&c.fps, "fps", "", "set frames per second")
	rootCmd.Flags().BoolVar(&c.stripAudio, "no-audio", false, "remove audio from input files")
	rootCmd.Flags().StringVar(&c.output, "output", "", "file or directory to write output")
	rootCmd.Flags().IntVar(&c.numWorkers, "parallel", 1, "number of files to process concurrently; defaults to synchronous operation")
	rootCmd.Flags().Float64Var(&c.speed, "speed", 0, "adjustment of videos speed")

	rootCmd.AddCommand(completionCmd(peg))

	return &rootCmd
}

func (c *cmdBuilder) runE(cmd *cobra.Command, args []string) error {
	if c.output != "" {
		isDir, err := validateOutput(c.output)
		if err != nil {
			return err
		}
		if !isDir && len(args) > 1 {
			return errors.New("attempting to write all file matches to a single file; did you mean to provide a directory?")
		}
		c.outputIsDir = isDir
	}

	sem := make(chan struct{}, c.validNumWorkers())
	errStream := make(chan error)

	wg := new(sync.WaitGroup)
	for _, arg := range args {
		sem <- struct{}{}
		wg.Add(1)
		go func(rawInputFile string) {
			defer wg.Done()
			defer func() { <-sem }()

			errStream <- c.runFFMPEG(rawInputFile)
		}(arg)
	}

	go func() {
		wg.Wait()
		close(sem)
		close(errStream)
	}()

	return readErrStream(c.ctx, errStream)
}

func (c *cmdBuilder) runFFMPEG(rawInputFile string) error {
	inputFile := filepath.Clean(rawInputFile)

	execArgs := append(c.globalFlags(), "-i", inputFile)
	execArgs = append(execArgs, c.inputFileFlags()...)
	outputFile := c.outputFile(inputFile)
	execArgs = append(execArgs, outputFile)

	execCmd := exec.CommandContext(c.ctx, "ffmpeg", execArgs...)

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr
	return execCmd.Run()
}

func (c *cmdBuilder) globalFlags() []string {
	var flags []string
	if c.force {
		flags = append(flags, "-y")
	}
	return flags
}

func (c *cmdBuilder) inputFileFlags() []string {
	var flags []string
	if c.fps != "" {
		flags = append(flags, "-vf", "fps=fps="+c.fps)
	}
	if c.stripAudio {
		flags = append(flags, "-an")
	}
	if c.speed > 0 && c.speed != 1 {
		flags = append(flags,
			"-vf", fmt.Sprintf("setpts=%0.6f*PTS", 1/c.speed),
			"-af", audioSpeedFlagValue(c.speed),
		)
	}
	return flags
}

func (c *cmdBuilder) outputFile(inputFile string) string {
	switch {
	case c.outputIsDir:
		return filepath.Join(c.output, setFileFormat(filepath.Base(inputFile), c.format))
	case c.output != "":
		return c.output
	case c.format != "":
		return setFileFormat(inputFile, c.format)
	}
	return inputFile
}

func (c *cmdBuilder) validNumWorkers() int {
	numCPU := c.numCPUFn()
	if maxWorkers := numCPU - 1; c.numWorkers > maxWorkers {
		return maxWorkers
	}
	if c.numWorkers < 1 {
		return 1
	}
	return c.numWorkers
}

func readErrStream(ctx context.Context, errStream <-chan error) error {
	errs := make(map[string]struct{})

	toErr := func() error {
		if len(errs) == 0 {
			return nil
		}
		errSlc := make([]string, 0, len(errs))
		for errMsg := range errs {
			errSlc = append(errSlc, errMsg)
		}
		sort.Strings(errSlc)
		return errors.New(strings.Join(errSlc, "\n"))
	}

Loop:
	for {
		select {
		case <-ctx.Done():
			return toErr()
		case err, ok := <-errStream:
			if !ok {
				break Loop
			}
			if err != nil {
				errs[err.Error()] = struct{}{}
			}
		}
	}

	return toErr()
}

func setFileFormat(file, format string) string {
	if format == "" {
		return file
	}

	ext := filepath.Ext(file)
	return file[0:len(file)-len(ext)] + "." + format
}

// docs from ffmpeg itself: http://trac.ffmpeg.org/wiki/How%20to%20speed%20up%20/%20slow%20down%20a%20video
func audioSpeedFlagValue(speed float64) string {
	var output []string
	for _, v := range splitSpeed(speed) {
		output = append(output, fmt.Sprintf("atempo=%0.6f", v))
	}
	return strings.Join(output, ",")
}

func splitSpeed(speed float64) []float64 {
	switch {
	case speed > 2:
		return speedUp(speed)
	case speed < 0.5:
		return slowDown(speed)
	default:
		return []float64{speed}
	}
}

func speedUp(speed float64) []float64 {
	var result []float64
	for speed > 2 {
		speed = speed / 2
		result = append(result, 2)
	}
	result = append(result, speed)
	return result
}

func slowDown(speed float64) []float64 {
	var result []float64
	for speed < 0.5 {
		speed = speed / 0.5
		result = append(result, 0.5)
	}
	result = append(result, speed)
	return result
}

func validateOutput(output string) (bool, error) {
	f, err := os.Stat(output)
	if os.IsNotExist(err) {
		if filepath.Ext(output) == "" {
			if err := os.Mkdir(output, os.ModePerm); err != nil {
				return false, err
			}
			return true, nil
		}
		return false, nil
	}
	return f != nil && f.IsDir(), err
}
