# usage> build
# short> build & inject mrx-tool
# long>  build & inject mrx-tool
#        _
#   __  | |  ___   __ _
#  / _| | | / _ \ / _` |
#  \__| |_| \___/ \__, |
#                 |___/
# export the commit ID for the build

source clogrc/core/inc.sh

# generate the version information for the compiler to use
GOCODE=$(
	cat <<-EOM
		package versionstr  //auto-generated (versionstr.go)
		const build = "$(git rev-list -1 HEAD)"
		const date = "$(date +%F)"
	EOM
)
echo "$GOCODE" >versionstr/versionstr-build-id.go

fnInfo "Building ${cE}_win${cT}mrxtool.exe (${cE}amd64${cT}) with version metadata"
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.UseLinkerOverrides=true$cX" -o win_mrxtool.exe

fnInfo "Building ${cW}_la${cT}mrxtool      (${cW}arm64${cT}) with version metadata"
GOOS=linux GOARCH=arm64 go build -ldflags "-X main.UseLinkerOverrides=true$cX" -o la_mrxtool

fnInfo "Building ${cC}_lx${cT}mrxtool      (${cC}amd64${cT}) with version metadata$cX"
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.UseLinkerOverrides=true$cX" -o lx_mrxtool

fnInfo "Building ${Cgreen}_da${cT}mrxtool      (${Cgreen}arm64${cT}) with version metadata$cX"
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.UseLinkerOverrides=true$cX" -o da_mrxtool
