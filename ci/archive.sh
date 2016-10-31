#! /bin/bash -ex

pushd `dirname $0`/.. > /dev/null
root=$(pwd -P)
popd > /dev/null

export GOPATH=$root/gopath

source $root/ci/vars.sh

mkdir -p $GOPATH $GOPATH/bin $GOPATH/src $GOPATH/pkg

PATH=$PATH:$GOPATH/bin

go version

# install metalinter
go get -u github.com/alecthomas/gometalinter
gometalinter --install

go get -v github.com/venicegeo/bf-handle/...


cd $GOPATH/src/github.com/venicegeo/bf-handle

# run unit tests w/ coverage collection
go test -v -coverprofile=$root/bf-handle.cov github.com/venicegeo/bf-handle/bf

# lint
gometalinter \
--deadline=60s \
--concurrency=6 \
--vendor \
--exclude="exported (var)|(method)|(const)|(type)|(function) [A-Za-z\.0-9]* should have comment" \
--exclude="comment on exported function [A-Za-z\.0-9]* should be of the form" \
--exclude="Api.* should be .*API" \
--exclude="Http.* should be .*HTTP" \
--exclude="Id.* should be .*ID" \
--exclude="Json.* should be .*JSON" \
--exclude="Url.* should be .*URL" \
--exclude="[iI][dD] can be fmt\.Stringer" \
--exclude=" duplicate of [A-Za-z\._0-9]*" \
./... | tee $root/lint.txt
wc -l $root/lint.txt

# gather some data about the repo
src=$GOPATH/bin/bf-handle

cd $root
cp $GOPATH/bin/$APP .
tar cvzf $APP.$EXT \
    $APP \
    bf-handle.cov \
    lint.txt \
tar tzf $APP.$EXT