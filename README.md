# Peg

`peg` is a CLI tool that wraps ffmpeg to provide a simple UX.
You won't need to know how to manage audio and video filters
for simple exercises.

### Examples

Working on a single video/audio input.

```shell
> peg --crop 360,640 foo.mp4   # Crop to width 360 and height 640
> peg --format gif foo.mp4     # Convert to gif
> peg --fps 20 foo.mp4         # Set frame rate to 20
> peg --no-audio foo.mp4       # Strip audio
> peg --reverse foo.mp4        # Reverse the audio/video
> peg --rotate clock foo.mp4   # Rotate 90 degrees clockwise
> peg --scale 720,-1 foo.mp4   # Scale to width 720, maintaining aspect ratio
> peg --speed 3 foo.mp4        # Triple the speed
> peg --trim 0:30,1:30 foo.mp4 # Trim from time 0:30 to 1:30
> peg --volume 2 foo.mp4       # Double the volume
```

Working with directories, all transformations for a single input can be used.

```shell
> peg --format mp4 $HOME/*.mov              # Convert all .mov videos to mp4 format
> peg --format mp4 $HOME/movies*/**/*.mov   # double star your way to paydirt
```

### Shell Completions

Usage: 

```shell
> peg completion -h

	Outputs shell completion for the given shell (bash, fish, oh-my-zsh, or zsh)
	OS X:
		$ source $(brew --prefix)/etc/bash_completion	# for bash users
		$ source <(peg completion bash)			# for bash users
		$ source <(peg completion oh-my-zsh)		# for oh-my-zsh users
		$ source <(peg completion zsh)			# for zsh users
	Ubuntu:
		$ source /etc/bash-completion	   	# for bash users
		$ source <(peg completion bash) 	# for bash users
		$ source <(peg completion oh-my-zsh) 	# for oh-my-zsh users
		$ source <(peg completion zsh)  	# for zsh users
	Additionally, you may want to add this to your .bashrc/.zshrc

Usage:
  peg completion [bash|fish|oh-my-zsh|powershell|zsh]
```

note: I haven't added instructions for the windows OS or the fish and powershell completions because its not shell's I use often.
Would be a great place for a contribution from the community :-).

### Why `peg`?

I saw [vdx](https://github.com/yuanqing/vdx) on hacker news and really like what they had done.
However, I did not want a CLI with a node dependency.
I've had a laundry list of shell funcs that wrap all the commands in vdx and then some.
I wanted to provide an interface that was easier to access.
To make it easier for users, I wanted to provide shell completions that would provide a helpful description.
I wanted to make the CLI as small as possible, with minimal dependencies.
Node was undesirable.
Node dependency is no more.
Shell completions are here.

### Inspired By

* [ffmpeg](https://ffmpeg.org/)
* [vdx](https://github.com/yuanqing/vdx)