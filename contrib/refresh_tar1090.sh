#/usr/bin/env bash
version=$1
if [ "$version" = "" ]; then
    echo "missing tar1090 version!"
    exit 1
fi
tmpdir=$(mktemp -d)
gitroot=$(git rev-parse --show-toplevel)
cd $tmpdir
git clone https://github.com/wiedehopf/tar1090
rm -rf $gitroot/tar1090
mkdir -p $gitroot/tar1090/
cp -va $tmpdir/tar1090/html $gitroot/tar1090/html
rm -rf $tmpdir/tar1090
rmdir $tmpdir
