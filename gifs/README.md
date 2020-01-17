To record a gif from a demo using [asciinema](https://asciinema.org/docs/installation) run:

```
asciinema rec -c "./DEMO_FILE.sh"
```

Then export the output using [asciicast2gif](https://github.com/asciinema/asciicast2gif):

- requires:
  - ImageMagick
  - gifsicle

```
asciicast2gif -s 5 PATH_TO_CAST PATH_TO_GIF
```