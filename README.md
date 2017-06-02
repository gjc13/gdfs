# Google Drive File System

## System

Ubuntu/OSX

## Required

1. Go >= 1.8.3
    
    You can find go install instruction [here](https://github.com/moovweb/gvm)

2. Make sure you can cross the GFW

## Installation

Here’s how to get going:

```shell
go get github.com/gjc13/gdfs
go get bazil.org/fuse
```

## Usage

1. You should get your **client_secret.json** by following [Step1](https://developers.google.com/drive/v3/web/quickstart/go)

2. Build
```shell
go build ./fuse/gdfs.go
```

3. Run
```shell
mv /path/to/client_secret.json ./fuse/
./fuse/gdfs /path/to/mountpoint
```

4. Have fun!