#!/usr/bin/bash

oname='ting55down'

arch1='linux-amd64'
arch2='linux-arm7'
arch3='windows-amd64'

ogoos=$GOOS
ogoarch=$GOARCH

echo "构建 $arch1"
export GOOS='linux'
export GOARCH='amd64'
export GOAMD64='v2'
go build -o ${oname}-${arch1} .

echo "构建 $arch2"
export GOOS='linux'
export GOARCH='arm'
export GOARM='7'
go build -o ${oname}-${arch2} .

echo "构建 $arch3"
export GOOS='windows'
export GOARCH='amd64'
export GOAMD64='v2'
go build -o ${oname}-${arch3}.exe .

echo "恢复原有go变量"
export GOOS=$ogoos
export GOARCH=$ogoarch
