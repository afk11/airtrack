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
cd tar1090
git checkout "$version"
rm -rf $gitroot/resources/tar1090
mkdir -p $gitroot/resources/tar1090/
cp -va $tmpdir/tar1090/html $gitroot/resources/tar1090/html
rm -rf $tmpdir/tar1090
rmdir $tmpdir
