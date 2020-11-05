package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
		out:      os.Stdout,
		err:      os.Stderr,
		numCPUFn: runtime.NumCPU,
	}
	return rootCmd.cmd()
}

type cmdBuilder struct {
	ctx context.Context
	out io.Writer
	err io.Writer

	crop        string
	force       bool
	format      string
	fps         string
	output      string
	outputIsDir bool
	numWorkers  int
	quiet       bool
	reverse     bool
	rotate      string
	scale       string
	showCommand bool
	speed       float64
	stripAudio  bool
	trim        string
	volume      string

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
	rootCmd.SetOut(c.out)
	rootCmd.SetErr(c.err)

	rootCmd.Flags().StringVar(&c.crop, "crop", "", "crop original media to provided dimensions")
	rootCmd.Flags().BoolVar(&c.force, "force", false, "force files to be override existing files")
	rootCmd.Flags().StringVar(&c.format, "format", "", "convert input to desired format")
	rootCmd.Flags().StringVar(&c.fps, "fps", "", "set frames per second")
	rootCmd.Flags().BoolVar(&c.stripAudio, "no-audio", false, "remove audio from input files")
	rootCmd.Flags().StringVar(&c.output, "output", "", "file or directory to write output")
	rootCmd.Flags().IntVar(&c.numWorkers, "parallel", 1, "number of files to process concurrently; defaults to synchronous operation")
	rootCmd.Flags().BoolVar(&c.quiet, "quiet", false, "trim ffmpeg output")
	rootCmd.Flags().BoolVar(&c.reverse, "reverse", false, "reverse the video and audio of media provided")
	rootCmd.Flags().StringVar(&c.rotate, "rotate", "", "rotate the video")
	rootCmd.Flags().StringVar(&c.scale, "scale", "", "scale media")
	rootCmd.Flags().BoolVar(&c.showCommand, "show-command", false, "shows the raw ffmpeg command to be run")
	rootCmd.Flags().Float64Var(&c.speed, "speed", 0, "adjustment of media speed")
	rootCmd.Flags().StringVar(&c.trim, "trim", "", "trim content of the media")
	rootCmd.Flags().StringVar(&c.volume, "volume", "", "adjustment of media volume")

	rootCmd.AddCommand(completionCmd(peg))

	return &rootCmd
}

func (c *cmdBuilder) runE(cmd *cobra.Command, args []string) error {
	cmd.SetOut(c.out)
	cmd.SetErr(c.err)

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
	execArgs = append(execArgs, c.inputFileFlags(rawInputFile)...)
	outputFile := c.outputFile(inputFile)
	execArgs = append(execArgs, outputFile)

	execCmd := exec.CommandContext(c.ctx, "ffmpeg", execArgs...)
	if c.showCommand {
		fmt.Println("ffmpeg", strings.Join(execCmd.Args, " "))
	}

	execCmd.Stdout, execCmd.Stderr = c.out, c.err

	if c.quiet {
		var stdout, stderr bytes.Buffer
		execCmd.Stdout = &stdout
		execCmd.Stderr = &stderr
	}
	return execCmd.Run()
}

func (c *cmdBuilder) globalFlags() []string {
	var flags []string
	if c.force {
		flags = append(flags, "-y")
	}
	return flags
}

func (c *cmdBuilder) inputFileFlags(inputFileExt string) []string {
	flags := append(c.commonFlags(), c.videoFlags()...)
	if inputFileExt != ".gif" && c.format != "gif" {
		flags = append(flags, c.audioFlags()...)
	}
	return flags
}

func (c *cmdBuilder) commonFlags() []string {
	cf := commonFilter{
		trim: c.trim,
	}

	var flags []string
	for _, f := range cf.flagValues() {
		flags = append(flags, f.rawFlagArgs()...)
	}
	return flags
}

func (c *cmdBuilder) audioFlags() []string {
	af := audioFilter{
		noAudio: c.stripAudio,
		reverse: c.reverse,
		speed:   c.speed,
		volume:  c.volume,
	}

	var flags []string
	for _, f := range af.flagValues() {
		flags = append(flags, f.rawFlagArgs()...)
	}
	return flags
}

func (c *cmdBuilder) videoFlags() []string {
	vf := videoFilter{
		crop:    c.crop,
		fps:     c.fps,
		reverse: c.reverse,
		rotate:  c.rotate,
		scale:   c.scale,
		speed:   c.speed,
	}

	var flags []string
	for _, f := range vf.flagValue() {
		flags = append(flags, f.rawFlagArgs()...)
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

type flagVal struct {
	name   string
	values []string
}

func (f flagVal) rawFlagArgs() []string {
	out := []string{f.name}
	if len(f.values) > 0 {
		out = append(out, strings.Join(f.values, ","))
	}
	return out
}

type commonFilter struct {
	trim string // https://superuser.com/questions/681885/how-can-i-remove-multiple-segments-from-a-video-using-ffmpeg
}

func (c commonFilter) flagValues() []flagVal {
	var flags []flagVal
	if trimParts := strings.Split(c.trim, ","); len(trimParts) == 2 {
		start, end := trimParts[0], trimParts[1]
		if start != "" {
			flags = append(flags, flagVal{
				name:   "-ss",
				values: []string{start},
			})
		}
		if end != "" {
			flags = append(flags, flagVal{
				name:   "-to",
				values: []string{end},
			})
		}
	}
	return flags
}

type audioFilter struct {
	noAudio bool    // https://walterebert.com/blog/removing-audio-from-video-with-ffmpeg/
	reverse bool    // https://ffmpeg.org/ffmpeg-filters.html#areverse
	speed   float64 // https://trac.ffmpeg.org/wiki/How%20to%20speed%20up%20/%20slow%20down%20a%20video
	volume  string  // https://trac.ffmpeg.org/wiki/AudioVolume
}

func (a audioFilter) flagValues() []flagVal {
	if a.noAudio {
		return []flagVal{{name: "-an"}}
	}

	ff := flagVal{
		name: "-af",
	}
	if a.reverse {
		ff.values = append(ff.values, "areverse")
	}
	if a.speed > 0 && a.speed != 1 {
		ff.values = append(ff.values, audioSpeedFlagValue(a.speed))
	}
	if a.volume != "" {
		ff.values = append(ff.values, fmt.Sprintf("volume=%s", a.volume))
	}

	if len(ff.values) == 0 {
		return nil
	}

	return []flagVal{ff}
}

type videoFilter struct {
	crop    string  // https://www.linuxuprising.com/2020/01/ffmpeg-how-to-crop-videos-with-examples.html
	fps     string  // https://trac.ffmpeg.org/wiki/ChangingFrameRate
	reverse bool    // https://ffmpeg.org/ffmpeg-filters.html#reverse
	rotate  string  // http://ffmpeg.org/ffmpeg-filters.html#transpose
	scale   string  // https://trac.ffmpeg.org/wiki/Scaling
	speed   float64 //https://trac.ffmpeg.org/wiki/How%20to%20speed%20up%20/%20slow%20down%20a%20video
}

func (v videoFilter) flagValue() []flagVal {
	ff := flagVal{name: "-vf"}
	if v.crop != "" {
		ff.values = append(ff.values, "crop="+v.crop)
	}
	if v.fps != "" {
		ff.values = append(ff.values, "fps=fps="+v.fps)
	}
	if v.rotate != "" {
		ff.values = append(ff.values, "transpose="+v.rotate)
	}
	if v.reverse {
		ff.values = append(ff.values, "reverse")
	}
	if v.scale != "" {
		ff.values = append(ff.values, "scale="+v.scale)
	}
	if v.speed > 0 && v.speed != 1 {
		ff.values = append(ff.values, fmt.Sprintf("setpts=%0.6f*PTS", 1/v.speed))
	}

	if len(ff.values) == 0 {
		return nil
	}

	return []flagVal{ff}
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
