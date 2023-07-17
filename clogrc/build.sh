# usage> build
# short> build & inject metadata into clog
# long>  build & inject metadata into clog
#        _
#   __  | |  ___   __ _
#  / _| | | / _ \ / _` |
#  \__| |_| \___/ \__, |
#                 |___/
# export the commit ID for the build

source clogrc/core/inc.sh

GOCODE=$(cat <<-EOM
package versionstr  //auto-generated (versionstr.go)
const build = "$(git rev-list -1 HEAD)"
const date = "$(date +%F)"
EOM
)
echo "$GOCODE" > versionstr/versionstr-build-id.go

fnInfo "Building ${cE}_win${cT}clog.exe (${cE}amd64${cT}) with metadata"
GOOS=windows     GOARCH=amd64       go build -ldflags "-X main.UseLinkerOverrides=true$cX"  -o win_mrxparse.exe

fnInfo "Building ${cW}_la${cT}clog      (${cW}arm64${cT}) with metadata"
GOOS=linux       GOARCH=arm64       go build -ldflags "-X main.UseLinkerOverrides=true$cX"  -o la_mrxparse

fnInfo "Building ${cC}_lx${cT}clog      (${cC}amd64${cT}) with metadata$cX"
GOOS=linux       GOARCH=amd64       go build -ldflags "-X main.UseLinkerOverrides=true$cX"  -o lx_mrxparse

# insert some macs here call them darwin they should still rin
fnInfo "Building ${cC}_da${cT}clog      (${cC}amd64${cT}) with metadata$cX"
GOOS=darwin      GOARCH=arm64       go build -ldflags "-X main.UseLinkerOverrides=true$cX"  -o da_mrxparse

fnInfo "Linking  ${cC}_lx${cT}clog to ${cC}./clog$cX"
rm clog
ln _lxclog clog
