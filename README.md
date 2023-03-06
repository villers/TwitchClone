# Twitch clone

# Run the server

```
go run cmd/srt
```

# Start straming with obs

Configure the stream to use the SRT protocol and the following settings:

srt://127.0.0.1:6000
streamid=demo:demo

# run the client

```
ffplay -fflags nobuffer "srt://localhost:6000?streamid=demo"
```
